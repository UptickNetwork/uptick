package app

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"

	v050 "github.com/UptickNetwork/uptick/app/upgrades/v050"
)

// v050Plan builds the plan the handler is executed for.
func v050Plan() upgradetypes.Plan {
	return upgradetypes.Plan{Name: v050.Upgrade.UpgradeName, Height: 42}
}

// migrationsAppliedProbeKey restates the replay marker the handler writes. It is
// written out by hand on purpose: the assertion is that an operator inspecting a
// halted node finds exactly these bytes in the x/upgrade store, so a handler that
// moved the marker elsewhere has to fail here rather than silently lose its
// replay protection.
const migrationsAppliedProbeKey = "v0.5.0/migrations-applied"

// TestV050UpgradeHandlerOnUpgradedState executes the real handler on the real
// application, from the starting state a testnet node is in.
//
// What this covers: the x/upgrade store key the replay guard needs really
// resolves, the modern branch runs through the chained v0.4.1 handler and
// returns a usable version map, and the marker lands in the store.
//
// What it cannot cover: the mainnet branch. Producing an absent EvmCoinInfo
// record means a v0.3.3 store, which is a different dependency tree (legacy
// ethermint) and cannot be assembled in this binary. That side is pinned by
// TestEvmCoinInfoProbe and TestRunMigrationSetMainnetPath in
// app/upgrades/v050, and the migration steps themselves by the v0.4.0 and
// v0.4.1 suites.
func TestV050UpgradeHandlerOnUpgradedState(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	// cosmos/evm's x/vm InitGenesis persists the EvmCoinInfo record, so a fresh
	// app is in the same shape as a chain that already ran the v0.4.0 set. If
	// this ever stops holding, this test silently starts exercising the mainnet
	// path, so it is asserted rather than assumed.
	require.True(t, ctx.KVStore(app.GetKey(evmtypes.StoreKey)).Has(evmtypes.KeyPrefixEvmCoinInfo),
		"the shared app must carry the EvmCoinInfo record, otherwise this test runs the wrong branch")

	upgradeStore := ctx.KVStore(app.GetKey(upgradetypes.StoreKey))
	require.False(t, upgradeStore.Has([]byte(migrationsAppliedProbeKey)),
		"the replay marker must not exist before the handler runs")
	t.Cleanup(func() {
		upgradeStore.Delete([]byte(migrationsAppliedProbeKey))
	})

	handler := v050.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())

	vm, err := handler(ctx, v050Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)
	require.NotEmpty(t, vm, "the handler must return the post-migration version map")

	require.True(t, upgradeStore.Has([]byte(migrationsAppliedProbeKey)),
		"the handler must record that its sequence completed; without the marker a replayed plan re-runs the non-idempotent v0.4.0 stage")

	// A replay (crash-restart, or the plan re-proposed at a new height) must
	// return a usable version map instead of erroring.
	vm, err = handler(ctx, v050Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)
	require.NotEmpty(t, vm)
}
