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
	v040 "github.com/UptickNetwork/uptick/app/upgrades/v040"
	v041 "github.com/UptickNetwork/uptick/app/upgrades/v041"
)

var (
	router = upgrades.NewUpgradeRouter().
		Register(v033.Upgrade).
		Register(v040.Upgrade).
		Register(v041.Upgrade)
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
