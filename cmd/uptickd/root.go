package main

import (
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/log"
	rosettaCmd "cosmossdk.io/tools/rosetta/cmd"
	"errors"
	"fmt"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/UptickNetwork/uptick/app/params"
	uptickparams "github.com/UptickNetwork/uptick/app/params"
	cmdcfg "github.com/UptickNetwork/uptick/cmd/config"
	tmcfg "github.com/cometbft/cometbft/config"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	srvflags "github.com/evmos/ethermint/server/flags"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"

	confixcmd "cosmossdk.io/tools/confix/cmd"
	"github.com/UptickNetwork/uptick/app"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	ethermintclient "github.com/evmos/ethermint/client"
	"github.com/evmos/ethermint/client/debug"
	"github.com/evmos/ethermint/crypto/hd"
	ethermintserver "github.com/evmos/ethermint/server"
	servercfg "github.com/evmos/ethermint/server/config"
)

const (
	EnvPrefix = "UPTICK"
)

// NewRootCmd creates a new root command for uptickd. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, params.EncodingConfig) {

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
		WithBroadcastMode(flags.FlagBroadcastMode).
		WithHomeDir(app.DefaultNodeHome).
		WithKeyringOptions(hd.EthSecp256k1Option()).
		WithViper(EnvPrefix)

	rootCmd := &cobra.Command{
		Use:   app.Name,
		Short: "Uptick Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			// This needs to go after ReadFromClientConfig, as that function
			// sets the RPC client needed for SIGN_MODE_TEXTUAL. This sign mode
			// is only available if the client is online.
			if !initClientCtx.Offline {
				enabledSignModes := append(tx.DefaultSignModes, signing.SignMode_SIGN_MODE_TEXTUAL) //nolint:gocritic
				txConfigOpts := tx.ConfigOptions{
					EnabledSignModes:           enabledSignModes,
					TextualCoinMetadataQueryFn: txmodule.NewGRPCCoinMetadataQueryFn(initClientCtx),
				}
				txConfig, err := tx.NewTxConfigWithOptions(
					initClientCtx.Codec,
					txConfigOpts,
				)
				if err != nil {
					return err
				}

				initClientCtx = initClientCtx.WithTxConfig(txConfig)
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customTMConfig := initTendermintConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customTMConfig)
		},
		SilenceUsage: true,
	}

	cfg := sdk.GetConfig()
	cfg.Seal()
	ac := appCreator{encodingConfig}
	rootCmd.AddCommand(
		ethermintclient.ValidateChainID(
			InitCmd(tempApplication.BasicModuleManager, app.DefaultNodeHome),
		),

		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{},
			app.DefaultNodeHome,
			genutiltypes.DefaultMessageValidator,
			tempApplication.GetTxConfig().SigningContext().ValidatorAddressCodec(),
		),

		MigrateGenesisCmd(),
		genutilcli.GenTxCmd(tempApplication.BasicModuleManager, tempApplication.GetTxConfig(), banktypes.GenesisBalancesIterator{}, app.DefaultNodeHome, tempApplication.GetTxConfig().SigningContext().ValidatorAddressCodec()),
		genutilcli.ValidateGenesisCmd(tempApplication.BasicModuleManager),
		AddGenesisAccountCmd(app.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		NewTestnetCmd(tempApplication.BasicModuleManager, banktypes.GenesisBalancesIterator{}),
		AddIbcCaclulateCommand(debug.Cmd()),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(ac.newApp, app.DefaultNodeHome),
		snapshot.Cmd(ac.newApp),
	)

	ethermintserver.AddCommands(
		rootCmd,
		ethermintserver.NewDefaultStartOptions(ac.newApp, app.DefaultNodeHome),
		ac.appExport,
		addModuleInitFlags,
	)

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(tempApplication.BasicModuleManager, encodingConfig),
		queryCommand(),
		txCommand(tempApplication.BasicModuleManager),
		ethermintclient.KeyCommands(app.DefaultNodeHome),
	)

	autoCliOpts := enrichAutoCliOpts(tempApplication.AutoCliOpts(), initClientCtx)
	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	rootCmd, err := srvflags.AddTxFlags(rootCmd)
	if err != nil {
		panic(err)
	}

	// add rosetta
	rootCmd.AddCommand(rosettaCmd.RosettaCommand(encodingConfig.InterfaceRegistry, encodingConfig.Codec))

	return rootCmd, encodingConfig
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

	app.ModuleBasics.AddQueryCommands(cmd)
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
	customAppTemplate, customAppConfig := servercfg.AppConfig(cmdcfg.BaseDenom)

	srvCfg, ok := customAppConfig.(servercfg.Config)
	if !ok {
		panic(fmt.Errorf("unknown app config type %T", customAppConfig))
	}

	srvCfg.StateSync.SnapshotInterval = 1500
	srvCfg.StateSync.SnapshotKeepRecent = 2
	srvCfg.IAVLDisableFastNode = false
	return customAppTemplate, srvCfg
}

type appCreator struct {
	encCfg uptickparams.EncodingConfig
}

// newApp is an appCreator
func (a appCreator) newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {

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
