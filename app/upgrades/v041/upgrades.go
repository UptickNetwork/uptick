package v041

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const upgradeName = "v0.4.1"

// Upgrade is the v0.4.1 upgrade. There is no state migration required; this
// upgrade ships the Keplr compatibility fix (legacy ethermint pubkey and
// EIP-712 extension option decoding) which lives entirely in the binary runtime
// (encoding config + ante handler), so the handler only runs module migrations.
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
			"change", "activate static precompiles; Keplr legacy ethermint pubkey / EIP-712 tx compatibility",
		)

		// Repair the EVM params: v0.4.0 introduced ActiveStaticPrecompiles but
		// left it empty, so every custom static precompile was inactive.
		if err := migrateActiveStaticPrecompiles(sdkCtx, box); err != nil {
			return nil, fmt.Errorf("migrate active static precompiles: %w", err)
		}

		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}
