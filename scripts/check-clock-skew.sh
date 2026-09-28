#!/usr/bin/env bash
#
# Preflight: can this host's clock follow a cometbft >= v0.38.22 chain?
#
# WHY THIS EXISTS
# cometbft v0.38.22 added `consensus.block_time_tolerance` (config/config.go),
# wired in node/node.go:393 to state's block executor and enforced in
# state/validation.go:
#
#   if tol := vopts.blockTimeTolerance; tol > 0 && !block.Time.Before(time.Now().Add(tol)) {
#       return fmt.Errorf("block time %v is too far in the future (wall clock %v + tolerance %v)", ...)
#   }
#
# So a block whose header time is at/after `local wall clock + tolerance` is
# rejected. Block time tracks the network's median clock, so the host that trips
# this is the one whose clock runs BEHIND: with the default tolerance of 1m0s, a
# host more than 60s slow rejects every block, stops signing and stalls at one
# height with a full-looking log (ApplyBlock -> ErrInvalidBlock).
#
# Two properties make this worth a preflight rather than a config edit:
#   1. The default is applied when the key is ABSENT from config.toml, with no
#      log line and no validation error. Nodes that never re-ran `init` (i.e.
#      every upgraded node) therefore get 60s without anyone choosing it.
#   2. `block_time_tolerance = 0` is NOT a way to disable it: config.go's
#      ValidateBasic rejects a non-positive value. The check is skipped only by
#      a value that is never reached -- i.e. do not "fix" this by setting 0.
#
# CHECKS
#   - the effective tolerance, read from <home>/config/config.toml, with a
#     commented-out key reported as ABSENT (it is not in effect);
#   - this host's offset against independent time sources, measured over HTTPS
#     `Date:` headers at 1-second resolution. That resolution is deliberate: the
#     threshold is tens of seconds, and an absolute comparison needs no
#     agreement with any tool's sign convention (sntp/chronyc differ).
#
# Usage:
#   scripts/check-clock-skew.sh [--home DIR] [--tolerance DUR]
#                               [--sources URL,URL,...] [--max-skew SECONDS]
#                               [--use-proxy] [--force-behind SECONDS]
#
#   --home DIR         node home; default $UPTICKD_HOME or $HOME/.uptickd
#   --tolerance DUR    override the tolerance (Go duration, e.g. 60s, 1m0s)
#   --sources LIST     comma-separated https URLs (default 3 public hosts)
#   --max-skew N       warn above N seconds of skew (default 5)
#   --use-proxy        honour *_proxy env vars; by default requests are direct,
#                      because a proxy answers with ITS OWN clock -- measuring a
#                      middlebox and reporting it as the node's skew is a false
#                      signal, not a failed measurement.
#   --force-behind N   skip measurement and evaluate a synthetic N-second
#                      local-behind offset. For rehearsing the failure path and
#                      for evaluating an offset measured by other monitoring.
#
# Exit codes:
#   0 = skew below the tolerance (a warning may still have been printed)
#   1 = local clock is behind by at least the tolerance; blocks would be rejected
#   2 = no measurement could be taken (tolerance/offset unknown) -- callers must
#       treat this as a failure, never as "no skew found"
#   3 = usage error
set -euo pipefail

home="${UPTICKD_HOME:-${HOME}/.uptickd}"
tolerance=""
sources="https://www.cloudflare.com,https://www.google.com,https://goproxy.io"
max_skew=5
use_proxy=0
force_behind=""

