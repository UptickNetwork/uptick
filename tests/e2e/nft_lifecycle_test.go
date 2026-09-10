package e2e

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/evm/crypto/ethsecp256k1"

	"github.com/UptickNetwork/uptick/app/params"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
)

// This file covers the module WRITE path on a real node (audit G-06). The RPC
// smoke test next door only proves the query routes answer; it never moves
// state. Everything here - issue a class, mint, transfer, burn - goes through
// the real tx pipeline (ante handler, msg server, state commit) on a node that
// actually produces blocks, which is the part `go test ./...` cannot reach: its
// keepers are constructed by hand and never see a TxDecoder, a fee check or a
// committed store.

// signDirectAndBroadcast signs exactly one message with SIGN_MODE_DIRECT using
// the legacy ethsecp256k1 public key - the path @cosmjs/stargate and Keplr's
// getOfflineSigner use - then waits for the tx to be committed and fails on a
// non-zero code.
//
// It is deliberately generic over the message: any sdk.Msg can be pushed
// through it, which is what makes it usable for the collection lifecycle here
// and the bank transfer in keplr_eip712_test.go.
func signDirectAndBroadcast(
	t *testing.T,
	enc params.EncodingConfig,
	rest string,
	privKey cryptotypes.PrivKey,
	pubKey *ethsecp256k1.PubKey,
	from sdk.AccAddress,
	msg sdk.Msg,
	gas uint64,
) string {
	t.Helper()

	// Query the sequence per transaction: every committed tx consumes one, so
	// a sequence cached across the lifecycle would be rejected on the second
	// send.
	accNum, seq := queryAccount(t, rest, from.String())

	msgAny, err := codectypes.NewAnyWithValue(msg)
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
		Fee: &txtypes.Fee{
			Amount:   sdk.NewCoins(sdk.NewCoin(e2eDenom, e2eFeeAmount)),
			GasLimit: gas,
		},
	}
	authInfoBz, err := enc.Codec.Marshal(authInfo)
	require.NoError(t, err)

	signDocBz, err := enc.Codec.Marshal(&txtypes.SignDoc{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		ChainId:       e2eChainID,
		AccountNumber: accNum,
	})
	require.NoError(t, err)

	sig, err := privKey.Sign(signDocBz)
	require.NoError(t, err)

	txRawBz, err := enc.Codec.Marshal(&txtypes.TxRaw{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		Signatures:    [][]byte{sig},
	})
	require.NoError(t, err)

	txHash := broadcastTx(t, rest, base64.StdEncoding.EncodeToString(txRawBz))

	var code int
	require.Eventuallyf(t, func() bool {
		c, height := queryTxResult(t, rest, txHash)
		if height == "" {
			return false
		}
		code = c
		return true
	}, 20*time.Second, 500*time.Millisecond, "%s tx %s was never committed", fmt.Sprintf("%T", msg), txHash)

	require.Zerof(t, code, "%s tx %s failed with code %d", fmt.Sprintf("%T", msg), txHash, code)
	t.Logf("%T committed: txhash=%s", msg, txHash)
	return txHash
}

