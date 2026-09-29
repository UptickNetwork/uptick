package app

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"cosmossdk.io/log"
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

// reapplyVoucher* describe the voucher the re-apply test seeds. The hash only
// has to be a 64-character hex string, which is what ibc-go derives.
const (
	reapplyVoucherDenom  = "ibc/1DEA000000000000000000000000000000000000000000000000000000000000"
	reapplyVoucherSource = "auoc"
	reapplyVoucherPath   = "transfer/channel-9/auoc"
)

// tailReport is what the three migrations at the tail of the v0.5.0 handler say
// they did during one run. Their summary lines are the only observable that
// separates "the guard fired" from "the step wrote the same value again", which
// end-state comparison cannot see.
type tailReport struct {
	enabledLines    int // "permissionless erc20 registration enabled"
	alreadyOnLines  int // "permissionless erc20 registration already enabled"
	normalized      int
	pairsRegistered int
	pairsAlreadyHad int
	pairsFailed     int
}

// tailState is everything the tail migrations are allowed to touch.
type tailState struct {
	params        erc20types.Params
	metadata      banktypes.Metadata
	metadataFound bool
	pairs         []string
}

func tailStateOf(app *Uptick, ctx sdk.Context, denom string) tailState {
	metadata, found := app.BankKeeper.GetDenomMetaData(ctx, denom)

	pairs := make([]string, 0, 8)
	for _, pair := range app.Erc20Keeper.GetTokenPairs(ctx) {
		pairs = append(pairs, pair.Denom+"="+pair.Erc20Address)
	}
	sort.Strings(pairs)

	return tailState{
		params:        app.Erc20Keeper.GetParams(ctx),
		metadata:      metadata,
		metadataFound: found,
		pairs:         pairs,
	}
}

// parseTailReport reads the JSON log lines the handler emitted. The logger is
// replaced per run (see the test below) so the report is per-run by construction.
func parseTailReport(t *testing.T, logged string) tailReport {
	t.Helper()

	var report tailReport
	for _, line := range strings.Split(logged, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		message, _ := entry["message"].(string)
		count := func(key string) int {
			if v, ok := entry[key].(float64); ok {
				return int(v)
			}
			return 0
		}

		switch message {
		case "permissionless erc20 registration enabled":
			report.enabledLines++
		case "permissionless erc20 registration already enabled":
			report.alreadyOnLines++
		case "ibc voucher erc20 decimals normalized":
			report.normalized = count("normalized")
		case "ibc voucher token pair backfill complete":
			report.pairsRegistered = count("registered")
			report.pairsAlreadyHad = count("already_registered")
			report.pairsFailed = count("failed")
		}
	}
	return report
}

