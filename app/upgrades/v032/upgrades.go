package v032

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/x/evm/statedb"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
)

var (
	create2FactoryAddress = "0x4e59b44847b379578588920cA78FbF26c0B4956C"
	create2FactoryRuntime = "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3"

	// EIP-3855 (PUSH0): activated via x/evm Params.ExtraEIPs, wired to vm.EnableEIP in ethermint.
	eip3855Extra = int64(3855)

	defaultFeemarketParams = feemarkettypes.Params{
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
	UpgradeName:               "v0.3.2",
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
		ctx.Logger().Info("executing upgrade plan", "name", Upgrade.UpgradeName)

		migrateConsensusParamsToXConsensus(ctx, box)
		if err := ensureConsensusBlockGasForEVM(ctx, box.ConsensusParamsKeeper); err != nil {
			return vm, fmt.Errorf("ensure consensus block gas for EVM: %w", err)
		}
		if err := setFeemarketParams(ctx, box); err != nil {
			return vm, err
		}
		if err := setWasmParams(ctx, box); err != nil {
			return vm, err
		}
		if err := initIBCNFTTransferGenesis(ctx, box); err != nil {
			return vm, err
		}
		initICAModule(ctx, m, vm)
		if err := injectEVMRuntimeContract(ctx, box); err != nil {
			return vm, fmt.Errorf("inject EVM runtime contract: %w", err)
		}
		if err := enableEVMShanghaiUpgrade(ctx, box); err != nil {
			return vm, fmt.Errorf("enable EVM Shanghai upgrade: %w", err)
		}

		return box.ModuleManager.RunMigrations(ctx, c, vm)
	}
}

func migrateConsensusParamsToXConsensus(ctx sdk.Context, box upgrades.Toolbox) {
	baseAppLegacySS := box.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())
	// Migrate Tendermint consensus parameters from x/params module to a dedicated x/consensus module.
	baseapp.MigrateParams(ctx, baseAppLegacySS, &box.ConsensusParamsKeeper.ParamsStore)
}

func buildFeemarketParams(enableHeight int64) feemarkettypes.Params {
	p := defaultFeemarketParams
	p.EnableHeight = enableHeight
	return p
}

func setFeemarketParams(ctx sdk.Context, box upgrades.Toolbox) error {
	if err := box.FeeMarketKeeper.SetParams(ctx, buildFeemarketParams(ctx.BlockHeight())); err != nil {
		return fmt.Errorf("set feemarket params: %w", err)
	}
	return nil
}

func setWasmParams(ctx sdk.Context, box upgrades.Toolbox) error {
	wasmParams := box.WasmKeeper.GetParams(ctx)
	wasmParams.CodeUploadAccess.Permission = wasmtypes.AccessTypeEverybody
	wasmParams.InstantiateDefaultPermission = wasmtypes.AccessTypeEverybody
	if err := box.WasmKeeper.SetParams(ctx, wasmParams); err != nil {
		return fmt.Errorf("set wasm params: %w", err)
	}
	return nil
}

func initIBCNFTTransferGenesis(ctx sdk.Context, box upgrades.Toolbox) error {
	gs := ibcnfttransfertypes.DefaultGenesisState()
	bz, err := ibcnfttransfertypes.ModuleCdc.MarshalJSON(gs)
	if err != nil {
		return fmt.Errorf("marshal %s genesis: %w", ibcnfttransfertypes.ModuleName, err)
	}
	if mod, ok := box.ModuleManager.Modules[ibcnfttransfertypes.ModuleName].(module.HasGenesis); ok {
		mod.InitGenesis(ctx, ibcnfttransfertypes.ModuleCdc, bz)
	}
	return nil
}

func ensureConsensusBlockGasForEVM(ctx sdk.Context, k consensusparamkeeper.Keeper) error {
	cp, err := k.ParamsStore.Get(ctx)
	if err != nil {
		return fmt.Errorf("get consensus params: %w", err)
	}
	changed := false
	if cp.Block == nil {
		def := cmttypes.DefaultConsensusParams().ToProto()
		cp.Block = def.Block
		changed = true
	} else if cp.Block.MaxGas == 0 {
		cp.Block.MaxGas = -1
		changed = true
	}
	if !changed {
		return nil
	}
	return k.ParamsStore.Set(ctx, cp)
}

func initICAModule(ctx sdk.Context, m *module.Manager, fromVM module.VersionMap) {
	icaModule := m.Modules[icatypes.ModuleName].(ica.AppModule)
	fromVM[icatypes.ModuleName] = icaModule.ConsensusVersion()
	controllerParams := icacontrollertypes.Params{}
	hostParams := icahosttypes.Params{
		HostEnabled:   true,
		AllowMessages: []string{"*"},
	}

	ctx.Logger().Info("initializing ica module (ics27)")
	icaModule.InitModule(ctx, controllerParams, hostParams)
}

func injectEVMRuntimeContract(ctx sdk.Context, box upgrades.Toolbox) error {
	contractAddr := common.HexToAddress(create2FactoryAddress)
	runtimeCodeHex := strings.TrimPrefix(create2FactoryRuntime, "0x")
	runtimeCode, err := hex.DecodeString(runtimeCodeHex)
	if err != nil {
		return fmt.Errorf("decode runtime bytecode: %w", err)
	}

	codeHash := crypto.Keccak256(runtimeCode)
	account := box.EvmKeeper.GetAccountWithoutBalance(ctx, contractAddr)
	if account != nil && bytes.Equal(account.CodeHash, codeHash) {
		ctx.Logger().Info("evm contract runtime already injected", "address", contractAddr.Hex())
		return nil
	}

	if account != nil && len(account.CodeHash) > 0 &&
		!bytes.Equal(account.CodeHash, evmtypes.EmptyCodeHash) &&
		!bytes.Equal(account.CodeHash, codeHash) {
		return fmt.Errorf("contract address %s already has different code", contractAddr.Hex())
	}

	if account == nil {
		account = &statedb.Account{
			Nonce:   1,
			Balance: big.NewInt(0),
		}
	}
	account.CodeHash = codeHash

	if err := box.EvmKeeper.SetAccount(ctx, contractAddr, *account); err != nil {
		return fmt.Errorf("set contract account: %w", err)
	}
	box.EvmKeeper.SetCode(ctx, codeHash, runtimeCode)

	ctx.Logger().Info("evm contract runtime injected", "address", contractAddr.Hex())
	return nil
}

// enableEVMShanghaiUpgrade sets ShanghaiBlock and CancunBlock in the EVM ChainConfig and
// appends EIP-3855 to ExtraEIPs so the interpreter enables PUSH0 (Solidity 0.8.20+ bytecode).
func enableEVMShanghaiUpgrade(ctx sdk.Context, box upgrades.Toolbox) error {
	evmParams := box.EvmKeeper.GetParams(ctx)
	zero := math.ZeroInt()
	evmParams.ChainConfig.ShanghaiBlock = &zero
	evmParams.ChainConfig.CancunBlock = &zero
	if !slices.Contains(evmParams.ExtraEIPs, eip3855Extra) {
		evmParams.ExtraEIPs = append(evmParams.ExtraEIPs, eip3855Extra)
	}
	if err := box.EvmKeeper.SetParams(ctx, evmParams); err != nil {
		return fmt.Errorf("set evm params: %w", err)
	}
	ctx.Logger().Info("EVM Shanghai fork heights set; EIP-3855 (PUSH0) enabled via ExtraEIPs")
	return nil
}
