#!/usr/bin/env bash
#
# Run a `go test` command and require that it actually passed some tests.
#
# `go test` exits 0 when every test is skipped, and also when the -run pattern
# matches nothing at all. Both cases make a CI gate silently green: rename a
# test, delete its file, or let the build tag / -Enabled flag stop applying, and
# the target keeps reporting success while checking nothing. This wrapper turns
# that into a failure.
#
# Usage:
#   scripts/test-gate.sh <min-passes> -- <go test arguments...>
#
# Requires -v in the go test arguments: without it there are no "--- PASS:" lines
# to count, and the gate would fail on a genuinely passing run.
set -euo pipefail

if [[ $# -lt 3 || "$2" != "--" ]]; then
  echo "usage: $0 <min-passes> -- <go test arguments...>" >&2
  exit 2
fi

min="$1"
shift 2

if ! [[ "$min" =~ ^[0-9]+$ ]]; then
  echo "error: <min-passes> must be a non-negative integer, got '$min'" >&2
  exit 2
fi

if [[ "$min" -gt 0 ]]; then
  case " $* " in
    *" -v "*|*" --v "*) ;;
    *)
      echo "error: -v is required so passed tests can be counted (got: go test $*)" >&2
      exit 2
      ;;
  esac
fi

log="$(mktemp)"
trap 'rm -f "$log"' EXIT

status=0
go test "$@" 2>&1 | tee "$log" || status=$?

# Top-level results only: nested subtests are indented, so they never match "^".
passed="$(grep -cE '^--- PASS: ' "$log" || true)"

if [[ "$status" -ne 0 ]]; then
  exit "$status"
fi

if [[ "$passed" -lt "$min" ]]; then
  echo >&2
  echo "FAIL: go test exited 0 but only $passed top-level test(s) passed (expected at least $min)." >&2
  echo "A renamed or deleted test, or a -run pattern that no longer matches, leaves this" >&2
  echo "gate permanently green while checking nothing. If the change was intentional," >&2
  echo "update the expected count in the Makefile." >&2
  exit 1
fi

echo "test-gate: $passed top-level test(s) passed (>= $min required)"
