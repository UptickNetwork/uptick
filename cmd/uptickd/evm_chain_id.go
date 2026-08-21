package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	upticktypes "github.com/UptickNetwork/uptick/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	srvflags "github.com/cosmos/evm/server/flags"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/spf13/cobra"
)

var evmChainIDLine = regexp.MustCompile(`(?m)^evm-chain-id\s*=\s*\d+`)

// applyEVMChainID overlays viper's evm.evm-chain-id with the EIP-155 id
// parsed from genesis / --chain-id. JSON-RPC eth_chainId reads this config,
// not the keeper's process-wide ChainConfig.
func applyEVMChainID(cmd *cobra.Command) {
	serverCtx := server.GetServerContextFromCmd(cmd)
	if serverCtx == nil || serverCtx.Viper == nil {
		return
	}
	home := serverCtx.Viper.GetString(flags.FlagHome)
	if home == "" {
		home, _ = cmd.Flags().GetString(flags.FlagHome)
	}
	chainID := upticktypes.ChainIDFromGenesisFile(home)
	if chainID == "" {
		chainID, _ = cmd.Flags().GetString(flags.FlagChainID)
	}
	evmID := upticktypes.ResolveEVMChainID(serverCtx.Viper, chainID)
	if evmID == 0 || evmID == evmtypes.DefaultEVMChainID {
		return
	}
	serverCtx.Viper.Set(srvflags.EVMChainID, evmID)
	_ = cmd.Flags().Set(srvflags.EVMChainID, strconv.FormatUint(evmID, 10))
}

func writeAppTomlEVMChainID(home string, id uint64) error {
	path := filepath.Join(home, "config", "app.toml")
	bz, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	next := evmChainIDLine.ReplaceAll(bz, []byte(fmt.Sprintf("evm-chain-id = %d", id)))
	return os.WriteFile(path, next, 0o644)
}
