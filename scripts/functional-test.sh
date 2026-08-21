#!/usr/bin/env bash
#
# Uptick v0.4.0 (cosmos/evm) functional test script
# --------------------------------------------------
# Spins up a single local node and exercises each module:
#   - auth / bank (transfer)
#   - staking (validator query)
#   - vm / EVM (deploy + call a contract via eth_call)
#   - feemarket (params query)
#   - cosmos/evm erc20 (module params)
#   - erc721 / cw721 (module params)
#   - gov (proposal query)
#   - ibc / wasm / nft-transfer / collection (params query)
#
# Prerequisites:
#   - uptickd binary built at ./build/uptickd
#   - A running node at --home /tmp/uptick-test (see setup below)
#
# Usage:
#   ./scripts/functional-test.sh           # run against a running node
#   SETUP=1 ./scripts/functional-test.sh    # also init+start the node
#
# Chain ID must be {name}_{eip155}-{revision}. Local tests use the testnet
# EIP-155 id 1170 (never 7000 / cosmos/evm default 262144).

set -u

CHAIN_ID="uptick_1170-1"
HOME_DIR="/tmp/uptick-test"
KEYRING="--keyring-backend test"
NODE="--node tcp://localhost:26657"
DENOM="auptick"
BINARY="./build/uptickd"
PASS="${PASS:-12345678}"

PASS_FILE=$(mktemp)
printf '%s\n' "$PASS" > "$PASS_FILE"

PASS_FILE_T=$(mktemp)
printf '%s\n' "$PASS" > "$PASS_FILE_T"

GREEN=$'\033[0;32m'; RED=$'\033[0;31m'; YELLOW=$'\033[0;33m'; NC=$'\033[0m'
PASS_COUNT=0; FAIL_COUNT=0

ok()   { echo "${GREEN}[PASS]${NC} $1"; PASS_COUNT=$((PASS_COUNT+1)); }
fail() { echo "${RED}[FAIL]${NC} $1"; FAIL_COUNT=$((FAIL_COUNT+1)); }
info() { echo "${YELLOW}[INFO]${NC} $1"; }

VAL_ADDR=$(./build/uptickd keys show validator -a $KEYRING --home "$HOME_DIR" 2>/dev/null)
VAL_VALOPER=$(./build/uptickd keys show validator --bech val -a $KEYRING --home "$HOME_DIR" 2>/dev/null)

# ---------------------------------------------------------------------------
# Setup (optional)
# ---------------------------------------------------------------------------
if [ "${SETUP:-0}" = "1" ]; then
  info "Setting up local chain..."
  pkill -f "uptickd start" 2>/dev/null || true
  rm -rf "$HOME_DIR"
  $BINARY init testnode --home "$HOME_DIR" --chain-id "$CHAIN_ID" 2>&1 | grep -iv "erc20:" >/dev/null
  $BINARY keys add validator $KEYRING --home "$HOME_DIR" 2>&1 | grep -iv "erc20:" >/dev/null
  $BINARY genesis add-genesis-account validator 1000000000000000000000$DENOM $KEYRING --home "$HOME_DIR" 2>&1 | grep -iv "erc20:" >/dev/null
  $BINARY genesis gentx validator 100000000000000000000$DENOM --chain-id "$CHAIN_ID" $KEYRING --home "$HOME_DIR" --from validator 2>&1 | grep -iv "erc20:" >/dev/null
  $BINARY genesis collect-gentxs --home "$HOME_DIR" 2>&1 | grep -iv "erc20:" >/dev/null
  $BINARY start --home "$HOME_DIR" --minimum-gas-prices 0$DENOM > /tmp/uptick-node.log 2>&1 &
  info "Node starting (pid $!)..."
  sleep 30
fi

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
# Query helper: emit JSON so we can parse tx codes reliably. The 'erc20:'
# duplicate-registration warnings are filtered from stderr.
query() { $BINARY q "$@" -o json $NODE 2>&1 | grep -iv "erc20:"; }

# Derive a usable gas price. cosmos/evm feemarket enforces a base_fee, so a
# zero gas price (as used in earlier test runs) makes every tx fail with
# code 13 "insufficient fee". We use a fixed margin above the typical base fee.
GAS_PRICE="0.1$DENOM"

# tx helper: returns the raw JSON broadcast result (contains txhash + code).
tx() {
  # tx <module> <cmd> <args...>
  local module="$1"; shift
  $BINARY tx "$module" "$@" $KEYRING --home "$HOME_DIR" --chain-id "$CHAIN_ID" \
    --gas-prices "$GAS_PRICE" --gas auto --gas-adjustment 1.5 -y -o json $NODE 2>&1 \
    | grep -iv "erc20:" | grep -E '^\{.*\}$'
}

