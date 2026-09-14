package main

import (
	"bytes"
	"context"
	"testing"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Guard for finding F-1: `uptickd export` used to emit its server log lines
// ("evm chain id", "cosmos pool max tx is non-positive") on STDOUT, ahead of
// the genesis document, so `export > genesis.json` produced a file that failed
// its own validate-genesis with `invalid character '\x1b'`.
//
// keepExportStdoutClean must re-point the server context logger at stderr for
// the export command only, and must leave the command's own writer alone (the
// genesis itself flows through cmd.OutOrStdout() in server/export.go).
//
// Reverse control: run with an overlay that disables the redirect (see the
// batch notes) -- the first test here then fails, because the log line lands
// on stdout again.

// newExportRoot builds a minimal root/export command pair with a server
// context, as PersistentPreRunE would have set up.
func newExportRoot(t *testing.T, use string) (*cobra.Command, *server.Context) {
	t.Helper()
	rootCmd := &cobra.Command{Use: "uptickd"}
	sub := &cobra.Command{Use: use, RunE: func(*cobra.Command, []string) error { return nil }}
	rootCmd.AddCommand(sub)

	serverCtx := server.NewDefaultContext()
	// Mirror svrcmd.Execute, which puts the server context under
	// server.ServerContextKey before PersistentPreRunE runs: without it
	// GetServerContextFromCmd hands back a fresh context instead of this one.
	sub.SetContext(context.WithValue(context.Background(), server.ServerContextKey, serverCtx))
	require.NoError(t, server.SetCmdServerContext(sub, serverCtx))
	return sub, serverCtx
}

func TestKeepExportStdoutCleanSendsServerLogsToStderr(t *testing.T) {
	exportCmd, serverCtx := newExportRoot(t, "export")

	var outBuf, errBuf bytes.Buffer
	exportCmd.SetOut(&outBuf)
	exportCmd.SetErr(&errBuf)
	// Plain output so the assertions read the log line verbatim; this also
	// exercises that the rebuilt logger honours --log-no-color.
	serverCtx.Viper.Set(flags.FlagLogNoColor, true)
	// Simulate the SDK: the logger aims at the command's stdout.
	serverCtx.Logger = log.NewLogger(&outBuf)

	require.NoError(t, keepExportStdoutClean(exportCmd))
	serverCtx.Logger.Info("evm chain id test line")

	require.Empty(t, outBuf.String(),
		"the export command's server logger must not write to stdout: export > genesis.json would capture log bytes")
	require.Contains(t, errBuf.String(), "evm chain id test line")
	require.Contains(t, errBuf.String(), "module=server",
		"the module=server decoration must survive the redirect, not vanish")

	// The genesis writer must be untouched: server/export.go copies the genesis
	// through cmd.OutOrStdout().
	require.Equal(t, &outBuf, exportCmd.OutOrStdout(),
		"keepExportStdoutClean must not re-point the command writer that carries the genesis")
}

func TestKeepExportStdoutCleanLeavesOtherCommandsAlone(t *testing.T) {
	startCmd, serverCtx := newExportRoot(t, "start")

	var outBuf, errBuf bytes.Buffer
	startCmd.SetOut(&outBuf)
	startCmd.SetErr(&errBuf)
	serverCtx.Logger = log.NewLogger(&outBuf)

	require.NoError(t, keepExportStdoutClean(startCmd))
	serverCtx.Logger.Info("node start test line")

	require.Contains(t, outBuf.String(), "node start test line",
		"non-export commands keep logging to stdout")
	require.Empty(t, errBuf.String())
}

func TestKeepExportStdoutCleanRespectsLogLevelFlags(t *testing.T) {
	exportCmd, serverCtx := newExportRoot(t, "export")

	var outBuf, errBuf bytes.Buffer
	exportCmd.SetOut(&outBuf)
	exportCmd.SetErr(&errBuf)
	serverCtx.Viper.Set(flags.FlagLogLevel, "error")
	serverCtx.Logger = log.NewLogger(&outBuf)

	require.NoError(t, keepExportStdoutClean(exportCmd))
	serverCtx.Logger.Info("suppressed info line")
	serverCtx.Logger.Error("kept error line")

	require.NotContains(t, errBuf.String(), "suppressed info line",
		"the redirected logger must honour --log-level")
	require.Contains(t, errBuf.String(), "kept error line")
}
