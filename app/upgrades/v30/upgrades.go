package v30

import (
	"context"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"fmt"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
)

var (
	feemarketParams = feemarkettypes.Params{
		NoBaseFee:                false,
		BaseFeeChangeDenominator: 8,
		ElasticityMultiplier:     4,
		BaseFee:                  math.NewInt(10000000000),
		MinGasPrice:              math.LegacyNewDecFromInt(math.NewInt(10000000000)),
		MinGasMultiplier:         math.LegacyNewDecWithPrec(5, 1),
	}
)

// Upgrade defines a struct containing necessary fields that a SoftwareUpgradeProposal
var Upgrade = upgrades.Upgrade{
	UpgradeName:               "v0.3.0",
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades:             &storetypes.StoreUpgrades{},
}

func upgradeHandlerConstructor(
	m *module.Manager,
	c module.Configurator,
	box upgrades.Toolbox,
) upgradetypes.UpgradeHandler {
	return func(context context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

		ctx := sdk.UnwrapSDKContext(context)
		ctx.Logger().Info("execute a upgrade plan...")
		////return box.ModuleManager.RunMigrations(ctx, c, vm)
		baseAppLegacySS := box.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())
		// Migrate Tendermint consensus parameters from x/params module to a dedicated x/consensus module.
		baseapp.MigrateParams(ctx, baseAppLegacySS, &box.ConsensusParamsKeeper.ParamsStore)

		if err := box.FeeMarketKeeper.SetParams(ctx, generateFeemarketParams(ctx.BlockHeight())); err != nil {
			panic(fmt.Errorf("failed to FeeMarketKeeper SetParams "))
		}

		wasmParams := box.WasmKeeper.GetParams(ctx)
		wasmParams.CodeUploadAccess.Permission = wasmtypes.AccessTypeEverybody
		wasmParams.InstantiateDefaultPermission = wasmtypes.AccessTypeEverybody
		if err := box.WasmKeeper.SetParams(ctx, wasmParams); err != nil {
			panic(fmt.Errorf("failed to wasmKeeper SetParams "))
		}

		gs := ibcnfttransfertypes.DefaultGenesisState()
		bz, err := ibcnfttransfertypes.ModuleCdc.MarshalJSON(gs)
		if err != nil {
			panic(fmt.Errorf("failed to ModuleCdc %s: %w", ibcnfttransfertypes.ModuleName, err))
		}
		if module, ok := box.ModuleManager.Modules[ibcnfttransfertypes.ModuleName].(module.HasGenesis); ok {
			module.InitGenesis(ctx, ibcnfttransfertypes.ModuleCdc, bz)
		}

		return box.ModuleManager.RunMigrations(ctx, c, vm)
	}
}

func generateFeemarketParams(enableHeight int64) feemarkettypes.Params {
	feemarketParams.EnableHeight = enableHeight
	return feemarketParams
}
