#!/usr/bin/env bash
#
# Keep app/upgrades/v040/ pinned to known, reviewed content, and require that the
# surviving (frontier) changes to that pin are labelled `fix(v040):`.
#
# WHAT app/upgrades/v040 IS
# -------------------------
# The frozen v0.4.0 upgrade handler plus the legacy registry shims it depends on
# (legacy/erc20_proposals.go, legacy/ethsecp256k1.go). Those files run exactly
# once, on live state, during a chain halt, so they must stay byte-identical to
# the build a validator already signed off on. Before this script the freeze was
# prose in .golangci.yml and nothing could go red.
#
# THE DESIGN: CONTENT PINNING + LABEL (never history structure)
# -------------------------------------------------------------
#   A. Content pin. The blob shas of every TRACKED file under app/upgrades/v040/,
#      read from the WORKING TREE (git ls-files + git hash-object), are compared,
#      as a SET, against the manifest scripts/v040-frozen.sha256. Reading the
#      working tree -- the SAME source `make v040-frozen-manifest` writes -- means
#      an uncommitted edit is caught, and "generator output == check expectation"
#      holds by construction, not by a one-off measurement. A tracked file that is
#      missing on disk is an error (never a silent skip). No git history and no
#      base, so no merge trick can hide a content DRIFT (the one shape A cannot
#      see -- a merge that edits a frozen file and the manifest TOGETHER -- is in
#      RESIDUAL). Added/removed files fail too. A missing or malformed manifest is
#      a hard failure (fail closed).
#   B. Label. Of the non-merge commits in base..HEAD that modified the manifest,
#      take the FRONTIER: those NOT superseded (as an ancestor) by another such
#      commit. These are exactly the pin writes whose content survives to HEAD,
#      so EVERY one of them must have a subject matching ^fix\(v040\):. Requiring
#      ALL in-range pin writes (not just the frontier) would be red-forever once
#      a bad commit lands -- no later commit could clear it -- which is how such
#      a gate gets neutered with `|| true`; a later `fix(v040):` re-pin is the
#      sanctioned remedy. Merges are deliberately NOT consulted here (RESIDUAL).
#
# WHY NOT ASK GIT "WHO CHANGED THIS PATH"
# ---------------------------------------
# Two earlier versions of this gate did exactly that and both were wrong; do not
# revive them.
#
#   (1) List the non-merge commits touching the path and require every subject
#       to be `fix(v040):` (git log --no-merges -- <path>). This MISSES the
#       "evil merge": a merge commit can put bytes into the frozen path that no
#       reviewed commit ever produced (a hand-resolved conflict, `git merge -X`,
#       an amended merge). One legitimate `fix(v040):` commit on a side branch
#       is enough to make this version report OK while the merge smuggles in
#       different content.
#
#   (2) Skip merges, then exempt a merge that is TREESAME to one of its parents.
#       Wrong in BOTH directions. TREESAME only says "same content as that
#       parent" -- not "that parent's bytes were ever reviewed". A merge can be
#       TREESAME to a side branch that never touched the path and whose bytes
#       date from BEFORE `base`, so unaudited pre-base content passes (false
#       GREEN). And two legitimate `fix(v040):` commits that change two
#       DIFFERENT frozen files auto-merge to content that differs from BOTH
#       parents, so a correct multi-branch collaboration is rejected (false
#       RED).
#
#   Lesson: any predicate built on "who changed this path" is at the mercy of
#   merge semantics. The predicate below is built on CONTENT instead, which has
#   no such ambiguity.
#
# WHY FRONTIER, NOT "NEWEST" (do NOT put `git log -1 -- <path>` back)
# ------------------------------------------------------------------
# `git log -1 -- <path>` orders by COMMITTER DATE. On a DAG that is NOT the pin
# state reflected at HEAD: a backdated commit that is a *child* of a labelled one
# wins on topology but loses on date, so the gate would trust the WRONG commit.
# Concrete bypass (reproduced): set GIT_COMMITTER_DATE=2024 on a `style(lint):`
# commit that re-pins, make it a child of a `fix(v040):` commit, then merge with
# `git merge --no-ff` so HEAD's parents are [fix, style] and HEAD carries the
# style commit's bytes -- `git log -1 -- <path>` returns the fix, rc=0, FALSE
# GREEN. The frontier test below is pure topology (`git merge-base --is-ancestor`
# / `git rev-list`), so no committer-date games can move a commit off the frontier.
#
# THE REMEDIATION FLOW (why a fix that only edits files is not enough)
# --------------------------------------------------------------------
# A `fix(v040):` commit that changes only the frozen files does NOT clear an
# already-landed bad change: the content must be RE-PINNED, because part A
# compares content to the manifest. The remedy is a commit whose subject starts
# with `fix(v040):` that contains BOTH the file change and a refreshed manifest:
#
#     make v040-frozen-manifest      # rewrites scripts/v040-frozen.sha256
#     git add app/upgrades/v040/... scripts/v040-frozen.sha256
#     git commit -m 'fix(v040): ...'
#
# Because part B checks only the FRONTIER of pin writes, this later re-pin commit
# is the sole surviving pin write and clears a previously-red range -- the way a
# review gate must behave if we want developers to fix it instead of reaching for
# `|| true`.
#
# `make v040-frozen-manifest` runs this script with --write-manifest and rewrites
# the pin from the working tree, so generator and check share one data source.
#
# WHAT THIS GATE CATCHES, AND WHAT IT DOES NOT (read before citing it)
# --------------------------------------------------------------------
#   CATCHES: content drift under app/upgrades/v040/ that is NOT accompanied by a
#   matching re-pin, however the bytes got there -- an ordinary edit, a rebase,
#   `git merge -X`, a hand-resolved conflict, an amended merge, pre-base bytes.
#   Part A compares content to the pin, so no merge trick hides a drift.
#   DOES NOT CATCH: (1a) a MERGE commit that, in the SAME commit, hand-edits a
#   frozen file AND hand-edits the manifest to match (part B ignores merges and
#   part A then sees content == pin); and (1b) the pin BIRTH WINDOW -- when the
#   pin did not exist at base, every in-range pin write is exempt, because there
#   is no baseline to compare against and the first pin IS the trust anchor. Both
#   are residual (1) below, not an unstated hole.
#
# RESIDUAL GAPS (deliberate -- do not pretend they are closed)
# ----------------------------------------------------------
# (1) Two sub-shapes B does not police, both by construction -- labelled (1a) and
#     (1b) so the runtime messages can point at them:
#   (1a) A merge commit that hand-edits a frozen file AND the manifest together is
#        not caught: B ignores merges and part A then sees content == pin.
#        Accepted on purpose: editing the manifest is an explicit, visible act,
#        unlike the accidental "edited a frozen file" shape the freeze exists to
#        stop. Covering it would require every legitimate multi-branch merge to
#        be named `fix(v040):`, a cost larger than the benefit.
#   (1b) The pin BIRTH WINDOW: when the pin does not exist at base, B exempts the
#        in-range pin writes, because there is no baseline to compare against --
#        the first pin IS the trust anchor, and the commit that introduces the
#        gate must not be RED the day it lands (how a gate gets `|| true`d). The
#        window is keyed on the ABSENCE of the pin at base, NOT on the pin's first
#        creation: a base that is itself a post-deletion pin-absent commit reopens
#        it. So the moment the pin exists at base, EVERY in-range commit that
#        (re)adds or rewrites the pin is a re-pin and must be labelled -- including
#        a re-add whose delete was hidden inside a merge (merges are dropped by
#        --no-merges, so a merge-only delete is neither counted nor named). When
#        this window is what a run is looking at, part B says so explicitly
#        ("... this range is the pin's BIRTH window") instead of the misleading
#        "pin unchanged".
#
# (2) FORGERY COST. A purely decorative `fix(v040):` commit -- one that only
#     reorders manifest lines or adds a blank line and touches no frozen file --
#     makes parts A and B both pass even after an earlier commit smuggled in bad
#     bytes: a wrongdoer can WHITEWASH the pin at the cost of naming a commit
#     `fix(v040):`. We ACCEPT this, because the alternative -- requiring the
#     ratifying commit to ALSO touch app/upgrades/v040/ -- would BREAK the remedy
#     we need: when an already-landed merge changed content without re-pinning,
#     the remedy commit touches ONLY the manifest, never the frozen files. The
#     two cannot both hold; we choose the remedy being available. Anyone who
#     would "tighten" this must first explain how to fix an already-merged,
#     content-only change without a manifest-only commit.
#
# (3) A narrower consequence of pinning a small generated file: two branches that
#     BOTH re-pin may hit a text conflict when their changed lines sit close
#     together (git hunk context overlaps). This is fail-CLOSED, not a false green
#     -- the conflicted manifest keeps its `<<<<<<<` markers, part A rejects it as
#     malformed and exits 2. Resolve it with the same sanctioned flow (`make
#     v040-frozen-manifest`, then one `fix(v040):` commit). Branches whose changed
#     files are far apart in the manifest still auto-merge cleanly.
#
# FAIL-CLOSED RULES (never exit 0 when we cannot check)
# ----------------------------------------------------
#   * manifest absent or malformed -> exit 2
#   * base not 40-hex / all-zero / unreachable -> exit 2
#   * a branch-creating push's `before` is the all-zero sha -> ignored, anchor
#     used instead
#   * no `git ... || true`: a git failure is fatal, never "nothing to check"
#
# Exit codes: 0 = frozen tree pinned and every frontier pin write is labelled;
# 1 = drift, or a frontier pin write is not a `fix(v040):` commit; 2 = could not
# check (fail closed).
set -euo pipefail

