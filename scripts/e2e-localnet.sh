#!/usr/bin/env bash
#
# Single-node localnet for the end-to-end suite (audit G-06).
#
# The e2e tests need a REAL node: they broadcast Keplr-style signed transactions
# over the REST gateway and probe the EVM JSON-RPC endpoint. `go test ./...` can
# never prove those paths work, so the CI e2e job starts this node, exports
# UPTICK_E2E_STRICT=1 and runs tests/e2e - where a missing node is a failure
# instead of a skip.
#
# Usage:
#   ./scripts/e2e-localnet.sh start    # init + start in the background, wait for REST/JSON-RPC
#   ./scripts/e2e-localnet.sh stop     # stop the node started by `start`
#   ./scripts/e2e-localnet.sh status   # show pid, log tail and latest height
#
# Environment (defaults match tests/e2e):
#   UPTICK_E2E_HOME         node home AND the test keyring dir  (default /tmp/uptick-keplr)
#   UPTICK_E2E_CHAIN_ID     bech32 chain id                     (default uptick_1170-1)
#   UPTICK_E2E_DENOM        base denom                          (default auptick)
#   UPTICK_E2E_REST         REST gateway                        (default http://127.0.0.1:1317)
#   UPTICK_E2E_RPC          EVM JSON-RPC                        (default http://127.0.0.1:8545)
#   UPTICKD                 uptickd binary                      (default uptickd on PATH)
#
# The key is named after UPTICK_E2E_KEY_NAME because tests/e2e looks it up by
# that name in the test keyring under UPTICK_E2E_HOME.
set -euo pipefail

HOME_DIR="${UPTICK_E2E_HOME:-/tmp/uptick-keplr}"
CHAIN_ID="${UPTICK_E2E_CHAIN_ID:-uptick_1170-1}"
DENOM="${UPTICK_E2E_DENOM:-auptick}"
REST="${UPTICK_E2E_REST:-http://127.0.0.1:1317}"
RPC="${UPTICK_E2E_RPC:-http://127.0.0.1:8545}"
BIN="${UPTICKD:-uptickd}"

KEY_NAME="${UPTICK_E2E_KEY_NAME:-keplr}"
KEYRING_BACKEND="${UPTICK_E2E_KEYRING_BACKEND:-test}"
# 1e26: enough for the suite's fee of 1e15 per transaction with room to spare.
GENESIS_BALANCE="${UPTICK_E2E_GENESIS_BALANCE:-100000000000000000000000000${DENOM}}"
GENTX_STAKE="${UPTICK_E2E_GENTX_STAKE:-1000000000000000000000${DENOM}}"
MIN_GAS_PRICES="${UPTICK_E2E_MIN_GAS_PRICES:-1000000000${DENOM}}"

PID_FILE="$HOME_DIR/e2e-localnet.pid"
LOG_FILE="$HOME_DIR/e2e-localnet.log"
STARTUP_TIMEOUT="${UPTICK_E2E_STARTUP_TIMEOUT:-180}"

log() { printf '[e2e-localnet] %s\n' "$*" >&2; }

# GNU sed and BSD sed disagree about the argument order of -i; CI is Linux but
# the same script is meant to be usable on a developer's macOS checkout.
replace_in_file() {
	local expr="$1" file="$2"
	if sed --version >/dev/null 2>&1; then
		sed -i "$expr" "$file"
	else
		sed -i '' "$expr" "$file"
	fi
}

die() {
	log "$*"
	exit 1
}

node_pid() {
	[ -f "$PID_FILE" ] && cat "$PID_FILE"
}

node_running() {
	local pid
	pid="$(node_pid)" || return 1
	[ -n "$pid" ] && kill -0 "$pid" 2>/dev/null
}

dump_log() {
	local lines="${1:-60}"
	if [ -f "$LOG_FILE" ]; then
		log "last $lines lines of $LOG_FILE:"
		tail -n "$lines" "$LOG_FILE" >&2 || true
	fi
}

require_bin() {
	command -v "$BIN" >/dev/null 2>&1 || die "$BIN not found on PATH; build it (make build) or set UPTICKD"
}

rest_ready() {
	curl -fsS -m 3 "$REST/cosmos/base/tendermint/v1beta1/node_info" >/dev/null 2>&1
}

jsonrpc_ready() {
	curl -fsS -m 3 -X POST -H 'Content-Type: application/json' \
		-d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
		"$RPC" >/dev/null 2>&1
}

