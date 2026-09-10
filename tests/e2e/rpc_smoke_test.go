package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/app/params"
)

// This file is the "is the chain actually wired up" gate the audit asked for
// (G-06). Unit tests construct keepers directly, so nothing there can tell
// whether the module manager, the REST gateway, the IBC router or the EVM
// JSON-RPC server are reachable on a running node. These tests need a real
// node; under UPTICK_E2E_STRICT=1 a missing node fails instead of skipping, so
// the CI e2e job can never go green by doing nothing.

func e2eRPCURL() string {
	return envOr("UPTICK_E2E_RPC", "http://127.0.0.1:8545")
}

func summarizeBody(body []byte) string {
	const limit = 200
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "..."
}

// httpStatus performs a GET and returns the status code together with the body.
func httpStatus(t *testing.T, url string) (int, []byte) {
	t.Helper()

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(url)
	require.NoError(t, err, "GET %s failed", url)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, buf.Bytes()
}

// jsonRPC performs one JSON-RPC call and returns the raw result.
func jsonRPC(t *testing.T, method string, params ...interface{}) json.RawMessage {
	t.Helper()

	if params == nil {
		params = []interface{}{}
	}
	payload, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, e2eRPCURL(), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	require.NoError(t, err, "JSON-RPC %s: is the node listening on %s?", method, e2eRPCURL())
	defer resp.Body.Close()

	var out struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Nilf(t, out.Error, "JSON-RPC %s returned an error: %+v", method, out.Error)
	return out.Result
}

func hexUint64(t *testing.T, raw json.RawMessage) uint64 {
	t.Helper()

	var s string
	require.NoError(t, json.Unmarshal(raw, &s))
	value, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	require.NoErrorf(t, err, "not a hex quantity: %q", s)
	return value
}

// TestNodeModuleRoutesSmoke walks every module route the audit is about. A
// missing route means the module is not reachable on a real node - keeper
// wiring, module ordering and the IBC router are exactly the things unit tests
// cannot check.
func TestNodeModuleRoutesSmoke(t *testing.T) {
	_, rest := e2eNode(t)

	// The node must serve the chain id the suite signs for.
	body := httpGet(t, rest+"/cosmos/base/tendermint/v1beta1/node_info")
	var nodeInfo struct {
		DefaultNodeInfo struct {
			Network string `json:"network"`
		} `json:"default_node_info"`
	}
	require.NoError(t, json.Unmarshal(body, &nodeInfo))
	require.Equal(t, e2eChainID, nodeInfo.DefaultNodeInfo.Network)

	// Each entry is a route plus a field it must return. Probing for the field
	// keeps the assertion about the module answering, not about its contents.
	routes := []struct {
		path  string
		field string
	}{
		{"/uptick/collection/nft/denoms", "denoms"},            // x/collection
		{"/uptick/erc721/v1/params", "params"},                 // x/erc721
		{"/uptick/erc721/v1/token_pairs", "token_pairs"},       // x/erc721
		{"/uptick/cw721/v1/params", "params"},                  // x/cw721
		{"/uptick/cw721/v1/token_pairs", "token_pairs"},        // x/cw721
		{"/ibc/apps/transfer/v1/params", "params"},             // ICS-20 router
		{"/ibc/core/client/v1/client_states", "client_states"}, // ICS-02 client
		{"/ibc/core/channel/v1/channels", "channels"},          // ICS-04 channel
		{"/cosmos/evm/vm/v1/params", "params"},                 // cosmos/evm
		{"/cosmos/evm/vm/v1/config", "config"},                 // eth chain config
		{"/cosmos/evm/vm/v1/base_fee", "base_fee"},             // feemarket
		{"/cosmos/evm/vm/v1/min_gas_price", "min_gas_price"},   // feemarket
	}

	for _, route := range routes {
		status, payload := httpStatus(t, rest+route.path)
		require.Equalf(t, http.StatusOK, status, "GET %s returned %d: %s",
			route.path, status, summarizeBody(payload))

		var decoded map[string]json.RawMessage
		require.NoErrorf(t, json.Unmarshal(payload, &decoded),
			"GET %s did not return a JSON object: %s", route.path, summarizeBody(payload))
		require.Containsf(t, decoded, route.field,
			"GET %s is missing the %q field: %s", route.path, route.field, summarizeBody(payload))
	}

	// The EVM module must report the chain's own denom, not a stock default.
	_, payload := httpStatus(t, rest+"/cosmos/evm/vm/v1/params")
	var evmParams struct {
		Params struct {
			EvmDenom string `json:"evm_denom"`
		} `json:"params"`
	}
	require.NoError(t, json.Unmarshal(payload, &evmParams))
	require.Equal(t, e2eDenom, evmParams.Params.EvmDenom,
		"the EVM denom must be the chain denom, otherwise every EVM amount is mis-scaled")
}

// TestEVMJSONRPCSmoke covers the JSON-RPC surface the audit calls out as
// ungated: the reported chain id, block production and the account mapping
// between the bank and the EVM state.
func TestEVMJSONRPCSmoke(t *testing.T) {
	home, _ := e2eNode(t)

	require.Equal(t, e2eEVMChainID, hexUint64(t, jsonRPC(t, "eth_chainId")),
		"eth_chainId must be derived from the chain id; a wrong value breaks every EIP-155 signer")

	var netVersion string
	require.NoError(t, json.Unmarshal(jsonRPC(t, "net_version"), &netVersion))
	require.Equal(t, strconv.FormatUint(e2eEVMChainID, 10), netVersion)

	var clientVersion string
	require.NoError(t, json.Unmarshal(jsonRPC(t, "web3_clientVersion"), &clientVersion))
	require.NotEmpty(t, clientVersion)

	var syncing bool
	require.NoError(t, json.Unmarshal(jsonRPC(t, "eth_syncing"), &syncing))
	require.False(t, syncing, "the single-node localnet must not be catching up")

	// The chain must keep producing blocks while the suite is otherwise idle,
	// otherwise the broadcast path has nothing to include a transaction in.
	first := hexUint64(t, jsonRPC(t, "eth_blockNumber"))
	require.Positive(t, first, "the node must have produced at least one block")

	advanced := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if hexUint64(t, jsonRPC(t, "eth_blockNumber")) > first {
			advanced = true
			break
		}
		time.Sleep(time.Second)
	}
	require.Truef(t, advanced, "block height stayed at %d: the node stopped producing blocks", first)

	// eth_getBalance reads the EVM view of the account the suite signs with. A
	// zero balance would mean the Cosmos -> EVM account mapping is broken, which
	// would also break the ERC721/EVM paths.
	enc := params.MakeEncodingConfig()
	privKey := loadKeplrPrivKey(t, home, enc.Codec)
	address := common.BytesToAddress(sdk.AccAddress(privKey.PubKey().Address().Bytes()))

	var balance string
	require.NoError(t, json.Unmarshal(jsonRPC(t, "eth_getBalance", address.Hex(), "latest"), &balance))
	require.NotEqual(t, "0x0", balance, "the funded account must have an EVM balance")
	require.NotEqual(t, "0x", balance)
}
