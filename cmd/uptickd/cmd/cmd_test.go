package cmd_test

import (
	"fmt"
	uptickd "github.com/UptickNetwork/uptick/cmd/uptickd/cmd"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	"github.com/UptickNetwork/uptick/app"
)

func TestInitCmd(t *testing.T) {
	rootCmd := uptickd.NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",        // Test the init cmd
		"uptick-test", // Moniker
		fmt.Sprintf("--%s=%s", cli.FlagOverwrite, "true"), // Overwrite genesis.json, in case it already exists
		fmt.Sprintf("--%s=%s", flags.FlagChainID, "uptick_7777-1"),
	})

	err := svrcmd.Execute(rootCmd, "uptick", app.DefaultNodeHome)
	require.NoError(t, err)
}
