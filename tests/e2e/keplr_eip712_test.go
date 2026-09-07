package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

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
	e2eChainID    = "uptick_1170-1"
	e2eEVMChainID = uint64(1170)
	e2eDenom      = "auptick"
)

func init() {
	cmdcfg.SetBech32Prefixes(sdk.GetConfig())
	cmdcfg.RegisterDenoms()
}

func TestKeplrEip712TransferEndToEnd(t *testing.T) {
	home := os.Getenv("UPTICK_E2E_HOME")
	if home == "" {
		home = "/tmp/uptick-keplr"
	}
	rest := os.Getenv("UPTICK_E2E_REST")
	if rest == "" {
		rest = "http://127.0.0.1:1317"
	}
	if !restReachable(rest) {
		// Audit CI-gap: in the dedicated e2e job (UPTICK_E2E_STRICT=1) a
		// missing node is a failure, not a skip — `go test ./...` passing
		// locally with this suite skipped proves nothing about real
		// Keplr broadcast paths (see M-01).
		if os.Getenv("UPTICK_E2E_STRICT") == "1" {
			t.Fatalf("no uptick node reachable at %s; the CI e2e job must never skip", rest)
		}
		t.Skipf("no uptick node reachable at %s; set UPTICK_E2E_REST to run the e2e test", rest)
	}

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
		Amount: sdk.NewCoins(sdk.NewInt64Coin(e2eDenom, 1000000000000000)),
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
func TestKeplrDirectTransferEndToEnd(t *testing.T) {
	home := os.Getenv("UPTICK_E2E_HOME")
	if home == "" {
		home = "/tmp/uptick-keplr"
	}
	rest := os.Getenv("UPTICK_E2E_REST")
	if rest == "" {
		rest = "http://127.0.0.1:1317"
	}
	if !restReachable(rest) {
		t.Skipf("no uptick node reachable at %s", rest)
	}

	enc := params.MakeEncodingConfig()
	banktypes.RegisterInterfaces(enc.InterfaceRegistry)
	banktypes.RegisterLegacyAminoCodec(enc.LegacyAmino)
	legacytx.RegressionTestingAminoCodec = enc.LegacyAmino

	privKey := loadKeplrPrivKey(t, home, enc.Codec)
	pubKey := privKey.PubKey().(*ethsecp256k1.PubKey)
	from := sdk.AccAddress(pubKey.Address().Bytes())

	accNum, seq := queryAccount(t, rest, from.String())
	msgs := []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: from.String(),
			ToAddress:   from.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin(e2eDenom, 1)),
		},
	}
	fee := legacytx.StdFee{
		Amount: sdk.NewCoins(sdk.NewInt64Coin(e2eDenom, 1000000000000000)),
		Gas:    200000,
	}

	msgAny, err := codectypes.NewAnyWithValue(msgs[0])
	require.NoError(t, err)
	pubAny := newLegacyAny(t, "/ethermint.crypto.v1.ethsecp256k1.PubKey", &ethsecp256k1.PubKey{Key: pubKey.Bytes()})

	body := &txtypes.TxBody{Messages: []*codectypes.Any{msgAny}}
	bodyBz, err := enc.Codec.Marshal(body)
	require.NoError(t, err)

	authInfo := &txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{
			{
				PublicKey: pubAny,
				ModeInfo: &txtypes.ModeInfo{
					Sum: &txtypes.ModeInfo_Single_{
						Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT},
					},
				},
				Sequence: seq,
			},
		},
		Fee: &txtypes.Fee{Amount: fee.Amount, GasLimit: fee.Gas},
	}
	authInfoBz, err := enc.Codec.Marshal(authInfo)
	require.NoError(t, err)

	signDoc := &txtypes.SignDoc{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		ChainId:       e2eChainID,
		AccountNumber: accNum,
	}
	signDocBz, err := enc.Codec.Marshal(signDoc)
	require.NoError(t, err)
	sig, err := privKey.Sign(signDocBz)
	require.NoError(t, err)

	txRaw := &txtypes.TxRaw{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		Signatures:    [][]byte{sig},
	}
	txRawBz, err := enc.Codec.Marshal(txRaw)
	require.NoError(t, err)

	txHash := broadcastTx(t, rest, base64.StdEncoding.EncodeToString(txRawBz))
	require.Eventually(t, func() bool {
		code, height := queryTxResult(t, rest, txHash)
		return height != "" && code >= 0
	}, 20*time.Second, 500*time.Millisecond)
	code, _ := queryTxResult(t, rest, txHash)
	require.Zero(t, code, "keplr direct transfer failed with code %d", code)
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
	kr, err := keyring.New("uptick", keyring.BackendTest, home, nil, cdc)
	require.NoError(t, err)
	rec, err := kr.Key("keplr")
	require.NoError(t, err)
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
