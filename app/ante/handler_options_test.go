package ante

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	corestore "cosmossdk.io/core/store"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	"github.com/stretchr/testify/require"
)

// stubKVStoreService satisfies corestore.KVStoreService with a nil store.
// The order tests never actually run the chain, so the store is never
// opened.
type stubKVStoreService struct{}

func (stubKVStoreService) OpenKVStore(_ context.Context) corestore.KVStore {
	return nil
}

// TestSetUpContextPrecedesWasmCountTX pins the fix from dfe8cee: SetUpContext
// MUST come before CountTX (and any other wasm KV-writing decorator), so the tx
// gas meter, not BaseApp's infinite preset meter, accounts for CountTX's KV
// writes. A regression here would let a malicious tx inflate the count without
// paying gas.
func TestSetUpContextPrecedesWasmCountTX(t *testing.T) {
	opts := HandlerOptions{
		TXCounterStoreService: stubKVStoreService{},
	}

	decorators := cosmosAnteDecorators(
		opts,
		&feemarkettypes.Params{},
		func(*codectypes.Any) bool { return false },
		ante.NewSigVerificationDecorator(nil, nil),
		nil,
	)

	setUpIdx, countTXIdx := -1, -1
	for i, d := range decorators {
		if _, ok := d.(ante.SetUpContextDecorator); ok {
			setUpIdx = i
			continue
		}
		if _, ok := d.(*wasmkeeper.CountTXDecorator); ok {
			countTXIdx = i
			continue
		}
	}

	require.NotEqual(t, -1, setUpIdx, "SetUpContextDecorator must be present in the chain")
	require.NotEqual(t, -1, countTXIdx, "CountTXDecorator must be present when TXCounterStoreService != nil")
	require.Less(t, setUpIdx, countTXIdx,
		"SetUpContextDecorator (index %d) must precede CountTXDecorator (index %d) so its KV writes are metered by the tx gas meter",
		setUpIdx, countTXIdx)
}

// TestCosmosAndEip712DecoratorOrderIdenticalExceptSig pins that the EIP-712
// path is byte-for-byte the same as the standard path except for the
// signature verifier decorator slot. This is the structural invariant that
// the cosmosAnteDecorators extraction enforces — a future refactor that
// drifts one path off the other fails this test.
func TestCosmosAndEip712DecoratorOrderIdenticalExceptSig(t *testing.T) {
	opts := HandlerOptions{
		IBCKeeper: &ibckeeper.Keeper{},
	}

	sigStandard := ante.NewSigVerificationDecorator(nil, nil)
	sigEip712 := NewEip712SigVerificationDecorator(nil, nil)

	chainStandard := cosmosAnteDecorators(opts, &feemarkettypes.Params{},
		func(*codectypes.Any) bool { return false }, sigStandard, nil)
	chainEip712 := cosmosAnteDecorators(opts, &feemarkettypes.Params{},
		func(*codectypes.Any) bool { return false }, sigEip712, nil)

	require.Equal(t, len(chainStandard), len(chainEip712),
		"cosmos and eip712 ante chains must have the same length")

	for i := range chainStandard {
		if fmt.Sprintf("%T", chainStandard[i]) == fmt.Sprintf("%T", chainEip712[i]) {
			continue
		}
		// Only the signature-verifier slot is allowed to differ.
		if isSigVerifySlot(chainStandard, chainStandard[i]) {
			continue
		}
		t.Fatalf("decorator slot %d differs between cosmos and eip712: standard=%T eip712=%T",
			i, chainStandard[i], chainEip712[i])
	}
}

// isSigVerifySlot returns true if the given decorator is the unique
// signature verifier in the chain. It compares by type name rather than
// slot index so the helper remains robust to future slot reorders (as
// long as the cosmos and eip712 chains reorder together).
func isSigVerifySlot(chain []sdk.AnteDecorator, candidate sdk.AnteDecorator) bool {
	seen := 0
	candName := fmt.Sprintf("%T", candidate)
	for _, d := range chain {
		if fmt.Sprintf("%T", d) == candName {
			seen++
		}
	}
	return seen == 1
}

// TestWasmDecoratorsOmittedWhenKeepersNil guards the conditional insertion:
// CountTX / GasRegister must NOT be added when their respective keepers
// are nil. A regression that always inserts them would crash on startup
// (nil pointer dereference).
func TestWasmDecoratorsOmittedWhenKeepersNil(t *testing.T) {
	decorators := cosmosAnteDecorators(
		HandlerOptions{},
		&feemarkettypes.Params{},
		func(*codectypes.Any) bool { return false },
		ante.NewSigVerificationDecorator(nil, nil),
		nil,
	)

	for i, d := range decorators {
		switch d.(type) {
		case *wasmkeeper.CountTXDecorator, *wasmkeeper.GasRegisterDecorator:
			t.Fatalf("decorator %d (%T) must not be inserted when its keeper is nil", i, d)
		}
	}
}

