package app

import (
	"context"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	"fmt"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	ibchooks "github.com/cosmos/ibc-apps/modules/ibc-hooks/v8"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v8/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	srvflags "github.com/evmos/ethermint/server/flags"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/UptickNetwork/uptick/app/ante"
	"github.com/UptickNetwork/uptick/app/keepers"
	uptickparams "github.com/UptickNetwork/uptick/app/params"
	"github.com/UptickNetwork/uptick/app/rpc"
	_ "github.com/UptickNetwork/uptick/client/docs/statik"
	evmostypes "github.com/UptickNetwork/uptick/types"
	nftmodule "github.com/UptickNetwork/uptick/x/collection/module"
	nfttypes "github.com/UptickNetwork/uptick/x/collection/types"
	erc20 "github.com/UptickNetwork/uptick/x/erc20"
	erc20client "github.com/UptickNetwork/uptick/x/erc20/client"
	erc20types "github.com/UptickNetwork/uptick/x/erc20/types"

	"cosmossdk.io/x/nft"
	"github.com/cosmos/cosmos-sdk/x/consensus"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/evmos/ethermint/x/feemarket"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	"cosmossdk.io/log"
	"cosmossdk.io/simapp"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmos "github.com/cometbft/cometbft/libs/os"
	dbm "github.com/cosmos/cosmos-db"
	sigtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/ibc-go/modules/capability"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"

	erc721 "github.com/UptickNetwork/evm-nft-convert"
	erc721types "github.com/UptickNetwork/evm-nft-convert/types"
	cw721 "github.com/UptickNetwork/wasm-nft-convert"
	cw721types "github.com/UptickNetwork/wasm-nft-convert/types"
	nfttransfer "github.com/bianjieai/nft-transfer"
	//ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".uptickd")

	// manually update the power reduction by replacing micro (u) -> atto (a) uptick
	sdk.DefaultPowerReduction = ethermint.PowerReduction
}

const (
	// Name defines the application binary name
	Name = "uptickd"

	// ProposalsEnabled If EnabledSpecificProposals is "", and this is "true", then enable all x/wasm proposals.
	// If EnabledSpecificProposals is "", and this is not "true", then disable all x/wasm proposals.
	ProposalsEnabled = "true"
	// EnableSpecificProposals If set to non-empty string it must be comma-separated list of values that are all a subset
	// of "EnableAllProposals" (takes precedence over ProposalsEnabled)
	// https://github.com/CosmWasm/wasmd/blob/02a54d33ff2c064f3539ae12d75d027d9c665f05/x/wasm/internal/types/proposal.go#L28-L34
	EnableSpecificProposals = ""
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
				// Uptick proposal types
				erc20client.RegisterCoinProposalHandler,
				erc20client.RegisterERC20ProposalHandler,
				erc20client.ToggleTokenRelayProposalHandler,
			},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		evm.AppModuleBasic{},
		feemarket.AppModuleBasic{},
		erc20.AppModuleBasic{},
		erc721.AppModuleBasic{},
		cw721.AppModuleBasic{},
		nftmodule.AppModuleBasic{},
		nfttransfer.AppModuleBasic{},
		wasm.AppModuleBasic{},
		ica.AppModuleBasic{},
		consensus.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},

		evmtypes.ModuleName:    {authtypes.Minter, authtypes.Burner}, // used for secure addition and subtraction of balance using module account
		erc20types.ModuleName:  {authtypes.Minter, authtypes.Burner},
		erc721types.ModuleName: nil,

		cw721types.ModuleName: nil,

		nfttypes.ModuleName: nil,
		// nft.ModuleName:      nil,
		wasmtypes.ModuleName: {authtypes.Burner},
		icatypes.ModuleName:  nil,
		nft.ModuleName:       nil,
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		distrtypes.ModuleName: true,
	}
)

var (
	_ servertypes.Application = (*Uptick)(nil)
	_ runtime.AppI            = (*Uptick)(nil)
)

