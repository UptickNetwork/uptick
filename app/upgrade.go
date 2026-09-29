package app

import (
	"errors"
	"fmt"
	"os"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
	// v030-v032 were removed: they were build-ignored historical upgrades with
	// SDK 0.50 API incompatibilities and stale ethermint fork fields.
	v033 "github.com/UptickNetwork/uptick/app/upgrades/v033"
	// v040 and v041 are deliberately not imported here. Their packages still
	// build into the binary -- v050 calls their handler constructors directly --
	// but neither name is registered as a plan this binary will execute. See the
	// note on `router` below: the absence is the guard.
	v050 "github.com/UptickNetwork/uptick/app/upgrades/v050"
)

// router holds exactly the upgrade plans this binary is meant to serve.
//
// The table is scoped per release AND per chain. x/upgrade's startup self-check
// (abci.go:38-62) requires the name of the last completed upgrade to have a
// handler, so the table must contain the plan name of EVERY chain this binary is
// pointed at -- and it should contain nothing else. Mainnet is the only chain
// v0.5.0 serves: it is stopped on v0.3.3 and upgrades to v0.5.0, which replays
// the v0.4.0 change set and then the v0.4.1 repairs itself (see v050's package
// doc for why the starting state, not a flag, decides which stage runs).
//
// # The roll-out, release by release
//
//	release   registers                serves
//	v0.5.0    v0.3.3, v0.5.0           mainnet (v0.3.3 -> v0.5.0); testnet NOT served
//	v0.5.1    + v0.4.1, + v0.5.1       both: mainnet (v0.5.0 -> v0.5.1) and
//	                                   testnet (v0.4.1 -> v0.5.1)
//
// Testnet stays on v0.4.1 through this release and follows a plan named v0.5.1
// later, so it is absent from the table above on purpose. That v0.5.1 row is not
// a preference: the same self-check forces it. Once mainnet has run its v0.5.0
// plan, v0.5.0 is mainnet's LAST COMPLETED name and every mainnet node aborts on
// start unless the running binary has a handler for it; testnet's last completed
// name stays v0.4.1 until its own plan executes, so v0.5.1 owes a handler for
// both names. v0.5.1 also carries the v0.5.0 handler's tail migrations forward,
// because testnet never runs that plan and would otherwise keep an erc20 switch
// that is off, voucher metadata that reports decimals() == 0 and voucher pairs
// that were never backfilled.
//
// v0.4.0 and v0.4.1 are absent HERE on purpose, and the absence is the guard:
//
//   - x/upgrade keeps a name schedulable until it has a done record, and mainnet
//     has none for either. Registering v0.4.0 would leave it reachable by
//     governance, and its legacy-pair deletion is not idempotent -- a replayed
//     plan deletes the pairs registered after the upgrade. Registering v0.4.1
//     would leave reachable a handler that cannot run on a v0.3.3 chain at all:
//     its first repair reads the ICA controller params (v041/upgrades.go:68, and
//     before the module manager gets to run), and ibc-go v10's controller keeper
//     PANICS rather than defaulting when the module store has no "params" key
//     (27-interchain-accounts/controller/keeper/keeper.go:307-315). That is the
//     state a chain which never ran the v0.4.0 change set is in, those params
//     having lived in the legacy x/params subspace.
//   - With neither name registered, such a plan cannot reach ApplyUpgrade: the
//     module fails the upgrade height with "UPGRADE NEEDED" and halts, and an
//     operator clears it with --unsafe-skip-upgrades. A halt is recoverable; a
//     replayed deletion is not.
//
// Neither removal touches the upgrade path: v050 builds both of its stages from
// the v040 and v041 packages directly, never by looking them up here.
//
// The cost, deliberate and temporary: this binary must NOT be handed to a node
// stopped on v0.4.1 -- that is the testnet, and it aborts on every start with
// "upgrade handler is missing for v0.4.1 upgrade plan". v0.5.1 registers v0.4.1
// back, which is a line in that release's table and not a change to this one.
var (
	router = upgrades.NewUpgradeRouter().
		Register(v033.Upgrade).
		Register(v050.Upgrade)
)

// RegisterUpgradePlans register a handler of upgrade plan
func (app *Uptick) RegisterUpgradePlans() {
	app.setupUpgradeStoreLoaders()
	app.setupUpgradeHandlers()
}

func (app *Uptick) toolbox() upgrades.Toolbox {
	return upgrades.Toolbox{
		AppCodec:      app.AppCodec(),
		ModuleManager: app.mm,
		ReaderWriter:  app,
		AppKeepers:    app.AppKeepers,
	}
}

// isMissingUpgradeInfo reports whether an error from ReadUpgradeInfoFromDisk
// means "no upgrade is scheduled" — the benign case for a fresh node or a chain
// with no upgrade in flight — as opposed to a corrupt or half-written
// upgrade-info.json, which must abort startup instead of silently skipping the
// store upgrades this release requires.
func isMissingUpgradeInfo(err error) bool {
	return os.IsNotExist(err) || errors.Is(err, os.ErrNotExist)
}

// configure store loader that checks if version == upgradeHeight and applies store upgrades
func (app *Uptick) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		if isMissingUpgradeInfo(err) {
			// No pending upgrade scheduled: the normal path for a fresh node or
			// a chain with no upgrade in flight.
			return
		}
		// The upgrade info file exists but cannot be read/parsed (corrupted or
		// half-written). Continuing would silently skip the store upgrades this
		// release requires and boot the node into an inconsistent state, so fail
		// loudly and let the operator repair or remove the file.
		panic(fmt.Errorf("failed to read upgrade info from disk: %w", err))
	}

	// If upgradeInfo has no height, return without setting up store loader.
	// Silent on purpose: this is the steady state of a node with nothing
	// scheduled, so logging it would print a line on every startup.
	if upgradeInfo.Height == 0 {
		return
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		// An operator asked for this height to be skipped. Say so - the node
		// will keep running the old logic while its peers migrate, and the only
		// other record of that decision is a number in app.toml.
		app.Logger().Info(
			"skipping the upgrade scheduled at this height",
			"name", upgradeInfo.Name,
			"height", upgradeInfo.Height,
		)
		return
	}

	// Check if the upgrade exists in our router
	upgrade, exists := router.Routers()[upgradeInfo.Name]
	if !exists {
		// A plan this binary does not know about: there are no store upgrades
		// to load, but do not return silently. This is what a binary that
		// reached a peer's upgrade height looks like, and an operator needs to
		// be able to tell it apart from "no upgrade pending".
		app.Logger().Info(
			"upgrade plan is not registered in this binary; no store loader installed",
			"name", upgradeInfo.Name,
			"height", upgradeInfo.Height,
		)
		return
	}

	app.SetStoreLoader(
		upgradetypes.UpgradeStoreLoader(
			upgradeInfo.Height,
			upgrade.StoreUpgrades,
		),
	)
}

func (app *Uptick) setupUpgradeHandlers() {
	box := app.toolbox()
	for upgradeName, upgrade := range router.Routers() {
		app.UpgradeKeeper.SetUpgradeHandler(
			upgradeName,
			upgrade.UpgradeHandlerConstructor(
				app.mm,
				app.configurator,
				box,
			),
		)
	}
}