// TestCollectionNFCLifecycleOnRealNode walks a full NFT lifecycle against a
// live chain.
//
// Ordering is deliberate: the burn happens while the minting account is still
// the owner, because the collection module authorises burn by ownership - a
// token transferred away can no longer be burned by its minter.
func TestCollectionNFTLifecycleOnRealNode(t *testing.T) {
	home, rest := e2eNode(t)

	enc := params.MakeEncodingConfig()
	privKey := loadKeplrPrivKey(t, home, enc.Codec)
	pubKey := privKey.PubKey().(*ethsecp256k1.PubKey)
	from := sdk.AccAddress(pubKey.Address().Bytes())

	// The denom id must satisfy ValidateIssueDenomID: it starts with a
	// lowercase letter, never with the reserved "ibc"/"uptick-" shapes, and is
	// neither a bech32 address nor a 40-nibble hex address. The 't'/'n' keep it
	// out of the hex-address shape, so the nanos suffix stays a valid id.
	denomID := fmt.Sprintf("e2enft%d", time.Now().UnixNano())
	const (
		tokenA = "token001"
		tokenB = "token002"
	)
	recipient := sdk.AccAddress([]byte{
		0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
	})

	const gas = 400_000

	// 1. Issue the class.
	signDirectAndBroadcast(t, enc, rest, privKey, pubKey, from, &collectiontypes.MsgIssueDenom{
		Id:     denomID,
		Name:   "E2E Collection",
		Symbol: "E2E",
		Sender: from.String(),
	}, gas)

	// /uptick/collection/nft/denoms/{id} (QueryDenom) is the route that returns
	// the class record itself; /collections/{id} (QueryCollection) wraps it in
	// a "collection" envelope.
	denom := getJSON(t, rest+"/uptick/collection/nft/denoms/"+denomID)
	var denomResp struct {
		Denom struct {
			Id      string `json:"id"`
			Name    string `json:"name"`
			Creator string `json:"creator"`
		} `json:"denom"`
	}
	require.NoError(t, json.Unmarshal(denom, &denomResp), "denom query returned %s", summarizeBody(denom))
	require.Equal(t, denomID, denomResp.Denom.Id, "denom query returned %s", summarizeBody(denom))
	require.Equal(t, from.String(), denomResp.Denom.Creator,
		"the creator must be the signer; a mismatch means the msg server ignored the sender")

	// 2. Mint two tokens.
	for _, tokenID := range []string{tokenA, tokenB} {
		signDirectAndBroadcast(t, enc, rest, privKey, pubKey, from, &collectiontypes.MsgMintNFT{
			Id:        tokenID,
			DenomId:   denomID,
			Name:      "E2E Token " + tokenID,
			URI:       "ipfs://e2e/" + tokenID,
			Sender:    from.String(),
			Recipient: from.String(),
		}, gas)
	}

	nft := getNFT(t, rest, denomID, tokenA)
	require.Equal(t, tokenA, nft.Id)
	require.Equal(t, from.String(), nft.Owner, "the minter must own the token")

	require.Equal(t, "2", getSupply(t, rest, denomID))

	// 3. Transfer tokenA away.
	signDirectAndBroadcast(t, enc, rest, privKey, pubKey, from, &collectiontypes.MsgTransferNFT{
		Id:        tokenA,
		DenomId:   denomID,
		Sender:    from.String(),
		Recipient: recipient.String(),
	}, gas)

	transferred := getNFT(t, rest, denomID, tokenA)
	require.Equal(t, recipient.String(), transferred.Owner,
		"ownership must follow the transfer; a stale owner breaks every downstream royalty/allowance check")

	// 4. Burn tokenB, which the signer still owns.
	signDirectAndBroadcast(t, enc, rest, privKey, pubKey, from, &collectiontypes.MsgBurnNFT{
		Id:      tokenB,
		DenomId: denomID,
		Sender:  from.String(),
	}, gas)

	status, body := httpStatus(t, rest+"/uptick/collection/nfts/"+denomID+"/"+tokenB)
	// The gateway surfaces the keeper's ErrNFTNotExists as HTTP 500 rather than
	// 404, so the assertion is "no longer queryable" instead of naming a status
	// code the SDK does not actually produce.
	require.NotEqual(t, http.StatusOK, status,
		"the burned token must not be queryable any more: %s", summarizeBody(body))

	require.Equal(t, "1", getSupply(t, rest, denomID),
		"burning must decrement the supply")
}

// getJSON fetches a URL and fails the test on a non-200 response, so a broken
// route surfaces as a route failure instead of a confusing JSON error.
func getJSON(t *testing.T, url string) []byte {
	t.Helper()

	status, body := httpStatus(t, url)
	require.Equalf(t, http.StatusOK, status, "GET %s returned %d: %s", url, status, summarizeBody(body))
	return body
}

type queriedNFT struct {
	Id    string `json:"id"`
	URI   string `json:"uri"`
	Owner string `json:"owner"`
}

func getNFT(t *testing.T, rest, denomID, tokenID string) queriedNFT {
	t.Helper()

	body := getJSON(t, rest+"/uptick/collection/nfts/"+denomID+"/"+tokenID)
	var out struct {
		NFT queriedNFT `json:"nft"`
	}
	require.NoError(t, json.Unmarshal(body, &out), "nft query returned %s", summarizeBody(body))
	return out.NFT
}

func getSupply(t *testing.T, rest, denomID string) string {
	t.Helper()

	body := getJSON(t, rest+"/uptick/collection/collections/"+denomID+"/supply")
	// The supply is a uint64, which the gateway renders as a decimal STRING
	// (protobuf JSON), so it is decoded as one rather than as a number.
	var out struct {
		Amount string `json:"amount"`
	}
	require.NoError(t, json.Unmarshal(body, &out), "supply query returned %s", summarizeBody(body))
	return out.Amount
}