// Uptick implements an extended ABCI application. It is an application
// that may process transactions through Ethereum's EVM running atop of
// Tendermint consensus.
type Uptick struct {
	*baseapp.BaseApp
	keepers.AppKeepers
	// encoding
	configurator      module.Configurator
	interfaceRegistry types.InterfaceRegistry
	codec             codec.Codec
	txConfig          client.TxConfig
	legacyAmino       *codec.LegacyAmino

	// the module manager
	mm                 *module.Manager
	BasicModuleManager module.BasicManager
	// simulation manager
	sm         *module.SimulationManager
	tpsCounter *tpsCounter
}

// NewUptick returns a reference to a new initialized Ethermint application.
func NewUptick(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	wasmOpts []wasmkeeper.Option,
	baseAppOptions ...func(*baseapp.BaseApp),
) *Uptick {

	encodingConfig := uptickparams.MakeEncodingConfig()

	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.LegacyAmino
	interfaceRegistry := encodingConfig.InterfaceRegistry
	txConfig := encodingConfig.TxConfig

	// Setup Mempool
	baseAppOptions = append(baseAppOptions, NoOpMempoolOption())

	// NOTE we use custom transaction decoder that supports the sdk.Tx interface instead of sdk.StdTx
	bApp := baseapp.NewBaseApp(
		Name,
		logger,
		db,
		txConfig.TxDecoder(),
		baseAppOptions...,
	)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	app := &Uptick{
		BaseApp:           bApp,
		codec:             appCodec,
		interfaceRegistry: interfaceRegistry,
		txConfig:          txConfig,
		legacyAmino:       legacyAmino,
	}

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	// Setup keepers
	app.AppKeepers = keepers.New(
		appCodec,
		bApp,
		legacyAmino,
		maccPerms,
		app.ModuleAccountAddrs(),
		app.BlockedModuleAccountAddrs(),
		skipUpgradeHeights,
		cast.ToString(appOpts.Get(flags.FlagHome)),
		cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
		logger,
		appOpts,
		wasmOpts,
	)

	/****  Module Options ****/
	skipGenesisInvariants := false
	opt := appOpts.Get(crisis.FlagSkipGenesisInvariants)
	if opt, ok := opt.(bool); ok {
		skipGenesisInvariants = opt
	}

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		// SDK app modules
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app.BaseApp,
			encodingConfig.TxConfig,
		),

		auth.NewAppModule(
			appCodec,
			app.AccountKeeper,
			authsims.RandomGenesisAccounts,
			app.GetSubspace(authtypes.ModuleName),
		),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(
			appCodec,
			app.BankKeeper,
			app.AccountKeeper,
			app.GetSubspace(banktypes.ModuleName),
		),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(*app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		// nftmodule.NewAppModule(appCodec, app.NF	app.mm.SetOrderBeginBlockers(TKeeper, app.AccountKeeper, app.BankKeeper),
		// ibc modules
		ibc.NewAppModule(app.IBCKeeper),
		app.TransferModule,
		app.IBCNftTransferModule,
		app.ICAModule,
		// Ethermint app modules
		evm.NewAppModule(app.EvmKeeper, app.AccountKeeper, app.GetSubspace(evmtypes.ModuleName)),
		feemarket.NewAppModule(app.FeeMarketKeeper, app.GetSubspace(feemarkettypes.ModuleName)),
		// Uptick app modules
		erc20.NewAppModule(*app.Erc20Keeper, app.AccountKeeper),
		erc721.NewAppModule(app.Erc721Keeper, app.AccountKeeper),
		cw721.NewAppModule(app.Cw721Keeper, app.AccountKeeper),
		nftmodule.NewAppModule(app.codec, app.NFTKeeper, app.AccountKeeper, app.BankKeeper),

		// this line is used by starport scaffolding # stargate/app/appModule
		// wasm.NewAppModule(appCodec, &app.wasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		ibchooks.NewAppModule(app.AccountKeeper),
		packetforward.NewAppModule(app.PacketForwardKeeper, app.GetSubspace(packetforwardtypes.ModuleName)),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		//ibcwasm.NewAppModule(app.IBCWasmKeeper),
		// always be last to make sure that it checks for all invariants and not only part of them
		ibctm.NewAppModule(),
	)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.BasicModuleManager = module.NewBasicManagerFromManager(
		app.mm,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
			govtypes.ModuleName: gov.NewAppModuleBasic(
				[]govclient.ProposalHandler{
					paramsclient.ProposalHandler,
				},
			),
		})
	app.BasicModuleManager.RegisterLegacyAminoCodec(legacyAmino)
	app.BasicModuleManager.RegisterInterfaces(interfaceRegistry)

	enabledSignModes := append([]sigtypes.SignMode(nil), authtx.DefaultSignModes...)
	enabledSignModes = append(enabledSignModes, sigtypes.SignMode_SIGN_MODE_TEXTUAL)

	txConfigOpts := authtx.ConfigOptions{
		EnabledSignModes:           enabledSignModes,
		TextualCoinMetadataQueryFn: txmodule.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
	}

	txConfig, err := authtx.NewTxConfigWithOptions(
		appCodec,
		txConfigOpts,
	)
	if err != nil {
		panic(err)
	}
	app.txConfig = txConfig

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: upgrade module must go first to handle software upgrades.
	// NOTE: staking module is required if HistoricalEntries param > 0.
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	app.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
		// Note: epochs' begin should be "real" start of epochs, we keep epochs beginblock at the beginning
		feemarkettypes.ModuleName,
		evmtypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		// no-op modules
		ibctransfertypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		erc20types.ModuleName,
		erc721types.ModuleName,
		cw721types.ModuleName,
		nfttypes.ModuleName,

		ibcnfttransfertypes.ModuleName,
		icatypes.ModuleName,
		wasmtypes.ModuleName,
		consensusparamtypes.ModuleName,
		ibchookstypes.ModuleName,
		packetforwardtypes.ModuleName,
		ibcwasmtypes.ModuleName,
	)

	// NOTE: fee market module must go last in order to retrieve the block gas used.
	app.mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		// no-op modules
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		minttypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		erc20types.ModuleName,
		erc721types.ModuleName,
		cw721types.ModuleName,
		nfttypes.ModuleName,
		ibcnfttransfertypes.ModuleName,
		icatypes.ModuleName,
		wasmtypes.ModuleName,
		consensusparamtypes.ModuleName,
		packetforwardtypes.ModuleName,
		ibchookstypes.ModuleName,
		ibcwasmtypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		// SDK modules
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		// NOTE: staking requires the claiming hook
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		ibcexported.ModuleName,
		// Ethermint modules
		// evm module denomination is used by the feesplit module, in AnteHandle
		evmtypes.ModuleName,
		// NOTE: feemarket module needs to be initialized before genutil module:
		// gentx transactions use MinGasPriceDecorator.AnteHandle
		feemarkettypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,

		erc20types.ModuleName,
		erc721types.ModuleName,
		crisistypes.ModuleName,
		nfttypes.ModuleName,
		ibcnfttransfertypes.ModuleName,
		icatypes.ModuleName,
		wasmtypes.ModuleName,
		consensusparamtypes.ModuleName,
		cw721types.ModuleName,
		ibchookstypes.ModuleName,
		packetforwardtypes.ModuleName,
		ibcwasmtypes.ModuleName,
	)
	// Create and set the configurator
	app.configurator = module.NewConfigurator(app.codec, app.MsgServiceRouter(), app.GRPCQueryRouter())

	app.mm.RegisterInvariants(app.CrisisKeeper)
	app.RegisterServices()

	// create the simulation manager and define the order of the modules for deterministic simulations
	app.sm = module.NewSimulationManager(
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		params.NewAppModule(app.ParamsKeeper),
		evidence.NewAppModule(*app.EvidenceKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		ibc.NewAppModule(app.IBCKeeper),
		app.TransferModule,
		evm.NewAppModule(app.EvmKeeper, app.AccountKeeper, app.GetSubspace(evmtypes.ModuleName)),
		feemarket.NewAppModule(app.FeeMarketKeeper, app.GetSubspace(feemarkettypes.ModuleName)),
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		nftmodule.NewAppModule(appCodec, app.NFTKeeper, app.AccountKeeper, app.BankKeeper),
		app.IBCNftTransferModule,
	)
	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(app.KvStoreKeys())
	app.MountTransientStores(app.TransientStoreKeys())
	app.MountMemoryStores(app.MemoryStoreKeys())

	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))
	options := ante.HandlerOptions{
		AccountKeeper:     app.AccountKeeper,
		BankKeeper:        app.BankKeeper,
		IBCKeeper:         app.IBCKeeper,
		TxCounterStoreKey: app.GetKey(wasm.StoreKey),
		WasmConfig:        app.WasmConfig,
		FeeMarketKeeper:   app.FeeMarketKeeper,
		EvmKeeper:         app.EvmKeeper,
		FeegrantKeeper:    app.FeeGrantKeeper,
		SignModeHandler:   txConfig.SignModeHandler(),
		SigGasConsumer:    SigVerificationGasConsumer,
		MaxTxGasWanted:    maxGasWanted,
	}

	if err := options.Validate(); err != nil {
		panic(err)
	}

	// initialize
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.SetPreBlocker(app.PreBlocker)

	app.RegisterUpgradePlans()

	// Register all module services
	//for _, m := range app.mm.Modules {
	//	if mod, ok := m.(module.HasServices); ok {
	//		mod.RegisterServices(app.configurator)
	//	}
	//}

	if manager := app.SnapshotManager(); manager != nil {
		err := manager.RegisterExtensions(
			wasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmKeeper),
		)
		if err != nil {
			panic("failed to register snapshot extension: " + err.Error())
		}
	}

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == "multiverse" && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{icacontrollertypes.StoreKey, icahosttypes.StoreKey},
		}

		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	// Finally start the tpsCounter.
	app.tpsCounter = newTPSCounter(logger)
	go func() {
		// Unfortunately golangci-lint is so pedantic
		// so we have to ignore this error explicitly.
		_ = app.tpsCounter.start(context.Background())
	}()

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}

		// Initialize and seal the capability keeper so all persistent capabilities
		// are loaded in-memory and prevent any further modules from creating scoped
		// sub-keepers.
		// This must be done during creation of baseapp rather than in InitChain so
		// that in-memory capabilities get regenerated on app restart.
		// Note that since this reads from the store, we can only perform it when
		// `loadLatest` is set to true.
		app.CapabilityKeeper.Seal()
	}

	return app
}

