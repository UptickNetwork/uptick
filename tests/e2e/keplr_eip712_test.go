package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/evm/ethereum/eip712"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/UptickNetwork/uptick/app/params"
	cmdcfg "github.com/UptickNetwork/uptick/cmd/config"
)

const (
	e2eDefaultChainID    = "uptick_1170-1"
	e2eDefaultEVMChainID = uint64(1170)
	e2eDefaultDenom      = "auptick"
	e2eDefaultFee        = int64(1000000000000000)
)

// Overridable so the suite can run against local test chains whose
// chain-id/denom differ from the shared testnet defaults.
var (
	e2eChainID    = envOr("UPTICK_E2E_CHAIN_ID", e2eDefaultChainID)
	e2eEVMChainID = uint64(envIntOr("UPTICK_E2E_EVM_CHAIN_ID", int64(e2eDefaultEVMChainID)))
	e2eDenom      = envOr("UPTICK_E2E_DENOM", e2eDefaultDenom)
	e2eFeeAmount  = math.NewInt(envIntOr("UPTICK_E2E_FEE", e2eDefaultFee))
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return def
}

func init() {
	cmdcfg.SetBech32Prefixes(sdk.GetConfig())
	cmdcfg.RegisterDenoms()
}

// e2eNode resolves the node the suite talks to.
//
// Under UPTICK_E2E_STRICT=1 an unreachable node is a hard failure. The
// dedicated e2e CI job starts a real node and sets that variable precisely so
// that a green run always means the node was really exercised - a suite that
// silently skips itself proves nothing about Keplr broadcast paths.
func e2eNode(t *testing.T) (home, rest string) {
	t.Helper()

	home = envOr("UPTICK_E2E_HOME", "/tmp/uptick-keplr")
	rest = envOr("UPTICK_E2E_REST", "http://127.0.0.1:1317")

	if !restReachable(rest) {
		if os.Getenv("UPTICK_E2E_STRICT") == "1" {
			t.Fatalf("no uptick node reachable at %s; UPTICK_E2E_STRICT=1 forbids skipping this suite", rest)
		}
		t.Skipf("no uptick node reachable at %s; set UPTICK_E2E_REST to run the e2e suite", rest)
	}
	return home, rest
}

func TestKeplrEip712TransferEndToEnd(t *testing.T) {
	home, rest := e2eNode(t)

	enc := params.MakeEncodingConfig()
	banktypes.RegisterInterfaces(enc.InterfaceRegistry)
	banktypes.RegisterLegacyAminoCodec(enc.LegacyAmino)
	legacytx.RegressionTestingAminoCodec = enc.LegacyAmino

	privKey := loadKeplrPrivKey(t, home, enc.Codec)
	pubKey := privKey.PubKey().(*ethsecp256k1.PubKey)
	from := sdk.AccAddress(pubKey.Address().Bytes())

	accNum, seq := queryAccount(t, rest, from.String())
	t.Logf("keplr account=%s account_number=%d sequence=%d", from.String(), accNum, seq)

	msgs := []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: from.String(),
			ToAddress:   from.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin(e2eDenom, 1)),
		},
	}
	fee := legacytx.StdFee{
		// The local feemarket base fee is 1e9 auptick, so provide a gas price
		// comfortably above it.
		Amount: sdk.NewCoins(sdk.NewCoin(e2eDenom, e2eFeeAmount)),
		Gas:    200000,
	}

	txRaw := buildSignedKeplrEip712Tx(t, enc, privKey, pubKey, from, msgs, fee, accNum, seq)
	txRawBz, err := enc.Codec.Marshal(txRaw)
	require.NoError(t, err)

	txHash := broadcastTx(t, rest, base64.StdEncoding.EncodeToString(txRawBz))
	t.Logf("broadcast txhash=%s", txHash)

	require.Eventually(t, func() bool {
		code, height := queryTxResult(t, rest, txHash)
		return height != "" && code >= 0
	}, 20*time.Second, 500*time.Millisecond, "tx %s was not committed", txHash)

	code, _ := queryTxResult(t, rest, txHash)
	require.Zero(t, code, "keplr eip712 transfer failed with code %d", code)
}

// TestKeplrDirectTransferEndToEnd validates the flow used by the current Keplr
// dApp path (window.keplr.getOfflineSigner + @cosmjs/stargate): SIGN_MODE_DIRECT
// with the legacy /ethermint.crypto.v1.ethsecp256k1.PubKey public key and no
// Web3 extension option.
//
// This path was previously skipped unconditionally, so nothing ever exercised
// it. It now honours the same node/strict contract as the EIP-712 test.
func TestKeplrDirectTransferEndToEnd(t *testing.T) {
	home, rest := e2eNode(t)

	enc := params.MakeEncodingConfig()
	banktypes.RegisterInterfaces(enc.InterfaceRegistry)
	banktypes.RegisterLegacyAminoCodec(enc.LegacyAmino)
	legacytx.RegressionTestingAminoCodec = enc.LegacyAmino

	privKey := loadKeplrPrivKey(t, home, enc.Codec)
	pubKey := privKey.PubKey().(*ethsecp256k1.PubKey)
	from := sdk.AccAddress(pubKey.Address().Bytes())

	// The SIGN_MODE_DIRECT construction lives in signDirectAndBroadcast so
	// there is exactly one implementation of it. Keeping a second copy here
	// meant a fix to the signing path could land in one place and be missed in
	// the other - the failure mode the audit found twice in the erc721/cw721
	// pair, and the reason this test now shares the collection lifecycle
	// suite's code instead of duplicating it.
	signDirectAndBroadcast(t, enc, rest, privKey, pubKey, from, &banktypes.MsgSend{
		FromAddress: from.String(),
		ToAddress:   from.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin(e2eDenom, 1)),
	}, 200_000)
}

