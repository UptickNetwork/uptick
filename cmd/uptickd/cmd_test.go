package main_test

import (
	"fmt"
	uptickd "github.com/UptickNetwork/uptick/cmd/uptickd"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
)

func TestInitCmd(t *testing.T) {
	rootCmd := uptickd.NewRootCmd()
	names := make([]string, 0, len(rootCmd.Commands()))
	for _, c := range rootCmd.Commands() {
		names = append(names, c.Name())
	}
	require.Contains(t, names, "testnet")

	homeDir := t.TempDir()
	rootCmd.SetArgs([]string{
		"init",        // Test the init cmd
		"uptick-test", // Moniker
		fmt.Sprintf("--%s=%s", cli.FlagOverwrite, "true"), // Overwrite genesis.json, in case it already exists
		fmt.Sprintf("--%s=%s", flags.FlagChainID, "uptick_1170-1"),
		fmt.Sprintf("--%s=%s", flags.FlagHome, homeDir),
	})

	err := svrcmd.Execute(rootCmd, "uptick", homeDir)
	require.NoError(t, err)
}
