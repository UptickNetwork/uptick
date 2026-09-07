package v041

import (
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
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

// migrateActiveStaticPrecompiles repairs the EVM params so the static
// precompiles are actually activated. The v0.4.0 upgrade introduced the
// ActiveStaticPrecompiles params field (the legacy ethermint params proto had no
// such field) but never populated it, leaving every custom precompile inactive:
// IsAvailableStaticPrecompile returns false, so GetStaticPrecompileInstance
// never loads the contract and precompile calls fail.
func migrateActiveStaticPrecompiles(ctx sdk.Context, box upgrades.Toolbox) error {
	params := box.EvmKeeper.GetParams(ctx)
	updated, changed := withDefaultActiveStaticPrecompiles(params)
	if !changed {
		return nil
	}
	if err := box.EvmKeeper.SetParams(ctx, updated); err != nil {
		return fmt.Errorf("set evm params: %w", err)
	}
	return nil
}

// withDefaultActiveStaticPrecompiles returns params with the default precompile
// list filled in when the stored list is empty. An already-populated list is
// preserved so governance removals are not silently reverted.
func withDefaultActiveStaticPrecompiles(params evmtypes.Params) (evmtypes.Params, bool) {
	if len(params.ActiveStaticPrecompiles) != 0 {
		return params, false
	}
	params.ActiveStaticPrecompiles = append([]string(nil), defaultActiveStaticPrecompiles...)
	return params, true
}
