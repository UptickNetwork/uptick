package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/UptickNetwork/uptick/app"
	"github.com/UptickNetwork/uptick/app/params"
	uptickparams "github.com/UptickNetwork/uptick/app/params"
	cmdcfg "github.com/UptickNetwork/uptick/cmd/config"
	upticktypes "github.com/UptickNetwork/uptick/types"
	tmcfg "github.com/cometbft/cometbft/config"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	evmclient "github.com/cosmos/evm/client"
	"github.com/cosmos/evm/client/debug"
	evmconfig "github.com/cosmos/evm/config"
	"github.com/cosmos/evm/crypto/hd"
	evmserver "github.com/cosmos/evm/server"
	srvflags "github.com/cosmos/evm/server/flags"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	EnvPrefix = "UPTICK"
)

// NewRootCmd creates a new root command for uptickd. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {

	initAppOptions := viper.New()
	tempDir := tempDir()
	initAppOptions.Set(flags.FlagHome, tempDir)
	tempApplication := app.NewUptick(log.NewNopLogger(), dbm.NewMemDB(), nil, true, initAppOptions, []wasmkeeper.Option{})
	encodingConfig := tempApplication.EncodingConfig()

	defer func() {
		if err := tempApplication.Close(); err != nil {
			panic(err)
		}
		if tempDir != app.DefaultNodeHome {
			os.RemoveAll(tempDir)
		}
	}()
	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.LegacyAmino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync).
		WithHomeDir(app.DefaultNodeHome).
		WithKeyringOptions(hd.EthSecp256k1Option()).
		WithViper(EnvPrefix)

	rootCmd := &cobra.Command{
		Use:   app.Name,
		Short: "Uptick Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err = client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customTMConfig := initTendermintConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customTMConfig)
		},
		SilenceUsage: true,
	}

	ac := appCreator{}
	rootCmd.AddCommand(
		InitCmd(tempApplication.BasicManager(), app.DefaultNodeHome),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{},
			app.DefaultNodeHome,
			genutiltypes.DefaultMessageValidator,
			tempApplication.GetTxConfig().SigningContext().ValidatorAddressCodec(),
		),
		MigrateGenesisCmd(),
		genutilcli.GenTxCmd(tempApplication.BasicManager(), tempApplication.GetTxConfig(), banktypes.GenesisBalancesIterator{}, app.DefaultNodeHome, tempApplication.GetTxConfig().SigningContext().ValidatorAddressCodec()),
		genutilcli.ValidateGenesisCmd(tempApplication.BasicManager()),
		AddGenesisAccountCmd(app.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		// NewTestnetCmd excluded (testnet.go build-constrained)
		AddIbcCaclulateCommand(debug.Cmd()),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(ac.newApp, app.DefaultNodeHome),
		snapshot.Cmd(ac.newApp),
	)

	evmserver.AddCommands(
		rootCmd,
		evmserver.NewDefaultStartOptions(ac.newEvmApp, app.DefaultNodeHome),
		ac.appExport,
		addModuleInitFlags,
	)

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(tempApplication.BasicManager(), encodingConfig),
		queryCommand(),
		txCommand(tempApplication.BasicManager()),
		evmclient.KeyCommands(app.DefaultNodeHome, false),
	)

	autoCliOpts := enrichAutoCliOpts(tempApplication.AutoCliOpts(), initClientCtx)
	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	rootCmd, err := srvflags.AddTxFlags(rootCmd)
	if err != nil {
		panic(err)
	}

	// TODO: rosetta disabled - incompatible with SDK 0.53
	// rootCmd.AddCommand(rosettaCmd.RosettaCommand(encodingConfig.InterfaceRegistry, encodingConfig.Codec))
	return rootCmd
}

func enrichAutoCliOpts(autoCliOpts autocli.AppOptions, clientCtx client.Context) autocli.AppOptions {
	autoCliOpts.AddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	autoCliOpts.ValidatorAddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	autoCliOpts.ConsensusAddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	autoCliOpts.ClientCtx = clientCtx

	return autoCliOpts
}