# wait_until blocks until the probe succeeds, the node dies, or the deadline
# passes. `what` names the endpoint for the failure message.
wait_until() {
	local what="$1" probe="$2"
	local deadline=$((SECONDS + STARTUP_TIMEOUT))

	while [ "$SECONDS" -lt "$deadline" ]; do
		if "$probe"; then
			log "$what is up"
			return 0
		fi
		if ! node_running; then
			dump_log
			die "the node exited before $what became reachable"
		fi
		sleep 2
	done

	dump_log
	# Stop the node we started before reporting the timeout, so a failed
	# startup does not leave an orphan holding the REST/JSON-RPC ports.
	# (This used to call an undefined `stop`, which made a timeout exit 127
	# with "command not found" instead of the diagnostic below.)
	cmd_stop
	die "timed out after ${STARTUP_TIMEOUT}s waiting for $what"
}

cmd_start() {
	require_bin

	if node_running; then
		die "a node is already running (pid $(node_pid)); run 'stop' first"
	fi

	log "resetting $HOME_DIR"
	rm -rf "$HOME_DIR"
	mkdir -p "$HOME_DIR"

	# `init` already writes the auptick denom everywhere it matters: the default
	# bond denom comes from --default-denom, and CustomizeDefaultGenesis sets
	# evm.params.evm_denom plus the matching bank denom metadata. No genesis
	# post-processing is needed.
	log "initialising $CHAIN_ID in $HOME_DIR"
	"$BIN" init e2e-localnet --chain-id "$CHAIN_ID" --home "$HOME_DIR" --overwrite >/dev/null

	log "creating the $KEY_NAME key (tests/e2e signs with this one)"
	"$BIN" keys add "$KEY_NAME" \
		--home "$HOME_DIR" --keyring-backend "$KEYRING_BACKEND" --algo eth_secp256k1 >/dev/null

	"$BIN" add-genesis-account "$KEY_NAME" "$GENESIS_BALANCE" \
		--home "$HOME_DIR" --keyring-backend "$KEYRING_BACKEND" >/dev/null
	"$BIN" gentx "$KEY_NAME" "$GENTX_STAKE" \
		--chain-id "$CHAIN_ID" --home "$HOME_DIR" --keyring-backend "$KEYRING_BACKEND" >/dev/null
	"$BIN" collect-gentxs --home "$HOME_DIR" >/dev/null
	"$BIN" validate-genesis --home "$HOME_DIR" >/dev/null

	# The suite runs while the chain is otherwise idle: empty blocks must keep
	# coming (the JSON-RPC smoke test asserts the height advances) and 1s
	# commits keep the whole job quick.
	replace_in_file 's/^create_empty_blocks = false/create_empty_blocks = true/' "$HOME_DIR/config/config.toml"
	replace_in_file 's/^timeout_commit = .*/timeout_commit = "1s"/' "$HOME_DIR/config/config.toml"

	log "starting the node (log: $LOG_FILE)"
	nohup "$BIN" start \
		--home "$HOME_DIR" \
		--api.enable \
		--json-rpc.enable \
		--json-rpc.api=eth,net,web3 \
		--minimum-gas-prices "$MIN_GAS_PRICES" \
		--log_level info \
		>"$LOG_FILE" 2>&1 &
	echo $! >"$PID_FILE"

	wait_until "REST at $REST" rest_ready
	wait_until "JSON-RPC at $RPC" jsonrpc_ready

	log "ready: home=$HOME_DIR chain-id=$CHAIN_ID rest=$REST rpc=$RPC"
}

cmd_stop() {
	local pid
	pid="$(node_pid)" || { log "no pid file; nothing to stop"; return 0; }

	if [ -z "$pid" ]; then
		log "empty pid file; nothing to stop"
		rm -f "$PID_FILE"
		return 0
	fi

	if kill -0 "$pid" 2>/dev/null; then
		log "stopping pid $pid"
		kill "$pid" 2>/dev/null || true
		for _ in $(seq 1 30); do
			kill -0 "$pid" 2>/dev/null || break
			sleep 1
		done
		kill -9 "$pid" 2>/dev/null || true
	fi
	rm -f "$PID_FILE"
	log "stopped"
}

cmd_status() {
	if node_running; then
		log "running: pid $(node_pid)"
	else
		log "not running"
	fi
	if rest_ready; then
		log "REST height: $(curl -fsS -m 3 "$REST/cosmos/base/tendermint/v1beta1/blocks/latest" || echo '?')"
	else
		log "REST not reachable at $REST"
	fi
	dump_log 10
}

case "${1:-}" in
start) cmd_start ;;
stop) cmd_stop ;;
status) cmd_status ;;
*) die "usage: $0 {start|stop|status}" ;;
esac
