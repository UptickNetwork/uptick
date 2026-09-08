#!/usr/bin/env bash
#
# govulncheck 基线过滤脚本。
#
# govulncheck 官方不支持 baseline，本脚本实现：
#   1. 全量扫描 ./...；
#   2. 提取影响本代码的漏洞 ID；
#   3. 与 scripts/vuln-baseline.txt 对比：
#      - 出现基线之外的新漏洞 -> 非零退出，CI 失败；
#      - 仅剩基线内漏洞       -> 通过（提示性输出）。
#
# 维护规则：
#   - 基线内条目出现修复版本（govulncheck 输出不再报它）时，应尽快
#     升级依赖并从基线中删除对应 ID；
#   - 严禁把"新出现的漏洞"直接加进基线，必须先完成人工评估并在
#     基线文件注释中记录理由。
set -euo pipefail

cd "$(dirname "$0")/.."

BASELINE_FILE="scripts/vuln-baseline.txt"
if [[ ! -f "$BASELINE_FILE" ]]; then
  echo "error: baseline file not found: $BASELINE_FILE" >&2
  exit 2
fi

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "govulncheck not found, installing..."
  go install golang.org/x/vuln/cmd/govulncheck@latest
  export PATH="$HOME/go/bin:$PATH"
fi

LOG="$(mktemp)"
trap 'rm -f "$LOG"' EXIT

# govulncheck 发现漏洞时退出码为 3，不能让 set -e 直接终止。
govulncheck ./... >"$LOG" 2>&1 || true

# Sanity check: make sure the scan actually ran. Otherwise a broken
# environment (go missing, network error, govulncheck crash) would
# silently produce an empty result and falsely pass.
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
  grep -E '^Vulnerability #|More info:|Found in:|Fixed in:' "$LOG" |
    grep -A3 -B1 "$(echo "$new" | head -1)" >&2 || sed -n '1,40p' "$LOG" >&2
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
