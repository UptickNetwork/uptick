package v041

import (
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// defaultActiveStaticPrecompiles is the list of static precompile addresses
// registered by cosmos/evm's precompilestypes.DefaultStaticPrecompiles, which
// Uptick wires into the EVM keeper. It intentionally excludes the vesting
// precompile (0x803), which Uptick does not register, to avoid the
// "precompiled contract not stored in memory" panic when the EVM resolves it.
var defaultActiveStaticPrecompiles = []string{
	evmtypes.P256PrecompileAddress,
	evmtypes.Bech32PrecompileAddress,
	evmtypes.StakingPrecompileAddress,
	evmtypes.DistributionPrecompileAddress,
	evmtypes.ICS20PrecompileAddress,
	evmtypes.BankPrecompileAddress,
	evmtypes.GovPrecompileAddress,
	evmtypes.SlashingPrecompileAddress,
}

// ConfigureDefaultStaticPrecompiles overrides the EVM module's default params
// so fresh chains also activate the static precompiles (the cosmos/evm default
// is an empty list). It must be called before any evmtypes.DefaultParams()
// invocation — i.e. before the module manager (and its DefaultGenesis) is
// built in app.go. It replaces the former package init(): a global mutation
// as an import side effect is invisible to readers and easy to lose during
// refactors, so the app wires it explicitly.
func ConfigureDefaultStaticPrecompiles() {
	evmtypes.DefaultStaticPrecompiles = defaultActiveStaticPrecompiles
}

// evmParamsStore is the slice of the EVM keeper this migration needs.
//
// It is an interface rather than the concrete keeper on purpose: the real keeper
// can only fail SetParams on invalid params, and this migration always writes a
// known-good list, so with the concrete type the error branch below would be
// unreachable from any test. Narrowing the dependency makes the failure path
// injectable instead of untested.
type evmParamsStore interface {
	GetParams(ctx sdk.Context) evmtypes.Params
	SetParams(ctx sdk.Context, params evmtypes.Params) error
}

// migrateActiveStaticPrecompiles repairs the EVM params so the static
// precompiles are actually activated. The v0.4.0 upgrade introduced the
// ActiveStaticPrecompiles params field (the legacy ethermint params proto had no
// such field) but never populated it, leaving every custom precompile inactive:
// IsAvailableStaticPrecompile returns false, so GetStaticPrecompileInstance
// never loads the contract and precompile calls fail.
func migrateActiveStaticPrecompiles(ctx sdk.Context, store evmParamsStore) error {
	params := store.GetParams(ctx)
	updated, changed := withDefaultActiveStaticPrecompiles(params)
	if !changed {
		return nil
	}
	if err := store.SetParams(ctx, updated); err != nil {
		return fmt.Errorf("set evm params: %w", err)
	}
	return nil
}

// withDefaultActiveStaticPrecompiles returns params with the default precompile
// list filled in when the stored list is empty. An already-populated list is
// preserved so governance removals are not silently reverted.
func withDefaultActiveStaticPrecompiles(params evmtypes.Params) (evmtypes.Params, bool) {
	// An empty list means "reset to the default set". Note that proto round-trips
	// collapse an explicitly-empty list to nil, so a governance decision to clear
	// the list (disable all static precompiles) is not distinguishable from
	// "never configured" here; both reset to the default.
	if len(params.ActiveStaticPrecompiles) != 0 {
		return params, false
	}
	params.ActiveStaticPrecompiles = append([]string(nil), defaultActiveStaticPrecompiles...)
	return params, true
}