# V040_FREEZE_ANCHOR is the parent commit of the v0.4.1 release commit; being the
# release's direct parent it is reachable from every branch off this release line.
V040_FREEZE_ANCHOR="5e8000f7594524c1321a0fbd1b6708e574267647"
FROZEN_PATH="app/upgrades/v040/"
MANIFEST="scripts/v040-frozen.sha256"
ZERO_SHA="0000000000000000000000000000000000000000"

# write_manifest regenerates MANIFEST from the WORKING TREE. Used by the
# `v040-frozen-manifest` Makefile target so the remediation flow (edit files,
# re-pin, commit both with a fix(v040): subject) is a single command.
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

# manifest_label_check is part B. It computes the FRONTIER of pin writes -- the
# non-merge commits in base..HEAD that RE-PINNED (the pin BIRTH WINDOW is exempt:
# when the pin does not exist at base, its in-range writes are the first pin, not
# a re-pin) and are NOT an ancestor of any other such commit -- which is exactly
# the set whose content survives to HEAD, and requires EVERY one of them to be an
# explicit fix. It deliberately does NOT use `git log -1 -- <path>`: that orders
# by committer date, which on a DAG is not topology, so a backdated child commit
# could hide behind a label (see WHY FRONTIER, NOT "NEWEST" above). Merges are
# ignored (RESIDUAL (1a)).
manifest_label_check() {
  local base="$1" c d subject superseded status
  local -a frontier=() pins=()

  while IFS= read -r c; do
    [[ -z "$c" ]] && continue
    # Exempt the pin BIRTH WINDOW only: the commit is exempt iff the pin did NOT
    # exist at base AND the parent does not carry it either. The window is keyed
    # on the ABSENCE of the pin at base, NOT on the pin's first creation, so a
    # base that is itself a post-deletion pin-absent commit reopens it. There is
    # then no baseline to compare against, and this is the very commit that
    # introduces the gate, so the job would otherwise be RED the day it lands (how
    # a gate gets neutered with `|| true`). As soon as the pin exists at base, ANY
    # in-range commit that (re)adds or rewrites the pin is a re-pin and must be
    # labelled -- including a re-add whose delete was hidden inside a merge
    # (merges are dropped by --no-merges, so a merge-only delete is neither
    # counted nor named).
    if ! git cat-file -e "${c}^:${MANIFEST}" 2>/dev/null &&
       ! git cat-file -e "${base}:${MANIFEST}" 2>/dev/null; then
      continue
    fi
    pins+=("$c")
  done < <(git rev-list --full-history --no-merges "${base}..HEAD" -- "$MANIFEST")

  if [[ "${#pins[@]}" -eq 0 ]]; then
    # Distinguish "no pin existed to change" (birth window) from "the pin existed
    # and simply was not touched". Calling the former "unchanged" would be false:
    # the pin is CREATED in this range and exempted by construction (RESIDUAL 1b).
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

# resolve_base picks the base commit using the documented precedence and prints
# its provenance so a false red is easy to explain.
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
