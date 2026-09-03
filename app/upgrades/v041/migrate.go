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

func init() {
	// Fix the EVM module's default params so fresh chains also activate the
	// static precompiles (the cosmos/evm default is an empty list).
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
	if len(params.ActiveStaticPrecompiles) != 0 {
		// Preserve any existing governance / on-chain decision. Overwriting an
		// already-populated list would silently revert an administrator's choice
		// (e.g. removing the distribution precompile to mitigate a vulnerability).
		return nil
	}
	params.ActiveStaticPrecompiles = defaultActiveStaticPrecompiles
	if err := box.EvmKeeper.SetParams(ctx, params); err != nil {
		return fmt.Errorf("set evm params: %w", err)
	}
	return nil
}