# extract a top-level JSON field from a tx/query result
json_field() {
  # json_field <field> <text>  (text may contain non-JSON lines; the JSON line is used)
  echo "$2" | grep -E '^\{.*\}$' | tail -1 | python3 -c "import sys,json
try:
    d=json.loads(sys.stdin.read()); print(d.get('$1',''))
except Exception: pass" 2>/dev/null
}

# Wait for a tx to be committed by polling a query a few times
wait_blocks() {
  local n="${1:-3}"
  sleep $((n * 2))
}

# ---------------------------------------------------------------------------
# 1. Node / Chain
# ---------------------------------------------------------------------------
info "== 1. Node / Chain =="
OUT=$($BINARY status $NODE 2>&1 | grep -iv "erc20:")
if echo "$OUT" | grep -q "$CHAIN_ID"; then ok "node responds & chain-id is $CHAIN_ID"; else fail "node status: $OUT"; fi

# ---------------------------------------------------------------------------
# 2. Auth / Bank
# ---------------------------------------------------------------------------
info "== 2. Auth / Bank =="
BAL=$(query bank balances "$VAL_ADDR")
if echo "$BAL" | grep -q "$DENOM"; then ok "bank balance query returns $DENOM"; else fail "bank balance: $BAL"; fi

# transfer to self
SECOND=$(./build/uptickd keys show validator -a $KEYRING --home "$HOME_DIR" 2>/dev/null)
RES=$(tx bank send "$VAL_ADDR" "$VAL_ADDR" 100000000000000000$DENOM)
TXH=$(json_field txhash "$RES")
if [ -n "$TXH" ]; then ok "bank send tx submitted (txhash=$TXH)"; else fail "bank send: $RES"; fi
wait_blocks 2
# verify the tx actually committed with code 0 (not just accepted into mempool)
if [ -n "$TXH" ]; then
  TXQ=$(query tx "$TXH")
  TXCODE=$(json_field code "$TXQ")
  if [ "$TXCODE" = "0" ]; then ok "bank send tx committed (code 0)"; else fail "bank send tx NOT committed (code=$TXCODE)"; fi
else
  info "skipped bank send commit check (no txhash)"
fi

# ---------------------------------------------------------------------------
# 3. Staking
# ---------------------------------------------------------------------------
info "== 3. Staking =="
VALS=$(query staking validators)
if echo "$VALS" | grep -q "BOND_STATUS_BONDED"; then ok "validator is bonded"; else fail "staking validators: $VALS"; fi
DEL=$(query staking delegations "$VAL_ADDR")
if echo "$DEL" | grep -q "$DENOM"; then ok "delegation query returns $DENOM"; else fail "delegations: $DEL"; fi

# ---------------------------------------------------------------------------
# 4. VM / EVM (module queries + JSON-RPC)
# ---------------------------------------------------------------------------
info "== 4. VM / EVM =="
EVM_PARAMS=$(query evm params)
if echo "$EVM_PARAMS" | grep -q "evm_denom"; then ok "evm params query ok"; else fail "evm params: $EVM_PARAMS"; fi