// Name returns the name of the App
func (app *Uptick) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *Uptick) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.mm.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *Uptick) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.mm.EndBlock(ctx)
}

//func (app *Uptick) DeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {
//	defer func() {
//		// TODO: Record the count along with the code and or reason so as to display
//		// in the transactions per second live dashboards.
//		if res.IsErr() {
//			app.tpsCounter.incrementFailure()
//		} else {
//			app.tpsCounter.incrementSuccess()
//		}
//	}()
//
//	return app.BaseApp.DeliverTx(req)
//}

// InitChainer application update at chain initialization
func (app *Uptick) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState simapp.GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, err
	}
	if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap()); err != nil {
		return nil, err
	}
	return app.mm.InitGenesis(ctx, app.codec, genesisState)
}

// LoadHeight loads state at a particular height
func (app *Uptick) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *Uptick) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedAddrs returns all the app's module account addresses that are not
// allowed to receive external tokens.
func (app *Uptick) BlockedAddrs() map[string]bool {
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	return blockedAddrs
}

// LegacyAmino returns Uptick's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *Uptick) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns Uptick's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *Uptick) AppCodec() codec.Codec {
	return app.codec
}

// InterfaceRegistry returns Uptick's InterfaceRegistry
func (app *Uptick) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// EncodingConfig returns Uptick's EncodingConfig
func (app *Uptick) EncodingConfig() uptickparams.EncodingConfig {
	return uptickparams.EncodingConfig{
		InterfaceRegistry: app.interfaceRegistry,
		Codec:             app.codec,
		TxConfig:          app.txConfig,
		LegacyAmino:       app.legacyAmino,
	}
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *Uptick) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *Uptick) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *Uptick) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}

}

