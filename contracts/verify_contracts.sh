#!/usr/bin/env bash
# Deterministic contract rebuild + bytecode comparison for CI (L-11, P3-14, P3-15).
#
# Usage:
#   ./verify_contracts.sh          # compile and compare against committed refs
#                                  # AND against the committed compiled_contracts/*.json
#   ./verify_contracts.sh write    # regenerate the committed reference hashes
#
# The build is pinned to solc 0.8.28 and @openzeppelin/contracts 4.9.6 (see
# package.json / package-lock.json). Contracts are compiled WITHOUT the
# optimizer (matching the committed artifacts) and with DEFAULT metadata-hash
# (ipfs). Audit P3-14: the previous --metadata-hash none setting produced
# bytecode that could never be tied to the committed compiled_contracts/*.json
# (different metadata plumbing), so the gate verified the rebuild against refs/
# but never against the artifacts that actually ship (the JSON embedded by
# x/erc721). The metadata ipfs hash is a function of sources + settings only,
# so the build stays reproducible across environments; this was verified by a
# full byte-for-byte rebuild comparison.
#
# Audit P3-15: an abstract contract produces no bytecode. That case is asserted
# explicitly (empty committed bin, empty refs hash) instead of silently
# "passing" an empty-vs-empty comparison.
#
# Audit O-05: compiled_contracts/*.json with no matching root .sol is a
# violation — such an artifact can be neither rebuilt nor reviewed, so it must
# not exist. The check at the bottom makes that an explicit gate instead of an
# implicit gap in the `for *.sol` loop (which is why the old orphan
# compiled_contracts/ERC20Burnable.json could sit unreferenced for so long).
set -euo pipefail
cd "$(dirname "$0")"

SOLC_VERSION="0.8.28"
TMP_DO="$(mktemp -d)"
trap 'rm -rf "$TMP_DO"' EXIT

# Hashing helper (Linux sha256sum / macOS shasum).
hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

# Hex-string equality between a rebuilt artifact and a field of the committed
# compiled_contracts/<name>.json. Args: 1=rebuilt file, 2=json file,
# 3=json field ("bin" or "bin-runtime"), 4=contract name.
check_against_committed() {
  local rebuilt_file="$1" json_file="$2" field="$3" name="$4"
  python3 - "$rebuilt_file" "$json_file" "$field" "$name" <<'PY' || return 1
import json, sys
rebuilt = open(sys.argv[1]).read().strip().lower()
committed = json.load(open(sys.argv[2])).get(sys.argv[3], "")
name, field = sys.argv[4], sys.argv[3]
if rebuilt == committed.lower():
    print(f"ok: {name}.{field} matches committed {sys.argv[2]}")
else:
    print(f"MISMATCH: {name}.{field} differs from committed {sys.argv[2]} "
          f"(rebuilt {len(rebuilt)} hex chars, committed {len(committed)})",
          file=sys.stderr)
    sys.exit(1)
PY
}

# Verify the toolchain matches the pinned versions.
actual_solc="$(solc --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)"
if [ "$actual_solc" != "$SOLC_VERSION" ]; then
  echo "error: solc $SOLC_VERSION required, found $actual_solc" >&2
  exit 1
fi
if [ ! -d node_modules/@openzeppelin/contracts ]; then
  echo "error: @openzeppelin/contracts not installed; run 'npm ci' in contracts/" >&2
  exit 1
fi

refs_dir="refs"
write_mode="${1:-}"
if [ "$write_mode" = "write" ]; then
  mkdir -p "$refs_dir"
fi

status=0
for sol in *.sol; do
  name="${sol%.sol}"
  solc --bin --bin-runtime \
    --base-path . --include-path node_modules --overwrite -o "$TMP_DO" "$sol" >/dev/null 2>&1 || true

  json_file="compiled_contracts/${name}.json"

  for kind in bin bin-runtime; do
    artifact="$TMP_DO/${name}.${kind}"
    ref_file="${refs_dir}/${name}.${kind}.sha256"

    # Audit P3-15: an abstract contract yields an EMPTY bytecode file (or none
    # at all). Assert that case explicitly instead of letting an empty-vs-empty
    # comparison silently "pass".
    if [ ! -s "$artifact" ]; then
      committed_bin="$(python3 -c "import json,sys; print(json.load(open(sys.argv[1])).get('bin',''))" "$json_file" 2>/dev/null || echo UNKNOWN)"
      if [ "$committed_bin" = "" ]; then
        echo "note: $name is abstract (empty solc bytecode); committed bin is empty as expected"
      else
        echo "error: $name.$kind rebuilt empty/missing but committed bin is NOT empty" >&2
        status=1
      fi
      continue
    fi

    actual="$(hash_file "$artifact")"

    if [ "$write_mode" = "write" ]; then
      echo "$actual" > "$ref_file"
      echo "wrote $ref_file ($actual)"
    else
      if [ ! -f "$ref_file" ]; then
        echo "error: missing reference $ref_file (run 'verify_contracts.sh write')" >&2
        status=1
      else
        expected="$(cat "$ref_file")"
        if [ "$actual" != "$expected" ]; then
          echo "MISMATCH: $name.$kind" >&2
          echo "  expected (committed refs): $expected" >&2
          echo "  actual   (rebuilt):        $actual" >&2
          status=1
        else
          echo "ok: $name.$kind $actual"
        fi
      fi
    fi

    # Audit P3-14: the rebuilt bytecode must ALSO match the committed
    # compiled_contracts/<name>.json — the artifact that actually ships
    # (embedded by x/erc721). Without this step a tampered or stale committed
    # JSON could pass while refs/ verified a rebuild of the pure sources.
    if [ "$write_mode" != "write" ] && [ -f "$json_file" ]; then
      if ! check_against_committed "$artifact" "$json_file" "$kind" "$name"; then
        status=1
      fi
    fi
  done
done

# Audit O-05: every committed artifact must be traceable to a root .sol source.
# The loop above only walks `*.sol`, so a stale artifact left behind by a
# removed or renamed contract would never be looked at (and, because it is
# outside the loop, never compared against anything either).
for json_file in compiled_contracts/*.json; do
  [ -e "$json_file" ] || continue
  base="$(basename "$json_file" .json)"
  if [ ! -f "${base}.sol" ]; then
    echo "error: $json_file has no matching ${base}.sol source; delete the orphan" >&2
    status=1
  fi
done

# In write mode also regenerate checksums.txt so the artifact integrity list can
# never drift from the artifacts it describes.
if [ "$write_mode" = "write" ]; then
  : > checksums.txt
  for json_file in compiled_contracts/*.json; do
    [ -e "$json_file" ] || continue
    echo "$(hash_file "$json_file")  $json_file" >> checksums.txt
    echo "wrote checksums.txt entry for $json_file"
  done
fi

exit "$status"
