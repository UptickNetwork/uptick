package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/version"

	evmconfig "github.com/cosmos/evm/config"
	cosmosevmutils "github.com/cosmos/evm/utils"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	cosmosnft "cosmossdk.io/x/nft"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sigtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	// 	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types" // removed in ibc-go v10
	srvflags "github.com/cosmos/evm/server/flags"

	"github.com/UptickNetwork/uptick/app/ante"
	"github.com/UptickNetwork/uptick/app/keepers"
	uptickparams "github.com/UptickNetwork/uptick/app/params"
	v041 "github.com/UptickNetwork/uptick/app/upgrades/v041"
	_ "github.com/UptickNetwork/uptick/client/docs/statik" // registers statik FS for the Swagger UI
	cmdcfg "github.com/UptickNetwork/uptick/cmd/config"
	nftmodule "github.com/UptickNetwork/uptick/x/collection/module"
	nfttypes "github.com/UptickNetwork/uptick/x/collection/types"
	cosmoserc20 "github.com/cosmos/evm/x/erc20"
	cosmoserc20types "github.com/cosmos/evm/x/erc20/types"

	upticktypes "github.com/UptickNetwork/uptick/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client/http"
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
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
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
	evmmempool "github.com/cosmos/evm/mempool"
	"github.com/cosmos/evm/x/feemarket"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	vm "github.com/cosmos/evm/x/vm"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	ibcsolomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	"cosmossdk.io/log"
	"cosmossdk.io/simapp"
	"cosmossdk.io/x/evidence"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	cw721 "github.com/UptickNetwork/uptick/x/cw721"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721 "github.com/UptickNetwork/uptick/x/erc721"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmos "github.com/cometbft/cometbft/libs/os"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".uptickd")

	// manually update the power reduction by replacing micro (u) -> atto (a) uptick
	sdk.DefaultPowerReduction = upticktypes.PowerReduction
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

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		evmtypes.ModuleName:            {authtypes.Minter, authtypes.Burner}, // used for secure addition and subtraction of balance using module account
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		erc721types.ModuleName:         nil,

		cosmoserc20types.ModuleName: {authtypes.Minter, authtypes.Burner},

		cw721types.ModuleName: nil,

		cosmosnft.ModuleName:           nil, // cosmossdk.io/x/nft module account required by collection keeper
		nfttypes.ModuleName:            nil, // x/collection
		wasmtypes.ModuleName:           {authtypes.Burner},
		icatypes.ModuleName:            nil,
		feemarkettypes.ModuleName:      nil,
		ibcnfttransfertypes.ModuleName: {authtypes.Minter, authtypes.Burner},
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

	// evm mempool (required by cosmos/evm JSON-RPC server)
	evmMempool *evmmempool.ExperimentalEVMMempool
	clientCtx  client.Context

	// the module manager
	mm *module.Manager
	bm module.BasicManager
	// simulation manager
	sm *module.SimulationManager
}

