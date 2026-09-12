#!/usr/bin/env bash
#
# Fail when prose documents a `make <target>` that the Makefile does not define.
#
# WHY THIS EXISTS
# ---------------
# CONTRIBUTING.md is a contract: it told contributors that `development` must
# never fail `make test-import`, but no `test-import` rule has ever existed in
# the Makefile (only a stale `.PHONY` mention). Running it fails with
# "No rule to make target 'test-import'" -- so a documented, mandatory gate did
# nothing but mislead. A doc that names a gate nobody can run is the same class
# of "green that proves nothing" this repository keeps finding in its CI.
#
# WHAT IT CHECKS -- and it checks BOTH places that name commands
# --------------------------------------------------------------
#   1. Inline code spans (`like this`), which is where CONTRIBUTING.md puts
#      runnable commands. A span is split on commas so one span listing several
#      targets (`make lint, make test, make test-race`) is handled piece by
#      piece, and only pieces of the exact form `make <target>` are considered.
#
#   2. Fenced (```-delimited) code blocks. These are where the CI gate commands
#      actually live -- `make test-sim-ci` and
#      `UPTICK_E2E_STRICT=1 make test-e2e-localnet` are inside a ```bash block.
#      Each fenced line is split on whitespace and every `make <target>` token
#      is taken, so an env-prefixed command still yields its target.
#
# Prose verbs are not commands and are never mistaken for targets: "make sure",
# "make changes" and "make a comment" appear only in plain sentences, and this
# script reads only spans and fenced blocks, never bare prose.
#
# USAGE
# -----
#   scripts/check-doc-make-targets.sh [DOC] [MAKEFILE]
# Defaults: DOC=CONTRIBUTING.md, MAKEFILE=Makefile.
#
# Exit codes: 0 = every documented target exists; 1 = at least one does not;
# 2 = a required input file is missing.
set -euo pipefail

doc="${1:-CONTRIBUTING.md}"
makefile="${2:-Makefile}"

if [[ ! -f "$doc" ]]; then
  echo "::error::doc file not found: $doc" >&2
  exit 2
fi
if [[ ! -f "$makefile" ]]; then
  echo "::error::Makefile not found: $makefile" >&2
  exit 2
fi

# Three backticks, built from octal escapes so the fence never has to be quoted
# inside this script.
FENCE=$'\x60\x60\x60'

span_hits="$(mktemp)"
word_hits="$(mktemp)"
targets_file="$(mktemp)"
trap 'rm -f "$span_hits" "$word_hits" "$targets_file"' EXIT

# One pass over the doc: collect inline spans outside fences and `make <target>`
# tokens inside fences.
in_fence=0
while IFS= read -r line; do
  trimmed="${line#"${line%%[![:space:]]*}"}"

  # A fence delimiter (``` or ```lang) toggles fenced-block state; the delimiter
  # line itself carries no command.
  if [[ "$trimmed" == "$FENCE"* ]]; then
    if [[ "$in_fence" -eq 0 ]]; then in_fence=1; else in_fence=0; fi
    continue
  fi

  if [[ "$in_fence" -eq 1 ]]; then
    # Fenced lines are shell commands: keep every `make <target>` token so an
    # env-prefix like `UPTICK_E2E_STRICT=1 make test-e2e-localnet` is handled.
    if ! printf '%s\n' "$line" | grep -oE 'make[[:space:]]+[A-Za-z0-9][A-Za-z0-9_.-]*' >> "$word_hits"; then
      :
    fi
  else
    if ! printf '%s\n' "$line" | grep -oE '`[^`]+`' >> "$span_hits"; then
      :
    fi
  fi
done < "$doc"

# Inline spans -> target names (split on commas, trim, exact `make <target>`).
while IFS= read -r span; do
  [[ -z "$span" ]] && continue
  span="${span#\`}"
  span="${span%\`}"
  IFS=',' read -ra parts <<< "$span"
  for part in "${parts[@]}"; do
    part="${part#"${part%%[![:space:]]*}"}"
    part="${part%"${part##*[![:space:]]}"}"
    if [[ "$part" =~ ^make[[:space:]]+([A-Za-z0-9][A-Za-z0-9_.-]*)$ ]]; then
      printf '%s\n' "${BASH_REMATCH[1]}" >> "$targets_file"
    fi
  done
done < "$span_hits"

# Fenced `make <target>` tokens -> target names.
while IFS= read -r token; do
  [[ -z "$token" ]] && continue
  if [[ "$token" =~ ^make[[:space:]]+([A-Za-z0-9][A-Za-z0-9_.-]*)$ ]]; then
    printf '%s\n' "${BASH_REMATCH[1]}" >> "$targets_file"
  fi
done < "$word_hits"

# Every documented target must have a real rule in the Makefile.
checked=0
status=0
while IFS= read -r target; do
  [[ -z "$target" ]] && continue
  checked=$((checked + 1))
  if grep -qE "^${target}:" "$makefile"; then
    echo "  ok: make ${target}"
  else
    echo "::error file=${doc}::documented target 'make ${target}' has no rule in ${makefile}"
    status=1
  fi
done < "$targets_file"

if [[ "$checked" -eq 0 ]]; then
  echo "check-doc-make-targets: no 'make <target>' references found in ${doc}"
fi

if [[ "$status" -ne 0 ]]; then
  echo "check-doc-make-targets: FAILED - ${doc} documents a target that ${makefile} does not define." >&2
  echo "Either add the rule or stop advertising the target." >&2
  exit 1
fi

echo "check-doc-make-targets: OK - ${checked} documented target(s) all exist in ${makefile}"
exit 0
