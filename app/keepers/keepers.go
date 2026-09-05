package keepers

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cmdcfg "github.com/UptickNetwork/uptick/cmd/config"
	upticktypes "github.com/UptickNetwork/uptick/types"
	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	nfttypes "github.com/UptickNetwork/uptick/x/collection/types"
	cw721keeper "github.com/UptickNetwork/uptick/x/cw721/keeper"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	"github.com/UptickNetwork/uptick/x/evmibc"
	evmIBCKeepr "github.com/UptickNetwork/uptick/x/evmibc/keeper"
	"github.com/UptickNetwork/uptick/x/internft"
	nfttransfer "github.com/bianjieai/nft-transfer"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	cosmoserc20 "github.com/cosmos/evm/x/erc20"
	cosmoserc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	cosmoserc20types "github.com/cosmos/evm/x/erc20/types"
	ica "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	icacontroller "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"

	"path/filepath"

	precompilestypes "github.com/cosmos/evm/precompiles/types"
	srvflags "github.com/cosmos/evm/server/flags"
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	icahost "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	transferv2 "github.com/cosmos/ibc-go/v10/modules/apps/transfer/v2"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	"github.com/spf13/cast"
)

// AppKeepers defines a structure used to consolidate all
// the keepers needed to run an iris appKeepers.
type AppKeepers struct {

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper       authkeeper.AccountKeeper
	BankKeeper          bankkeeper.Keeper
	StakingKeeper       *stakingkeeper.Keeper
	SlashingKeeper      slashingkeeper.Keeper
	MintKeeper          mintkeeper.Keeper
	DistrKeeper         distrkeeper.Keeper
	GovKeeper           *govkeeper.Keeper
	CrisisKeeper        *crisiskeeper.Keeper
	UpgradeKeeper       *upgradekeeper.Keeper
	ParamsKeeper        paramskeeper.Keeper
	IBCKeeper           *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	ICAControllerKeeper icacontrollerkeeper.Keeper
	EvidenceKeeper      *evidencekeeper.Keeper
	IBCTransferKeeper   ibctransferkeeper.Keeper
	FeeGrantKeeper      feegrantkeeper.Keeper

	ICAHostKeeper icahostkeeper.Keeper

	IBCNFTTransferKeeper  ibcnfttransferkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	// cosmos/evm keepers
	EvmKeeper       *evmkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper
	// ERC20 keeper (cosmos/evm v0.6.1)
	Erc20Keeper cosmoserc20keeper.Keeper
	// Uptick keepers

	Erc721Keeper erc721keeper.Keeper
	Cw721Keeper  cw721keeper.Keeper
	EVMIBCKeeper evmIBCKeepr.Keeper
	NFTKeeper    nftkeeper.Keeper
	// wasm keepers
	WasmKeeper           wasmkeeper.Keeper
	WasmConfig           wasmtypes.NodeConfig
	ContractKeeper       *wasmkeeper.PermissionedKeeper
	TransferModule       transfer.AppModule
	ICAModule            ica.AppModule
	IBCNftTransferModule nfttransfer.AppModule
}

// GetKVStoreKey returns the KVStoreKey registered for the given module name.
// It is used by upgrade handlers to access module stores directly for raw
// state migrations (e.g. deleting a deprecated store prefix).
func (ak *AppKeepers) GetKVStoreKey(moduleName string) *storetypes.KVStoreKey {
	return ak.keys[moduleName]
}