func buildSignedKeplrEip712Tx(
	t *testing.T,
	enc params.EncodingConfig,
	privKey cryptotypes.PrivKey,
	pubKey *ethsecp256k1.PubKey,
	from sdk.AccAddress,
	msgs []sdk.Msg,
	fee legacytx.StdFee,
	accNum, seq uint64,
) *txtypes.TxRaw {
	t.Helper()

	signBytes := legacytx.StdSignBytes(e2eChainID, accNum, seq, 0, fee, msgs, "")
	feeDelegation := &eip712.FeeDelegationOptions{FeePayer: from}
	typedData, err := eip712.LegacyWrapTxToTypedData(enc.Codec, e2eEVMChainID, msgs[0], signBytes, feeDelegation)
	require.NoError(t, err)

	sigHash, _, err := apitypes.TypedDataAndHash(typedData)
	require.NoError(t, err)
	sig, err := privKey.Sign(sigHash)
	require.NoError(t, err)
	sig[ethcrypto.RecoveryIDOffset] += 27

	msgAny, err := codectypes.NewAnyWithValue(msgs[0])
	require.NoError(t, err)
	extAny := newLegacyAny(t, "/ethermint.types.v1.ExtensionOptionsWeb3Tx", &eip712.ExtensionOptionsWeb3Tx{
		TypedDataChainID: e2eEVMChainID,
		FeePayer:         from.String(),
		FeePayerSig:      sig,
	})
	pubAny := newLegacyAny(t, "/ethermint.crypto.v1.ethsecp256k1.PubKey", &ethsecp256k1.PubKey{Key: pubKey.Bytes()})

	body := &txtypes.TxBody{
		Messages:         []*codectypes.Any{msgAny},
		ExtensionOptions: []*codectypes.Any{extAny},
	}
	bodyBz, err := enc.Codec.Marshal(body)
	require.NoError(t, err)

	authInfo := &txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{
			{
				PublicKey: pubAny,
				ModeInfo: &txtypes.ModeInfo{
					Sum: &txtypes.ModeInfo_Single_{
						Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON},
					},
				},
				Sequence: seq,
			},
		},
		Fee: &txtypes.Fee{
			Amount:   fee.Amount,
			GasLimit: fee.Gas,
			Payer:    from.String(),
		},
	}
	authInfoBz, err := enc.Codec.Marshal(authInfo)
	require.NoError(t, err)

	return &txtypes.TxRaw{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		Signatures:    [][]byte{{}},
	}
}

func loadKeplrPrivKey(t *testing.T, home string, cdc codec.Codec) cryptotypes.PrivKey {
	t.Helper()

	// The service name decides the on-disk keyring directory
	// (<home>/<service>-test), so it must match the one the CLI uses. The CLI
	// goes through sdk.KeyringServiceName(); hardcoding a different name here
	// made the key invisible and would strand this test on first contact with a
	// real node.
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, home, nil, cdc)
	require.NoError(t, err)
	rec, err := kr.Key("keplr")
	require.NoError(t, err, "no %q key in the test keyring under %s", "keplr", home)
	local := rec.GetLocal()
	require.NotNil(t, local, "keplr key is not a local key")
	priv, ok := local.PrivKey.GetCachedValue().(cryptotypes.PrivKey)
	require.True(t, ok, "failed to extract keplr private key")
	return priv
}

func restReachable(rest string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(rest + "/cosmos/base/tendermint/v1beta1/node_info")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func queryAccount(t *testing.T, rest, address string) (accNum, seq uint64) {
	t.Helper()
	body := httpGet(t, rest+"/cosmos/auth/v1beta1/accounts/"+address)
	var out struct {
		Account struct {
			AccountNumber string `json:"account_number"`
			Sequence      string `json:"sequence"`
		} `json:"account"`
	}
	require.NoError(t, json.Unmarshal(body, &out))
	_, err := fmt.Sscanf(out.Account.AccountNumber, "%d", &accNum)
	require.NoError(t, err)
	_, err = fmt.Sscanf(out.Account.Sequence, "%d", &seq)
	require.NoError(t, err)
	return accNum, seq
}

func broadcastTx(t *testing.T, rest, txBase64 string) string {
	t.Helper()
	payload := map[string]string{
		"tx_bytes": txBase64,
		"mode":     "BROADCAST_MODE_SYNC",
	}
	bz, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, rest+"/cosmos/tx/v1beta1/txs", bytes.NewReader(bz))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var out struct {
		TxResponse struct {
			TxHash string `json:"txhash"`
			Code   int    `json:"code"`
			RawLog string `json:"raw_log"`
		} `json:"tx_response"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Zero(t, out.TxResponse.Code, "broadcast rejected: %s", out.TxResponse.RawLog)
	return out.TxResponse.TxHash
}

func queryTxResult(t *testing.T, rest, txHash string) (code int, height string) {
	t.Helper()
	body := httpGet(t, rest+"/cosmos/tx/v1beta1/txs/"+txHash)
	var out struct {
		TxResponse struct {
			Code   int    `json:"code"`
			Height string `json:"height"`
			RawLog string `json:"raw_log"`
		} `json:"tx_response"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return -1, ""
	}
	return out.TxResponse.Code, out.TxResponse.Height
}

func httpGet(t *testing.T, url string) []byte {
	t.Helper()
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)
	return buf.Bytes()
}

func newLegacyAny(t *testing.T, typeURL string, msg proto.Message) *codectypes.Any {
	t.Helper()
	anyMsg, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	anyMsg.TypeUrl = typeURL
	return anyMsg
}
