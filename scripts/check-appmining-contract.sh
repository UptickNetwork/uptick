#!/usr/bin/env bash
#
# UGE (x/appmining) contract guard.
# Executable twin: x/appmining/contract_guard_test.go
#
# Rules:
#   [1] No EVM / go-ethereum imports.
#   [2] Do not json/proto-unmarshal opaque hash fields as structured payloads.
#   [3] No literal "chain/" app_id key prefix.
#
# Exit codes: 0 = clean; 1 = findings; 2 = could not check.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: appmining-contract needs a git working copy." >&2
  exit 2
fi

mod_dir="x/appmining"

if [[ ! -d "$mod_dir" ]]; then
  echo "appmining-contract: OK (pre-module) - ${mod_dir} does not exist yet."
  exit 0
fi

# Include tracked and untracked (but not ignored) Go files so the gate works
# before the first commit that lands the module.
mod_go="$(git ls-files --cached --others --exclude-standard -- "${mod_dir}" | grep '\.go$' || true)"
if [[ -z "$mod_go" ]]; then
  on_disk="$(find "$mod_dir" -name '*.go' ! -name '*_test.go' ! -name '*.pb.go' ! -name '*.pb.gw.go' -type f 2>/dev/null || true)"
  if [[ -z "$on_disk" ]]; then
    echo "appmining-contract: OK (pre-module) - no production Go files yet."
    exit 0
  fi
  echo "error: ${mod_dir} has production .go files on disk but none are visible to git." >&2
  exit 2
fi

# Production files: exclude tests and generated protobuf.
prod_go="$(printf '%s\n' "$mod_go" | grep -v '_test\.go$' | grep -v '\.pb\.go$' | grep -v '\.pb\.gw\.go$' || true)"
if [[ -z "$prod_go" ]]; then
  echo "error: no non-test, non-generated Go files under ${mod_dir}." >&2
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
  printf '%s\n' "$2" | tr '\n' '\0' | xargs -0 grep -nIE "$1" 2>/dev/null || true
}

raw="$(scan '"github\.com/cosmos/evm|"github\.com/ethereum/go-ethereum' "$prod_go")"
if [[ -n "$raw" ]]; then
  report 1 'x/appmining must not import EVM packages' "$raw"
fi

raw="$(scan 'json\.Unmarshal\([^)]*(attestation|AttestationHash|evidence|EvidenceHash|result_commitment|ResultCommitment)|proto\.Unmarshal\([^)]*(AttestationHash|EvidenceHash|ResultCommitment)' "$prod_go")"
if [[ -n "$raw" ]]; then
  report 2 'x/appmining must not decode opaque hash fields as structured payloads' "$raw"
fi

raw="$(scan '"chain/"' "$prod_go")"
if [[ -n "$raw" ]]; then
  report 3 'x/appmining must not use a "chain/" app_id key prefix' "$raw"
fi

if [[ "$findings" -ne 0 ]]; then
  echo "appmining-contract: FAILED - ${findings} finding group(s)." >&2
  exit 1
fi

echo "appmining-contract: OK - ${prod_count} production Go file(s) scanned in ${mod_dir}"
exit 0
