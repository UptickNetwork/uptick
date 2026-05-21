package v033

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const upgradeName = "v0.3.3"

// Upgrade activates the erc20 IBC outbound refund security fix: refunds on
// error acknowledgements require MsgTransferERC20 provenance instead of
// user-controlled memo substrings. Provenance uses a new prefix in the
// existing erc20 KVStore; no additional module store is required.
var Upgrade = upgrades.Upgrade{
	UpgradeName:               upgradeName,
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades:             &storetypes.StoreUpgrades{},
}

func upgradeHandlerConstructor(
	_ *module.Manager,
	c module.Configurator,
	box upgrades.Toolbox,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(
			"executing upgrade plan",
			"name", upgradeName,
			"change", "erc20 IBC outbound refund requires transfer provenance",
		)
		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}
