#!/usr/bin/env bash
#
# UGE (x/appmining) contract guard.
#
# x/appmining has no code in this tree yet. This gate exists BEFORE the module
# lands, because three of its four rules are about what must never appear, and a
# rule nobody checks is a rule that erodes the first time someone is in a hurry.
# The executable version is x/appmining/contract_guard_test.go.
#
# WHY THESE THREE RULES
#
#   [1] No EVM import.
#       UGE is a Cosmos-side protocol: its emitter is CosmWasm, its economics are
#       bank transfers out of a module account. The moment x/appmining imports
#       cosmos/evm, the module inherits the EVM account model -- a contract may
#       become the caller -- and every permission check written against
#       sdk.AccAddress has to be re-derived against a caller that can be code.
#       That is a design change, not an implementation detail, so it has to be a
#       deliberate one.
#
#   [2] Do not parse PavCommitment's internal fields.
#       The commitment is opaque by construction: submitter commits, a third
#       party challenges, settlement reads the verdict. Any code that reaches
#       INTO the commitment (unmarshalling it, reading a field, comparing a
#       field) is re-deriving what the commitment was supposed to make
#       underivable, and it does so with consensus-critical consequences --
#       a challenge window or an epoch boundary then lives in two places and
#       drifts. The verdict and the height are the whole interface.
#
#   [3] chain/appmining (the app_id prefix) may not be minted.
#       app_id is base64url-no-padding of a 32-byte SHA-256 -- 43 chars, an
#       opaque identifier the chain deliberately does not parse (UGE confirmed
#       the chain end never resolves it). Storing it under a prefixed key is
#       what keeps it opaque at the storage layer too. A prefix would make
#       "app_id" and "chain/<app_id>" two spellings of one thing and put the
#       separator inside the key space.
#
#   Note [3] is deliberately narrow: it bans only the literal below. Storing
#   app_id under a module-owned prefix (0x01 | len | app_id, say) is fine and is
#   in fact the expected shape; the ban is on the string "chain/" appearing as
#   an app_id key prefix, because that specific prefix is a protocol claim
#   ("this chain owns this app") rather than a storage detail.
#
# Exit codes: 0 = clean; 1 = findings reported; 2 = could not check.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: appmining-contract needs a git working copy: the file list comes from 'git ls-files'." >&2
  exit 2
fi

mod_dir="x/appmining"

# Pre-module is the normal state today, not an error: the gate is armed ahead of
# the implementation. What it must never do is report a clean run for a module it
# never looked at.
if [[ ! -d "$mod_dir" ]]; then
  echo "appmining-contract: OK (pre-module) - ${mod_dir} does not exist yet; nothing to scan."
  echo "appmining-contract: the three rules are armed and will apply to the first commit that creates the module."
  exit 0
fi

# Probe the filesystem first, so "the module has not landed" stays distinguishable
# from "the file list is broken". Only the latter is an error.
on_disk_prod="$(find "$mod_dir" -name '*.go' ! -name '*_test.go' -type f 2>/dev/null || true)"
if [[ -z "$on_disk_prod" ]]; then
  echo "appmining-contract: OK (pre-module) - ${mod_dir} carries no production Go file yet; rules armed and inert."
  echo "appmining-contract: they apply automatically to the first commit that lands one."
  exit 0
fi

# The module HAS production code on disk. From here the file list must be
# trustworthy: git ls-files is what keeps the scan bounded to tracked files, and
# an empty result here would mean every file is untracked -- a silently clean run.
mod_go="$(git ls-files -- "${mod_dir}/*.go")"
if [[ -z "$mod_go" ]]; then
  echo "error: ${mod_dir} has production .go files on disk but 'git ls-files ${mod_dir}/*.go' is empty." >&2
  echo "error: they are all untracked, so this scan would pass vacuously. Add them first." >&2
  exit 2
fi

prod_go="$(printf '%s\n' "$mod_go" | grep -v '_test\.go$' || true)"
if [[ -z "$prod_go" ]]; then
  echo "error: ${mod_dir} production files exist on disk but none of them are tracked non-test Go files." >&2
  exit 2
fi
prod_count="$(printf '%s\n' "$prod_go" | wc -l | tr -d ' ')"

findings=0
report() {
  echo "::error::appmining-contract[$1]: $2" >&2
  printf '%s\n' "$3" >&2
  findings=$((findings + 1))
}

scan() {
  printf '%s\n' "$2" | tr '\n' '\0' | xargs -0 grep -nE "$1" 2>/dev/null || true
}

# --- [1] no EVM import in the module ---------------------------------------
# Matched on the import path, not on a symbol, so an aliased import is caught
# too.
raw="$(scan '"github\.com/cosmos/evm' "$prod_go")"
hits="$raw"
if [[ -n "$hits" ]]; then
  report 1 \
    'x/appmining imports cosmos/evm; UGE is a Cosmos-side protocol and adding an EVM import makes a contract a possible caller, which invalidates every sdk.AccAddress permission check' \
    "$hits"
fi

# --- [2] no reaching into PavCommitment internals ---------------------------
# The commitment is a blob. Reading a field of it is the failure mode; the type
# may be referenced (it is the Msg's field type), so this matches member access
# and per-field getters rather than the bare identifier.
raw="$(scan '(GetAppId|GetEpoch|GetCommitment|GetChallenger|GetChallengeReason)[[:space:]]*\(' "$prod_go" | grep -iE 'commit|pav' || true)"
hits="$raw"
if [[ -n "$hits" ]]; then
  report 2 \
    'x/appmining reads individual fields off the PAV commitment; the commitment is opaque by design and per-field access duplicates the challenge window or epoch boundary into a second place that then drifts' \
    "$hits"
fi

# --- [3] no chain/ app_id key prefix ----------------------------------------
raw="$(scan '"chain/"' "$prod_go")"
hits="$raw"
if [[ -n "$hits" ]]; then
  report 3 \
    'x/appmining uses a "chain/" app_id key prefix; app_id is an opaque base64url-no-padding SHA-256 the chain deliberately does not parse, and a prefix contradicts that at the storage layer' \
    "$hits"
fi

if [[ "$findings" -ne 0 ]]; then
  echo "appmining-contract: FAILED - ${findings} finding group(s) above." >&2
  exit 1
fi

echo "appmining-contract: OK - ${prod_count} non-test Go file(s) scanned in ${mod_dir}; no EVM import, no PavCommitment field access, no chain/ app_id prefix"
exit 0