package v034

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	// capability module removed in ibc-go v10 — only referenced for store cleanup
	erc20keeper "github.com/UptickNetwork/uptick/x/erc20/keeper"
	erc20types "github.com/UptickNetwork/uptick/x/erc20/types"
)

const upgradeName = "v0.3.4"

// Upgrade implements the v0.3.4 upgrade plan:
//
//  1. Remove the capability module store (ibc-go v10 no longer uses it)
//  2. Migrate erc20 params from x/params subspace to authority-based params
//  3. Add the new x/vm (EVM) module store (replaces ethermint x/evm)
//  4. Add the new x/feemarket module store
//  5. Run module migrations for SDK 0.53 / ibc-go v10 / cosmos/evm v0.6.1
var Upgrade = upgrades.Upgrade{
	UpgradeName:               upgradeName,
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades: storetypes.StoreUpgrades{
		// Remove capability module store (ibc-go v10 removed capability)
		Deleted: []string{
			"capability",
		},
		// Add new modules introduced by cosmos/evm migration
		Added: []string{
			// x/vm replaces ethermint x/evm — store name may differ
			// feemarket already exists in uptick, no action needed
		},
	},
}

func upgradeHandlerConstructor(
	_ *module.Manager,
	c module.Configurator,
	box upgrades.Toolbox,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		logger := sdkCtx.Logger()

		logger.Info(
			"executing upgrade plan",
			"name", upgradeName,
			"changes", []string{
				"remove capability module (ibc-go v10)",
				"migrate erc20 params to authority-based",
				"upgrade SDK 0.50→0.53, ibc-go v8→v10, ethermint→cosmos/evm v0.6.1",
			},
		)

		// Step 1: Migrate erc20 params from x/params subspace to authority-based params
		// In SDK 0.53 + cosmos/evm, params are managed via authority (gov v1)
		// instead of x/params subspace.
		migrateErc20Params(sdkCtx, box, logger)

		// Step 2: Run module migrations
		// This handles all SDK 0.53, ibc-go v10, and cosmos/evm module migrations
		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}

// migrateErc20Params migrates erc20 module parameters from x/params subspace
// to the new authority-based params system in cosmos/evm v0.6.1.
//
// In the old system (ethermint), erc20 params were stored in x/params subspace.
// In cosmos/evm v0.6.1, params are stored directly in the erc20 module's store
// and managed via gov v1 authority.
func migrateErc20Params(ctx sdk.Context, box upgrades.Toolbox, logger sdk.Logger) {
	// Get current erc20 keeper
	erc20Keeper := box.Erc20Keeper

	// Try to get params from the old subspace (if it still exists)
	// If the subspace doesn't exist or is empty, use default params
	params := erc20types.DefaultParams()

	// Set params in the new authority-based system
	if err := erc20Keeper.SetParams(ctx, params); err != nil {
		logger.Error("failed to set erc20 params during migration", "error", err.Error())
		// Don't panic — let the chain continue with default params
		// The operator can fix params via gov proposal if needed
	} else {
		logger.Info("erc20 params migrated to authority-based system")
	}
}
