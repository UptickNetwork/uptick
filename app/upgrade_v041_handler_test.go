package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/core/appmodule"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"

	v041 "github.com/UptickNetwork/uptick/app/upgrades/v041"
)

// Audit O-01: the v0.4.1 upgrade handler had no execution test. Only its pure
// helpers (the precompile list, the fill-in rule, the ICA predicate) were
// covered, so nothing proved that `upgradeHandlerConstructor` actually wires
// them to the keepers, that the repaired state lands in the store, or that a
// failure from any step reaches the caller instead of being swallowed - an
// upgrade handler that returns nil on error stops the chain in the middle of a
// migration and calls it success.
//
// These tests run the real handler against a real application. A mocked Toolbox
// is not an option: Toolbox embeds the concrete keepers.AppKeepers, so the only
// honest way to execute the handler is on an app.

// expectedActivePrecompiles is restated here on purpose. Asserting against
// evmtypes.DefaultStaticPrecompiles would compare the migration output with the
// very global the app configures from the same list, so a typo shared by both
// would pass. This is the list the v0.4.1 spec promises to activate.
var expectedActivePrecompiles = []string{
	evmtypes.P256PrecompileAddress,
	evmtypes.Bech32PrecompileAddress,
	evmtypes.StakingPrecompileAddress,
	evmtypes.DistributionPrecompileAddress,
	evmtypes.ICS20PrecompileAddress,
	evmtypes.BankPrecompileAddress,
	evmtypes.GovPrecompileAddress,
	evmtypes.SlashingPrecompileAddress,
}

// v041Plan builds the plan the handler is executed for.
func v041Plan() upgradetypes.Plan {
	return upgradetypes.Plan{Name: v041.Upgrade.UpgradeName, Height: 42}
}

// restoreSharedState captures the EVM and ICA params and puts them back when the
// test ends. The shared app is a process-wide singleton (see
// shared_testapp_test.go), so a test that leaves repaired state behind would
// make the next test's starting point depend on execution order - the same
// class of bug as the unrestored evmtypes.DefaultStaticPrecompiles global
// (audit O-02).
func restoreSharedState(t *testing.T, app *Uptick) {
	t.Helper()

	evmParams := app.EvmKeeper.GetParams(sharedTestAppCtx)
	icaParams := app.ICAControllerKeeper.GetParams(sharedTestAppCtx)
	t.Cleanup(func() {
		require.NoError(t, app.EvmKeeper.SetParams(sharedTestAppCtx, evmParams))
		app.ICAControllerKeeper.SetParams(sharedTestAppCtx, icaParams)
	})
}

// TestV041UpgradeHandlerRepairsPrecompilesAndICA is the release-blocking case:
// a chain upgrading from v0.4.0 carries an EMPTY ActiveStaticPrecompiles list
// and (for chains derived from the legacy genesis template) a disabled ICA
// controller. Both repairs must land, or every static precompile call fails and
// every ICA channel registration is rejected after the upgrade.
func TestV041UpgradeHandlerRepairsPrecompilesAndICA(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	// Reproduce the pre-upgrade state.
	params := app.EvmKeeper.GetParams(ctx)
	require.NotEmpty(t, params.ActiveStaticPrecompiles, "the app should start with the precompiles active")
	params.ActiveStaticPrecompiles = nil
	require.NoError(t, app.EvmKeeper.SetParams(ctx, params))
	require.Empty(t, app.EvmKeeper.GetParams(ctx).ActiveStaticPrecompiles)

	app.ICAControllerKeeper.SetParams(ctx, icacontrollertypes.NewParams(false))
	require.False(t, app.ICAControllerKeeper.GetParams(ctx).ControllerEnabled)

	handler := v041.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())

	// v0.4.1 bumps no consensus version, so a chain coming from v0.4.0 hands the
	// handler a version map equal to the current consensus versions. That is
	// exactly what GetVersionMap() returns, and it is also why the handler must
	// not carry the UpgradeAlreadyApplied guard: the guard would report
	// "applied" on the first legitimate run and skip both repairs.
	vm, err := handler(ctx, v041Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)
	require.NotEmpty(t, vm, "the handler must return the post-migration version map")

	got := app.EvmKeeper.GetParams(ctx)
	require.Equal(t, expectedActivePrecompiles, got.ActiveStaticPrecompiles,
		"the handler must activate exactly the shipped precompile set")
	require.NotContains(t, got.ActiveStaticPrecompiles, evmtypes.VestingPrecompileAddress,
		"activating the vesting precompile panics the EVM with 'precompiled contract not stored in memory'")

	require.True(t, app.ICAControllerKeeper.GetParams(ctx).ControllerEnabled,
		"ICA controller must be enabled, otherwise every register fails with 'controller submodule is disabled'")
}

