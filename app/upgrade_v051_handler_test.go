package app

import (
	"testing"

	storetypes "cosmossdk.io/store/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"

	v051 "github.com/UptickNetwork/uptick/app/upgrades/v051"
	appminingtypes "github.com/UptickNetwork/uptick/x/appmining/types"
)

// testBaselineAuthority is a syntactically valid stand-in for the release ops
// multisig: the handler test pins the authority write without depending on
// the real address, which is configured in v1BaselineAuthority before any
// release build. Lazily derived: a package-level var would render before the
// package init() flips the bech32 prefix to "uptick" and produce a cosmos1...
// string that params.Validate rejects.
var testBaselineAuthority = func() string {
	return sdk.AccAddress([]byte("v051-baseline-authority")).String()
}

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

	// Inject a stand-in ops authority: the release gate refuses an empty
	// v1BaselineAuthority, and this test pins the write, not the address.
	// The pause authority is stubbed to empty as well: the release build fills
	// both constants, and the fallback (pause == authority) this test asserts
	// only triggers on an empty slot.
	v051.SetBaselineAuthorityForTest(t, testBaselineAuthority())
	v051.SetBaselinePauseAuthorityForTest(t, "")

	handler := v051.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	newVM, err := handler(ctx, v051Plan(), vm)
	require.NoError(t, err)
	require.Equal(t, uint64(2), newVM[appminingtypes.ModuleName],
		"appmining must be initialized at ConsensusVersion 2 (challenge indexes + timeout sweep)")

	require.True(t, upgradeStore.Has([]byte(migrationsAppliedV051Key)))
	require.NotNil(t, app.AccountKeeper.GetModuleAccount(ctx, appminingtypes.RewardPoolName),
		"appmining_reward_pool module account must exist after upgrade")
	require.NotNil(t, app.AccountKeeper.GetModuleAccount(ctx, appminingtypes.ChallengeEscrowName),
		"appmining_challenge_escrow module account must exist after upgrade")
	acc := app.AccountKeeper.GetModuleAccount(ctx, appminingtypes.ModuleName)
	require.NotNil(t, acc)

	params := app.AppminingKeeper.GetParams(ctx)
	require.NoError(t, params.Validate())

	// V1 baseline must be live at the upgrade height: denom derived from the
	// chain's staking bond denom, fund path unlocked, default-deny lists open.
	bondDenom, err := app.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, bondDenom, "test app must have a staking bond denom")
	require.Equal(t, bondDenom, params.ChallengeBondDenom,
		"baseline bond denom must follow the chain, not the binary default")
	require.Equal(t, bondDenom, params.ChallengeDepositDenom)
	require.Equal(t, []string{bondDenom}, params.RewardDenomAllowlist,
		"reward allowlist must gate on the chain denom or pool/fund/budget all fail")
	require.Equal(t, "1000000", params.EntitlementEnableMinFunded,
		"empty min_funded would leave V1 silently disabled")
	require.NotEmpty(t, params.EntitlementAllowedRuleVersions,
		"empty rule allowlist denies every approval (error 77)")
	require.NotEmpty(t, params.EntitlementAllowedAssessmentVersions,
		"empty assessment allowlist denies every credential")
	require.True(t, params.CredentialRequired)
	require.Equal(t, appminingtypes.ChallengeOpenAccessDeposit, params.ChallengeOpenAccess)
	require.NotEmpty(t, params.ChallengeDepositAmount)
	require.Equal(t, testBaselineAuthority(), params.Authority,
		"upgrade must install the ops authority so remaining setup runs as plain txs")
	// Empty v1BaselinePauseAuthority falls back to the authority address
	// (kill switch held by the same multisig).
	require.Equal(t, testBaselineAuthority(), params.PauseAuthority,
		"pause authority must default to the ops authority when not split")

	// Replay must be a no-op beyond RunMigrations: baseline params survive.
	_, err = handler(ctx, v051Plan(), newVM)
	require.NoError(t, err)
	afterReplay := app.AppminingKeeper.GetParams(ctx)
	require.Equal(t, params, afterReplay,
		"replayed plan must not re-write (or wipe) the baseline params")
}

// TestV051UpgradeHandlerSplitsPauseAuthority pins the independent slot: when
// v1BaselinePauseAuthority is configured, the kill switch goes to that address
// instead of mirroring the admin authority.
func TestV051UpgradeHandlerSplitsPauseAuthority(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	upgradeStore := ctx.KVStore(app.GetKey(upgradetypes.StoreKey))
	require.False(t, upgradeStore.Has([]byte(migrationsAppliedV051Key)))
	t.Cleanup(func() {
		upgradeStore.Delete([]byte(migrationsAppliedV051Key))
	})

	vm := app.mm.GetVersionMap()
	delete(vm, appminingtypes.ModuleName)

	v051.SetBaselineAuthorityForTest(t, testBaselineAuthority())
	splitPause := func() string { return sdk.AccAddress([]byte("v051-oncall-pause-authority")).String() }
	v051.SetBaselinePauseAuthorityForTest(t, splitPause())

	handler := v051.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	_, err := handler(ctx, v051Plan(), vm)
	require.NoError(t, err)

	params := app.AppminingKeeper.GetParams(ctx)
	require.Equal(t, testBaselineAuthority(), params.Authority)
	require.Equal(t, splitPause(), params.PauseAuthority,
		"a configured pause authority must be installed verbatim, not overwritten by the authority fallback")
	require.NoError(t, params.Validate())
}