// genesisCommand builds genesis-related `simd genesis` command. Users may provide application specific commands as a parameter
func genesisCommand(basicManager module.BasicManager, encodingConfig params.EncodingConfig, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.Commands(
		encodingConfig.TxConfig,
		basicManager,
		app.DefaultNodeHome,
	)

	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
	wasm.AddModuleInitFlags(startCmd)
	// wasmd flags default to 3M query gas / 100 MiB cache. Those defaults win
	// over app.toml via viper, which would starve CW721 metadata queries.
	cfg := cmdcfg.DefaultWasmNodeConfig()
	queryGas := fmt.Sprintf("%d", cfg.SmartQueryGasLimit)
	if f := startCmd.Flags().Lookup("wasm.query_gas_limit"); f != nil {
		f.DefValue = queryGas
		_ = f.Value.Set(queryGas)
	}
	cacheSize := fmt.Sprintf("%d", cfg.MemoryCacheSize)
	if f := startCmd.Flags().Lookup("wasm.memory_cache_size"); f != nil {
		f.DefValue = cacheSize
		_ = f.Value.Set(cacheSize)
	}
	if cfg.SimulationGasLimit != nil {
		simGas := fmt.Sprintf("%d", *cfg.SimulationGasLimit)
		if f := startCmd.Flags().Lookup("wasm.simulation_gas_limit"); f != nil {
			f.DefValue = simGas
			_ = f.Value.Set(simGas)
		}
	}
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.ValidatorCommand(),
		server.QueryBlocksCmd(),
		server.QueryBlockCmd(),
		server.QueryBlockResultsCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),

		//authcmd.GetAccountCmd(),
		//rpc.BlockCommand(),
		//rpc.QueryEventForTxCmd(),
	)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func txCommand(basicManager module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
	)

	basicManager.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// cosmos/evm provides the EVM/JSONRPC/TLS app.toml configuration.
	evmChainID := upticktypes.MainnetEVMChainID
	customAppTemplate, customAppConfig := evmconfig.InitAppConfig(cmdcfg.BaseDenom, evmChainID)
	customAppTemplate += cmdcfg.WasmConfigTemplate()

	// apply snapshot / fast-node settings via the concrete EVMAppConfig type.
	if cfg, ok := customAppConfig.(evmconfig.EVMAppConfig); ok {
		cfg.StateSync.SnapshotInterval = 1500
		cfg.StateSync.SnapshotKeepRecent = 2
		cfg.IAVLDisableFastNode = false
		return customAppTemplate, cfg
	}
	return customAppTemplate, customAppConfig
}

type appCreator struct {
	encCfg uptickparams.EncodingConfig
}

// newApp is an appCreator
func (a appCreator) newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	return a.newEvmApp(logger, db, traceStore, appOpts)
}

func (a appCreator) newEvmApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) evmserver.Application {

	var wasmOpts []wasmkeeper.Option
	if cast.ToBool(appOpts.Get("telemetry.enabled")) {
		wasmOpts = append(wasmOpts, wasmkeeper.WithVMCacheMetrics(prometheus.DefaultRegisterer))
	}
	baseappOptions := server.DefaultBaseappOptions(appOpts)
	return app.NewUptick(
		logger,
		db,
		traceStore,
		true,
		appOpts,
		wasmOpts,
		baseappOptions...,
	)
}

// createIrisappAndExport creates a new irisapp (optionally at a given height) and exports state.
func (ac appCreator) appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var uptickApp *app.Uptick
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home is not set")
	}

	var loadLatest bool
	if height == -1 {
		loadLatest = true
	}
	var emptyWasmOpts []wasmkeeper.Option
	uptickApp = app.NewUptick(logger, db, traceStore, loadLatest, appOpts, emptyWasmOpts)
	if height != -1 {
		if err := uptickApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return uptickApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// initTendermintConfig helps to override default Tendermint Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

var tempDir = func() string {
	dir, err := os.MkdirTemp("", ".uptickd")
	if err != nil {
		panic(fmt.Sprintf("failed creating temp directory: %s", err.Error()))
	}
	defer os.RemoveAll(dir)

	return dir
}