die_usage() {
  echo "usage: $0 [--home DIR] [--tolerance DUR] [--sources URL,URL] [--max-skew SECONDS] [--use-proxy] [--force-behind SECONDS]" >&2
  exit 3
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --home)         [[ $# -ge 2 ]] || die_usage; home="$2"; shift 2 ;;
    --tolerance)    [[ $# -ge 2 ]] || die_usage; tolerance="$2"; shift 2 ;;
    --sources)      [[ $# -ge 2 ]] || die_usage; sources="$2"; shift 2 ;;
    --max-skew)     [[ $# -ge 2 ]] || die_usage; max_skew="$2"; shift 2 ;;
    --use-proxy)    use_proxy=1; shift ;;
    --force-behind) [[ $# -ge 2 ]] || die_usage; force_behind="$2"; shift 2 ;;
    -h|--help)      sed -n '2,50p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *)              die_usage ;;
  esac
done

[[ "${max_skew}" =~ ^[0-9]+$ ]] || { echo "error: --max-skew must be a non-negative integer" >&2; exit 3; }
if [[ -n "${force_behind}" ]]; then
  [[ "${force_behind}" =~ ^-?[0-9]+$ ]] || { echo "error: --force-behind must be an integer" >&2; exit 3; }
fi

# Go duration ("1m0s", "60s", "1h30m", "8760h") -> whole seconds, rounded up so
# a sub-second tolerance never reads as 0 and quietly disables the comparison.
duration_to_seconds() {
  awk -v d="$1" '
    BEGIN {
      total = 0; found = 0
      while (match(d, /[0-9]+(\.[0-9]+)?(ns|us|ms|s|m|h)/)) {
        tok = substr(d, RSTART, RLENGTH)
        unit = tok; sub(/^[0-9.]+/, "", unit)
        num = tok;  sub(/(ns|us|ms|s|m|h)$/, "", num)
        if      (unit == "ns") total += num / 1000000000
        else if (unit == "us") total += num / 1000000
        else if (unit == "ms") total += num / 1000
        else if (unit == "s")  total += num
        else if (unit == "m")  total += num * 60
        else if (unit == "h")  total += num * 3600
        d = substr(d, RSTART + RLENGTH)
        found = 1
      }
      if (!found) exit 1
      printf "%d", (total == int(total) ? total : int(total) + 1)
    }'
}

# Read [consensus] block_time_tolerance. A commented key is ABSENT, not zero and
# not "the default" -- cometbft applies 1m0s either way, and saying so out loud is
# the point: nobody chose that value.
read_configured_tolerance() {
  local cfg="$1"
  awk '
    /^[ \t]*\[/ {
      line = $0; sub(/^[ \t]*\[/, "", line); sub(/\].*$/, "", line)
      gsub(/[ \t"]/, "", line)
      in_consensus = (line == "consensus")
      next
    }
    in_consensus && /^[ \t]*#/ && /block_time_tolerance/ { commented = 1; next }
    in_consensus && /^[ \t]*block_time_tolerance[ \t]*=/ {
      v = $0; sub(/^[^=]*=[ \t]*/, "", v)
      gsub(/"/, "", v)
      sub(/[ \t\r]+$/, "", v)
      print v; found = 1; exit
    }
    END { if (!found) { print (commented ? "__COMMENTED__" : "__ABSENT__") } }
  ' "$cfg"
}

# "Tue, 24 Sep 2026 03:20:00 GMT" -> epoch seconds. GNU coreutils first
# (-d), then BSD/macOS (-j -f): the invalid flag makes the other try fail.
parse_http_date() {
  local s="$1" out
  if out="$(date -u -d "${s}" +%s 2>/dev/null)" && [[ -n "${out}" ]]; then printf '%s' "${out}"; return 0; fi
  if out="$(date -u -j -f '%a, %d %b %Y %H:%M:%S GMT' "${s}" +%s 2>/dev/null)" && [[ -n "${out}" ]]; then printf '%s' "${out}"; return 0; fi
  return 1
}

fetch_date_header() {
  local url="$1" args=(-sSI --max-time 10)
  [[ "${use_proxy}" -eq 1 ]] || args+=(--noproxy '*')
  curl "${args[@]}" "${url}" 2>/dev/null | awk '
    { if (tolower(substr($0, 1, 5)) == "date:") {
        v = $0; sub(/^[^:]*:[ \t]*/, "", v); sub(/\r$/, "", v); print v; exit } }'
}

tol_secs=""
tol_source=""
if [[ -n "${tolerance}" ]]; then
  tol_secs="$(duration_to_seconds "${tolerance}")" || { echo "error: --tolerance is not a Go duration: ${tolerance}" >&2; exit 3; }
  tol_source="--tolerance"
else
  cfg="${home}/config/config.toml"
  raw=""
  if [[ -f "${cfg}" ]]; then
    raw="$(read_configured_tolerance "${cfg}")"
  else
    raw="__NOCONFIG__"
  fi
  case "${raw}" in
    __COMMENTED__|__ABSENT__|__NOCONFIG__)
      tol_secs=60
      case "${raw}" in
        __COMMENTED__) tol_source="commented out in ${cfg} -> cometbft default 1m0s" ;;
        __ABSENT__)    tol_source="absent from ${cfg} -> cometbft default 1m0s" ;;
        __NOCONFIG__)  tol_source="${cfg} not found -> cometbft default 1m0s" ;;
      esac ;;
    "")
      echo "error: block_time_tolerance is present but empty in ${cfg}" >&2; exit 3 ;;
    *)
      tol_secs="$(duration_to_seconds "${raw}")" || { echo "error: block_time_tolerance is not a Go duration: ${raw}" >&2; exit 3; }
      tol_source="${cfg}";;
  esac
fi

if [[ "${tol_secs}" -le 0 ]]; then
  echo "NOTE: tolerance resolves to 0s. cometbft ValidateBasic rejects a non-positive value, and" >&2
  echo "      the future-block check is skipped only because an unreachable value was written." >&2
  echo "      Treat 0s as 'check disabled', not as 'host verified'." >&2
  exit 2
fi

echo "node home      : ${home}"
echo "tolerance      : ${tol_secs}s  (${tol_source})"
echo "warn threshold : ${max_skew}s of absolute skew"

# A positive number means this host is BEHIND the source, which is the direction
# that gets blocks rejected. Worst observation across sources wins: the check
# should fail on the strongest evidence of slowness, not the friendliest.
worst_behind=0
measured=0

if [[ -n "${force_behind}" ]]; then
  worst_behind="${force_behind}"
  measured=1
  echo
  echo "offset         : ${force_behind}s behind (synthetic, --force-behind; not measured)"
else
  echo
  printf '%-32s %10s  %s\n' "source" "offset(s)" "local vs source"
  IFS=',' read -r -a source_list <<< "${sources}"
  for src in "${source_list[@]}"; do
    src_trimmed="${src#"${src%%[![:space:]]*}"}"
    src_trimmed="${src_trimmed%"${src_trimmed##*[![:space:]]}"}"
    [[ -n "${src_trimmed}" ]] || continue
    t0="$(date -u +%s)"
    hdr="$(fetch_date_header "${src_trimmed}")" || hdr=""
    t1="$(date -u +%s)"
    if [[ -z "${hdr}" ]]; then
      printf '%-32s %10s  %s\n' "${src_trimmed}" "-" "unreachable / no Date header"
      continue
    fi
    server="$(parse_http_date "${hdr}")" || { printf '%-32s %10s  %s\n' "${src_trimmed}" "-" "unparseable Date: ${hdr}"; continue; }
    local_mid=$(( (t0 + t1) / 2 ))
    delta=$(( local_mid - server ))
    behind=$(( 0 - delta ))
    measured=$(( measured + 1 ))
    if [[ "${behind}" -gt "${worst_behind}" ]]; then worst_behind="${behind}"; fi
    if [[ "${delta}" -gt 0 ]]; then direction="${delta}s ahead"; elif [[ "${delta}" -lt 0 ]]; then direction="${behind}s behind"; else direction="in step"; fi
    printf '%-32s %10s  %s\n' "${src_trimmed}" "${behind}" "${direction}"
  done
fi

echo
if [[ "${measured}" -eq 0 ]]; then
  echo "VERDICT: UNKNOWN -- no time source answered, so this host's skew was not verified." >&2
  echo "         Re-run with a reachable --sources list (or --use-proxy if this host needs one)." >&2
  echo "         Do NOT read this as a pass: with tolerance ${tol_secs}s an unverified clock can still halt the node." >&2
  exit 2
fi

if [[ -n "${force_behind}" ]]; then
  echo "worst offset   : ${worst_behind}s behind (synthetic; no source queried)"
else
  echo "worst offset   : ${worst_behind}s behind (${measured} source(s) answered)"
fi

if [[ "${worst_behind}" -ge "${tol_secs}" ]]; then
  echo "VERDICT: FAIL -- this host is behind by ${worst_behind}s, at or past the ${tol_secs}s tolerance." >&2
  echo "         cometbft will reject every block with" >&2
  echo "         'block time ... is too far in the future (wall clock ... + tolerance ...)'" >&2
  echo "         and this node will stall at one height. Fix time sync (systemd-timesyncd," >&2
  echo "         chrony, or ntp) and re-run before upgrading." >&2
  exit 1
fi

verdict="PASS"
detail="worst offset ${worst_behind}s behind is within the ${tol_secs}s tolerance."
if [[ "${worst_behind}" -gt "${max_skew}" ]]; then
  verdict="PASS (warn)"
  detail="${detail} That is above the ${max_skew}s warn threshold: time sync is not doing its job on this host."
fi
if [[ -n "${force_behind}" ]]; then
  detail="${detail} Offset is synthetic (--force-behind), not measured."
fi
echo "VERDICT: ${verdict} -- ${detail}"
exit 0
