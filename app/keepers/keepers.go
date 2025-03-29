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
	erc721keeper "github.com/UptickNetwork/evm-nft-convert/keeper"
	erc721types "github.com/UptickNetwork/evm-nft-convert/types"
	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	nfttypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/erc20"
	erc20keeper "github.com/UptickNetwork/uptick/x/erc20/keeper"
	erc20types "github.com/UptickNetwork/uptick/x/erc20/types"
	"github.com/UptickNetwork/uptick/x/evmIBC"
	evmIBCKeepr "github.com/UptickNetwork/uptick/x/evmIBC/keeper"
	"github.com/UptickNetwork/uptick/x/internft"
	cw721keeper "github.com/UptickNetwork/wasm-nft-convert/keeper"
	cw721types "github.com/UptickNetwork/wasm-nft-convert/types"
	nfttransfer "github.com/bianjieai/nft-transfer"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
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
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclient "github.com/cosmos/ibc-go/v8/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	srvflags "github.com/evmos/ethermint/server/flags"
	ethermint "github.com/evmos/ethermint/types"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/evmos/ethermint/x/evm/vm/geth"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
	"github.com/spf13/cast"
	"path/filepath"
)

var wasmCapabilities = []string{
	"stargaze",
	"token_factory",
}

// AppKeepers defines a structure used to consolidate all
// the keepers needed to run an iris appKeepers.
type AppKeepers struct {

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    *stakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	MintKeeper       mintkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper
	GovKeeper        *govkeeper.Keeper
	CrisisKeeper     *crisiskeeper.Keeper
	UpgradeKeeper    *upgradekeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper
	IBCKeeper        *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly

	EvidenceKeeper    *evidencekeeper.Keeper
	IBCTransferKeeper ibctransferkeeper.Keeper
	FeeGrantKeeper    feegrantkeeper.Keeper

	//ICAControllerKeeper icacontrollerkeeper.Keeper
	ICAHostKeeper icahostkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper
	//scopedIBCMockKeeper       capabilitykeeper.ScopedKeeper
	ScopedNFTTransferKeeper   capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper

	IBCNFTTransferKeeper ibcnfttransferkeeper.Keeper

	ConsensusParamsKeeper consensusparamkeeper.Keeper

	AuthzKeeper authzkeeper.Keeper
	// Ethermint keepers
	EvmKeeper       *evmkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper
	// Uptick keepers
	Erc20Keeper  *erc20keeper.Keeper
	Erc721Keeper erc721keeper.Keeper
	Cw721Keeper  cw721keeper.Keeper
	EVMIBCKeeper evmIBCKeepr.Keeper

	NFTKeeper nftkeeper.Keeper

	//Add ICS721 for nft ibc transfer
	// ICS721Keeper ibcnfttransferkeeper.Keeper

	// this line is used by starport scaffolding # stargate/app/keeperDeclaration
	WasmKeeper       wasmkeeper.Keeper
	WasmConfig       wasmtypes.WasmConfig
	ContractKeeper   *wasmkeeper.PermissionedKeeper
	ScopedWasmKeeper capabilitykeeper.ScopedKeeper
	//IBCWasmKeeper       ibcwasmkeeper.Keeper
	//IBCHooksKeeper      ibchookskeeper.Keeper
	//Ics20WasmHooks      *ibchooks.WasmHooks
	//HooksICS4Wrapper    ibchooks.ICS4Middleware
	//PacketForwardKeeper *packetforwardkeeper.Keeper

	TransferModule       transfer.AppModule
	ICAModule            ica.AppModule
	IBCNftTransferModule nfttransfer.AppModule
}

