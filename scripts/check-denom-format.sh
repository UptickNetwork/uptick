#!/usr/bin/env bash
#
# Denom format guard.
#
# app/denom_format_pin_test.go is the executable version of the rules below and
# carries the live-chain reads this rests on.
#
# THE THREE SHAPES, AND WHY CASE IS NEVER OURS TO CHOOSE
#   ibc/<UPPER-64-hex>  ibc-go renders Denom.Hash() through cmtbytes.HexBytes,
#                       whose String() is strings.ToUpper(hex). Protocol
#                       invariant.
#   erc20:<EIP-55>      cosmos/go-ethereum's Address.String() is checksummed,
#                       i.e. deliberately MIXED case.
#   auptick / auoc      native denoms, lowercase since genesis.
#
# Every denom lookup is byte-exact: bank's SupplyKey (0x00||denom), the erc20
# denom index (GetTokenPairID -> store.Get([]byte(denom)) for anything that is
# not a bare hex address), the IBC denom map. Folding case on the way in
# normalises nothing -- it turns a hit into a silent miss -- and
# sdk.ValidateDenom stays quiet, because its regex accepts A-Za-z.
#
# WHAT THIS FAILS ON
#   [1] the legacy v0.3.3 prefix "erc20/" as a string literal in non-test Go
#       code. v0.4.0 switched to "erc20:" and no migration exists for the old
#       shape, so a reappearance means either a resurrected legacy path or a
#       copy-paste from an old branch.
#   [2] a hardcoded "erc20:" literal in non-test Go code instead of the upstream
#       constant, which would drift silently the moment cosmos/evm renames it.
#   [3] case folding applied to something named *denom*.
#
# Test files are exempt from [1] and [2] on purpose: app/erc20_inflight_refund_test.go
# and app/upgrades/v040/erc20_legacy_pairs_test.go both build legacy "erc20/"
# fixtures on purpose, to prove the migration behaves on that shape.
#
# Exemptions live in scripts/denom-format-allowlist.txt and may name either a
# whole path or one path:line. An exemption that no longer matches anything is
# reported as a note -- that is how a gate rots silently.
#
# Exit codes: 0 = clean; 1 = findings reported; 2 = could not check.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

allowlist_file="scripts/denom-format-allowlist.txt"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: denom-format needs a git working copy: the file list comes from 'git ls-files'." >&2
  exit 2
fi

all_go="$(git ls-files -- '*.go')"
if [[ -z "$all_go" ]]; then
  echo "error: 'git ls-files -- *.go' returned nothing; refusing to report a clean run." >&2
  exit 2
fi

prod_go="$(printf '%s\n' "$all_go" | grep -v '_test\.go$' || true)"
if [[ -z "$prod_go" ]]; then
  echo "error: no non-test Go files were found; refusing to report a clean run." >&2
  exit 2
fi
prod_count="$(printf '%s\n' "$prod_go" | wc -l | tr -d ' ')"

allow=""
if [[ -f "$allowlist_file" ]]; then
  allow="$(sed -e 's/#.*$//' -e 's/[[:space:]]*$//' "$allowlist_file" | grep -v '^$' || true)"
fi

# allow_res is an extended-regex alternation of the exempt entries, dots
# escaped. Appending ':' when matching makes one rule cover both granularities:
# an entry "a/b.go" matches "a/b.go:<line>:<text>", and an entry "a/b.go:12"
# matches "a/b.go:12:<text>" and nothing else.
allow_res=""
if [[ -n "$allow" ]]; then
  allow_res="$(printf '%s\n' "$allow" | sed 's/\./\\./g' | paste -sd'|' -)"
fi

# scan <extended-regex> <newline-separated file list> -> "path:line:text" hits
scan() {
  printf '%s\n' "$2" | tr '\n' '\0' | xargs -0 grep -nE "$1" 2>/dev/null || true
}

# drop_allowlisted removes hits whose path (or path:line) is exempt.
drop_allowlisted() {
  if [[ -n "$allow_res" ]]; then
    grep -vE "^(${allow_res}):" || true
  else
    cat
  fi
}

findings=0
report() {
  echo "::error::denom-format[$1]: $2" >&2
  printf '%s\n' "$3" >&2
  findings=$((findings + 1))
}

raw_all=""

# --- [1] legacy v0.3.3 prefix, as a string literal, in production Go ---------
raw="$(scan '"erc20/"' "$prod_go")"
raw_all="${raw_all}${raw}"$'\n'
hits="$(printf '%s\n' "$raw" | drop_allowlisted)"
if [[ -n "$hits" ]]; then
  report 1 \
    'the v0.3.3 "erc20/" denom prefix reappeared in non-test Go code; v0.4.0 moved to "erc20:" and nothing migrates the old shape' \
    "$hits"
fi

# --- [2] hardcoded "erc20:" prefix in production Go --------------------------
raw="$(scan '"erc20:"' "$prod_go")"
raw_all="${raw_all}${raw}"$'\n'
hits="$(printf '%s\n' "$raw" | drop_allowlisted)"
if [[ -n "$hits" ]]; then
  report 2 \
    'hardcoded "erc20:" denom prefix; use cosmoserc20types.Erc20NativeCoinDenomPrefix (or CreateDenom) so an upstream rename breaks the build instead of diverging silently' \
    "$hits"
fi

# --- [3] case folding applied to a denom ------------------------------------
raw="$(scan '(ToLower|ToUpper|EqualFold)\(' "$prod_go" | grep -i 'denom' || true)"
raw_all="${raw_all}${raw}"$'\n'
hits="$(printf '%s\n' "$raw" | drop_allowlisted)"
if [[ -n "$hits" ]]; then
  report 3 \
    'case folding applied to something named *denom*; denom lookups are byte-exact, so this turns a hit into a silent miss' \
    "$hits"
fi

# --- stale exemptions -------------------------------------------------------
if [[ -n "$allow" ]]; then
  while IFS= read -r entry; do
    [[ -z "$entry" ]] && continue
    esc="$(printf '%s' "$entry" | sed 's/\./\\./g')"
    if ! printf '%s\n' "$raw_all" | grep -qE "^(${esc}):"; then
      echo "note: denom-format: exemption '${entry}' in ${allowlist_file} matched nothing; drop it or fix the path"
    fi
  done <<< "$allow"
fi

if [[ "$findings" -ne 0 ]]; then
  echo "denom-format: FAILED - ${findings} finding group(s) above." >&2
  echo "If a finding is legitimate, add its path (or path:line) to ${allowlist_file} with the reason." >&2
  exit 1
fi

if [[ -n "$allow" ]]; then
  echo "denom-format: $(printf '%s\n' "$allow" | wc -l | tr -d ' ') exemption(s) applied from ${allowlist_file}"
fi
echo "denom-format: OK - ${prod_count} non-test Go file(s) scanned; no legacy prefix, no hardcoded prefix, no denom case folding"
exit 0
