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

// Upgrade is a historical no-op kept for replay compatibility. It originally
// activated an erc20 IBC outbound refund security fix; that self-developed
// erc20 module (and MsgTransferERC20) was removed in v0.4.0 in favor of
// cosmos/evm's x/erc20, so this handler now only runs module migrations.
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