// NewUptick returns a reference to a new initialized Ethermint application.
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

	appKeepers.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[consensusparamtypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.EventService{},
	)
	// set the BaseApp's parameter store
	bApp.SetParamStore(&appKeepers.ConsensusParamsKeeper.ParamsStore)

	// add capability keeper and ScopeToModule for ibc module
	appKeepers.CapabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec,
		appKeepers.keys[capabilitytypes.StoreKey],
		appKeepers.memKeys[capabilitytypes.MemStoreKey],
	)
	appKeepers.ScopedIBCKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	appKeepers.ScopedTransferKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	appKeepers.ScopedNFTTransferKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibcnfttransfertypes.ModuleName)
	appKeepers.ScopedICAHostKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)

	appKeepers.ScopedWasmKeeper = appKeepers.CapabilityKeeper.ScopeToModule(wasmtypes.ModuleName)
	appKeepers.ScopedICAControllerKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)

	appKeepers.CapabilityKeeper.Seal()

	// use custom Ethermint account for contracts
	appKeepers.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[authtypes.StoreKey]),
		ethermint.ProtoAccount,
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
		appKeepers.keys[ibcexported.StoreKey],
		appKeepers.GetSubspace(ibcexported.ModuleName),
		appKeepers.StakingKeeper,
		appKeepers.UpgradeKeeper,
		appKeepers.ScopedIBCKeeper,
		authtypes.NewModuleAddress(ibcexported.ModuleName).String(),
	)

	// Initialize ICA Host keeper
	appKeepers.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[icahosttypes.StoreKey],
		appKeepers.GetSubspace(icahosttypes.SubModuleName),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.ScopedICAHostKeeper,
		bApp.MsgServiceRouter(),
		authtypes.NewModuleAddress(icahosttypes.SubModuleName).String(),
	)

	appKeepers.ICAHostKeeper.WithQueryRouter(bApp.GRPCQueryRouter())

	appKeepers.ICAModule = ica.NewAppModule(nil, &appKeepers.ICAHostKeeper)
	icaHostIBCModule := icahost.NewIBCModule(appKeepers.ICAHostKeeper)

	appKeepers.NFTKeeper = nftkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[nfttypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
	)

	// Uptick Keeper
	appKeepers.Erc20Keeper = erc20keeper.NewKeeper(
		appCodec,
		appKeepers.keys[erc20types.StoreKey],
		appKeepers.GetSubspace(erc20types.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.EvmKeeper,
	)

	// Create Transfer Keepers
	appKeepers.IBCTransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibctransfertypes.StoreKey],
		appKeepers.GetSubspace(ibctransfertypes.ModuleName),
		appKeepers.Erc20Keeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.ScopedTransferKeeper,
		authtypes.NewModuleAddress(ibctransfertypes.ModuleName).String(),
	)

	appKeepers.Erc20Keeper.SetICS4Wrapper(appKeepers.IBCKeeper.ChannelKeeper)
	appKeepers.Erc20Keeper.SetIBCKeeper(appKeepers.IBCTransferKeeper)

	appKeepers.TransferModule = transfer.NewAppModule(appKeepers.IBCTransferKeeper)
	transferIBCModule := transfer.NewIBCModule(appKeepers.IBCTransferKeeper)
	transferStack := erc20.NewIBCMiddleware(*appKeepers.Erc20Keeper, transferIBCModule)

	appKeepers.IBCNFTTransferKeeper = ibcnfttransferkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibcnfttransfertypes.StoreKey],
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		internft.NewInterNftKeeper(appCodec, appKeepers.NFTKeeper, appKeepers.AccountKeeper),
		appKeepers.ScopedNFTTransferKeeper,
	)
	appKeepers.IBCNftTransferModule = nfttransfer.NewAppModule(appKeepers.IBCNFTTransferKeeper)
	nftTransferIBCModule := nfttransfer.NewIBCModule(appKeepers.IBCNFTTransferKeeper)
	ercTransferStack := evmIBC.NewIBCMiddleware(appKeepers.EVMIBCKeeper, nftTransferIBCModule)

	// routerModule := router.NewAppModule(app.RouterKeeper, transferIBCModule)
	// create static IBC router, add transfer route, then set and seal it
	//var icaControllerStack porttypes.IBCModule
	//icaControllerStack = icacontroller.NewIBCMiddleware(icaControllerStack, appKeepers.ICAControllerKeeper)

	ibcRouter := porttypes.NewRouter().
		AddRoute(icahosttypes.SubModuleName, icaHostIBCModule).
		//AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(ibcnfttransfertypes.ModuleName, ercTransferStack).
		AddRoute(wasmtypes.ModuleName, wasm.NewIBCHandler(appKeepers.WasmKeeper, appKeepers.IBCKeeper.ChannelKeeper, appKeepers.IBCKeeper.ChannelKeeper))

	// Set IBC Router
	appKeepers.IBCKeeper.SetRouter(ibcRouter)

	// Create Ethermint keepers
	appKeepers.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		appKeepers.keys[feemarkettypes.StoreKey],
		appKeepers.tkeys[feemarkettypes.TransientKey],
		appKeepers.GetSubspace(feemarkettypes.ModuleName),
	)

	appKeepers.EvmKeeper = evmkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[evmtypes.StoreKey],
		appKeepers.tkeys[evmtypes.TransientKey],
		authtypes.NewModuleAddress(govtypes.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		appKeepers.FeeMarketKeeper,
		nil,
		geth.NewEVM,
		cast.ToString(appOpts.Get(srvflags.EVMTracer)),
		appKeepers.GetSubspace(evmtypes.ModuleName),
	)

	appKeepers.EvmKeeper = appKeepers.EvmKeeper.SetHooks(
		evmkeeper.NewMultiEvmHooks(
			appKeepers.Erc20Keeper.Hooks(),
		),
	)

	// create evidence keeper with router
	appKeepers.EvidenceKeeper = evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[evidencetypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.SlashingKeeper,
		appKeepers.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)

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
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(appKeepers.ParamsKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(appKeepers.IBCKeeper.ClientKeeper)).
		AddRoute(erc20types.RouterKey, erc20.NewErc20ProposalHandler(appKeepers.Erc20Keeper))

	appKeepers.GovKeeper = govKeeper.SetHooks(govtypes.NewMultiGovHooks(
		govtypes.NewMultiGovHooks(),
	))

	// Set legacy router for backwards compatibility with gov v1beta1
	govKeeper.SetLegacyRouter(govRouter)

	// Initialize ICA Controller keeper
	//appKeepers.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
	//	appCodec,
	//	appKeepers.keys[icacontrollertypes.StoreKey],
	//	appKeepers.GetSubspace(icacontrollertypes.SubModuleName),
	//	appKeepers.IBCKeeper.ChannelKeeper,
	//	appKeepers.IBCKeeper.ChannelKeeper,
	//	appKeepers.IBCKeeper.PortKeeper,
	//	appKeepers.ScopedICAControllerKeeper,
	//	bApp.MsgServiceRouter(),
	//	authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	//)

	// Configure the hooks keeper
	//appKeepers.IBCHooksKeeper = ibchookskeeper.NewKeeper(
	//	appKeepers.keys[ibchookstypes.StoreKey],
	//)

	//stargazePrefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
	//wasmHooks := ibchooks.NewWasmHooks(&appKeepers.IBCHooksKeeper, nil, stargazePrefix) // The contract keeper needs to be set later
	//appKeepers.Ics20WasmHooks = &wasmHooks
	//appKeepers.HooksICS4Wrapper = ibchooks.NewICS4Middleware(
	//	appKeepers.IBCKeeper.ChannelKeeper,
	//	appKeepers.Ics20WasmHooks,
	//)

	// Initialize the packet forward middleware Keeper
	//appKeepers.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
	//	appCodec,
	//	appKeepers.keys[packetforwardtypes.StoreKey],
	//	appKeepers.IBCTransferKeeper,
	//	appKeepers.IBCKeeper.ChannelKeeper,
	//	appKeepers.BankKeeper,
	//	// The ICS4Wrapper is replaced by the HooksICS4Wrapper instead of the channel so that sending can be overridden by the middleware
	//	appKeepers.HooksICS4Wrapper,
	//	authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	//)

	appKeepers.Erc721Keeper = erc721keeper.NewKeeper(
		appKeepers.keys[erc721types.StoreKey],
		appCodec,
		appKeepers.GetSubspace(erc721types.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.NFTKeeper,
		appKeepers.EvmKeeper,
		appKeepers.IBCNFTTransferKeeper,
	)

	// this line is used by starport scaffolding # stargate/app/keeperDefinition
	wasmDir := filepath.Join(homePath, "data")
	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}
	appKeepers.WasmConfig = wasmConfig
	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	appKeepers.WasmKeeper = wasmkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[wasmtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		distrkeeper.NewQuerier(appKeepers.DistrKeeper),
		nil,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.PortKeeper,
		appKeepers.ScopedWasmKeeper,
		appKeepers.IBCTransferKeeper,
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		GetWasmCapabilities(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		wasmOpts...,
	)

	appKeepers.Cw721Keeper = cw721keeper.NewKeeper(
		appKeepers.keys[cw721types.StoreKey],
		appCodec,
		appKeepers.GetSubspace(cw721types.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.NFTKeeper,
		appKeepers.WasmKeeper,
		appKeepers.ContractKeeper,
		appKeepers.IBCNFTTransferKeeper,
	)

	appKeepers.EVMIBCKeeper.SetCw721Keeper(appKeepers.Cw721Keeper)
	appKeepers.EVMIBCKeeper.SetErc721Keeper(appKeepers.Erc721Keeper)

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
	//paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName).WithKeyTable(banktypes.ParamKeyTable())
	// paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName).WithKeyTable(stakingtypes.ParamKeyTable())
	// paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName).WithKeyTable(minttypes.ParamKeyTable())
	// paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName).WithKeyTable(distrtypes.ParamKeyTable())
	// paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName).WithKeyTable(slashingtypes.ParamKeyTable())
	//paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable())
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	// paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName).WithKeyTable(crisistypes.ParamKeyTable())
	paramsKeeper.Subspace(ibctransfertypes.ModuleName).WithKeyTable(ibctransfertypes.ParamKeyTable())
	paramsKeeper.Subspace(ibcexported.ModuleName).WithKeyTable(keyTable)

	// ethermint subspaces
	paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable())
	paramsKeeper.Subspace(feemarkettypes.ModuleName).WithKeyTable(feemarkettypes.ParamKeyTable())

	// uptick subspaces
	paramsKeeper.Subspace(erc20types.ModuleName).WithKeyTable(erc20types.ParamKeyTable())
	paramsKeeper.Subspace(erc721types.ModuleName).WithKeyTable(erc721types.ParamKeyTable())
	paramsKeeper.Subspace(icahosttypes.SubModuleName).WithKeyTable(icahosttypes.ParamKeyTable())

	paramsKeeper.Subspace(wasmtypes.ModuleName)
	paramsKeeper.Subspace(ibcnfttransfertypes.ModuleName)
	paramsKeeper.Subspace(cw721types.ModuleName).WithKeyTable(cw721types.ParamKeyTable())

	//paramsKeeper.Subspace(icacontrollertypes.SubModuleName).WithKeyTable(icacontrollertypes.ParamKeyTable())

	return paramsKeeper
}

func GetWasmCapabilities() []string {
	return append(wasmkeeper.BuiltInCapabilities(), wasmCapabilities...)
}
