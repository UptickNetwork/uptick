package params

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/evm/ethereum/eip712"
	"github.com/cosmos/gogoproto/proto"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// TestTxDecoderAcceptsLegacyKeplrEIP712Types verifies the protobuf tx decoder
// accepts a Keplr-style EIP-712 transaction that still carries the legacy
// ethermint extension option and pubkey type URLs. Regression guard for the
// v0.4.0 "unable to resolve type URL /ethermint.types.v1.ExtensionOptionsWeb3Tx"
// and "does not have a Descriptor() method" decode failures.
func TestTxDecoderAcceptsLegacyKeplrEIP712Types(t *testing.T) {
	enc := MakeEncodingConfig()

	web3Any := newLegacyAny(t, "/ethermint.types.v1.ExtensionOptionsWeb3Tx", &eip712.ExtensionOptionsWeb3Tx{
		TypedDataChainID: 1170,
		FeePayer:         "uptick1n6ak2gnzsxgg6q6ankz53zwc2zacrhvrxt4433",
		FeePayerSig:      make([]byte, 65),
	})

	body := &txtypes.TxBody{
		ExtensionOptions: []*codectypes.Any{web3Any},
	}
	bodyBz, err := enc.Codec.Marshal(body)
	require.NoError(t, err)

	pubkeyAny := newLegacyAny(t, "/ethermint.crypto.v1.ethsecp256k1.PubKey", &ethsecp256k1.PubKey{Key: make([]byte, 33)})

	authInfo := &txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{
			{
				PublicKey: pubkeyAny,
				ModeInfo: &txtypes.ModeInfo{
					Sum: &txtypes.ModeInfo_Single_{
						Single: &txtypes.ModeInfo_Single{
							Mode: signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
						},
					},
				},
				Sequence: 0,
			},
		},
		Fee: &txtypes.Fee{
			Amount:   sdk.NewCoins(sdk.NewInt64Coin("auoc", 1)),
			GasLimit: 200000,
		},
	}
	authInfoBz, err := enc.Codec.Marshal(authInfo)
	require.NoError(t, err)

	txRaw := &txtypes.TxRaw{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authInfoBz,
		Signatures:    [][]byte{make([]byte, 65)},
	}
	txRawBz, err := enc.Codec.Marshal(txRaw)
	require.NoError(t, err)

	_, err = enc.TxConfig.TxDecoder()(txRawBz)
	require.NoError(t, err)
}

func newLegacyAny(t *testing.T, typeURL string, msg proto.Message) *codectypes.Any {
	t.Helper()
	anyMsg, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	anyMsg.TypeUrl = typeURL
	return anyMsg
}
