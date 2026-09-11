package ante

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"
)

// The anteinterfaces fields are populated by embedding the interface in an
// empty struct. The struct then satisfies every method of the interface without
// the test having to implement — and maintain — the whole surface. Validate
// only compares each field against nil and never calls a method, so the
// embedded nil interface is never dereferenced.
//
// Note that a plain typed nil, e.g. (*stubAccountKeeper)(nil), would NOT work
// here: assigning it to an interface field yields a non-nil interface value,
// which is the opposite of what these tests need to control.
type stubAccountKeeper struct{ anteinterfaces.AccountKeeper }
type stubBankKeeper struct{ anteinterfaces.BankKeeper }
type stubFeeMarketKeeper struct{ anteinterfaces.FeeMarketKeeper }
type stubEvmKeeper struct{ anteinterfaces.EVMKeeper }
type stubFeegrantKeeper struct{ ante.FeegrantKeeper }

// stubCodec satisfies codec.BinaryCodec the same way.
type stubCodec struct{ codec.BinaryCodec }

// completeHandlerOptions returns options with every REQUIRED field populated
// and every optional field left at its zero value. The tests below either clear
// one required field or add optional ones.
func completeHandlerOptions() HandlerOptions {
	return HandlerOptions{
		AccountKeeper:   stubAccountKeeper{},
		BankKeeper:      stubBankKeeper{},
		IBCKeeper:       &ibckeeper.Keeper{},
		FeeMarketKeeper: stubFeeMarketKeeper{},
		EvmKeeper:       stubEvmKeeper{},
		SignModeHandler: &txsigning.HandlerMap{},
		SigGasConsumer: func(storetypes.GasMeter, signing.SignatureV2, authtypes.Params) error {
			return nil
		},
		Cdc: stubCodec{},
	}
}

// TestValidateRejectsMissingRequiredField pins the fail-fast contract for every
// field that cosmosAnteDecorators hands to a decorator unconditionally. A
// regression here means a nil keeper reaches the ante chain and the node panics
// on the first transaction instead of refusing to boot.
//
// The ibc keeper and signature gas consumer rows are the ones added by this
// fix: both were dereferenced unconditionally while Validate did not require
// them, so a wiring mistake surfaced as a first-tx panic rather than a startup
// error.
func TestValidateRejectsMissingRequiredField(t *testing.T) {
	cases := []struct {
		name  string
		clear func(*HandlerOptions)
		want  string
	}{
		{"account keeper", func(o *HandlerOptions) { o.AccountKeeper = nil }, "account keeper is required"},
		{"bank keeper", func(o *HandlerOptions) { o.BankKeeper = nil }, "bank keeper is required"},
		{"ibc keeper", func(o *HandlerOptions) { o.IBCKeeper = nil }, "ibc keeper is required"},
		{"fee market keeper", func(o *HandlerOptions) { o.FeeMarketKeeper = nil }, "fee market keeper is required"},
		{"evm keeper", func(o *HandlerOptions) { o.EvmKeeper = nil }, "evm keeper is required"},
		{"sign mode handler", func(o *HandlerOptions) { o.SignModeHandler = nil }, "sign mode handler is required"},
		{"signature gas consumer", func(o *HandlerOptions) { o.SigGasConsumer = nil }, "signature gas consumer is required"},
		{"codec", func(o *HandlerOptions) { o.Cdc = nil }, "cdc is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := completeHandlerOptions()
			require.NoError(t, opts.Validate(),
				"sanity: with every required field set Validate must succeed, so the failure below is attributable to the cleared field")

			tc.clear(&opts)

			err := opts.Validate()
			require.Error(t, err, "a nil %s must be rejected", tc.name)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

// TestValidateAcceptsNilOptionalFields is the counter-sentinel for
// TestValidateRejectsMissingRequiredField. Tightening Validate must not turn a
// supported minimal wire-up into a boot failure:
//
//   - FeegrantKeeper nil means "feegrant disabled" to cosmos-sdk's
//     DeductFeeDecorator, and upstream's own Validate omits it for that reason.
//   - WasmKeeper / WasmNodeConfig / TXCounterStoreService nil is exactly the
//     input TestWasmDecoratorsOmittedWhenKeepersNil exists to protect.
//
// Without this test, the next person to "complete" Validate can silently break
// both contracts and still see a green suite.
func TestValidateAcceptsNilOptionalFields(t *testing.T) {
	opts := completeHandlerOptions()
	opts.FeegrantKeeper = nil
	opts.WasmKeeper = nil
	opts.WasmNodeConfig = nil
	opts.TXCounterStoreService = nil
	opts.DisabledAuthzMsgs = nil
	opts.MaxTxGasWanted = 0
	opts.TxFeeChecker = nil

	require.NoError(t, opts.Validate(),
		"optional fields must stay optional: feegrant nil = disabled, wasm nil = decorator omitted")
}

// TestValidateAcceptsFullyPopulatedOptions is the positive control: the shape
// app.go actually wires (every optional field present) must validate. It runs
// the value-typed stubs through the same code path the node uses.
func TestValidateAcceptsFullyPopulatedOptions(t *testing.T) {
	opts := completeHandlerOptions()
	opts.FeegrantKeeper = stubFeegrantKeeper{}
	opts.WasmNodeConfig = &wasmtypes.NodeConfig{}
	// stubKVStoreService is declared in handler_options_test.go and satisfies
	// corestore.KVStoreService with a nil store, which Validate never opens.
	opts.TXCounterStoreService = stubKVStoreService{}

	require.NoError(t, opts.Validate())
}
