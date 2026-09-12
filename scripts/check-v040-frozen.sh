#!/usr/bin/env bash
#
# Keep app/upgrades/v040/ pinned to reviewed content, and require that the
# surviving (frontier) changes to that pin are labelled `fix(v040):`.
#
# WHAT app/upgrades/v040 IS
# The frozen v0.4.0 upgrade handler plus the legacy registry shims it depends on
# (legacy/erc20_proposals.go, legacy/ethsecp256k1.go). They run exactly once, on
# live state, during a chain halt, so they must stay byte-identical to the build
# a validator signed off on. Before this script the freeze was prose in
# .golangci.yml and nothing could go red.
#
# THE DESIGN: CONTENT PINNING + LABEL (never history structure)
#   A. Content pin. The blob shas of every TRACKED file under app/upgrades/v040/,
#      read from the WORKING TREE (git ls-files + git hash-object), are compared,
#      as a SET, against scripts/v040-frozen.sha256. The working tree is the same
#      source `make v040-frozen-manifest` writes, so an uncommitted edit is caught
#      and "generator output == check expectation" holds by construction. No
#      history and no base are consulted. A tracked file missing on disk, an
#      added/removed file, or a missing/malformed manifest is a hard failure (fail
#      closed); the one shape A cannot see is RESIDUAL (1a).
#   B. Label. Of the non-merge commits in base..HEAD that modified the manifest,
#      take the FRONTIER: those NOT superseded (as an ancestor) by another such
#      commit. These are exactly the pin writes whose content survives to HEAD,
#      so every one must match ^fix\(v040\):. Requiring ALL in-range pin writes
#      would be red-forever once a bad commit lands -- no later commit could clear
#      it -- which is how a gate gets neutered with `|| true`; a later
#      `fix(v040):` re-pin is the sanctioned remedy. Merges are not consulted
#      (RESIDUAL (1a)); the pin BIRTH WINDOW is exempt (RESIDUAL (1b)).
#
# WHY NOT ASK GIT "WHO CHANGED THIS PATH"
# Two earlier designs did, both wrong; do not revive them. Requiring every
# path-touching subject to be `fix(v040):` MISSES the "evil merge" (a merge can put
# bytes into the path that no reviewed commit produced, while one side-branch
# `fix(v040):` makes the check report OK). Skipping merges and exempting one
# TREESAME to a parent is wrong BOTH ways: a side branch whose bytes predate base
# passes (TREESAME is not "reviewed"), and two `fix(v040):` commits touching
# DIFFERENT files auto-merge to content differing from BOTH parents (false RED).
# Lesson: such a predicate rides on merge semantics; the one below is on CONTENT.
#
# WHY FRONTIER, NOT "NEWEST" (do NOT put `git log -1 -- <path>` back)
# `git log -1 -- <path>` orders by COMMITTER DATE; on a DAG a backdated commit that
# is a *child* of a labelled one wins on topology but loses on date, so the gate
# trusts the WRONG commit. Reproduced false GREEN: a re-pinning `style(lint):`
# commit backdated via GIT_COMMITTER_DATE=2024, made a child of a `fix(v040):`
# commit and merged with `git merge --no-ff`, leaves HEAD carrying the style
# commit's bytes while `git log -1` returns the fix (rc=0). The frontier test is
# pure topology (git merge-base --is-ancestor).
#
# THE REMEDIATION FLOW (why a fix that only edits files is not enough)
# A `fix(v040):` commit that changes only the frozen files does NOT clear an
# already-landed bad change: part A compares content to the manifest, so the
# content must be RE-PINNED. The remedy is a `fix(v040):` commit carrying BOTH the
# file change and a refreshed manifest:
#
#     make v040-frozen-manifest      # rewrites scripts/v040-frozen.sha256
#     git add app/upgrades/v040/... scripts/v040-frozen.sha256
#     git commit -m 'fix(v040): ...'
#
# Because part B checks only the FRONTIER, this later re-pin is the sole surviving
# pin write and clears a previously-red range. `make v040-frozen-manifest` runs
# this script with --write-manifest, so generator and check share one data source.
#
# WHAT THIS GATE CATCHES, AND WHAT IT DOES NOT (read before citing it)
#   CATCHES: content drift under app/upgrades/v040/ NOT accompanied by a matching
#   re-pin, however the bytes got there (ordinary edit, rebase, merge -X,
#   hand-resolved conflict, amended merge, pre-base bytes).
#   DOES NOT CATCH: RESIDUAL (1a) and (1b) below.
#
# RESIDUAL GAPS (deliberate -- do not pretend they are closed)
# (1) Two sub-shapes B does not police, labelled (1a)/(1b) so the runtime
#     messages can point at them:
#   (1a) A merge commit that hand-edits a frozen file AND the manifest together is
#        not caught (B ignores merges; part A then sees content == pin). Accepted
#        on purpose: editing the manifest is an explicit, visible act, unlike the
#        accidental "edited a frozen file" shape the freeze exists to stop.
#        Covering it would require every legitimate multi-branch merge to be named
#        `fix(v040):`.
#   (1b) The pin BIRTH WINDOW: when the pin does not exist at base, B exempts the
#        in-range pin writes -- there is no baseline, the first pin IS the trust
#        anchor, and the commit that introduces the gate must not be RED the day it
#        lands (how a gate gets `|| true`d). The window is keyed on the ABSENCE of
#        the pin at base, NOT on its first creation: a base that is itself a
#        post-deletion pin-absent commit reopens it. Once the pin exists at base,
#        EVERY in-range commit that (re)adds or rewrites the pin is a re-pin and
#        must be labelled -- including a re-add whose delete was hidden inside a
#        merge (merges are dropped by --no-merges). When a run is looking at this
#        window, part B says so explicitly ("... this range is the pin's BIRTH
#        window") instead of the misleading "pin unchanged".
# (2) FORGERY COST: a decorative `fix(v040):` commit -- only reordering manifest
#     lines or adding a blank line, touching no frozen file -- makes A and B pass
#     even after an earlier commit smuggled in bad bytes, i.e. a wrongdoer can
#     WHITEWASH the pin by naming a commit `fix(v040):`. Accepted, because the
#     alternative (the ratifying commit must ALSO touch app/upgrades/v040/) would
#     BREAK the needed remedy: fixing an already-merged, content-only change
#     requires a manifest-only commit. To "tighten" this, first show how to fix
#     such a change without one.
# (3) Pinning a small generated file means two branches that BOTH re-pin may hit a
#     text conflict when their changed lines sit close (hunk context overlaps).
#     Fail-CLOSED, not a false green: the conflicted manifest keeps its `<<<<<<<`
#     markers, part A rejects it as malformed and exits 2. Resolve it with the same
#     flow (`make v040-frozen-manifest`, then one `fix(v040):` commit).
#
# FAIL-CLOSED RULES (never exit 0 when we cannot check)
#   * manifest absent or malformed -> exit 2
#   * base not 40-hex / all-zero / unreachable -> exit 2
#   * a branch-creating push's `before` is the all-zero sha -> ignored, anchor used
#   * no `git ... || true`: a git failure is fatal, never "nothing to check"
#
# Exit codes: 0 = pinned and every frontier pin write is labelled; 1 = drift, or
# a frontier pin write is not a `fix(v040):` commit; 2 = could not check.
set -euo pipefail

