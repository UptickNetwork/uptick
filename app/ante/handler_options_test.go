package ante

import (
	"context"
	"fmt"
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

// TestSetUpContextPrecedesWasmCountTX pins the dfe8cee (round 18) G-5 fix:
// SetUpContextDecorator MUST come before CountTXDecorator (and any other
// wasm KV-writing decorator), so the tx gas meter, not the infinite
// BaseApp preset meter, accounts for CountTX's KV writes. A regression
// here would let a malicious tx inflate the count without paying gas.
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
// counterpart to TestWasmDecoratorsOmittedWhenKeepersNil: WasmNodeConfig
// is set but WasmKeeper is nil, so GasRegisterDecorator must NOT appear.
// End-to-end integration tests in app_test.go exercise the real
// (non-nil) WasmKeeper branch.
func TestGasRegisterDecoratorRequiresWasmKeeper(t *testing.T) {
	decorators := cosmosAnteDecorators(
		HandlerOptions{WasmNodeConfig: &wasmtypes.NodeConfig{}},
		&feemarkettypes.Params{},
		func(*codectypes.Any) bool { return false },
		ante.NewSigVerificationDecorator(nil, nil),
		nil,
	)

	for _, d := range decorators {
		if fmt.Sprintf("%T", d) == "*wasmkeeper.GasRegisterDecorator" {
			t.Fatalf("GasRegisterDecorator must NOT appear when WasmKeeper == nil")
		}
	}
}
