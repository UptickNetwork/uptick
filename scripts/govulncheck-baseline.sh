#!/usr/bin/env bash
#
# govulncheck baseline filter: full scan, extract the GO-IDs that affect this
# code, then diff them against scripts/vuln-baseline.txt. New IDs outside the
# baseline fail CI; baseline-only IDs pass with informational output.
#
# Maintenance: when a baseline entry gets a fixed version (govulncheck stops
# reporting it), upgrade the dependency and delete the ID. NEVER add a new
# finding to the baseline directly -- it needs a manual assessment and a
# justification comment in the baseline file.
set -euo pipefail

cd "$(dirname "$0")/.."

BASELINE_FILE="scripts/vuln-baseline.txt"
if [[ ! -f "$BASELINE_FILE" ]]; then
  echo "error: baseline file not found: $BASELINE_FILE" >&2
  exit 2
fi

# G-02: the scanner is pinned, not floating. `@latest` resolved to a different
# binary on different days (v1.8.0 as of 2026-09-13), so "no new vulnerabilities"
# was a statement about an unknown tool. The pin below must stay in lockstep with
# .github/workflows/security.yml (GOVULNCHECK_VERSION), which installs the binary
# this script then picks up from $PATH: the assertion further down runs against
# whatever is on $PATH in BOTH places, so if one copy is bumped alone the other
# fails loudly instead of quietly scanning with a different scanner.
#
# v1.7.0 is the newest x/vuln that go.mod's toolchain (go 1.25.13) can build;
# v1.8.0 requires `go >= 1.26.0`. Bump this only together with the toolchain.
PINNED_GOVULNCHECK_VERSION="v1.7.0"

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "govulncheck not found, installing ${PINNED_GOVULNCHECK_VERSION}..."
  go install "golang.org/x/vuln/cmd/govulncheck@${PINNED_GOVULNCHECK_VERSION}"
  # `go install` writes into $(go env GOBIN), or GOPATH/bin when GOBIN is unset
  # (this used to hardcode $HOME/go/bin, which is wrong whenever GOBIN is set).
  install_bin="$(go env GOBIN)"
  [[ -n "$install_bin" ]] || install_bin="$(go env GOPATH)/bin"
  export PATH="$install_bin:$PATH"
fi

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "FAIL: govulncheck is still not on PATH after installing ${PINNED_GOVULNCHECK_VERSION}." >&2
  exit 2
fi

LOG="$(mktemp)"
trap 'rm -f "$LOG"' EXIT

# Record which scanner produced the verdict below, and refuse to continue with a
# different one. The `Scanner:` line is read from the binary's own build info
# (x/vuln internal/scan/run.go, scannerVersion), so a look-alike placed earlier
# on $PATH cannot satisfy it. The `Go:` line next to it is the ambient `go` from
# $PATH and is printed for traceability only, never asserted on.
version_out="$(govulncheck -version 2>&1 || true)"
echo "$version_out"
installed_version="$(printf '%s\n' "$version_out" | sed -n 's/^Scanner: govulncheck@//p' | tr -d '\r')"
if [[ "$installed_version" != "$PINNED_GOVULNCHECK_VERSION" ]]; then
  echo "FAIL: govulncheck reports '${installed_version:-<unparseable>}', expected '${PINNED_GOVULNCHECK_VERSION}'." >&2
  echo "Refusing to scan: the verdict would describe a scanner this gate does not pin, so" >&2
  echo "'no new vulnerabilities' would not be a statement about a fixed tool." >&2
  echo "Fix by installing the pin:  go install golang.org/x/vuln/cmd/govulncheck@${PINNED_GOVULNCHECK_VERSION}" >&2
  echo "If the pin is genuinely moving, change it here AND GOVULNCHECK_VERSION in" >&2
  echo ".github/workflows/security.yml in the same commit." >&2
  exit 2
fi

# govulncheck exits 3 for findings and other non-zero codes for build/network
# errors; capture the rc instead of letting set -e abort, for the checks below.
govulncheck_rc=0
govulncheck ./... >"$LOG" 2>&1 || govulncheck_rc=$?

# Three validity checks so a broken environment cannot produce a silent false
# PASS: govulncheck must have actually run, its rc must be 0 (clean) or 3
# (findings) -- anything else is a build/network/toolchain error -- and the
# report must carry its signature section (Symbol Results / No vulnerabilities).
if [[ "$govulncheck_rc" -ne 0 && "$govulncheck_rc" -ne 3 ]]; then
  echo "FAIL: govulncheck exited with unexpected code $govulncheck_rc (build/network/tooling error)." >&2
  echo "Treating this as a CI failure to avoid silently passing on a broken environment." >&2
  cat "$LOG" >&2
  exit 2
fi

if ! grep -qE '^(=== Symbol Results ===|No vulnerabilities found)' "$LOG"; then
  echo "FAIL: govulncheck did not produce a valid report. Raw output:" >&2
  cat "$LOG" >&2
  exit 2
fi

found="$(grep -oE 'GO-[0-9]{4}-[0-9]+' "$LOG" | sort -u || true)"
if [[ -z "$found" ]]; then
  echo "OK: govulncheck found no vulnerabilities."
  exit 0
fi

baseline="$(grep -E '^GO-[0-9]{4}-[0-9]+' "$BASELINE_FILE" | sort -u || true)"

new="$(comm -13 <(printf '%s\n' "$baseline") <(printf '%s\n' "$found"))"
allowed="$(comm -12 <(printf '%s\n' "$baseline") <(printf '%s\n' "$found"))"

status=0
if [[ -n "$new" ]]; then
  echo "FAIL: new vulnerabilities NOT in baseline:" >&2
  echo "$new" >&2
  echo >&2
  echo "Details:" >&2
  # Print detailed context for EVERY new vulnerability, not just the first.
  for v in $new; do
    echo "--- $v ---" >&2
    grep -A3 -B1 "$v" "$LOG" >&2 || sed -n '1,40p' "$LOG" >&2
  done
  echo >&2
  echo "If (and only if) these are confirmed unfixable/accepted, add the IDs to" >&2
  echo "$BASELINE_FILE with a justification comment." >&2
  status=1
fi

if [[ -n "$allowed" ]]; then
  echo "PASS (with baseline): known no-fix vulnerabilities:"
  sed 's/^/  - /' <<<"$allowed"
fi

if [[ "$status" -eq 0 ]]; then
  echo "govulncheck baseline check: PASS"
fi
exit "$status"