# V040_FREEZE_ANCHOR is the v0.4.1 release commit's parent, so it is reachable
# from every branch off this release line.
V040_FREEZE_ANCHOR="5e8000f7594524c1321a0fbd1b6708e574267647"
FROZEN_PATH="app/upgrades/v040/"
MANIFEST="scripts/v040-frozen.sha256"
ZERO_SHA="0000000000000000000000000000000000000000"

# write_manifest regenerates MANIFEST from the WORKING TREE; used by the
# `v040-frozen-manifest` target so the remediation flow is a single command.
write_manifest() {
  local paths tmp count
  paths="$(git ls-files -- "$FROZEN_PATH")"
  if [[ -z "$paths" ]]; then
    echo "error: no tracked files under $FROZEN_PATH" >&2
    exit 2
  fi

  tmp="$(mktemp)"
  while IFS= read -r path; do
    [[ -z "$path" ]] && continue
    if [[ ! -e "$path" ]]; then
      echo "error: tracked frozen file '$path' is missing from the working tree" >&2
      rm -f "$tmp"
      exit 2
    fi
    # hash-object reads the working-tree file, so an unstaged edit is captured.
    printf '%s  %s\n' "$(git hash-object -- "$path")" "$path" >> "$tmp"
  done <<< "$paths"

  LC_ALL=C sort -k2,2 "$tmp" > "$MANIFEST"
  count="$(wc -l < "$MANIFEST" | tr -d ' ')"
  rm -f "$tmp"
  echo "v040-freeze: wrote ${count} entr(ies) to $MANIFEST"
  exit 0
}

