package app

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"

	v051 "github.com/UptickNetwork/uptick/app/upgrades/v051"
	appminingtypes "github.com/UptickNetwork/uptick/x/appmining/types"
)

func v051Plan() upgradetypes.Plan {
	return upgradetypes.Plan{Name: v051.Upgrade.UpgradeName, Height: 100}
}

const migrationsAppliedV051Key = "v0.5.1/migrations-applied"

// TestV051UpgradeHandlerAddsAppmining executes the real handler on the shared
// app (cosmos/evm starting state, same shape as post-v0.4.1 testnet / post-v0.5.0
// mainnet) and checks that appmining lands in the version map and the module
// account exists.
func TestV051UpgradeHandlerAddsAppmining(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	require.True(t, ctx.KVStore(app.GetKey(evmtypes.StoreKey)).Has(evmtypes.KeyPrefixEvmCoinInfo),
		"shared app must carry EvmCoinInfo or the handler refuses the upgrade")

	upgradeStore := ctx.KVStore(app.GetKey(upgradetypes.StoreKey))
	require.False(t, upgradeStore.Has([]byte(migrationsAppliedV051Key)))
	t.Cleanup(func() {
		upgradeStore.Delete([]byte(migrationsAppliedV051Key))
	})

	// Drop appmining from the version map so RunMigrations treats it as new
	// (fresh shared apps may already include it from genesis wiring).
	vm := app.mm.GetVersionMap()
	delete(vm, appminingtypes.ModuleName)

	handler := v051.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	newVM, err := handler(ctx, v051Plan(), vm)
	require.NoError(t, err)
	require.Equal(t, uint64(1), newVM[appminingtypes.ModuleName],
		"appmining must be initialized at ConsensusVersion 1")

	require.True(t, upgradeStore.Has([]byte(migrationsAppliedV051Key)))
	acc := app.AccountKeeper.GetModuleAccount(ctx, appminingtypes.ModuleName)
	require.NotNil(t, acc)

	params := app.AppminingKeeper.GetParams(ctx)
	require.NoError(t, params.Validate())
	require.NotEmpty(t, params.AllowedProfiles)

	// Replay must be a no-op beyond RunMigrations.
	_, err = handler(ctx, v051Plan(), newVM)
	require.NoError(t, err)
}

func TestV051UpgradeHandlerRefusesLegacyState(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	evmStore := ctx.KVStore(app.GetKey(evmtypes.StoreKey))
	require.True(t, evmStore.Has(evmtypes.KeyPrefixEvmCoinInfo))
	evmStore.Delete(evmtypes.KeyPrefixEvmCoinInfo)
	t.Cleanup(func() {
		// Shared app: leave store as we found it is hard; restoreSharedState on
		// other tests rebuilds from genesis. Re-seed a placeholder so later
		// suites still see the key if they share the process.
		restoreSharedState(t, app)
	})

	handler := v051.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	_, err := handler(ctx, v051Plan(), app.mm.GetVersionMap())
	require.Error(t, err)
	require.Contains(t, err.Error(), "legacy ethermint")
}
