package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/UptickNetwork/uptick/app"
	cmdcfg "github.com/UptickNetwork/uptick/cmd/config"
	upticktypes "github.com/UptickNetwork/uptick/types"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/go-bip39"
)

type printInfo struct {
	Moniker    string          `json:"moniker" yaml:"moniker"`
	ChainID    string          `json:"chain_id" yaml:"chain_id"`
	NodeID     string          `json:"node_id" yaml:"node_id"`
	GenTxsDir  string          `json:"gentxs_dir" yaml:"gentxs_dir"`
	AppMessage json.RawMessage `json:"app_message" yaml:"app_message"`
}

func newPrintInfo(moniker, chainID, nodeID, genTxsDir string, appMessage json.RawMessage) printInfo {
	return printInfo{
		Moniker:    moniker,
		ChainID:    chainID,
		NodeID:     nodeID,
		GenTxsDir:  genTxsDir,
		AppMessage: appMessage,
	}
}

func displayInfo(info printInfo) error {
	out, err := json.MarshalIndent(info, "", " ")
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(os.Stderr, "%s\n", string(sdk.MustSortJSON(out))); err != nil {
		return err
	}

	return nil
}

// InitCmd returns a command that initializes all files needed for Tendermint
// and the respective application.
func InitCmd(mbm module.BasicManager, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize private validator, p2p, genesis, and application configuration files",
		Long:  `Initialize validators's and node's configuration files.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			config.P2P.MaxNumInboundPeers = 100
			config.P2P.MaxNumOutboundPeers = 30
			config.Mempool.Size = 10000
			config.StateSync.TrustPeriod = 112 * time.Hour

			config.SetRoot(clientCtx.HomeDir)

			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			switch {
			case chainID != "":
			case clientCtx.ChainID != "":
				chainID = clientCtx.ChainID
			default:
				chainID = MainnetChainID
			}

			// Get bip39 mnemonic
			var mnemonic string

			recoverKey, _ := cmd.Flags().GetBool(genutilcli.FlagRecover)
			if recoverKey {
				inBuf := bufio.NewReader(cmd.InOrStdin())
				value, err := input.GetString("Enter your bip39 mnemonic", inBuf)
				if err != nil {
					return err
				}

				mnemonic = value
				if !bip39.IsMnemonicValid(mnemonic) {
					return errors.New("invalid mnemonic")
				}
			}

			nodeID, _, err := genutil.InitializeNodeValidatorFilesFromMnemonic(config, mnemonic)
			if err != nil {
				return err
			}

			config.Moniker = args[0]

			genFile := config.GenesisFile()
			overwrite, _ := cmd.Flags().GetBool(genutilcli.FlagOverwrite)
			defaultDenom, _ := cmd.Flags().GetString(genutilcli.FlagDefaultBondDenom)

			// use os.Stat to check if the file exists
			_, statErr := os.Stat(genFile)
			if !overwrite && !os.IsNotExist(statErr) {
				return fmt.Errorf("genesis.json file already exists: %v", genFile)
			}

			// Overwrites the SDK default denom for side-effects
			if defaultDenom != "" {
				sdk.DefaultBondDenom = defaultDenom
			}

			genesis := mbm.DefaultGenesis(cdc)
			app.CustomizeDefaultGenesis(cdc, genesis)
			appState, err := json.MarshalIndent(genesis, "", " ")
			if err != nil {
				return errors.Wrap(err, "Failed to marshal default genesis state")
			}

			appGenesis := &types.AppGenesis{}
			if statErr == nil {
				// Reuse the stat result above: a stat error here means the file is
				// missing/unreadable, so start from a fresh AppGenesis.
				appGenesis, err = types.AppGenesisFromFile(genFile)
				if err != nil {
					return errors.Wrap(err, "Failed to read genesis doc from file")
				}
			} else if !os.IsNotExist(statErr) {
				return statErr
			}

			// Get initial height
			initHeight, _ := cmd.Flags().GetInt64(flags.FlagInitHeight)

			appGenesis.AppName = version.AppName
			// appGenesis.AppVersion = version.Version
			appGenesis.ChainID = chainID
			appGenesis.AppState = appState
			appGenesis.InitialHeight = initHeight
			appGenesis.Consensus = &types.ConsensusGenesis{
				Validators: nil,
			}

			if err := genutil.ExportGenesisFile(appGenesis, genFile); err != nil {
				return errors.Wrap(err, "Failed to export genesis file")
			}

			if evmID, err := upticktypes.ParseEIP155ChainID(chainID); err == nil {
				if err := writeAppTomlEVMChainID(clientCtx.HomeDir, evmID); err != nil {
					return errors.Wrap(err, "failed to write evm-chain-id")
				}
			}

			toPrint := newPrintInfo(config.Moniker, chainID, nodeID, "", appState)

			cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)
			return displayInfo(toPrint)
		},
	}

	cmd.Flags().String(cli.HomeFlag, defaultNodeHome, "node's home directory")
	cmd.Flags().BoolP(genutilcli.FlagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().Bool(genutilcli.FlagRecover, false, "provide seed phrase to recover existing key instead of creating")
	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, defaults to the mainnet chain id if left blank")
	cmd.Flags().String(genutilcli.FlagDefaultBondDenom, cmdcfg.BaseDenom, "defines the default denom to use in genesis file")

	return cmd
}
