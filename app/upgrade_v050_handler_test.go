package app

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	erc20types "github.com/cosmos/evm/x/erc20/types"
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

// handlerVoucherDenom is a voucher unique to this test. The hash only has to be
// the 64-character hex string ibc-go derives, since the ERC20 address is hashed
// out of it.
const (
	handlerVoucherDenom  = "ibc/C0FFEE0000000000000000000000000000000000000000000000000000000000"
	handlerVoucherSource = "auoc"
	handlerVoucherPath   = "transfer/channel-7/auoc"
)

// TestV050UpgradeHandlerBackfillsIBCVoucherDecimalsAndPairs runs the real handler
// against a real voucher on the real application, which is the only place the
// migrations' wiring to the bank and erc20 keepers is proven.
//
// The two assertions are the two halves of the v0.5.0 repair: the metadata comes
// back in the shape that makes decimals() report 18 instead of 0, and the denom
// gains a token pair it could never have acquired while the inbound gate is
// closed. The v050 package's own tests pin the rewrite rules; this one pins that
// the handler actually calls them, on the same starting state a testnet node is
// in.
func TestV050UpgradeHandlerBackfillsIBCVoucherDecimalsAndPairs(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	// Seed the voucher the way ibc-go does: supply in the bank, and metadata
	// whose Display is the full denom path with a single unit at exponent 0.
	//
	// The supply is created through the EVM module account -- the one account
	// this app grants both Minter and Burner -- so the fixture can be undone with
	// BurnCoins, which is what removes the denom from the supply table again.
	coins := sdk.NewCoins(sdk.NewInt64Coin(handlerVoucherDenom, 1_000_000))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, evmtypes.ModuleName, coins))
	app.BankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
		Description: "IBC token from " + handlerVoucherPath,
		DenomUnits:  []*banktypes.DenomUnit{{Denom: handlerVoucherSource, Exponent: 0}},
		Base:        handlerVoucherDenom,
		Display:     handlerVoucherPath,
		Name:        handlerVoucherPath + " IBC token",
		Symbol:      "AUOC",
	})

	upgradeStore := ctx.KVStore(app.GetKey(upgradetypes.StoreKey))
	require.False(t, upgradeStore.Has([]byte(migrationsAppliedProbeKey)),
		"the replay marker must not exist before the handler runs")
	require.False(t, app.Erc20Keeper.IsDenomRegistered(ctx, handlerVoucherDenom),
		"the fixture must start without a pair, otherwise the backfill proves nothing")

	// The shared app is a process-wide singleton: undo the seeded supply, the
	// pair and the marker so no other test inherits this fixture.
	t.Cleanup(func() {
		if pair, found := app.Erc20Keeper.GetTokenPair(ctx, app.Erc20Keeper.GetTokenPairID(ctx, handlerVoucherDenom)); found {
			app.Erc20Keeper.DeleteTokenPair(ctx, pair)
			app.Erc20Keeper.DeleteDynamicPrecompile(ctx, pair.GetERC20Contract())
			require.NoError(t, app.Erc20Keeper.UnRegisterERC20CodeHash(ctx, pair.GetERC20Contract()))
		}
		require.NoError(t, app.BankKeeper.BurnCoins(ctx, evmtypes.ModuleName, coins))
		upgradeStore.Delete([]byte(migrationsAppliedProbeKey))
	})

	handler := v050.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	_, err := handler(ctx, v050Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)

	// A-01: the metadata is in shape C and is legal bank state at last.
	metadata, found := app.BankKeeper.GetDenomMetaData(ctx, handlerVoucherDenom)
	require.True(t, found)
	require.Equal(t, handlerVoucherSource, metadata.Display,
		"Display must be the source denom so the precompile's last-segment rule finds the exponent unit")
	require.Equal(t, []*banktypes.DenomUnit{
		{Denom: handlerVoucherDenom, Exponent: 0},
		{Denom: handlerVoucherSource, Exponent: 18},
	}, metadata.DenomUnits, "auoc is atto-prefixed, so decimals() must come back as 18")
	require.NoError(t, metadata.Validate(),
		"the on-chain metadata must stop being the illegal shape the export side had to work around")

	// A-02: the voucher has an EVM representation, backed by a dynamic precompile
	// rather than by a deployed contract.
	require.True(t, app.Erc20Keeper.IsDenomRegistered(ctx, handlerVoucherDenom),
		"the backfill must register a pair for the seeded voucher")
	pair, found := app.Erc20Keeper.GetTokenPair(ctx, app.Erc20Keeper.GetTokenPairID(ctx, handlerVoucherDenom))
	require.True(t, found)
	require.Equal(t, erc20types.OWNER_MODULE, pair.ContractOwner,
		"the pair must be module-owned: the address is hashed out of the voucher, nothing is deployed")
	require.True(t, pair.Enabled)
	require.True(t, app.Erc20Keeper.IsDynamicPrecompileAvailable(ctx, pair.GetERC20Contract()),
		"the derived address must be an active dynamic precompile, otherwise calls to it fail")
}

// TestV050UpgradeHandlerEnablesPermissionlessRegistration pins the x/erc20
// parameter the release ships, on the starting state a testnet node is in.
//
// Why it has to run here and not merely in the v050 unit tests: the value comes
// from the real keeper's params store, and the only way a chain already on
// v0.4.x can reach it is this handler's own step -- v0.4.0's migrateErc20Params
// wrote the switch off and will never run again on that chain. The fixture is
// therefore the recorded state of such a chain, not a fresh app, whose genesis
// would default the parameter on and prove nothing.
func TestV050UpgradeHandlerEnablesPermissionlessRegistration(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	// restoreSharedState covers the EVM, ICA and feemarket params, not erc20's,
	// so this test has to undo its own fixture: the app is a process-wide
	// singleton and the parameter would otherwise leak into every test after it.
	previous := app.Erc20Keeper.GetParams(ctx)
	t.Cleanup(func() { require.NoError(t, app.Erc20Keeper.SetParams(ctx, previous)) })

	require.NoError(t, app.Erc20Keeper.SetParams(ctx, erc20types.NewParams(true, false)))
	require.False(t, app.Erc20Keeper.GetParams(ctx).PermissionlessRegistration,
		"the fixture must start with the switch off, which is exactly the state the migration exists for")

	upgradeStore := ctx.KVStore(app.GetKey(upgradetypes.StoreKey))
	t.Cleanup(func() { upgradeStore.Delete([]byte(migrationsAppliedProbeKey)) })

	handler := v050.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())
	_, err := handler(ctx, v050Plan(), app.mm.GetVersionMap())
	require.NoError(t, err)

	got := app.Erc20Keeper.GetParams(ctx)
	require.True(t, got.PermissionlessRegistration,
		"v0.5.0 must ship the switch ON, and on this starting state nothing else can turn it on")
	require.True(t, got.EnableErc20,
		"the flip is a read-modify-write; a rebuild from DefaultParams would hide a legacy chain that had conversion off")
}
