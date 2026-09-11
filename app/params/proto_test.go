package params

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	antetypes "github.com/cosmos/evm/ante/types"
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

// TestTxDecoderRejectsLegacyEthermintDynamicFeeExtensionOption pins the
// *current* behaviour that the pre-v0.4.0 ethermint type URL for the
// dynamic-fee extension option does not decode. This is a "reverse pin": it
// freezes an accepted incompatibility on purpose, so that a later re-baseline
// of the compat layer has to happen deliberately instead of by accident.
//
// Why the incompatibility is accepted:
//   - ethermint served this message under
//     /ethermint.types.v1.ExtensionOptionDynamicFeeTx. cosmos/evm renamed the
//     proto package to cosmos.evm.ante.v1, so the message now lives under
//     /cosmos.evm.ante.v1.ExtensionOptionDynamicFeeTx.
//   - RegisterCompatInterfaces (app/upgrades/v041/compat.go:47-50) remaps the
//     legacy pubkey and EIP-712 extension option URLs on purpose, but not this
//     one.
//   - The only known producer is Hermes' chain-registry option
//     `type = 'ethermint_dynamic_fee'`, which is off by default
//     (relayer-cli/src/chain_registry.rs:168). No transaction on uptick_117-1
//     has ever carried it.
//   - Dropping the option costs nothing today: it never routed anywhere in
//     v0.3.3 either (the v0.3.3 ante whitelist only accepted it as
//     extension_options[0]), while in v0.4.1 the dynamic-fee treatment is the
//     default for every Cosmos tx — ante.HasDynamicFeeExtensionOption only
//     *adds* the option (app/ante/handler_options.go:222).
//
// If this test starts failing, a legacy client began sending the option. Do
// NOT simply add a second registerCustomTypeURL call. The ethermint field was a
// math.Int while the cosmos/evm field is a math.LegacyDec; both marshal as a
// bare string, so an alias would resolve cleanly and silently reinterpret the
// value by 10^18 (a relayed "500000000000" would read as 0.0000005). Decide the
// value semantics first, then ship the alias together with a rescaling step.
func TestTxDecoderRejectsLegacyEthermintDynamicFeeExtensionOption(t *testing.T) {
	const (
		legacyURL  = "/ethermint.types.v1.ExtensionOptionDynamicFeeTx"
		currentURL = "/cosmos.evm.ante.v1.ExtensionOptionDynamicFeeTx"
		// What Hermes puts in ethermint_dynamic_fee dynamic_fee_price.
		priceASCII = "500000000000"
	)

	enc := MakeEncodingConfig()

	// Built by hand rather than via NewAnyWithValue: the ethermint message was
	// `string max_priority_price = 1` backed by math.Int, so the value bytes are
	// the bare ASCII digits with no decimal point. Hand-assembling keeps the
	// test independent of whichever Go type happens to be compiled in.
	legacyValue := append([]byte{0x0a, byte(len(priceASCII))}, priceASCII...)
	legacyAny := &codectypes.Any{TypeUrl: legacyURL, Value: legacyValue}

	// Control: the same option under the current type URL must still decode, so
	// that a failure below can only be attributed to the type URL.
	currentAny := newLegacyAny(t, currentURL, &antetypes.ExtensionOptionDynamicFeeTx{
		MaxPriorityPrice: math.LegacyMustNewDecFromStr(priceASCII),
	})

	for _, tc := range []struct {
		name    string
		ext     *codectypes.Any
		wantOK  bool
		wantMsg string
	}{
		// The failure mode matters as much as the failure: resolution of the
		// type URL is what breaks, not the option's value encoding.
		{name: "legacy ethermint type URL is rejected", ext: legacyAny, wantOK: false, wantMsg: "unable to resolve type URL " + legacyURL},
		{name: "current cosmos/evm type URL is accepted", ext: currentAny, wantOK: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := enc.TxConfig.TxDecoder()(txBytesWithExtensionOption(t, enc, tc.ext))
			if tc.wantOK {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

// txBytesWithExtensionOption returns the encoded bytes of a minimal TxRaw that
// carries a single extension option, ready to feed into TxDecoder.
func txBytesWithExtensionOption(t *testing.T, enc EncodingConfig, ext *codectypes.Any) []byte {
	t.Helper()

	bodyBz, err := enc.Codec.Marshal(&txtypes.TxBody{ExtensionOptions: []*codectypes.Any{ext}})
	require.NoError(t, err)

	authInfoBz, err := enc.Codec.Marshal(&txtypes.AuthInfo{
		Fee: &txtypes.Fee{
			Amount:   sdk.NewCoins(sdk.NewInt64Coin("auoc", 1)),
			GasLimit: 200000,
		},
	})
	require.NoError(t, err)

	txBz, err := enc.Codec.Marshal(&txtypes.TxRaw{BodyBytes: bodyBz, AuthInfoBytes: authInfoBz})
	require.NoError(t, err)

	return txBz
}

func newLegacyAny(t *testing.T, typeURL string, msg proto.Message) *codectypes.Any {
	t.Helper()
	anyMsg, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	anyMsg.TypeUrl = typeURL
	return anyMsg
}