func TestV051UpgradeHandlerRefusesLegacyState(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	evmStore := ctx.KVStore(app.GetKey(evmtypes.StoreKey))
	require.True(t, evmStore.Has(evmtypes.KeyPrefixEvmCoinInfo))
	// The shared app outlives this test: capture the record so the cleanup can
	// put it back. Without it every later upgrade-handler test trips the
	// legacy-state gate, which makes the suite fail purely by execution order.
	original := append([]byte{}, evmStore.Get(evmtypes.KeyPrefixEvmCoinInfo)...)
	evmStore.Delete(evmtypes.KeyPrefixEvmCoinInfo)
	t.Cleanup(func() {
		evmStore.Set(evmtypes.KeyPrefixEvmCoinInfo, original)
		restoreSharedState(t, app)
	})

	handler := v051.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	_, err := handler(ctx, v051Plan(), app.mm.GetVersionMap())
	require.Error(t, err)
	require.Contains(t, err.Error(), "legacy ethermint")
}

// TestV051UpgradeFromV050LeavesModuleConsistent covers the path a v0.5.0
// chain actually takes, as opposed to a chain that already ran v0.5.1: x/appmining
// is a NEW module there, so RunMigrations runs InitGenesis instead of the 1->2
// migration (SDK types/module: a module missing from the version map is
// initialized, never migrated). Three consequences follow, and none of them is
// visible from the keeper tests:
//
//   - the module lands in the version map at ConsensusVersion 2 directly, so a
//     later upgrade will not replay Migrate1to2;
//   - InitGenesis' own BackfillChallengeIndexes leaves the derived indexes
//     complete at the upgrade height, so the first block has nothing to rebuild
//     (the bare-swap path is the opposite: it rebuilds in the first block);
//   - the legacy pause posture comes from DefaultParams (all four flows paused),
//     not from Migrate1to2, so a v0.5.0->v0.5.1 chain and a bare-swapped v0.5.1
//     chain reach the same posture through different code paths.
func TestV051UpgradeFromV050LeavesModuleConsistent(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	upgradeStore := ctx.KVStore(app.GetKey(upgradetypes.StoreKey))
	require.False(t, upgradeStore.Has([]byte(migrationsAppliedV051Key)))
	t.Cleanup(func() {
		upgradeStore.Delete([]byte(migrationsAppliedV051Key))
	})

	// v0.5.0 has no x/appmining: drop it from the version map so the manager
	// treats it exactly as it would on that chain.
	vm := app.mm.GetVersionMap()
	delete(vm, appminingtypes.ModuleName)

	// The shared app has already produced one block, and that block's EndBlocker
	// marked the indexes ready. Delete the marker to restore the state a store
	// created at the upgrade height actually has: the module exists, nothing has
	// swept it yet.
	appStore := ctx.KVStore(app.GetKey(appminingtypes.StoreKey))
	appStore.Delete(appminingtypes.ChallengeIndexStateKey())
	require.False(t, app.AppminingKeeper.ChallengeIndexesReady(ctx),
		"test setup: the marker must be absent before the upgrade runs")

	v051.SetBaselineAuthorityForTest(t, testBaselineAuthority())
	v051.SetBaselinePauseAuthorityForTest(t, "")

	handler := v051.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	newVM, err := handler(ctx, v051Plan(), vm)
	require.NoError(t, err)
	require.Equal(t, uint64(2), newVM[appminingtypes.ModuleName])

	// x/appmining's InitGenesis ends with BackfillChallengeIndexes (keeper/genesis.go):
	// imported state must offer the same guarantees as state built by open/resolve,
	// so the module seals pre-field OPEN records and rebuilds the derived indexes
	// before returning. A module initialized at the upgrade height is therefore
	// already consistent, and the first block after the upgrade has no rebuild to do.
	//
	// This is the one behavioural difference from the bare-swap path on a chain that
	// already ran v0.5.1: there the indexes are absent and the EndBlocker rebuilds
	// them in the first block (see workorder/UPTICK-APPMINING-SWAP-VERIFY-0.1.md).
	require.True(t, app.AppminingKeeper.ChallengeIndexesReady(ctx),
		"InitGenesis must leave the indexes complete: a v0.5.0 chain must not need a first-block scan")

	params := app.AppminingKeeper.GetParams(ctx)
	require.True(t, params.PauseState.AttestationPaused, "legacy attestation must be paused via DefaultParams")
	require.True(t, params.PauseState.ChallengePaused, "legacy challenge must be paused via DefaultParams")
	require.True(t, params.PauseState.SettlementPaused, "legacy settlement must be paused via DefaultParams")
	require.True(t, params.PauseState.GasCreditPaused, "legacy gas credit must be paused via DefaultParams")

	// The first block must be a no-op for this module, not a rebuild. Run the
	// whole EndBlock on a cache branch: the ordering is what has to hold, and
	// the shared app's store must not keep the writes.
	cctx, _ := ctx.CacheContext()
	// EndBlock runs under an infinite block gas meter in baseapp (abci.go), and
	// the shared context has none: give the branch what a real block provides.
	cctx = cctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	_, err = app.mm.EndBlock(cctx)
	require.NoError(t, err)
	require.True(t, app.AppminingKeeper.ChallengeIndexesReady(cctx),
		"the EndBlocker must find the indexes already complete")

	// Steady state: the next block re-reads the marker and writes nothing new.
	_, err = app.mm.EndBlock(cctx)
	require.NoError(t, err)
	require.True(t, app.AppminingKeeper.ChallengeIndexesReady(cctx))
}
