package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
)

// TestStaticPrecompilesResolveToContracts covers the second half of the
// precompile wiring. The upgrade-handler tests assert the *params* list; the
// EVM keeper resolves a call through its own precompile map, and the two are
// independent. If params names an address the keeper holds no contract for,
// GetStaticPrecompileInstance panics with "precompiled contract not stored in
// memory" — on the first call, on chain, not at startup.
//
// That pairing is also what makes the ordering in app/keepers/keepers.go
// matter: WithStaticPrecompiles is called only after the ERC20 keeper exists
// (the comment there says so), because the ICS20 precompile captures it. A
// precompile map built from zero-value keepers would still resolve here, so
// this test does not prove the ordering — nothing observable does. What it
// proves is that the activated set and the registered set agree.
func TestStaticPrecompilesResolveToContracts(t *testing.T) {
	app, ctx := sharedTestApp(t)

	params := app.EvmKeeper.GetParams(ctx)
	require.NotEmpty(t, params.ActiveStaticPrecompiles, "the chain must ship with the precompiles active")

	for _, addr := range params.ActiveStaticPrecompiles {
		instance, available, err := app.EvmKeeper.GetStaticPrecompileInstance(&params, common.HexToAddress(addr))
		require.NoError(t, err, "resolving %s must not fail", addr)
		require.True(t, available, "%s is listed in ActiveStaticPrecompiles", addr)
		require.NotNil(t, instance, "no contract registered for the activated precompile %s", addr)
	}

	// The vesting precompile is the concrete case behind the exclusion list: it
	// is part of cosmos/evm's default set but Uptick never registers it, so
	// activating it panics the first time anything calls 0x803.
	_, available, err := app.EvmKeeper.GetStaticPrecompileInstance(
		&params, common.HexToAddress(evmtypes.VestingPrecompileAddress))
	require.NoError(t, err)
	require.False(t, available, "the vesting precompile must stay inactive: Uptick registers no contract for it")
}