// TestGasRegisterDecoratorRequiresWasmKeeper is the negative-control
// counterpart to TestWasmDecoratorsOmittedWhenKeepersNil: WasmNodeConfig is set
// but WasmKeeper is nil, so GasRegisterDecorator must NOT appear. The non-nil
// branch has no unit assertion here; it is exercised only end-to-end (tests/e2e).
func TestGasRegisterDecoratorRequiresWasmKeeper(t *testing.T) {
	decorators := cosmosAnteDecorators(
		HandlerOptions{WasmNodeConfig: &wasmtypes.NodeConfig{}},
		&feemarkettypes.Params{},
		func(*codectypes.Any) bool { return false },
		ante.NewSigVerificationDecorator(nil, nil),
		nil,
	)

	require.Equal(t, -1, decoratorIndex(decorators, (*wasmkeeper.GasRegisterDecorator)(nil)),
		"GasRegisterDecorator must NOT appear when WasmKeeper == nil")
}

// decoratorIndex returns the index of the first decorator whose concrete type
// equals prototype's, or -1.
//
// It compares reflect.Type values rather than type-name strings on purpose. The
// wasmd keeper package is imported as `wasmkeeper` but declares `package
// keeper`, so fmt.Sprintf("%T", d) prints "*keeper.GasRegisterDecorator" and any
// literal built from the import alias can never match - the assertion this
// helper replaces had exactly that shape and could not fail.
//
// A typed nil works for pointer decorators: reflect.TypeOf recovers the pointer
// type without allocating.
func decoratorIndex(chain []sdk.AnteDecorator, prototype any) int {
	want := reflect.TypeOf(prototype)
	for i, d := range chain {
		if reflect.TypeOf(d) == want {
			return i
		}
	}
	return -1
}

// TestAnteDecoratorOrdering pins the two orderings in cosmosAnteDecorators
// whose inversion breaks behaviour silently: without this test a plausible
// reorder would surface only in production as "the wasm simulation ignores its
// gas cap" or "signature verification became free".
//
// The rest of the chain is deliberately not pinned. Most pairs only decide which
// error surfaces first, and swapping them leaves the same tx accepted or
// rejected, just reported differently. That includes the RejectMessagesDecorator
// / extension-option adjacency flagged in review: they are not adjacent (the
// authz limiter and the simulation-gas decorator sit between them), and swapping
// any of the three changes only which error surfaces.
func TestAnteDecoratorOrdering(t *testing.T) {
	sigVerify := ante.NewSigVerificationDecorator(nil, nil)
	decorators := cosmosAnteDecorators(
		HandlerOptions{TXCounterStoreService: stubKVStoreService{}},
		&feemarkettypes.Params{},
		func(*codectypes.Any) bool { return false },
		sigVerify,
		nil,
	)

	// SetUpContext installs the tx's own gas meter; LimitSimulationGas then
	// replaces it with the wasm simulation cap. Reversed, SetUpContext would
	// overwrite the cap and SimulationGasLimit would never bind - a simulation
	// of an expensive contract would run unbounded.
	setUpIdx := decoratorIndex(decorators, ante.SetUpContextDecorator{})
	simGasIdx := decoratorIndex(decorators, (*wasmkeeper.LimitSimulationGasDecorator)(nil))
	require.NotEqual(t, -1, setUpIdx, "SetUpContextDecorator must be present in the chain")
	require.NotEqual(t, -1, simGasIdx, "LimitSimulationGasDecorator must be present in the chain")
	require.Less(t, setUpIdx, simGasIdx,
		"SetUpContextDecorator (index %d) must precede LimitSimulationGasDecorator (index %d), otherwise the simulation gas cap is overwritten",
		setUpIdx, simGasIdx)

	// SetPubKey -> SigGasConsume -> SigVerification is the canonical
	// sub-order: verification resolves the pubkey SetPubKey just installed, and
	// the gas it costs is charged before it runs. Reversing the last two hands
	// out signature verification without charging for it.
	setPubKeyIdx := decoratorIndex(decorators, ante.SetPubKeyDecorator{})
	sigGasIdx := decoratorIndex(decorators, ante.SigGasConsumeDecorator{})
	sigVerifyIdx := decoratorIndex(decorators, sigVerify)
	require.NotEqual(t, -1, setPubKeyIdx, "SetPubKeyDecorator must be present in the chain")
	require.NotEqual(t, -1, sigGasIdx, "SigGasConsumeDecorator must be present in the chain")
	require.NotEqual(t, -1, sigVerifyIdx, "the caller's signature verifier must be present in the chain")
	require.Less(t, setPubKeyIdx, sigGasIdx,
		"SetPubKeyDecorator (index %d) must precede SigGasConsumeDecorator (index %d)", setPubKeyIdx, sigGasIdx)
	require.Less(t, sigGasIdx, sigVerifyIdx,
		"SigGasConsumeDecorator (index %d) must precede the signature verifier (index %d) so verification is paid for",
		sigGasIdx, sigVerifyIdx)
}