// NewUptick returns a reference to a new initialized Uptick application.
func New(
	appCodec codec.Codec,
	bApp *baseapp.BaseApp,
	legacyAmino *codec.LegacyAmino,
	maccPerms map[string][]string,
	modAccAddrs map[string]bool,
	blockedAddress map[string]bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	logger log.Logger,
	appOpts servertypes.AppOptions,
	wasmOpts []wasmkeeper.Option,

) AppKeepers {

	appKeepers := AppKeepers{}
	// Set keys KVStoreKey, TransientStoreKey, MemoryStoreKey
	appKeepers.genStoreKeys()

	// configure state listening capabilities using AppOptions
	// we are doing nothing with the returned streamingServices and waitGroup in this case
	if err := bApp.RegisterStreamingServices(appOpts, appKeepers.keys); err != nil {
		panic(err)
	}

	// init params keeper and subspaces
	appKeepers.ParamsKeeper = initParamsKeeper(
		appCodec,
		legacyAmino,
		appKeepers.keys[paramstypes.StoreKey],
		appKeepers.tkeys[paramstypes.TStoreKey],
	)

	// consensus params keeper
	appKeepers.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[consensusparamtypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.EventService{},
	)
	// set the BaseApp's parameter store
	bApp.SetParamStore(&appKeepers.ConsensusParamsKeeper.ParamsStore)

	// use BaseAccount for contracts (cosmos/evm replaces Ethermint EthAccount)
	appKeepers.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[authtypes.StoreKey]),
		upticktypes.ProtoAccount,
		maccPerms,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[feegrant.StoreKey]),
		appKeepers.AccountKeeper,
	)

	appKeepers.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[banktypes.StoreKey]),
		appKeepers.AccountKeeper,
		blockedAddress,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		logger,
	)

	appKeepers.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[stakingtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	appKeepers.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[minttypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[distrtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(appKeepers.keys[slashingtypes.StoreKey]),
		appKeepers.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[crisistypes.StoreKey]),
		invCheckPeriod,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		appKeepers.AccountKeeper.AddressCodec(),
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	appKeepers.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			appKeepers.DistrKeeper.Hooks(),
			appKeepers.SlashingKeeper.Hooks(),
		),
	)

	appKeepers.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(appKeepers.keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		bApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(appKeepers.keys[authzkeeper.StoreKey]),
		appCodec,
		bApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
	)

	// Create IBC Keeper
	appKeepers.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibcexported.StoreKey]),
		appKeepers.GetSubspace(ibcexported.ModuleName),
		appKeepers.UpgradeKeeper,
		authtypes.NewModuleAddress(ibcexported.ModuleName).String(),
	)

	// Initialize ICA Host keeper
	appKeepers.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[icahosttypes.StoreKey]),
		appKeepers.GetSubspace(icahosttypes.SubModuleName),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.AccountKeeper,
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		authtypes.NewModuleAddress(icahosttypes.SubModuleName).String(),
	)

	// ICA Controller keeper must be constructed before NewIBCMiddleware copies
	// it by value into the IBC router. Constructing it after SetRouter left
	// the middleware holding a zero-value keeper.
	appKeepers.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[icacontrollertypes.StoreKey]),
		appKeepers.GetSubspace(icacontrollertypes.SubModuleName),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		bApp.MsgServiceRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.ICAModule = ica.NewAppModule(&appKeepers.ICAControllerKeeper, &appKeepers.ICAHostKeeper)
	icaHostIBCModule := icahost.NewIBCModule(appKeepers.ICAHostKeeper)

	appKeepers.NFTKeeper = nftkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[nfttypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
	)

	// Create cosmos/evm keepers
	appKeepers.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		appKeepers.keys[feemarkettypes.StoreKey],
		appKeepers.tkeys[feemarkettypes.TransientKey],
	)

	// Gov Keeper must be initialized before EVMKeeper because the EVM precompiles
	// (DefaultStaticPrecompiles) reference it.
	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[govtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		appKeepers.DistrKeeper,
		bApp.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// register the proposal types
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(appKeepers.ParamsKeeper))

	appKeepers.GovKeeper = govKeeper.SetHooks(govtypes.NewMultiGovHooks(
		govtypes.NewMultiGovHooks(),
	))

	// Set legacy router for backwards compatibility with gov v1beta1
	govKeeper.SetLegacyRouter(govRouter)

	// cosmos/evm v0.6.1: Create EVM Keeper first with an ERC20 proxy that is
	// filled in after the ERC20 keeper exists (there is no SetErc20Keeper).
	erc20Proxy := &erc20KeeperProxy{}
	// Genesis is the source of truth for a started node. appOpts chain-id can
	// still be leftover client.toml (e.g. ~/.uptickd chain-id=testnet) and must
	// not override {name}_{eip155}-{revision} from this home's genesis.
	cosmosChainID := upticktypes.ChainIDFromGenesisFile(homePath)
	if cosmosChainID == "" {
		cosmosChainID = cast.ToString(appOpts.Get(flags.FlagChainID))
	}
	evmChainID := upticktypes.ResolveEVMChainID(appOpts, cosmosChainID)
	logger.Info("evm chain id", "cosmos_chain_id", cosmosChainID, "evm_chain_id", evmChainID, "home", homePath)

	appKeepers.EvmKeeper = evmkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[evmtypes.StoreKey],
		appKeepers.tkeys[evmtypes.TransientKey],
		appKeepers.keys, // map of all KVStoreKeys for cross-module access
		authtypes.NewModuleAddress(govtypes.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		appKeepers.FeeMarketKeeper,
		&appKeepers.ConsensusParamsKeeper,
		erc20Proxy,
		evmChainID,
		cast.ToString(appOpts.Get(srvflags.EVMTracer)),
	).WithDefaultEvmCoinInfo(evmtypes.EvmCoinInfo{
		Denom:         cmdcfg.BaseDenom,
		ExtendedDenom: cmdcfg.BaseDenom,
		DisplayDenom:  cmdcfg.DisplayDenom,
		Decimals:      upticktypes.BaseDenomUnit,
	}) // NOTE: WithStaticPrecompiles is called AFTER ERC20 keeper is created

	// Create Transfer Keeper (no longer takes ERC20 keeper as ICS4Wrapper in v0.6.1)
	appKeepers.IBCTransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibctransfertypes.StoreKey]),
		appKeepers.GetSubspace(ibctransfertypes.ModuleName),
		appKeepers.IBCKeeper.ChannelKeeper, // ICS4Wrapper
		appKeepers.IBCKeeper.ChannelKeeper,
		bApp.MsgServiceRouter(), // MessageRouter
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.NewModuleAddress(ibctransfertypes.ModuleName).String(),
	)

	appKeepers.TransferModule = transfer.NewAppModule(appKeepers.IBCTransferKeeper)

	// cosmos/evm v0.6.1 ERC20 Keeper — created with the now-available EVMKeeper
	appKeepers.Erc20Keeper = cosmoserc20keeper.NewKeeper(
		appKeepers.keys[cosmoserc20types.StoreKey],
		appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName), // authority
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.EvmKeeper, // EVMKeeper
		appKeepers.StakingKeeper,
		&appKeepers.IBCTransferKeeper, // transfer keeper for IBC callbacks
	)
	erc20Proxy.keeper = &appKeepers.Erc20Keeper

	// Wire static precompiles (bank, staking, distribution, ics20, etc.)
	// Uses cosmos/evm's DefaultStaticPrecompiles with the concrete ERC20 keeper.
	appKeepers.EvmKeeper.WithStaticPrecompiles(
		precompilestypes.DefaultStaticPrecompiles(
			*appKeepers.StakingKeeper,
			appKeepers.DistrKeeper,
			appKeepers.BankKeeper,
			&appKeepers.Erc20Keeper,
			&appKeepers.IBCTransferKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			*appKeepers.GovKeeper,
			appKeepers.SlashingKeeper,
			appCodec,
		),
	)

	// IBC transfer stack wrapped with cosmos/evm's ERC20 middleware, which
	// auto-converts incoming IBC coins to their ERC20 representation on recv
	// (and refunds to ERC20 on error ack / timeout). The ERC20 keeper's
	// ibc_callbacks.go implements the conversion; the middleware invokes them.
	transferIBCModule := transfer.NewIBCModule(appKeepers.IBCTransferKeeper)
	transferStack := cosmoserc20.NewIBCMiddleware(appKeepers.Erc20Keeper, transferIBCModule)

	appKeepers.IBCNFTTransferKeeper = ibcnfttransferkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibcnfttransfertypes.StoreKey],
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.AccountKeeper,
		internft.NewInterNftKeeper(appCodec, appKeepers.NFTKeeper, appKeepers.AccountKeeper),
	)

	wasmDir := filepath.Join(homePath, "data")
	nodeConfig, err := wasm.ReadNodeConfig(appOpts)
	if err != nil {
		panic("error while reading wasm node config: " + err.Error())
	}
	appKeepers.WasmConfig = nodeConfig
	// wasmd v0.61 NewKeeper signature uses NodeConfig + VMConfig instead of WasmConfig
	vmConfig := wasmtypes.VMConfig{}

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	appKeepers.WasmKeeper = wasmkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[wasmtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		distrkeeper.NewQuerier(appKeepers.DistrKeeper),
		appKeepers.IBCKeeper.ChannelKeeper, // ICS4Wrapper
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeperV2,
		appKeepers.IBCTransferKeeper, // ICS20TransferPortSource
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		wasmDir,
		nodeConfig,
		vmConfig,
		GetWasmCapabilities(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		wasmOpts...,
	)
	appKeepers.ContractKeeper = wasmkeeper.NewDefaultPermissionKeeper(appKeepers.WasmKeeper)

	appKeepers.Cw721Keeper = cw721keeper.NewKeeper(
		appKeepers.keys[cw721types.StoreKey],
		appCodec,
		appKeepers.AccountKeeper,
		appKeepers.NFTKeeper,
		cw721keeper.WrapWasmKeeper(&appKeepers.WasmKeeper),
		appKeepers.IBCNFTTransferKeeper,
	)

	appKeepers.Erc721Keeper = erc721keeper.NewKeeper(
		appKeepers.keys[erc721types.StoreKey],
		appCodec,
		appKeepers.AccountKeeper,
		appKeepers.NFTKeeper,
		appKeepers.EvmKeeper,
		appKeepers.IBCNFTTransferKeeper,
	)

	appKeepers.EVMIBCKeeper = evmIBCKeepr.NewKeeper(appKeepers.IBCNFTTransferKeeper)

	appKeepers.EVMIBCKeeper.SetCw721Keeper(appKeepers.Cw721Keeper)
	appKeepers.EVMIBCKeeper.SetErc721Keeper(appKeepers.Erc721Keeper)

	appKeepers.IBCNftTransferModule = nfttransfer.NewAppModule(appKeepers.IBCNFTTransferKeeper)
	nftTransferIBCModule := nfttransfer.NewIBCModule(appKeepers.IBCNFTTransferKeeper)
	ercTransferStack := evmibc.NewIBCMiddleware(appKeepers.EVMIBCKeeper, nftTransferIBCModule, appKeepers.IBCKeeper.ChannelKeeper)
	if err := appKeepers.Erc721Keeper.SetICS4Wrapper(appKeepers.IBCKeeper.ChannelKeeper); err != nil {
		panic(err)
	}

	// create static IBC router, add transfer route, then set and seal it
	icaControllerStack := icacontroller.NewIBCMiddleware(appKeepers.ICAControllerKeeper)

	ibcRouter := porttypes.NewRouter().
		AddRoute(icahosttypes.SubModuleName, icaHostIBCModule).
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(ibcnfttransfertypes.PortID, ercTransferStack).
		AddRoute(wasmtypes.ModuleName, wasm.NewIBCHandler(appKeepers.WasmKeeper, appKeepers.IBCKeeper.ChannelKeeper, appKeepers.IBCTransferKeeper, appVersionGetterWrapper{appKeepers.IBCKeeper}))

	// Set IBC Router
	appKeepers.IBCKeeper.SetRouter(ibcRouter)

	ibcRouterV2 := ibcapi.NewRouter().
		AddRoute(ibctransfertypes.PortID, transferv2.NewIBCModule(appKeepers.IBCTransferKeeper)).
		AddPrefixRoute(wasmkeeper.PortIDPrefixV2, wasmkeeper.NewIBC2Handler(appKeepers.WasmKeeper))
	appKeepers.IBCKeeper.SetRouterV2(ibcRouterV2)

	// cosmos/evm v0.6.1 ERC20 module does not use EVM hooks — IBC callbacks
	// are handled internally via ibc_callbacks.go in the ERC20 keeper.

	// create evidence keeper with router
	appKeepers.EvidenceKeeper = evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[evidencetypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.SlashingKeeper,
		appKeepers.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)

	// this line is used by starport scaffolding # stargate/app/keeperDefinition

	return appKeepers
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec,
	legacyAmino *codec.LegacyAmino,
	key,
	tkey storetypes.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// register the key tables for legacy param subspaces
	keyTable := ibcclienttypes.ParamKeyTable()
	keyTable.RegisterParamSet(&ibcconnectiontypes.Params{})

	// SDK subspaces
	// paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(authtypes.ModuleName).WithKeyTable(authtypes.ParamKeyTable())
	// paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName).WithKeyTable(banktypes.ParamKeyTable())
	// paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName).WithKeyTable(stakingtypes.ParamKeyTable())
	// paramsKeeper.Subspace(minttypes.ModuleName)
	// TODO: minttypes.ParamKeyTable() removed in SDK 0.53 - params managed via authority
	// paramsKeeper.Subspace(minttypes.ModuleName).WithKeyTable(minttypes.ParamKeyTable())
	// paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName).WithKeyTable(distrtypes.ParamKeyTable())
	// paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName).WithKeyTable(slashingtypes.ParamKeyTable())
	// paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable())
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	// paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName).WithKeyTable(crisistypes.ParamKeyTable())
	paramsKeeper.Subspace(ibctransfertypes.ModuleName).WithKeyTable(ibctransfertypes.ParamKeyTable())
	paramsKeeper.Subspace(ibcexported.ModuleName).WithKeyTable(keyTable)

	// cosmos/evm subspaces
	// TODO: evmtypes.ParamKeyTable() and feemarkettypes.ParamKeyTable() removed in cosmos/evm v0.6.1
	// paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable())
	// paramsKeeper.Subspace(feemarkettypes.ModuleName).WithKeyTable(feemarkettypes.ParamKeyTable())

	// uptick subspaces
	paramsKeeper.Subspace(icahosttypes.SubModuleName).WithKeyTable(icahosttypes.ParamKeyTable())

	// Audit P3-6: subspaces must not accept unvalidated param writes.
	//
	// wasmd: upstream removed ParamKeyTable() in wasmd v0.61 (params moved to
	// NodeConfig), so there is no KeyTable to attach; the subspace is kept only
	// because wasm.NewAppModule still takes one. It is never written through
	// governance — do NOT add param-change proposals against it.
	paramsKeeper.Subspace(wasmtypes.ModuleName)
	// ibcnft-transfer: the Uptick fork (v1.3.0-ibc-v10) removed params entirely
	// (no types.ParamKeyTable, keeper takes no paramSpace) — dead registration.
	// x/cw721: never had params — dead registration. Both removed (audit P3-6).

	paramsKeeper.Subspace(icacontrollertypes.SubModuleName).WithKeyTable(icacontrollertypes.ParamKeyTable())

	return paramsKeeper
}

func GetWasmCapabilities() []string {
	return wasmkeeper.BuiltInCapabilities()
}

// appVersionGetterWrapper wraps ibc-go v10's IBCKeeper to satisfy wasmd's
// appVersionGetter interface (GetAppVersion).
type appVersionGetterWrapper struct {
	ik *ibckeeper.Keeper
}

func (w appVersionGetterWrapper) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return w.ik.ChannelKeeper.GetAppVersion(ctx, portID, channelID)
}
