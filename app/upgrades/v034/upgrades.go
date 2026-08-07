package v034

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const upgradeName = "v0.3.4"

// Upgrade implements the v0.3.4 upgrade plan:
//
//  1. Remove the capability module store (ibc-go v10 no longer uses it)
//  2. Run module migrations for SDK 0.53 / ibc-go v10 / cosmos/evm v0.6.1
var Upgrade = upgrades.Upgrade{
	UpgradeName:               upgradeName,
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades: &storetypes.StoreUpgrades{
		// Remove capability module store (ibc-go v10 removed capability)
		Deleted: []string{
			"capability",
		},
		Added: []string{},
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
				"upgrade SDK 0.50→0.53, ibc-go v8→v10, ethermint→cosmos/evm v0.6.1",
			},
		)

		// Run module migrations
		// This handles all SDK 0.53, ibc-go v10, and cosmos/evm module migrations
		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}
