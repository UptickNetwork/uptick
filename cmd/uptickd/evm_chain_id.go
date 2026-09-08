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
func applyEVMChainID(cmd *cobra.Command) error {
	serverCtx := server.GetServerContextFromCmd(cmd)
	if serverCtx == nil || serverCtx.Viper == nil {
		return nil
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
		return nil
	}
	serverCtx.Viper.Set(srvflags.EVMChainID, evmID)
	// The evm.evm-chain-id flag is registered by the EVM server command family
	// (start) only. Other commands (init, status, query, tx, keys, ...) do not
	// own it: writing viper alone is enough for the JSON-RPC server, and
	// attempting to set a missing cobra flag here used to hard-fail every
	// non-start command with "no such flag -evm.evm-chain-id".
	if f := cmd.Flags().Lookup(srvflags.EVMChainID); f != nil {
		if err := cmd.Flags().Set(srvflags.EVMChainID, strconv.FormatUint(evmID, 10)); err != nil {
			return err
		}
	}
	return nil
}

func writeAppTomlEVMChainID(home string, id uint64) error {
	path := filepath.Join(home, "config", "app.toml")
	bz, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	next := evmChainIDLine.ReplaceAll(bz, []byte(fmt.Sprintf("evm-chain-id = %d", id)))
	//nolint:gosec // G703: path is derived from the operator's own --home flag on a local CLI, not untrusted input
	return os.WriteFile(path, next, 0o600)
}