// RegisterServices implements the Application.RegisterTxService method.
func (app *Uptick) RegisterServices() {
	for _, mod := range app.mm.Modules {
		m, ok := mod.(module.AppModule)
		if !ok {
			panic("unable to cast mod into AppModule")
		}
		rpc.RegisterService(app.codec, m, app.configurator, app.AppKeepers)
	}
}

func (app *Uptick) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *Uptick) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// RegisterNodeService registers the node gRPC service on the provided
func (app *Uptick) RegisterNodeService(clientCtx client.Context, c config.Config) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), c)
}

//// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
//func (app *Uptick) DefaultGenesis() evmostypes.GenesisState {
//	return app.BasicModuleManager.DefaultGenesis(app.codec)
//}

// GetBaseApp implements the TestingApp interface.
func (app *Uptick) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetStakingKeeper implements the TestingApp interface.
func (app *Uptick) GetStakingKeeper() stakingkeeper.Keeper {
	return *app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *Uptick) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *Uptick) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// GetTxConfig implements the TestingApp interface.
func (app *Uptick) GetTxConfig() client.TxConfig {
	return app.txConfig
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router) {
	statikFS, err := fs.NewWithNamespace("uptick")
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// BlockedModuleAccountAddrs returns all the app's blocked module account
// addresses.
func (app *Uptick) BlockedModuleAccountAddrs() map[string]bool {
	modAccAddrs := app.ModuleAccountAddrs()

	// remove module accounts that are ALLOWED to received funds
	// delete(modAccAddrs, authtypes.NewModuleAddress(grouptypes.ModuleName).String())
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	return modAccAddrs
}

// Deprecated.
//func wasmParamsKeyTable() paramstypes.KeyTable {
//
//	var addrees []string
//	return paramstypes.NewKeyTable(
//		paramstypes.NewParamSetPair(
//			wasmtypes.ParamStoreKeyUploadAccess, wasmtypes.AccessConfig{
//				Permission: wasmtypes.AccessTypeEverybody,
//				Addresses:  addrees,
//			}, validateAccessConfig,
//		),
//		paramstypes.NewParamSetPair(
//			wasmtypes.ParamStoreKeyInstantiateAccess, wasmtypes.AccessTypeEverybody, validateAccessType,
//		),
//	)
//}

func validateAccessConfig(i interface{}) error {
	v, ok := i.(wasmtypes.AccessConfig)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return v.ValidateBasic()
}

func validateAccessType(i interface{}) error {
	a, ok := i.(wasmtypes.AccessType)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if a == wasmtypes.AccessTypeUnspecified {
		return fmt.Errorf("ErrEmpty: %T", i)
		// errorsmod.Wrap(ErrEmpty, "type")
	}
	for _, v := range wasmtypes.AllAccessTypes {
		if v == a {
			return nil
		}
	}
	return fmt.Errorf("unknown type: %q", a)
}

// NoOpMempoolOption returns a function that sets up a no-op mempool for the given BaseApp.
//
// The function takes a pointer to a BaseApp as a parameter and returns nothing.
func NoOpMempoolOption() func(*baseapp.BaseApp) {
	return func(app *baseapp.BaseApp) {
		memPool := mempool.NoOpMempool{}
		app.SetMempool(memPool)
		handler := baseapp.NewDefaultProposalHandler(memPool, app)
		app.SetPrepareProposal(handler.PrepareProposalHandler())
		app.SetProcessProposal(handler.ProcessProposalHandler())
	}
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *Uptick) DefaultGenesis() evmostypes.GenesisState {
	return app.BasicModuleManager.DefaultGenesis(app.AppCodec())
}

// PreBlocker application updates every pre block
func (app *Uptick) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.mm.PreBlock(ctx)
}
