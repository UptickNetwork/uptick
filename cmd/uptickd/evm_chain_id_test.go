package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	srvflags "github.com/cosmos/evm/server/flags"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// TestApplyEVMChainID* pin the bootstrap precedence of the EIP-155 chain id:
//
//	genesis.json chain_id  >  --chain-id  >  app.toml evm-chain-id  >  cosmos/evm default
//
// The types package covers the arithmetic (TestResolveEVMChainID); what is
// tested here is the glue around it in applyEVMChainID - which source it reads
// first, and that it survives a command that does not own the
// evm.evm-chain-id flag. That last point is a regression from earlier rounds:
// setting a cobra flag the command never registered used to hard-fail every
// non-start command with "no such flag -evm.evm-chain-id". Every case below
// uses a bare command with no such flag, so the old behaviour would fail all
// of them.

// newServerCmd builds a command carrying a server context backed by v, the way
// the real root command does.
//
// The value is installed directly rather than through server.SetCmdServerContext:
// that helper copies into an *already present* context and silently does
// nothing when the command has none, and the reader
// (server.GetServerContextFromCmd) then falls back to a fresh default context
// with an empty viper - so a test built on it would exercise the fallback, not
// the wiring under test.
func newServerCmd(t *testing.T, v *viper.Viper) *cobra.Command {
	t.Helper()

	serverCtx := server.NewContext(v, cmtcfg.DefaultConfig(), log.NewNopLogger())

	cmd := &cobra.Command{}
	cmd.SetContext(context.WithValue(context.Background(), server.ServerContextKey, serverCtx))
	return cmd
}

// writeGenesis drops a genesis.json carrying chainID under home/config.
func writeGenesis(t *testing.T, home, chainID string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(home+"/config", 0o755))
	require.NoError(t, os.WriteFile(
		home+"/config/genesis.json",
		[]byte(`{"chain_id":"`+chainID+`"}`),
		0o644,
	))
}

func TestApplyEVMChainIDPrefersGenesisOverAppToml(t *testing.T) {
	home := t.TempDir()
	writeGenesis(t, home, "uptick_9000-1")

	v := viper.New()
	v.Set(flags.FlagHome, home)
	// A deliberately different, stale value, as app.toml would carry after a
	// chain-id change. Genesis must win, otherwise eth_chainId reports an id
	// no wallet can sign for.
	v.Set(srvflags.EVMChainID, uint64(117))

	cmd := newServerCmd(t, v)
	// A third, also-different value, so the two higher-priority sources are in
	// real conflict. Without this the test would pass on a chain that reads
	// --chain-id first and only falls back to genesis.
	cmd.Flags().String(flags.FlagChainID, "", "")
	require.NoError(t, cmd.Flags().Set(flags.FlagChainID, "uptick_1170-3"))

	require.NoError(t, applyEVMChainID(cmd))

	require.Equal(t, uint64(9000), v.GetUint64(srvflags.EVMChainID),
		"the genesis chain_id must win over both --chain-id and the app.toml evm-chain-id")
}

func TestApplyEVMChainIDFallsBackToChainIDFlag(t *testing.T) {
	home := t.TempDir() // no genesis.json

	v := viper.New()
	v.Set(flags.FlagHome, home)
	v.Set(srvflags.EVMChainID, evmtypes.DefaultEVMChainID)

	cmd := newServerCmd(t, v)
	cmd.Flags().String(flags.FlagChainID, "", "")
	require.NoError(t, cmd.Flags().Set(flags.FlagChainID, "uptick_1170-3"))

	require.NoError(t, applyEVMChainID(cmd))

	require.Equal(t, uint64(1170), v.GetUint64(srvflags.EVMChainID),
		"with no genesis to read, --chain-id is the next source")
}

func TestApplyEVMChainIDPassesThroughAppTomlValue(t *testing.T) {
	home := t.TempDir() // no genesis.json, no --chain-id

	v := viper.New()
	v.Set(flags.FlagHome, home)
	v.Set(srvflags.EVMChainID, uint64(117))

	require.NoError(t, applyEVMChainID(newServerCmd(t, v)))

	require.Equal(t, uint64(117), v.GetUint64(srvflags.EVMChainID),
		"an operator-set app.toml value is the last source before the default")
}

func TestApplyEVMChainIDSyncsRegisteredFlag(t *testing.T) {
	home := t.TempDir()
	writeGenesis(t, home, "uptick_9000-1")

	v := viper.New()
	v.Set(flags.FlagHome, home)

	// The start command registers the flag; the JSON-RPC server reads viper,
	// but a later flag lookup must not see the pre-bootstrap value.
	cmd := newServerCmd(t, v)
	cmd.Flags().Uint64(srvflags.EVMChainID, evmtypes.DefaultEVMChainID, "")

	require.NoError(t, applyEVMChainID(cmd))

	got, err := cmd.Flags().GetUint64(srvflags.EVMChainID)
	require.NoError(t, err)
	require.Equal(t, uint64(9000), got, "a registered flag is synced with the resolved id")
	require.Equal(t, uint64(9000), v.GetUint64(srvflags.EVMChainID))
}

// writeAppTomlEVMChainID rewrites the app.toml entry in place, because the JSON
// RPC server reads app.toml's evm-chain-id rather than the keeper's
// process-wide ChainConfig. The failure it used to have was silence:
// regexp.ReplaceAll reports success whether or not it matched, so an app.toml
// whose entry was commented out, quoted or missing was written back unchanged
// and `init` reported success while eth_chainId kept answering the old id.
func TestWriteAppTomlEVMChainIDRewritesTheLine(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(home+"/config", 0o755))
	require.NoError(t, os.WriteFile(home+"/config/app.toml", []byte(
		"[evm]\n# evm-chain-id = 262144\nevm-chain-id = 262144\n"), 0o600))

	require.NoError(t, writeAppTomlEVMChainID(home, 117))

	got, err := os.ReadFile(home + "/config/app.toml")
	require.NoError(t, err)
	require.Contains(t, string(got), "evm-chain-id = 117")
	require.Contains(t, string(got), "# evm-chain-id = 262144",
		"the commented example is documentation and must stay commented")
}

func TestWriteAppTomlEVMChainIDFailsWhenThereIsNothingToRewrite(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"absent", "[evm]\n"},
		{"commented out", "[evm]\n# evm-chain-id = 262144\n"},
		{"quoted value", "[evm]\nevm-chain-id = \"262144\"\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			require.NoError(t, os.MkdirAll(home+"/config", 0o755))
			path := home + "/config/app.toml"
			require.NoError(t, os.WriteFile(path, []byte(tc.body), 0o600))

			err := writeAppTomlEVMChainID(home, 117)
			require.Error(t, err, "doing nothing silently is the bug this guards")
			require.Contains(t, err.Error(), "evm-chain-id")

			after, rerr := os.ReadFile(path)
			require.NoError(t, rerr)
			require.Equal(t, tc.body, string(after), "a rejected write must leave the file untouched")
		})
	}
}