# manifest_pin_check is part A: pure content comparison, no history, no base.
manifest_pin_check() {
  if [[ ! -f "$MANIFEST" ]]; then
    echo "::error::v040-freeze: manifest $MANIFEST is missing; refusing to certify the frozen tree" >&2
    exit 2
  fi

  # Every non-empty line must be "<40-hex-blob-sha>  <path>" (two spaces).
  local lineno=0 malformed=0 line
  while IFS= read -r line; do
    lineno=$((lineno + 1))
    [[ -z "$line" ]] && continue
    if [[ ! "$line" =~ ^[0-9a-f]{40}\ \ [^[:space:]].*$ ]]; then
      echo "::error::v040-freeze: $MANIFEST:$lineno is not '<40-hex-sha>  <path>': $line" >&2
      malformed=1
    fi
  done < "$MANIFEST"
  if [[ "$malformed" -ne 0 ]]; then
    exit 2
  fi

  # Part A reads the WORKING TREE -- the same source `make v040-frozen-manifest`
  # writes -- so an uncommitted edit is caught and the two legs cannot disagree.
  local expected actual actual_tmp missing_tmp path
  expected="$(grep -v '^[[:space:]]*$' "$MANIFEST" | LC_ALL=C sort -k2,2)"

  actual_tmp="$(mktemp)"; missing_tmp="$(mktemp)"
  while IFS= read -r path; do
    [[ -z "$path" ]] && continue
    if [[ ! -e "$path" ]]; then
      printf '%s\n' "$path" >> "$missing_tmp"
      continue
    fi
    printf '%s  %s\n' "$(git hash-object -- "$path")" "$path" >> "$actual_tmp"
  done < <(git ls-files -- "$FROZEN_PATH")

  if [[ -s "$missing_tmp" ]]; then
    echo "::error::v040-freeze: tracked frozen file(s) missing from the working tree:" >&2
    cat "$missing_tmp" >&2
    rm -f "$actual_tmp" "$missing_tmp"
    exit 1
  fi
  actual="$(LC_ALL=C sort -k2,2 "$actual_tmp")"
  rm -f "$actual_tmp" "$missing_tmp"

  if [[ -z "$expected" ]]; then
    echo "::error::v040-freeze: $MANIFEST pins no file" >&2
    exit 2
  fi

  if [[ "$expected" != "$actual" ]]; then
    local diff_out
    echo "::error::v040-freeze: app/upgrades/v040/ does not match $MANIFEST (content or file set changed)" >&2
    echo "--- diff ('<' expected from $MANIFEST, '>' actual in working tree) ---" >&2
    if ! diff_out="$(diff <(printf '%s\n' "$expected") <(printf '%s\n' "$actual") 2>&1)"; then
      printf '%s\n' "$diff_out" >&2
    fi
    cat >&2 <<'EOF'
The v040 tree is frozen. If the change is intentional, land a commit whose subject
starts with 'fix(v040):' that contains BOTH the file changes and a refreshed
manifest (run: make v040-frozen-manifest). That re-pin becomes the surviving
(frontier) pin write, which is what this gate checks -- so it clears this failure.
EOF
    exit 1
  fi

  local pinned
  pinned="$(printf '%s\n' "$expected" | wc -l | tr -d ' ')"
  echo "v040-freeze: pin OK - app/upgrades/v040/ matches $MANIFEST (${pinned} file(s))"
}

