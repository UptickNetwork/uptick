package ante

import (
	"context"
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
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	address "cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"

	"github.com/UptickNetwork/uptick/app/params"
)

// TestVerifyKeplrEip712Signature reproduces the exact transaction Keplr builds
// when signing a bank transfer on an ethermint chain (legacy EIP-712): a
// SIGN_MODE_LEGACY_AMINO_JSON signer with an empty body signature, the EIP-712
// signature stored in the /ethermint.types.v1.ExtensionOptionsWeb3Tx extension
// option, and the /ethermint.crypto.v1.ethsecp256k1.PubKey public key.

// testCodec builds a fully-populated codec once for tests that call
// verifyEip712Signature directly (the decorator now takes its codec by
// injection instead of a package-level global).
func testCodec() codec.Codec {
	enc := params.MakeEncodingConfig()
	banktypes.RegisterInterfaces(enc.InterfaceRegistry)
	banktypes.RegisterLegacyAminoCodec(enc.LegacyAmino)
	return enc.Codec
}

func TestVerifyKeplrEip712Signature(t *testing.T) {
	verify := buildKeplrEip712Tx(t, nil)
	require.NoError(t, verifyEip712Signature(testCodec(), verify.pubKey, verify.signerData, verify.sigData, verify.tx))
}

func TestVerifyKeplrEip712SignatureRejectsTamperedSig(t *testing.T) {
	verify := buildKeplrEip712Tx(t, func(sig []byte) {
		// Flip the first byte of the signature so verification must fail.
		sig[0] ^= 0xff
	})
	require.Error(t, verifyEip712Signature(testCodec(), verify.pubKey, verify.signerData, verify.sigData, verify.tx))
}

// fakeAccountKeeper is a minimal AccountKeeper backed by a single in-memory
// account, so decorator-level tests can drive AnteHandle end to end.
type fakeAccountKeeper struct {
	acc sdk.AccountI
}

func (k *fakeAccountKeeper) NewAccountWithAddress(context.Context, sdk.AccAddress) sdk.AccountI {
	panic("not implemented")
}
func (k *fakeAccountKeeper) GetModuleAddress(string) sdk.AccAddress { return nil }
func (k *fakeAccountKeeper) GetAccount(context.Context, sdk.AccAddress) sdk.AccountI {
	return k.acc
}
func (k *fakeAccountKeeper) SetAccount(context.Context, sdk.AccountI)    {}
func (k *fakeAccountKeeper) RemoveAccount(context.Context, sdk.AccountI) {}
func (k *fakeAccountKeeper) GetParams(context.Context) authtypes.Params  { return authtypes.Params{} }
func (k *fakeAccountKeeper) GetSequence(context.Context, sdk.AccAddress) (uint64, error) {
	return k.acc.GetSequence(), nil
}
func (k *fakeAccountKeeper) AddressCodec() address.Codec {
	return codecaddress.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
}
func (k *fakeAccountKeeper) UnorderedTransactionsEnabled() bool             { return false }
func (k *fakeAccountKeeper) RemoveExpiredUnorderedNonces(sdk.Context) error { return nil }
func (k *fakeAccountKeeper) TryAddUnorderedNonce(sdk.Context, []byte, time.Time) error {
	return nil
}

// TestEip712DecoratorAnteHandleEndToEnd drives AnteHandle through the real
// decorator (constructed via NewEip712SigVerificationDecorator) with a valid
// Keplr-built tx. Regression guard for the injected-codec wiring: a decorator
// built without its codec must not be able to pass this path (it would panic
// inside LegacyWrapTxToTypedData on a nil AnyUnpacker).
func TestEip712DecoratorAnteHandleEndToEnd(t *testing.T) {
	verify := buildKeplrEip712Tx(t, nil)

	acc := authtypes.NewBaseAccount(
		sdk.AccAddress(verify.pubKey.Address().Bytes()),
		verify.pubKey,
		7, 3,
	)
	ak := &fakeAccountKeeper{acc: acc}

	key := storetypes.NewKVStoreKey("eip712-test")
	tkey := storetypes.NewTransientStoreKey("eip712-test-t")
	ctx := testutil.DefaultContext(key, tkey).
		WithChainID("uptick_1170-1").
		WithBlockHeight(10)

	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	dec := NewEip712SigVerificationDecorator(ak, testCodec())
	_, err := dec.AnteHandle(ctx, verify.tx, false, next)
	require.NoError(t, err)
	require.True(t, nextCalled, "valid EIP-712 tx must pass the decorator and call next")
}

// TestEip712DecoratorAnteHandleRejectsTamperedSig drives AnteHandle with a
// tampered signature: the decorator must return an error (not panic) and must
// not call next.
func TestEip712DecoratorAnteHandleRejectsTamperedSig(t *testing.T) {
	verify := buildKeplrEip712Tx(t, func(sig []byte) {
		sig[0] ^= 0xff
	})

	acc := authtypes.NewBaseAccount(
		sdk.AccAddress(verify.pubKey.Address().Bytes()),
		verify.pubKey,
		7, 3,
	)
	ak := &fakeAccountKeeper{acc: acc}

	key := storetypes.NewKVStoreKey("eip712-test")
	tkey := storetypes.NewTransientStoreKey("eip712-test-t")
	ctx := testutil.DefaultContext(key, tkey).
		WithChainID("uptick_1170-1").
		WithBlockHeight(10)

	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	dec := NewEip712SigVerificationDecorator(ak, testCodec())
	_, err := dec.AnteHandle(ctx, verify.tx, false, next)
	require.Error(t, err)
	require.False(t, nextCalled, "invalid EIP-712 tx must be rejected before next")
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
	typedData, err := eip712.LegacyWrapTxToTypedData(enc.Codec, evmChainID, msgs[0], signBytes, feeDelegation)
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
