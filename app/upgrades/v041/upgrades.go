package v041

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const upgradeName = "v0.4.1"

// Upgrade is the v0.4.1 upgrade. It repairs state left behind by earlier
// versions: activates the static precompiles and enables the ICA controller
// submodule. The Keplr compatibility fix (legacy ethermint pubkey and EIP-712
// extension option decoding) lives entirely in the binary runtime (encoding
// config + ante handler), so no migration is needed for it.
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
			"change", "activate static precompiles; enable ICA controller; Keplr legacy ethermint pubkey / EIP-712 tx compatibility",
		)

		// Repair the EVM params: v0.4.0 introduced ActiveStaticPrecompiles but
		// left it empty, so every custom static precompile was inactive.
		if err := migrateActiveStaticPrecompiles(sdkCtx, box); err != nil {
			return nil, fmt.Errorf("migrate active static precompiles: %w", err)
		}

		// Enable the ICA controller submodule: genesis templates derived from
		// the legacy x/params defaults carry controller_enabled=false, and the
		// ibc-go v10 param migration keeps the stored value as-is, so every
		// ICA register fails with "controller submodule is disabled".
		migrateICAControllerParams(sdkCtx, box)

		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}

// migrateICAControllerParams flips the ICA controller submodule on if it is
// currently disabled. Chains initialized from legacy genesis templates (e.g.
// the origin testnet) store controller_enabled=false; fresh chains already
// default to true and are left untouched. Host params are not modified.
func migrateICAControllerParams(ctx sdk.Context, box upgrades.Toolbox) {
	params := box.ICAControllerKeeper.GetParams(ctx)
	if params.ControllerEnabled {
		return
	}
	params.ControllerEnabled = true
	box.ICAControllerKeeper.SetParams(ctx, icacontrollertypes.NewParams(true))
	ctx.Logger().Info("ica controller submodule enabled", "upgrade", upgradeName)
}
