#!/usr/bin/env bash
# check-entitlement-pool-funding.sh — preflight for UGE V1 entitlement enable gate.
#
# Usage:
#   ./scripts/check-entitlement-pool-funding.sh <node_rpc> <pool_id> [min_funded]
#
# Compares QueryPool.funded / module_balance against entitlement_enable_min_funded
# (or an explicit min_funded override). Exit 0 when the V1 flow may be enabled;
# exit 1 on shortfall; exit 2 on query failure.
#
# Does not mint or transfer. Report any gap to chain ops; keep entitlement msgs disabled.

set -euo pipefail

RPC="${1:-}"
POOL_ID="${2:-}"
MIN_OVERRIDE="${3:-}"

if [[ -z "$RPC" || -z "$POOL_ID" ]]; then
  echo "usage: $0 <node_rpc> <pool_id> [min_funded_smallest_units]" >&2
  exit 2
fi

BIN="${UPTICKD_BIN:-uptickd}"

PARAMS_JSON="$("$BIN" q appmining params --node "$RPC" -o json)"
MIN="${MIN_OVERRIDE}"
if [[ -z "$MIN" ]]; then
  MIN="$(echo "$PARAMS_JSON" | jq -r '.params.entitlement_enable_min_funded // empty')"
fi
if [[ -z "$MIN" || "$MIN" == "0" ]]; then
  echo "FAIL: entitlement_enable_min_funded unset or zero — V1 flow stays disabled" >&2
  echo "gap: configure params.entitlement_enable_min_funded (e.g. 5e8 * 10^decimals)" >&2
  exit 1
fi

POOL_JSON="$("$BIN" q appmining pool "$POOL_ID" --node "$RPC" -o json)"
FUNDED="$(echo "$POOL_JSON" | jq -r '.pool.funded')"
AVAILABLE="$(echo "$POOL_JSON" | jq -r '.pool.available')"
RESERVED="$(echo "$POOL_JSON" | jq -r '.pool.reserved')"
PAID="$(echo "$POOL_JSON" | jq -r '.pool.paid')"
MOD_BAL="$(echo "$POOL_JSON" | jq -r '.module_balance')"
ENABLED="$(echo "$POOL_JSON" | jq -r '.entitlement_flow_enabled')"

echo "pool_id=$POOL_ID funded=$FUNDED available=$AVAILABLE reserved=$RESERVED paid=$PAID module_balance=$MOD_BAL min=$MIN enabled=$ENABLED"

if [[ "$ENABLED" != "true" ]]; then
  echo "FAIL: entitlement_flow_enabled=false" >&2
  echo "gap: funded ($FUNDED) < min ($MIN) and/or pool paused / module balance < reserved" >&2
  exit 1
fi

echo "OK: V1 entitlement flow enable gate satisfied"
exit 0