// NewUptick returns a reference to a new initialized Uptick application.
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

	// NOTE: the EVM mempool is configured later (after keepers are set up) via
	// configureEVMMempool, which requires the EVMKeeper and FeeMarketKeeper.

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

	// ibc-go v10 ClientKeeper only auto-registers localhost. Tendermint and
	// solomachine must be routed with the same LightClientModule instances that
	// are passed to their AppModules, otherwise client updates fail with
	// ErrRouteNotFound.
	storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()
	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	app.IBCKeeper.ClientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)
	smLightClientModule := ibcsolomachine.NewLightClientModule(appCodec, storeProvider)
	app.IBCKeeper.ClientKeeper.AddRoute(ibcsolomachine.ModuleName, &smLightClientModule)

	/****  Module Options ****/
	// Wire the EVM default static precompiles explicitly: fresh-chain genesis
	// defaults must activate Uptick's precompile set before the module manager
	// (and its DefaultGenesis -> evmtypes.DefaultParams) is built.
	v041.ConfigureDefaultStaticPrecompiles()

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
		// ibc modules
		ibc.NewAppModule(app.IBCKeeper),
		ibctm.NewAppModule(tmLightClientModule),
		ibcsolomachine.NewAppModule(smLightClientModule),
		app.TransferModule,
		app.IBCNftTransferModule,
		app.ICAModule,
		// cosmos/evm app modules
		vm.NewAppModule(app.EvmKeeper, app.AccountKeeper, app.BankKeeper, app.AccountKeeper.AddressCodec()),
		feemarket.NewAppModule(app.FeeMarketKeeper),
		// Uptick app modules
		cosmoserc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
		erc721.NewAppModule(app.Erc721Keeper, app.AccountKeeper),
		cw721.NewAppModule(app.Cw721Keeper, app.AccountKeeper),
		nftmodule.NewAppModule(app.codec, app.NFTKeeper, app.AccountKeeper, app.BankKeeper),

		// this line is used by starport scaffolding # stargate/app/appModule
		// wasm.NewAppModule(appCodec, &app.wasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
	)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.bm = module.NewBasicManagerFromManager(
		app.mm,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
			govtypes.ModuleName: gov.NewAppModuleBasic(
				[]govclient.ProposalHandler{
					paramsclient.ProposalHandler,
				},
			),
		})
	app.bm.RegisterLegacyAminoCodec(legacyAmino)
	app.bm.RegisterInterfaces(interfaceRegistry)
	// The legacy EIP-712 verification path (Keplr) reconstructs the amino
	// StdSignBytes for signature checking, which requires the app's fully
	// populated amino codec.
	legacytx.RegressionTestingAminoCodec = legacyAmino

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

	// NOTE: upgrade module is required to be prioritized
	// cosmos/evm requires auth and evm modules as preblockers.
	app.mm.SetOrderPreBlockers(
		upgradetypes.ModuleName,
		authtypes.ModuleName,
		evmtypes.ModuleName,
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: upgrade module must go first to handle software upgrades.
	// NOTE: staking module is required if HistoricalEntries param > 0.
	app.mm.SetOrderBeginBlockers(

		// Note: epochs' begin should be "real" start of epochs, we keep epochs beginblock at the beginning
		feemarkettypes.ModuleName,
		evmtypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibctm.ModuleName,
		ibcsolomachine.ModuleName,
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
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		cosmoserc20types.ModuleName,
		erc721types.ModuleName,
		cw721types.ModuleName,
		nfttypes.ModuleName,

		ibcnfttransfertypes.ModuleName,
		icatypes.ModuleName,
		wasmtypes.ModuleName,
		consensusparamtypes.ModuleName,
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
		ibctm.ModuleName,
		ibcsolomachine.ModuleName,
		ibctransfertypes.ModuleName,
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
		cosmoserc20types.ModuleName,
		erc721types.ModuleName,
		cw721types.ModuleName,
		nfttypes.ModuleName,
		ibcnfttransfertypes.ModuleName,
		icatypes.ModuleName,
		wasmtypes.ModuleName,
		consensusparamtypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	app.mm.SetOrderInitGenesis(
		// SDK modules
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		// NOTE: staking requires the claiming hook
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		ibcexported.ModuleName,
		ibctm.ModuleName,
		ibcsolomachine.ModuleName,
		// cosmos/evm modules
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

		cosmoserc20types.ModuleName,
		erc721types.ModuleName,
		crisistypes.ModuleName,
		nfttypes.ModuleName,
		ibcnfttransfertypes.ModuleName,
		icatypes.ModuleName,
		wasmtypes.ModuleName,
		cw721types.ModuleName,
		consensusparamtypes.ModuleName,
	)

	app.mm.SetOrderExportGenesis(
		// SDK modules
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		// NOTE: staking requires the claiming hook
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		ibcexported.ModuleName,
		ibctm.ModuleName,
		ibcsolomachine.ModuleName,
		// cosmos/evm modules
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

		cosmoserc20types.ModuleName,
		erc721types.ModuleName,
		crisistypes.ModuleName,
		nfttypes.ModuleName,
		ibcnfttransfertypes.ModuleName,
		icatypes.ModuleName,
		wasmtypes.ModuleName,
		cw721types.ModuleName,
		consensusparamtypes.ModuleName,
	)
	// Create and set the configurator
	app.configurator = module.NewConfigurator(app.codec, app.MsgServiceRouter(), app.GRPCQueryRouter())

	app.mm.RegisterInvariants(app.CrisisKeeper)
	if err := app.mm.RegisterServices(app.configurator); err != nil {
		panic(err)
	}

	// create the simulation manager and define the order of the modules for deterministic simulations
	app.sm = module.NewSimulationManager(
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
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
		vm.NewAppModule(app.EvmKeeper, app.AccountKeeper, app.BankKeeper, app.AccountKeeper.AddressCodec()),
		feemarket.NewAppModule(app.FeeMarketKeeper),
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
		Cdc:                   appCodec,
		AccountKeeper:         app.AccountKeeper,
		BankKeeper:            app.BankKeeper,
		IBCKeeper:             app.IBCKeeper,
		FeeMarketKeeper:       app.FeeMarketKeeper,
		EvmKeeper:             app.EvmKeeper,
		FeegrantKeeper:        app.FeeGrantKeeper,
		SignModeHandler:       txConfig.SignModeHandler(),
		SigGasConsumer:        SigVerificationGasConsumer,
		MaxTxGasWanted:        maxGasWanted,
		WasmKeeper:            &app.WasmKeeper,
		WasmNodeConfig:        &app.WasmConfig,
		TXCounterStoreService: runtime.NewKVStoreService(app.GetKey(wasm.StoreKey)),
		DisabledAuthzMsgs: []string{
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&vestingtypes.MsgCreateVestingAccount{}),
			// Governance messages: prevent delegation of governance actions via authz
			sdk.MsgTypeURL(&govv1.MsgSubmitProposal{}),
			sdk.MsgTypeURL(&govv1.MsgVote{}),
			sdk.MsgTypeURL(&govv1.MsgVoteWeighted{}),
			sdk.MsgTypeURL(&govv1.MsgDeposit{}),
			sdk.MsgTypeURL(&upgradetypes.MsgSoftwareUpgrade{}),
			sdk.MsgTypeURL(&upgradetypes.MsgCancelUpgrade{}),
			sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
			sdk.MsgTypeURL(&ibcclienttypes.MsgUpgradeClient{}),
		},
	}

	if err := options.Validate(); err != nil {
		panic(err)
	}

	// initialize
	app.SetAnteHandler(ante.NewAnteHandler(options))
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.RegisterUpgradePlans()

	// Build a client context for the EVM mempool (used for broadcasting pending
	// EVM transactions). This is required by the EVM mempool's broadcast helper.
	evmClientCtx := client.Context{}.
		WithCodec(appCodec).
		WithInterfaceRegistry(interfaceRegistry).
		WithTxConfig(txConfig).
		WithLegacyAmino(legacyAmino).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithHomeDir(cast.ToString(appOpts.Get(flags.FlagHome))).
		WithBroadcastMode(flags.BroadcastSync)
	// Wire a CometBFT RPC client so the mempool can broadcast pending EVM txs.
	if nodeURI := cast.ToString(appOpts.Get(flags.FlagNode)); nodeURI != "" {
		if rpcClient, err := rpcclient.New(nodeURI, "/websocket"); err == nil {
			evmClientCtx = evmClientCtx.WithClient(rpcClient)
		}
	}
	app.clientCtx = evmClientCtx

	// Configure the EVM mempool (required for the EVM JSON-RPC server).
	// Must run before the BaseApp is sealed by LoadLatestVersion.
	app.configureEVMMempool(appOpts, logger)

	if manager := app.SnapshotManager(); manager != nil {
		err := manager.RegisterExtensions(
			wasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmKeeper),
		)
		if err != nil {
			panic("failed to register snapshot extension: " + err.Error())
		}
	}

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}

		// Pinned codes live in wasmvm's in-memory cache and are not persisted
		// across process restarts. Reload them from state after the store is mounted.
		ctx := app.NewUncachedContext(true, tmproto.Header{})
		if err := app.WasmKeeper.InitializePinnedCodes(ctx); err != nil {
			tmos.Exit("failed to initialize pinned wasm codes: " + err.Error())
		}
	}

	return app
}