// TestV041UpgradeHandlerPreservesConfiguredPrecompiles covers the "already
// configured" case and the replay path in one go: both repairs are idempotent,
// and a non-empty list is a governance decision that must survive. Re-running
// the (idempotent) handler after the repair is what a crash-restart does.
func TestV041UpgradeHandlerPreservesConfiguredPrecompiles(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	handler := v041.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())

	custom := []string{evmtypes.BankPrecompileAddress}
	params := app.EvmKeeper.GetParams(ctx)
	params.ActiveStaticPrecompiles = custom
	require.NoError(t, app.EvmKeeper.SetParams(ctx, params))

	// First run: the store already holds a populated list, so nothing is
	// overwritten...
	_, err := handler(ctx, v041Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)
	require.Equal(t, custom, app.EvmKeeper.GetParams(ctx).ActiveStaticPrecompiles,
		"a populated list is a governance decision; re-filling it would silently revert it")

	// ...and a replay of the same plan stays on the same values (idempotency),
	// which is what makes the missing UpgradeAlreadyApplied guard safe.
	_, err = handler(ctx, v041Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)
	require.Equal(t, custom, app.EvmKeeper.GetParams(ctx).ActiveStaticPrecompiles)
}

// TestV041UpgradeHandlerPropagatesRunMigrationsError pins the last line of the
// handler. `return box.ModuleManager.RunMigrations(...)` is the only thing that
// turns a failed module migration into a failed upgrade; writing `return vm,
// nil` there would make a broken migration look like a successful upgrade and
// let the chain start on half-migrated state.
//
// The failure is produced with the real configurator: the fake module reports
// consensus version 2 while the incoming version map says 1, and no migration
// is registered for it - so RunMigrations reports ErrNotFound, which the
// handler must pass on.
func TestV041UpgradeHandlerPropagatesRunMigrationsError(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	manager := module.NewManagerFromMap(map[string]appmodule.AppModule{"fakebump": fakeVersionedModule{}})
	toolbox := app.toolbox()
	toolbox.ModuleManager = manager

	handler := v041.Upgrade.UpgradeHandlerConstructor(manager, app.configurator, toolbox)

	_, err := handler(ctx, v041Plan(), module.VersionMap{"fakebump": 1})

	require.Error(t, err, "a failed module migration must fail the upgrade")
	require.ErrorIs(t, err, sdkerrors.ErrNotFound)
	require.ErrorContains(t, err, "fakebump",
		"the error must name the module so an operator knows what blocked the upgrade")
}

// fakeVersionedModule is the smallest module that makes RunMigrations take the
// migration path: a name, the two appmodule marker methods, and a consensus
// version higher than the one the test hands in.
type fakeVersionedModule struct{}

func (fakeVersionedModule) Name() string             { return "fakebump" }
func (fakeVersionedModule) IsOnePerModuleType()      {}
func (fakeVersionedModule) IsAppModule()             {}
func (fakeVersionedModule) ConsensusVersion() uint64 { return 2 }
