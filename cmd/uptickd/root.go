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

			if err := server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customTMConfig); err != nil {
				return err
			}
			if err := keepExportStdoutClean(cmd); err != nil {
				return err
			}
			if err := applyEVMChainID(cmd); err != nil {
				return err
			}
			return nil
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
		PrecheckCollectionMigrationCmd(app.DefaultNodeHome),
		genutilcli.GenTxCmd(tempApplication.BasicManager(), tempApplication.GetTxConfig(), banktypes.GenesisBalancesIterator{}, app.DefaultNodeHome, tempApplication.GetTxConfig().SigningContext().ValidatorAddressCodec()),
		genutilcli.ValidateGenesisCmd(tempApplication.BasicManager()),
		AddGenesisAccountCmd(app.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		NewTestnetCmd(tempApplication.BasicManager(), banktypes.GenesisBalancesIterator{}),
		AddIBCDenomCommand(debug.Cmd()),
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
	// The export command must carry the degraded-export reporting step. Without
	// it, an export whose diagnostics report could not be written is again
	// indistinguishable from a clean one for every consumer that does not read
	// the node log -- the very defect this step exists to close. Failing fast
	// at construction time is the same treatment the two panics below give to
	// the other construction-time invariants.
	if !wrapExportCommand(rootCmd) {
		panic("uptickd: the SDK export command was not found; the degraded-export exit code cannot be installed")
	}

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(tempApplication.BasicManager(), encodingConfig),
		queryCommand(),
		txCommand(tempApplication.BasicManager()),
		evmclient.KeyCommands(app.DefaultNodeHome, true),
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

		// authcmd.GetAccountCmd(),
		// rpc.BlockCommand(),
		// rpc.QueryEventForTxCmd(),
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

type appCreator struct{}

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
func (a appCreator) appExport(
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

// wrapExportCommand makes a degraded export visible to the operator and to a
// pipeline, once the genesis has already been written to stdout (or to
// --output-document).
//
// The export command itself must never fail because of the diagnostics sidecar
// (a full disk must not block disaster recovery -- see the invariant in
// app/export.go and app/export_diagnostics.go), so returning an error from here
// is not an option: that would make a perfectly usable genesis look like a
// failed export and could make a caller throw it away. The process exit code is
// therefore the only channel left, and it stays meaningful:
//
//	0  the export succeeded and, if it degraded, the degradations are on disk
//	3  the export succeeded but the degradations could NOT be recorded durably
//	   (app.ExportDiagnosticsDegradedExitCode)
//	1  the export itself failed (cobra's own error path)
//
// On top of the code, a degraded export always prints a standing one-line
// marker on stderr, so a human sees it even without inspecting the exit code.
//
// The command is located by name and its RunE is composed, rather than
// reconstructed, so the SDK keeps owning the export logic. It reports whether
// the command was found, so the caller can refuse to start a CLI whose export
// silently lost the reporting step.
func wrapExportCommand(rootCmd *cobra.Command) bool {
	exportCmd, _, err := rootCmd.Find([]string{"export"})
	if err != nil || exportCmd == nil || exportCmd.Name() != "export" || exportCmd.RunE == nil {
		return false
	}
	exportCmd.RunE = composeExportRunE(exportCmd.RunE)
	return true
}

// keepExportStdoutClean moves the server logger of the export command to
// stderr.
//
// The SDK builds the server logger against `cmd.OutOrStdout()`
// (server/util.go InterceptConfigsPreRunHandler -> CreateSDKLogger), so the
// two startup log lines ("evm chain id", "cosmos pool max tx is non-positive")
// went to stdout, right ahead of the genesis document. The documented backup
// path is `uptickd export > genesis.json`, and the result failed its own
// validate-genesis with `invalid character '\x1b'` -- the ANSI bytes of the
// log lines (finding F-1).
//
// The genesis itself MUST keep flowing through cmd.OutOrStdout()
// (server/export.go copies it there), so the writer on the command is left
// alone and only the server context's logger is rebuilt against stderr, with
// the same level / format / color options the SDK would have used. The app
// inherits its logger from the server context (server/export.go passes
// serverCtx.Logger to the app creator), so this one swap covers every log
// line an export can produce.
func keepExportStdoutClean(cmd *cobra.Command) error {
	if cmd.Name() != "export" {
		return nil
	}
	serverCtx := server.GetServerContextFromCmd(cmd)
	if serverCtx == nil {
		return nil
	}
	logger, err := server.CreateSDKLogger(serverCtx, cmd.ErrOrStderr())
	if err != nil {
		return fmt.Errorf("cannot redirect the export logger to stderr: %w", err)
	}
	// Same decoration the SDK applies to its own logger; without the module
	// key the "module=server" lines would vanish instead of moving to stderr.
	serverCtx.Logger = logger.With(log.ModuleKey, "server")
	return nil
}

// composeExportRunE appends the diagnostics report step to the SDK's export
// RunE. It is separate from wrapExportCommand so the composition can be
// exercised without a real node home and a real database.
func composeExportRunE(inner func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := inner(cmd, args); err != nil {
			return err
		}
		return finishExport(cmd.ErrOrStderr(), app.LastExportDiagnosticsStatus())
	}
}

// finishExport prints the standing marker of a degraded export and, when the
// degradations could not be recorded durably, terminates the process with the
// degraded exit code. The genesis has already been emitted at this point, so
// exiting directly -- rather than returning an error -- is what keeps the
// output usable while still letting a pipeline notice.
func finishExport(stderr io.Writer, status app.ExportDiagnosticsStatus) error {
	if msg := status.CLIMessage(); msg != "" {
		fmt.Fprintln(stderr, msg)
	}
	if code := status.CLIExitCode(); code != 0 {
		os.Exit(code)
	}
	return nil
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