// Name returns the name of the App
func (app *Uptick) Name() string { return app.BaseApp.Name() }

// GetMempool returns the app's mempool.
// Required by cosmos/evm server.Application interface.
func (app *Uptick) GetMempool() sdkmempool.ExtMempool {
	if extMempool, ok := app.BaseApp.Mempool().(sdkmempool.ExtMempool); ok {
		return extMempool
	}
	// Fallback: return nil if the mempool doesn't implement ExtMempool
	return nil
}

// RegisterPendingTxListener registers a pending tx listener.
// Required by cosmos/evm server.Application interface.
func (app *Uptick) RegisterPendingTxListener(listener func(common.Hash)) {
	// no-op: Uptick does not use pending tx listeners
}

// SetClientCtx sets the client context on the app.
// Required by cosmos/evm server.Application interface.
func (app *Uptick) SetClientCtx(clientCtx client.Context) {
	app.clientCtx = clientCtx
}

const (
	defaultCosmosPoolMaxTx  = 5000
	defaultEVMBlockGasLimit = ^uint64(0)
)

// configureEVMMempool sets up the cosmos/evm experimental EVM mempool and the
// related ABCI handlers. Required for the EVM JSON-RPC server to function.
// Modeled on cosmos/evm's evmd.ConfigureEVMMempool.
func (app *Uptick) configureEVMMempool(appOpts servertypes.AppOptions, logger log.Logger) {
	if evmtypes.GetChainConfig() == nil {
		logger.Debug("evm chain config is not set, skipping EVM mempool configuration")
		return
	}

	// Read operator-configurable mempool knobs from the cosmos/evm app.toml
	// ([evm] section), falling back to genesis/consensus values where relevant.
	cosmosPoolMaxTx := evmconfig.GetCosmosPoolMaxTx(appOpts, logger)
	// Critical fix: cosmos-sdk's mempool.DefaultMaxTx = -1. When --mempool.max-txs is
	// not set, GetCosmosPoolMaxTx returns -1 and PriorityNonceMempool.Insert hits the
	// `MaxTx < 0` branch and returns nil without inserting. On upgraded/unconfigured
	// chains this silently drops Cosmos transactions and leaves blocks permanently
	// empty (num_txs=0). Normalize non-positive values to 5000 to match the mainnet
	// convention.
	if cosmosPoolMaxTx <= 0 {
		logger.Warn(
			"cosmos pool max tx is non-positive, defaulting to configured fallback",
			"got", cosmosPoolMaxTx,
			"fallback", defaultCosmosPoolMaxTx,
		)
		cosmosPoolMaxTx = defaultCosmosPoolMaxTx
	}

	blockGasLimit := evmconfig.GetBlockGasLimit(appOpts, logger)
	if blockGasLimit == 0 {
		logger.Warn(
			"evm mempool block gas limit is zero, using unlimited fallback",
			"fallback", defaultEVMBlockGasLimit,
		)
		blockGasLimit = defaultEVMBlockGasLimit
	}

	mempoolConfig := &evmmempool.EVMMempoolConfig{
		AnteHandler:      app.AnteHandler(),
		LegacyPoolConfig: evmconfig.GetLegacyPoolConfig(appOpts, logger),
		// Do not use evmconfig.GetBlockGasLimit: it reads from the SDK AppGenesis
		// ConsensusParams in genesis.json, while v0.3.3 genesis only writes CometBFT's
		// `consensus` field (no `consensus_params`). After SDK v0.53 parsing,
		// ConsensusParams is nil and the function returns 0, so the EVM mempool rejects
		// every Cosmos tx as over the limit (num_txs=0, no Cosmos tx can be packed
		// after the upgrade). The block gas limit is enforced by the consensus layer,
		// so use MaxUint64 here (no pre-filtering), matching Uptick's max_gas=-1
		// semantics.
		BlockGasLimit: blockGasLimit,
		MinTip:        evmconfig.GetMinTip(appOpts, logger),
		// Critical fix: disable the default promote broadcast to avoid a deadlock.
		//
		// cosmos/evm's ExperimentalEVMMempool.Insert holds m.mtx and calls
		// txPool.Add(sync=true), whose requestPromoteExecutables blocks on <-done
		// waiting for the asynchronous runReorg to finish. runReorg invokes
		// BroadcastTxFn after promoting transactions; the default implementation
		// synchronously calls clientCtx.BroadcastTxSync, which re-enters CometBFT's
		// CheckTx → app-side mempool.Insert → m.mtx.Lock() while m.mtx is already held
		// by the outer Insert blocking on <-done, forming a deadlock that stalls the
		// chain at the PrepareProposal stage.
		//
		// Transactions already reach the mempool through eth_sendRawTransaction via
		// CometBFT broadcast, so there is no need to broadcast again during promote;
		// making this a no-op breaks the deadlock cycle.
		BroadCastTxFn: func(txs []*ethtypes.Transaction) error {
			return nil
		},
	}

	evmMempool := evmmempool.NewExperimentalEVMMempool(
		app.CreateQueryContext,
		logger,
		app.EvmKeeper,
		app.FeeMarketKeeper,
		app.txConfig,
		app.clientCtx,
		mempoolConfig,
		cosmosPoolMaxTx,
	)
	app.evmMempool = evmMempool
	app.SetMempool(evmMempool)
	checkTxHandler := evmmempool.NewCheckTxHandler(evmMempool)
	app.SetCheckTxHandler(checkTxHandler)

	abciProposalHandler := baseapp.NewDefaultProposalHandler(evmMempool, app)
	abciProposalHandler.SetSignerExtractionAdapter(
		evmmempool.NewEthSignerExtractionAdapter(
			sdkmempool.NewDefaultSignerExtractionAdapter(),
		),
	)
	app.SetPrepareProposal(abciProposalHandler.PrepareProposalHandler())

}

