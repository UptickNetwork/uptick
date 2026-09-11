package app

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"

	v041 "github.com/UptickNetwork/uptick/app/upgrades/v041"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
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

// restoreSharedState captures the EVM, ICA and feemarket params and puts them
// back when the test ends. The shared app is a process-wide singleton (see
// shared_testapp_test.go), so a test that leaves repaired state behind would
// make the next test's starting point depend on execution order - the same
// class of bug as the unrestored evmtypes.DefaultStaticPrecompiles global
// (audit O-02).
func restoreSharedState(t *testing.T, app *Uptick) {
	t.Helper()

	evmParams := app.EvmKeeper.GetParams(sharedTestAppCtx)
	icaParams := app.ICAControllerKeeper.GetParams(sharedTestAppCtx)
	feeMarketParams := app.FeeMarketKeeper.GetParams(sharedTestAppCtx)
	t.Cleanup(func() {
		require.NoError(t, app.EvmKeeper.SetParams(sharedTestAppCtx, evmParams))
		app.ICAControllerKeeper.SetParams(sharedTestAppCtx, icaParams)
		require.NoError(t, app.FeeMarketKeeper.SetParams(sharedTestAppCtx, feeMarketParams))
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

// legacyFeeMarketParamsHex is the pre-upgrade feemarket Params protobuf byte for
// byte, as ethermint v0.24.1-uptick wrote it:
//
//	field 2  base_fee_change_denominator = 8
//	field 3  elasticity_multiplier       = 2
//	field 5  enable_height               = 0
//	field 6  base_fee                    = "1000000000"  (math.Int ASCII, 1 gwei)
//	field 7  min_gas_price               = "0"
//	field 8  min_gas_multiplier          = "500000000000000000"
//
// Field 6 is the payload that changes meaning: math.Int stores the value
// directly, math.LegacyDec stores it scaled by 10^18. The v041 package's
// migrate_test.go pins that decoding difference against the real cosmos/evm
// type; this test proves the consequence on the real app, real store and real
// keeper.
const legacyFeeMarketParamsHex = "100818022800320a313030303030303030303a01304212353030303030303030303030303030303030"

// TestV041UpgradeHandlerRepairsLegacyFeeMarketBaseFee is the case the
// feemarket repair exists for. A chain that ran the v0.4.0 upgrade holds
// feemarket Params written by ethermint, whose base_fee is a math.Int. The
// replaced cosmos/evm module reads the same field as a math.LegacyDec, so the
// stored 1 gwei decodes as 10^-9 - a valid protobuf, a wrong number, and no
// error anywhere. feemarket's BeginBlock then recomputes and writes the base fee
// back every block, so the wrong value is promoted into consensus state and is
// what block.basefee reports to every contract.
func TestV041UpgradeHandlerRepairsLegacyFeeMarketBaseFee(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	// Reproduce the pre-upgrade store contents directly, so the test does not
	// depend on the migration under test to build its own input.
	legacy, err := hex.DecodeString(legacyFeeMarketParamsHex)
	require.NoError(t, err)
	ctx.KVStore(app.GetKey(feemarkettypes.StoreKey)).Set(feemarkettypes.ParamsKey, legacy)

	// The mis-read is silent. This is the chain state after v0.4.0.
	require.Equal(t, "0.000000001000000000", app.FeeMarketKeeper.GetParams(ctx).BaseFee.String(),
		"the legacy 1 gwei must decode 10^18 too small, otherwise this test is not reproducing the bug")

	handler := v041.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	_, err = handler(ctx, v041Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)

	require.Equal(t, feemarkettypes.DefaultBaseFee.String(), app.FeeMarketKeeper.GetParams(ctx).BaseFee.String(),
		"the handler must restore 1 gwei; a ~0 base fee disables the EIP-1559 fee market and the fee burn")

	// A crash-restart replays the plan. The rescale must not apply twice: the
	// repaired value is a whole number of wei, so it falls outside the guard.
	_, err = handler(ctx, v041Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)
	require.Equal(t, feemarkettypes.DefaultBaseFee.String(), app.FeeMarketKeeper.GetParams(ctx).BaseFee.String(),
		"a replayed plan must not multiply the base fee by another 10^18")
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

// ---------------------------------------------------------------------------
// ERC721 conversion index prune
// ---------------------------------------------------------------------------

// erc721UIDIndexSnapshot reads one of the two ERC721 conversion indexes out of
// the shared app's store. The prune is the only v0.4.1 repair that writes these
// prefixes, so comparing the whole prefix across a replayed plan is a direct
// idempotency check instead of a restatement of the implementation.
func erc721UIDIndexSnapshot(ctx sdk.Context, app *Uptick, prefix []byte) map[string]string {
	iter := storetypes.KVStorePrefixIterator(ctx.KVStore(app.GetKey(erc721types.StoreKey)), prefix)
	defer iter.Close()

	out := make(map[string]string)
	for ; iter.Valid(); iter.Next() {
		out[string(iter.Key())] = string(iter.Value())
	}
	return out
}

// TestV041UpgradeHandlerPrunesDuplicateERC721UIDIndex is the case the prune
// exists for. Mainnet holds 1110 forward conversion keys against 1078 reverse
// entries; the 32 unpaired leftovers make every genesis export report degraded,
// which is what buries real corruption (an undecodable pair, an orphaned refund
// key) in fixed noise on the one path that matters most.
//
// The fixture is mainnet's shape, not a synthetic one: the reverse index points
// at the lowercase spelling and the checksummed one is the pre-v0.4.0 leftover
// that the post-v0.4.1 write path can no longer reach.
func TestV041UpgradeHandlerPrunesDuplicateERC721UIDIndex(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	const (
		batchTokenID  = "1703751205993357472"
		lowerBatch    = "0x3bc44cb88233f75b858d0748a45d196be8375159"
		checksumBatch = "0x3bc44CB88233f75B858d0748a45d196bE8375159"
		batchClassID  = "uptick-3bc44cb88233f75b858d0748a45d196be8375159"
		batchNFTID    = "uptick1703751205993357472"
	)

	authority := erc721types.CreateTokenUID(lowerBatch, batchTokenID)
	shadow := erc721types.CreateTokenUID(checksumBatch, batchTokenID)
	nftUID := erc721types.CreateNFTUID(batchClassID, batchNFTID)
	require.NotEqual(t, authority, shadow, "the fixture has to differ in spelling to be a duplicate")

	app.Erc721Keeper.SetNFTUIDPairByNFTUID(ctx, nftUID, authority)
	app.Erc721Keeper.SetNFTUIDPairByTokenUID(ctx, authority, nftUID)
	app.Erc721Keeper.SetNFTUIDPairByTokenUID(ctx, shadow, nftUID)

	// The shared app is a process-wide singleton, so the seeded rows have to go
	// away again or the next test's starting state depends on execution order.
	t.Cleanup(func() {
		app.Erc721Keeper.DeleteNFTUIDPairByTokenUID(sharedTestAppCtx, authority)
		app.Erc721Keeper.DeleteNFTUIDPairByTokenUID(sharedTestAppCtx, shadow)
		app.Erc721Keeper.DeleteNFTUIDPairByNFTUID(sharedTestAppCtx, nftUID)
	})

	handler := v041.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())

	vm, err := handler(ctx, v041Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)
	require.NotEmpty(t, vm, "the handler must return the post-migration version map")

	require.Equal(t, nftUID, string(app.Erc721Keeper.GetNFTUIDPairByTokenUID(ctx, authority)),
		"the key the reverse index points at must survive the prune")
	require.Empty(t, app.Erc721Keeper.GetNFTUIDPairByTokenUID(ctx, shadow),
		"the duplicate key must be gone, otherwise every genesis export stays degraded")
	require.Equal(t, authority, string(app.Erc721Keeper.GetTokenUIDPairByNFTUID(ctx, nftUID)),
		"the prune deletes forward duplicates only; the reverse index keeps pointing where it did")

	// A crash-restart replays the plan, and v0.4.1 deliberately has no
	// UpgradeAlreadyApplied guard to stop it. This second run is the one that
	// would double-apply any repair that is not idempotent.
	forwardBefore := erc721UIDIndexSnapshot(ctx, app, erc721types.KeyPrefixNFTUIDPairByTokenUID)
	reverseBefore := erc721UIDIndexSnapshot(ctx, app, erc721types.KeyPrefixNFTUIDPairByNFTUID)

	_, err = handler(ctx, v041Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)

	require.Equal(t, forwardBefore,
		erc721UIDIndexSnapshot(ctx, app, erc721types.KeyPrefixNFTUIDPairByTokenUID),
		"a replayed plan must leave the forward index exactly as the first pass left it")
	require.Equal(t, reverseBefore,
		erc721UIDIndexSnapshot(ctx, app, erc721types.KeyPrefixNFTUIDPairByNFTUID),
		"the reverse index is never rewritten by the prune")
}