// TestV050TailMigrationsAreSafeToReapplyWithoutTheMarker pins the property the
// NEXT release depends on, not a property of this one.
//
// The v0.5.1 release has to reach two chains that are in different places. The
// testnet stops on v0.4.1 and never runs the v0.5.0 plan at all, so the three
// migrations at the tail of this handler are the only route by which its erc20
// parameter, its voucher metadata and its voucher pairs ever get repaired.
// Mainnet, already past v0.5.0, runs those same three migrations for a second
// time. v0.5.1 therefore reuses them directly, and does NOT inherit this plan's
// replay marker: the marker is keyed to the string "v0.5.0/migrations-applied".
//
// This test deletes that marker and applies the handler twice, which is what
// v0.5.1 will do to mainnet. It asserts two things the reuse rests on:
//
//   - each step reports that its guard fired, not that it repaired something
//     (the counters below); and
//   - the end state is unchanged, and the second application returns no error.
//
// The fixture is a legacy chain that had coin conversion OFF, because EnableErc20
// is the field a rebuild-from-DefaultParams would silently flip to true, and the
// second application would then be a real change rather than a no-op.
func TestV050TailMigrationsAreSafeToReapplyWithoutTheMarker(t *testing.T) {
	app, ctx := sharedTestApp(t)
	restoreSharedState(t, app)

	// erc20 params are outside restoreSharedState, as in the test above.
	previousParams := app.Erc20Keeper.GetParams(ctx)
	t.Cleanup(func() { require.NoError(t, app.Erc20Keeper.SetParams(ctx, previousParams)) })
	require.NoError(t, app.Erc20Keeper.SetParams(ctx, erc20types.NewParams(false, false)))
	require.False(t, app.Erc20Keeper.GetParams(ctx).PermissionlessRegistration)

	coins := sdk.NewCoins(sdk.NewInt64Coin(reapplyVoucherDenom, 1_000_000))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, evmtypes.ModuleName, coins))
	// Shape A, exactly as ibc-go writes it: Display is the full denom path and
	// the only unit carries exponent 0, which is why decimals() reads 0.
	app.BankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
		Description: "IBC token from " + reapplyVoucherPath,
		DenomUnits:  []*banktypes.DenomUnit{{Denom: reapplyVoucherSource, Exponent: 0}},
		Base:        reapplyVoucherDenom,
		Display:     reapplyVoucherPath,
		Name:        reapplyVoucherPath + " IBC token",
		Symbol:      "AUOC",
	})

	upgradeStore := ctx.KVStore(app.GetKey(upgradetypes.StoreKey))
	require.False(t, upgradeStore.Has([]byte(migrationsAppliedProbeKey)),
		"the replay marker must not exist before the handler runs")
	t.Cleanup(func() {
		if pair, found := app.Erc20Keeper.GetTokenPair(ctx, app.Erc20Keeper.GetTokenPairID(ctx, reapplyVoucherDenom)); found {
			app.Erc20Keeper.DeleteTokenPair(ctx, pair)
			app.Erc20Keeper.DeleteDynamicPrecompile(ctx, pair.GetERC20Contract())
			require.NoError(t, app.Erc20Keeper.UnRegisterERC20CodeHash(ctx, pair.GetERC20Contract()))
		}
		require.NoError(t, app.BankKeeper.BurnCoins(ctx, evmtypes.ModuleName, coins))
		upgradeStore.Delete([]byte(migrationsAppliedProbeKey))
	})

	handler := v050.Upgrade.UpgradeHandlerConstructor(app.mm, app.configurator, app.toolbox())

	// Each run gets its own logger, so the report describes one run rather than
	// the accumulated output of the test.
	run := func(t *testing.T) tailReport {
		t.Helper()

		var logged bytes.Buffer
		runCtx := ctx.WithLogger(log.NewLogger(&logged, log.OutputJSONOption()))

		_, err := handler(runCtx, v050Plan(), app.mm.GetVersionMap())
		require.NoError(t, err)

		return parseTailReport(t, logged.String())
	}

	first := run(t)
	require.Equal(t, 1, first.enabledLines,
		"the first run must turn the switch on; an unchanged parameter means the fixture no longer reproduces the state the migration exists for")
	require.Zero(t, first.alreadyOnLines)
	require.NotZero(t, first.normalized,
		"the first run must repair the seeded voucher, otherwise the second run's zero proves nothing")
	require.NotZero(t, first.pairsRegistered,
		"the first run must backfill the seeded voucher's pair")

	afterFirst := tailStateOf(app, ctx, reapplyVoucherDenom)

	// This is the line v0.5.1 will not have. What follows is a chain that has
	// already run these migrations being asked to run them again.
	upgradeStore.Delete([]byte(migrationsAppliedProbeKey))

	second := run(t)
	require.Zero(t, second.enabledLines,
		"the switch was already on: writing it again means the step stopped being a read-modify-write and would clobber whatever a legacy chain had set")
	require.Equal(t, 1, second.alreadyOnLines,
		"the second run has to report the guard firing, which is the only observable that distinguishes a skipped write from a repeated one")
	require.Zero(t, second.normalized,
		"the voucher was already in shape C: re-deriving the decimals a second time is how a real asset gets silently rescaled")
	require.Zero(t, second.pairsRegistered,
		"the voucher already had its pair: registering a second time must not be attempted")
	require.NotZero(t, second.pairsAlreadyHad)
	require.Zero(t, second.pairsFailed,
		"the backfill must not treat an existing pair as an error")

	require.Equal(t, afterFirst, tailStateOf(app, ctx, reapplyVoucherDenom),
		"re-applying the tail migrations must leave erc20 params, voucher metadata and the token pair set exactly as they were")
}
