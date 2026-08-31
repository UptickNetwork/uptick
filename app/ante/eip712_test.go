package ante

import (
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/evm/ethereum/eip712"
	"github.com/cosmos/gogoproto/proto"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/UptickNetwork/uptick/app/params"
)

// TestVerifyKeplrEip712Signature reproduces the exact transaction Keplr builds
// when signing a bank transfer on an ethermint chain (legacy EIP-712): a
// SIGN_MODE_LEGACY_AMINO_JSON signer with an empty body signature, the EIP-712
// signature stored in the /ethermint.types.v1.ExtensionOptionsWeb3Tx extension
// option, and the /ethermint.crypto.v1.ethsecp256k1.PubKey public key.
func TestVerifyKeplrEip712Signature(t *testing.T) {
	verify := buildKeplrEip712Tx(t, nil)
	require.NoError(t, verifyEip712Signature(verify.pubKey, verify.signerData, verify.sigData, verify.tx))
}

func TestVerifyKeplrEip712SignatureRejectsTamperedSig(t *testing.T) {
	verify := buildKeplrEip712Tx(t, func(sig []byte) {
		// Flip the first byte of the signature so verification must fail.
		sig[0] ^= 0xff
	})
	require.Error(t, verifyEip712Signature(verify.pubKey, verify.signerData, verify.sigData, verify.tx))
}

type keplrEip712Verify struct {
	pubKey     cryptotypes.PubKey
	signerData authsigning.SignerData
	sigData    signing.SignatureData
	tx         authsigning.Tx
}

func buildKeplrEip712Tx(t *testing.T, mutateSig func([]byte)) keplrEip712Verify {
	t.Helper()

	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	pubKey := priv.PubKey().(*ethsecp256k1.PubKey)
	from := sdk.AccAddress(pubKey.Address().Bytes())

	const (
		chainID    = "uptick_1170-1"
		evmChainID = uint64(1170)
		accNum     = uint64(7)
		seq        = uint64(3)
	)

	enc := params.MakeEncodingConfig()
	banktypes.RegisterInterfaces(enc.InterfaceRegistry)
	banktypes.RegisterLegacyAminoCodec(enc.LegacyAmino)
	legacytx.RegressionTestingAminoCodec = enc.LegacyAmino

	msgs := []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: from.String(),
			ToAddress:   from.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("auptick", 1)),
		},
	}
	fee := legacytx.StdFee{
		Amount: sdk.NewCoins(sdk.NewInt64Coin("auptick", 100)),
		Gas:    200000,
	}

	signBytes := legacytx.StdSignBytes(chainID, accNum, seq, 0, fee, msgs, "")

	feeDelegation := &eip712.FeeDelegationOptions{FeePayer: from}
	typedData, err := eip712.LegacyWrapTxToTypedData(evmCodec, evmChainID, msgs[0], signBytes, feeDelegation)
	require.NoError(t, err)

	sigHash, _, err := apitypes.TypedDataAndHash(typedData)
	require.NoError(t, err)
	sig, err := priv.Sign(sigHash)
	require.NoError(t, err)
	// Keplr/MetaMask transform the recovery id from 0/1 to 27/28.
	sig[ethcrypto.RecoveryIDOffset] += 27
	if mutateSig != nil {
		mutateSig(sig)
	}

	msgAny, err := codectypes.NewAnyWithValue(msgs[0])
	require.NoError(t, err)
	extAny := newLegacyAny(t, "/ethermint.types.v1.ExtensionOptionsWeb3Tx", &eip712.ExtensionOptionsWeb3Tx{
		TypedDataChainID: evmChainID,
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
						Single: &txtypes.ModeInfo_Single{
							Mode: signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
						},
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

	txRaw := &txtypes.TxRaw{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		Signatures:    [][]byte{{}},
	}
	txRawBz, err := enc.Codec.Marshal(txRaw)
	require.NoError(t, err)

	decoded, err := enc.TxConfig.TxDecoder()(txRawBz)
	require.NoError(t, err)

	sigTx, ok := decoded.(authsigning.SigVerifiableTx)
	require.True(t, ok)
	authTx, ok := decoded.(authsigning.Tx)
	require.True(t, ok)
	sigs, err := sigTx.GetSignaturesV2()
	require.NoError(t, err)
	require.Len(t, sigs, 1)

	signerData := authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: accNum,
		Sequence:      seq,
	}

	var pub cryptotypes.PubKey = pubKey
	return keplrEip712Verify{
		pubKey:     pub,
		signerData: signerData,
		sigData:    sigs[0].Data,
		tx:         authTx,
	}
}

func newLegacyAny(t *testing.T, typeURL string, msg proto.Message) *codectypes.Any {
	t.Helper()
	anyMsg, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	anyMsg.TypeUrl = typeURL
	return anyMsg
}
