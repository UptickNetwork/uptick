package app

import (
	"fmt"

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
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	app.SetStoreLoader(
		upgradetypes.UpgradeStoreLoader(
			upgradeInfo.Height,
			router.UpgradeInfo(upgradeInfo.Name).StoreUpgrades,
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