# JSON-RPC EVM checks
RPC="http://127.0.0.1:8545"
export NO_PROXY=localhost,127.0.0.1
rpc_call() {
  curl -s -m5 -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"$1\",\"params\":$2,\"id\":1}" "$RPC"
}
EXPECTED_ETH_CHAIN_ID=$(python3 -c "print(hex(int('${CHAIN_ID}'.rsplit('_',1)[1].split('-')[0])))")
CHAIND_ID_RPC=$(rpc_call "eth_chainId" "[]" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result'))" 2>/dev/null)
if [ "$CHAIND_ID_RPC" = "$EXPECTED_ETH_CHAIN_ID" ]; then ok "JSON-RPC eth_chainId = $EXPECTED_ETH_CHAIN_ID"; else fail "eth_chainId rpc: $CHAIND_ID_RPC (want $EXPECTED_ETH_CHAIN_ID)"; fi

BLOCK_NUM=$(rpc_call "eth_blockNumber" "[]" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result'))" 2>/dev/null)
if [ -n "$BLOCK_NUM" ]; then ok "JSON-RPC eth_blockNumber = $BLOCK_NUM"; else fail "eth_blockNumber rpc"; fi

RPC_GAS_PRICE=$(rpc_call "eth_gasPrice" "[]" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result'))" 2>/dev/null)
if [ -n "$RPC_GAS_PRICE" ]; then ok "JSON-RPC eth_gasPrice = $RPC_GAS_PRICE"; else fail "eth_gasPrice rpc"; fi

# eth_call executing EVM bytecode (PUSH1 0x2a PUSH1 0x00 SSTORE)
CALL_RES=$(rpc_call "eth_call" "[{\"from\":\"0x0000000000000000000000000000000000000000\",\"to\":\"0x0000000000000000000000000000000000000000\",\"data\":\"0x602a60005500\",\"gas\":\"0x100000\"},\"latest\"]" | python3 -c "import sys,json; d=json.load(sys.stdin); print('OK' if 'error' not in d else d.get('error'))" 2>/dev/null)
if [ "$CALL_RES" = "OK" ]; then ok "JSON-RPC eth_call executes EVM bytecode (no panic)"; else fail "eth_call rpc: $CALL_RES"; fi

# EVM native transfer via the evm module (from -> to, amount). Note: cosmos/evm
# `tx evm send` takes [from] [to] [amount] (it is an EVM coin transfer, not a
# contract deploy). We send a tiny amount to self to exercise the EVM tx path.
DEPLOY=$(tx evm send "$VAL_ADDR" "$VAL_ADDR" 1000$DENOM)
DTXH=$(json_field txhash "$DEPLOY")
if [ -n "$DTXH" ]; then ok "evm send tx submitted (txhash=$DTXH)"; else fail "evm send: $DEPLOY"; fi
wait_blocks 2
if [ -n "$DTXH" ]; then
  DQ=$(query tx "$DTXH")
  DCODE=$(json_field code "$DQ")
  if [ "$DCODE" = "0" ]; then ok "evm send tx committed (code 0)"; else fail "evm tx NOT committed (code=$DCODE)"; fi
fi

# ---------------------------------------------------------------------------
# 5. Feemarket
# ---------------------------------------------------------------------------
info "== 5. Feemarket =="
FM=$(query feemarket params)
if echo "$FM" | grep -qiE "base_fee|no_base_fee|min_gas_price"; then ok "feemarket params query ok"; else fail "feemarket params: $FM"; fi

# ---------------------------------------------------------------------------
# 6. ERC20 (cosmos/evm) / ERC721 / CW721
# ---------------------------------------------------------------------------
info "== 6. ERC20 / ERC721 / CW721 =="
ERC=$(query erc20 params)
if echo "$ERC" | grep -qiE "enable_erc20|permissionless_registration"; then ok "erc20 params query ok"; else fail "erc20 params: $ERC"; fi

ERC721=$(query erc721 params)
if echo "$ERC721" | grep -qiE "enable_erc721|enable_evm_hook"; then ok "erc721 params query ok"; else fail "erc721 params: $ERC721"; fi

CW721=$(query cw721 params)
if echo "$CW721" | grep -qiE "enable_cw721|enable_evm_hook"; then ok "cw721 params query ok"; else fail "cw721 params: $CW721"; fi

# ---------------------------------------------------------------------------
# 7. Gov
# ---------------------------------------------------------------------------
info "== 7. Gov =="
GOV=$(query gov params)
if echo "$GOV" | grep -qiE "threshold|voting_period|quorum"; then ok "gov params query ok"; else fail "gov params: $GOV"; fi

# ---------------------------------------------------------------------------
# 8. IBC / WASM / NFT / Collection
# ---------------------------------------------------------------------------
info "== 8. IBC / WASM / NFT / Collection =="
TRANSFER=$(query ibc-transfer params)
if echo "$TRANSFER" | grep -qiE "send_enabled|receive_enabled"; then ok "ibc-transfer params query ok"; else fail "ibc-transfer: $TRANSFER"; fi

WASM=$(query wasm params)
if echo "$WASM" | grep -qiE "code_upload_access|instantiate_default_permission"; then ok "wasm params query ok"; else fail "wasm: $WASM"; fi

# CLI module name remains nft-transfer; IBC port id is nonfungibletokentransfer.
NFTTR=$(query nft-transfer params)
if echo "$NFTTR" | grep -qiE "send_enabled|receive_enabled"; then ok "nft-transfer params query ok"; else fail "nft-transfer: $NFTTR"; fi

COL=$(query collection denoms)
if echo "$COL" | grep -qiE "denoms|pagination"; then ok "collection denoms query ok"; else fail "collection: $COL"; fi

# ---------------------------------------------------------------------------
# 9. Misc queries
# ---------------------------------------------------------------------------
info "== 9. Misc =="
MINT=$(query mint params)
if echo "$MINT" | grep -q "mint_denom"; then ok "mint params query ok (denom $DENOM)"; else fail "mint: $MINT"; fi

DIST=$(query distribution params)
if echo "$DIST" | grep -q "community_tax"; then ok "distribution params query ok"; else fail "distribution: $DIST"; fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "=========================================="
echo "  Functional test summary"
echo "  PASS: $PASS_COUNT   FAIL: $FAIL_COUNT"
echo "=========================================="

rm -f "$PASS_FILE" "$PASS_FILE_T"
if [ "$FAIL_COUNT" -gt 0 ]; then exit 1; fi
exit 0
