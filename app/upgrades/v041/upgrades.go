package v041

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
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

		// No UpgradeAlreadyApplied guard here (unlike v040): v0.4.1 bumps no
		// module ConsensusVersion, so a chain upgrading from v0.4.0 already
		// has a version map equal to the current consensus versions and the
		// guard would wrongly skip the one-shot repairs below on their first
		// (and only legitimate) run. Both repairs are inherently idempotent —
		// migrateActiveStaticPrecompiles only writes when the param is empty
		// and migrateICAControllerParams only flips a disabled flag — so a
		// replayed plan is harmless without the guard.

		sdkCtx.Logger().Info(
			"executing upgrade plan",
			"name", upgradeName,
			"change", "activate static precompiles; enable ICA controller; Keplr legacy ethermint pubkey / EIP-712 tx compatibility",
		)

		// Repair the EVM params: v0.4.0 introduced ActiveStaticPrecompiles but
		// left it empty, so every custom static precompile was inactive.
		if err := migrateActiveStaticPrecompiles(sdkCtx, box.EvmKeeper); err != nil {
			return nil, fmt.Errorf("migrate active static precompiles: %w", err)
		}

		// Enable the ICA controller submodule: genesis templates derived from
		// the legacy x/params defaults carry controller_enabled=false, and the
		// ibc-go v10 param migration keeps the stored value as-is, so every
		// ICA register fails with "controller submodule is disabled".
		migrateICAControllerParams(sdkCtx, box.ICAControllerKeeper)

		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}

// icaControllerParamsStore is the slice of the ICA controller keeper this
// migration needs.
//
// It is an interface rather than the concrete keeper for the same reason as
// evmParamsStore above, plus one specific to this call: the write is guarded by
// a read of the value it is about to write, and against the real keeper that
// guard is invisible. ibc-go's SetParams returns nothing (it panics on a store
// error) and writing true over an already-true param produces byte-identical
// state, so "did we write when we should not have?" cannot be observed from the
// store. Recording the calls makes the guard fail-able from a test.
type icaControllerParamsStore interface {
	GetParams(ctx sdk.Context) icacontrollertypes.Params
	SetParams(ctx sdk.Context, params icacontrollertypes.Params)
}

// migrateICAControllerParams flips the ICA controller submodule on if it is
// currently disabled. Chains initialized from legacy genesis templates (e.g.
// the origin testnet) store controller_enabled=false; fresh chains already
// default to true and are left untouched. Host params are not modified.
//
// SetParams has no error to propagate: ibc-go's controller keeper panics
// internally if the store rejects the write, so the only failure mode left for
// this function to get wrong is the guard.
func migrateICAControllerParams(ctx sdk.Context, store icaControllerParamsStore) {
	if store.GetParams(ctx).ControllerEnabled {
		return
	}
	store.SetParams(ctx, icacontrollertypes.NewParams(true))
	ctx.Logger().Info("ica controller submodule enabled", "upgrade", upgradeName)
}