// BeginBlocker application updates every begin block
func (app *Uptick) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.mm.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *Uptick) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.mm.EndBlock(ctx)
}

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
	app.bm.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
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
func (app *Uptick) GetScopedIBCKeeper() interface{} {
	return nil
}

// GetTxConfig implements the TestingApp interface.
func (app *Uptick) GetTxConfig() client.TxConfig {
	return app.txConfig
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router) {
	// The swagger UI assets are embedded under the default statik namespace
	// (client/docs/statik). Reading a named "uptick" namespace here fails with
	// "statik/fs: no zip data registered" because no such namespace is ever
	// registered, which panics any node that enables api.swagger.
	statikFS, err := fs.New()
	if err != nil {
		// Do not panic when the statik namespace is unavailable -- a chain that
		// enables api.swagger without the UI asset must degrade gracefully
		// (log and leave the /swagger route unregistered) instead of crashing.
		_, _ = fmt.Fprintf(os.Stderr, "failed to register swagger UI, swagger route disabled: %v\n", err)
		return
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// BlockedModuleAccountAddrs returns all the app's blocked module account
// addresses.
func (app *Uptick) BlockedModuleAccountAddrs() map[string]bool {
	modAccAddrs := app.ModuleAccountAddrs()

	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	blockedPrecompilesHex := append([]string{}, evmtypes.AvailableStaticPrecompiles...)
	for _, addr := range corevm.PrecompiledAddressesPrague {
		blockedPrecompilesHex = append(blockedPrecompilesHex, addr.Hex())
	}
	for _, precompile := range blockedPrecompilesHex {
		modAccAddrs[cosmosevmutils.Bech32StringFromHexAddress(precompile)] = true
	}

	return modAccAddrs
}

// NoOpMempoolOption returns a function that sets up a no-op mempool for the given BaseApp.
//
// The function takes a pointer to a BaseApp as a parameter and returns nothing.
func NoOpMempoolOption() func(*baseapp.BaseApp) {
	return func(app *baseapp.BaseApp) {
		memPool := sdkmempool.NoOpMempool{}
		app.SetMempool(memPool)
		handler := baseapp.NewDefaultProposalHandler(memPool, app)
		app.SetPrepareProposal(handler.PrepareProposalHandler())
		app.SetProcessProposal(handler.ProcessProposalHandler())
	}
}

// CustomizeDefaultGenesis overlays Uptick-specific defaults onto a
// BasicManager genesis map. cosmos/evm defaults EvmDenom to "aatom"; Uptick
// uses "auptick" and must also inject matching bank denom metadata.
func CustomizeDefaultGenesis(cdc codec.JSONCodec, genesis upticktypes.GenesisState) {
	evmDenom := cmdcfg.BaseDenom // "auptick"

	var evmGenState evmtypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[evmtypes.ModuleName], &evmGenState)
	evmGenState.Params.EvmDenom = evmDenom
	genesis[evmtypes.ModuleName] = cdc.MustMarshalJSON(&evmGenState)

	var bankGenState banktypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[banktypes.ModuleName], &bankGenState)
	bankGenState.DenomMetadata = append(bankGenState.DenomMetadata, banktypes.Metadata{
		Description: "Uptick mainnet token",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: evmDenom, Exponent: 0},
			{Denom: cmdcfg.DisplayDenom, Exponent: uint32(upticktypes.BaseDenomUnit)},
		},
		Base:    evmDenom,
		Display: cmdcfg.DisplayDenom,
		Name:    "Uptick",
		Symbol:  strings.ToUpper(cmdcfg.DisplayDenom),
	})
	genesis[banktypes.ModuleName] = cdc.MustMarshalJSON(&bankGenState)
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *Uptick) DefaultGenesis() upticktypes.GenesisState {
	genesis := app.bm.DefaultGenesis(app.AppCodec())
	CustomizeDefaultGenesis(app.AppCodec(), genesis)
	return genesis
}

// PreBlocker application updates every pre block
func (app *Uptick) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.mm.PreBlock(ctx)
}

// AutoCliOpts returns the autocli options for the app.
func (app *Uptick) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule, 0)
	for _, m := range app.mm.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		AddressCodec:          authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// BasicManager return the basic manager
func (app *Uptick) BasicManager() module.BasicManager {
	return app.bm
}
