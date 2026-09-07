package ante

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	antetypes "github.com/cosmos/evm/ante/types"
	"github.com/cosmos/evm/ethereum/eip712"

	protov2 "google.golang.org/protobuf/proto"

	storetypes "cosmossdk.io/store/types"
)

// M-01 regression: the EIP-712 ante handler's extension pre-check must accept
// the Web3 extension that Keplr attaches (both the current cosmos.evm type URL
// and the legacy ethermint one) and reject everything else, in lockstep with
// the "exactly one Web3Tx option" rule enforced by verifyEip712Signature.
func TestHasWeb3ExtensionOption(t *testing.T) {
	// Current /cosmos.evm.eip712.v1.ExtensionOptionsWeb3Tx.
	cur, err := types.NewAnyWithValue(&eip712.ExtensionOptionsWeb3Tx{})
	require.NoError(t, err)
	require.True(t, HasWeb3ExtensionOption(cur))

	// Legacy /ethermint.types.v1.ExtensionOptionsWeb3Tx: registered onto the
	// same Go type by app/upgrades/v041.RegisterCompatInterfaces, so the cached
	// value still asserts to *eip712.ExtensionOptionsWeb3Tx.
	legacy := newLegacyAny(t, "/ethermint.types.v1.ExtensionOptionsWeb3Tx", &eip712.ExtensionOptionsWeb3Tx{})
	require.True(t, HasWeb3ExtensionOption(legacy))

	// DynamicFee belongs to the Ethereum tx path and must stay rejected here.
	dyn, err := types.NewAnyWithValue(&antetypes.ExtensionOptionDynamicFeeTx{})
	require.NoError(t, err)
	require.False(t, HasWeb3ExtensionOption(dyn))

	// An unpackable Any (nil cached value) must be rejected, not panic.
	unknown := &types.Any{TypeUrl: "/unknown.ExtensionOption", Value: []byte{0x01}}
	require.False(t, HasWeb3ExtensionOption(unknown))
}

// TestExtensionOptionsDecoratorAcceptsKeplrEip712Tx drives the production
// decorator installed in newCosmosAnteHandlerEip712 with a fully built,
// validly signed Keplr EIP-712 tx. Under M-01 this exact combination failed
// with "unknown extension options" before Eip712SigVerificationDecorator could
// run, even though the signature itself was fine.
func TestExtensionOptionsDecoratorAcceptsKeplrEip712Tx(t *testing.T) {
	verify := buildKeplrEip712Tx(t, nil)

	key := storetypes.NewKVStoreKey("ext-checker-test")
	tkey := storetypes.NewTransientStoreKey("ext-checker-test-t")
	ctx := testutil.DefaultContext(key, tkey).WithChainID("uptick_1170-1")

	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	dec := ante.NewExtensionOptionsDecorator(HasWeb3ExtensionOption)
	_, err := dec.AnteHandle(ctx, verify.tx, false, next)
	require.NoError(t, err)
	require.True(t, nextCalled, "valid Keplr EIP-712 tx must pass the extension pre-check")
}

// fakeExtTx is a minimal sdk.Tx + ante.HasExtensionOptionsTx carrying just
// extension options, so the decorator's rejection path can be driven without
// building a complete transaction.
type fakeExtTx struct {
	opts []*types.Any
}

func (f *fakeExtTx) GetMsgs() []sdk.Msg                           { return nil }
func (f *fakeExtTx) GetMsgsV2() ([]protov2.Message, error)        { return nil, nil }
func (f *fakeExtTx) GetExtensionOptions() []*types.Any            { return f.opts }
func (f *fakeExtTx) GetNonCriticalExtensionOptions() []*types.Any { return f.opts }

func TestExtensionOptionsDecoratorRejectsDynamicFeeTx(t *testing.T) {
	dyn, err := types.NewAnyWithValue(&antetypes.ExtensionOptionDynamicFeeTx{})
	require.NoError(t, err)

	key := storetypes.NewKVStoreKey("ext-checker-test")
	tkey := storetypes.NewTransientStoreKey("ext-checker-test-t")
	ctx := testutil.DefaultContext(key, tkey)

	nextCalled := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	dec := ante.NewExtensionOptionsDecorator(HasWeb3ExtensionOption)
	_, err = dec.AnteHandle(ctx, &fakeExtTx{opts: []*types.Any{dyn}}, false, next)
	require.Error(t, err, "dynamic-fee-only extension must stay rejected on the EIP-712 chain")
	require.False(t, nextCalled)
}