# manifest_label_check is part B: it takes the FRONTIER of the in-range pin writes
# (the pin BIRTH WINDOW is exempt, RESIDUAL (1b)) -- the set whose content survives
# to HEAD -- and requires EVERY one to be an explicit fix. It does NOT use
# `git log -1 -- <path>` (committer date, not topology; see WHY FRONTIER above).
# Merges are ignored (RESIDUAL (1a)).
manifest_label_check() {
  local base="$1" c d subject superseded status
  local -a frontier=() pins=()

  while IFS= read -r c; do
    [[ -z "$c" ]] && continue
    # Exempt the pin BIRTH WINDOW only: exempt iff the pin did not exist at base
    # AND the parent does not carry it. Keyed on the ABSENCE of the pin at base,
    # not on its first creation (see RESIDUAL (1b) in the header).
    if ! git cat-file -e "${c}^:${MANIFEST}" 2>/dev/null &&
       ! git cat-file -e "${base}:${MANIFEST}" 2>/dev/null; then
      continue
    fi
    pins+=("$c")
  done < <(git rev-list --full-history --no-merges "${base}..HEAD" -- "$MANIFEST")

  if [[ "${#pins[@]}" -eq 0 ]]; then
    # Distinguish "no pin existed to change" (birth window) from "the pin existed
    # and was not touched"; calling the former "unchanged" would be false (RESIDUAL (1b)).
    if ! git cat-file -e "${base}:${MANIFEST}" 2>/dev/null; then
      echo "v040-freeze: pin absent at ${base}: this range is the pin's BIRTH window;"
      echo "v040-freeze:   part B has no baseline and exempts every pin write in it (RESIDUAL (1b))"
    else
      echo "v040-freeze: pin unchanged by any non-merge commit in ${base}..HEAD"
    fi
    return 0
  fi

  # frontier = pin writes not superseded by another pin write in range.
  for c in "${pins[@]}"; do
    superseded=0
    for d in "${pins[@]}"; do
      [[ "$c" == "$d" ]] && continue
      if git merge-base --is-ancestor "$c" "$d" 2>/dev/null; then
        superseded=1
        break
      fi
    done
    [[ "$superseded" -eq 1 ]] && continue
    frontier+=("$c")
  done

  status=0
  for c in "${frontier[@]}"; do
    subject="$(git log -1 --format='%s' "$c")"
    if [[ ! "$subject" =~ ^fix\(v040\): ]]; then
      echo "::error::v040-freeze: ${c} ${subject}" >&2
      echo "::error::v040-freeze: ${c} repinned $MANIFEST and is not superseded by a later 'fix(v040):' re-pin" >&2
      status=1
    fi
  done
  if [[ "$status" -ne 0 ]]; then
    return 1
  fi

  echo "v040-freeze: every frontier pin write (${#frontier[@]}) is a 'fix(v040):' commit"
  return 0
}

# resolve_base picks the base commit by the documented precedence and prints its
# provenance so a false red is easy to explain.
resolve_base() {
  if [[ -n "${V040_FREEZE_BASE:-}" ]]; then
    echo "v040-freeze: base source = V040_FREEZE_BASE (explicit override)" >&2
    printf '%s' "$V040_FREEZE_BASE"
  elif [[ -n "${V040_PR_BASE:-}" ]]; then
    echo "v040-freeze: base source = V040_PR_BASE (pull_request event)" >&2
    printf '%s' "$V040_PR_BASE"
  elif [[ -n "${V040_PUSH_BEFORE:-}" && "${V040_PUSH_BEFORE}" != "$ZERO_SHA" ]]; then
    echo "v040-freeze: base source = V040_PUSH_BEFORE (push event)" >&2
    printf '%s' "$V040_PUSH_BEFORE"
  else
    echo "v040-freeze: base source = V040_FREEZE_ANCHOR (fallback constant)" >&2
    printf '%s' "$V040_FREEZE_ANCHOR"
  fi
}

case "${1:-}" in
  --write-manifest)
    write_manifest
    ;;
  "")
    ;;
  *)
    echo "usage: $0 [--write-manifest]" >&2
    exit 2
    ;;
esac

# --- A: content pin (independent of base and fetch-depth; runs first) --------
manifest_pin_check

# --- B: base resolution, validated fail-closed -------------------------------
base="$(resolve_base)"

if [[ ! "$base" =~ ^[0-9a-f]{40}$ ]]; then
  echo "::error::v040-freeze: resolved base '$base' is not a 40-char lowercase hex sha" >&2
  exit 2
fi
if [[ "$base" == "$ZERO_SHA" ]]; then
  echo "::error::v040-freeze: resolved base is the all-zero sha" >&2
  exit 2
fi
if ! git cat-file -e "${base}^{commit}" 2>/dev/null; then
  echo "::error::v040-freeze: base '$base' is not a reachable commit (is actions/checkout missing fetch-depth: 0?)" >&2
  exit 2
fi

range_size="$(git rev-list --count "${base}..HEAD")"
echo "v040-freeze: base=${base}"
echo "v040-freeze: range=${base}..HEAD (${range_size} commit(s))"

manifest_label_check "$base" || {
  echo "v040-freeze: FAILED - a frontier commit that repinned $MANIFEST is not a 'fix(v040):' commit." >&2
  echo "Re-pin it as a commit whose subject starts with 'fix(v040):'." >&2
  exit 1
}

echo "v040-freeze: OK - frozen tree pinned, and every frontier pin write is a fix(v040): commit"
exit 0
