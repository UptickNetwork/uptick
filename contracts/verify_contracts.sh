#!/usr/bin/env bash
# Deterministic contract rebuild + bytecode comparison for CI (L-11).
#
# Usage:
#   ./verify_contracts.sh          # compile and compare against committed refs
#   ./verify_contracts.sh write    # regenerate the committed reference hashes
#
# The build is pinned to solc 0.8.28 and @openzeppelin/contracts 4.9.6 (see
# package.json / package-lock.json). Contracts are compiled WITHOUT the
# optimizer (matching the committed artifacts) and with --metadata-hash none so
# the bytecode is reproducible across environments (no source-path/IPFS metadata
# variance). Each contract's creation and runtime bytecode is hashed and compared
# to the committed reference below.
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
  solc --bin --bin-runtime --metadata-hash none \
    --base-path . --include-path node_modules --overwrite -o "$TMP_DO" "$sol" >/dev/null

  for kind in bin bin-runtime; do
    artifact="$TMP_DO/${name}.${kind}"
    if [ ! -f "$artifact" ]; then
      echo "error: missing artifact $artifact" >&2
      status=1
      continue
    fi
    actual="$(hash_file "$artifact")"
    ref_file="${refs_dir}/${name}.${kind}.sha256"

    if [ "$write_mode" = "write" ]; then
      echo "$actual" > "$ref_file"
      echo "wrote $ref_file ($actual)"
      continue
    fi

    if [ ! -f "$ref_file" ]; then
      echo "error: missing reference $ref_file (run 'verify_contracts.sh write')" >&2
      status=1
      continue
    fi

    expected="$(cat "$ref_file")"
    if [ "$actual" != "$expected" ]; then
      echo "MISMATCH: $name.$kind" >&2
      echo "  expected (committed): $expected" >&2
      echo "  actual   (rebuilt):   $actual" >&2
      status=1
    else
      echo "ok: $name.$kind $actual"
    fi
  done
done

exit "$status"
