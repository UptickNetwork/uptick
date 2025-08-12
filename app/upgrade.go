package app

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
	v030 "github.com/UptickNetwork/uptick/app/upgrades/v030"
)

var (
	router = upgrades.NewUpgradeRouter().
		Register(v030.Upgrade)
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

// configure store loader that checks if version == upgradeHeight and applies store upgrades
func (app *Uptick) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		// If there's no upgrade info, just return without setting up store loader
		return
	}

	// If upgradeInfo has no height, return without setting up store loader
	if upgradeInfo.Height == 0 {
		return
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	// Check if the upgrade exists in our router
	upgrade, exists := router.Routers()[upgradeInfo.Name]
	if !exists {
		// If upgrade doesn't exist in our router, return without setting up store loader
		return
	}

	// Always apply store upgrades for v0.3.0 upgrade to ensure icacontroller store is added
	if upgradeInfo.Name == "v0.3.0" && upgrade.StoreUpgrades != nil {
		app.SetStoreLoader(
			upgradetypes.UpgradeStoreLoader(
				upgradeInfo.Height,
				upgrade.StoreUpgrades,
			),
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
