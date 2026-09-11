package main

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/crypto/hd"
)

// The tests below drive readInitArgs / readStartArgs against the real commands.
// That is the point of extracting them from the RunE closures: it turns "every
// flag this command reads is actually registered on it" into an executable
// assertion. Under the previous `args.x, _ = cmd.Flags().GetString(...)` form
// the same mistake produced an empty value and no signal at all.

// TestReadInitArgsSeesEveryRegisteredFlag asserts both halves of the contract:
// no lookup fails (every name is registered) and the values that arrive are the
// registered defaults, not zero values.
func TestReadInitArgsSeesEveryRegisteredFlag(t *testing.T) {
	cmd := testnetInitFilesCmd(nil, banktypes.GenesisBalancesIterator{})

	args, err := readInitArgs(cmd)
	require.NoError(t, err, "every flag read by readInitArgs must be registered on the command")

	// Defaults, one per flag type (string / int), so a silently-zeroed read for
	// any of them fails here rather than in a generated genesis file.
	require.Equal(t, "./.testnets", args.outputDir)
	require.Equal(t, "node", args.nodeDirPrefix)
	require.Equal(t, "uptickd", args.nodeDaemonHome)
	require.Equal(t, "192.168.0.1", args.startingIPAddress)
	require.Equal(t, 4, args.numValidators)
	require.Equal(t, string(hd.EthSecp256k1Type), args.algo)
	require.Equal(t, flags.DefaultKeyringBackend, args.keyringBackend)
	require.NotEmpty(t, args.minGasPrices, "min-gas-prices has a non-empty default (denom-aware)")

	// --chain-id is the one flag with an intentionally empty default (the value
	// is generated per testnet), so an empty read is legitimate here. Assert it
	// explicitly so a future reader does not "fix" this into a non-empty check.
	require.Empty(t, args.chainID)
}

// TestReadStartArgsSeesEveryRegisteredFlag is the `testnet start` counterpart.
func TestReadStartArgsSeesEveryRegisteredFlag(t *testing.T) {
	cmd := testnetStartCmd()

	args, err := readStartArgs(cmd)
	require.NoError(t, err, "every flag read by readStartArgs must be registered on the command")

	require.Equal(t, "./.testnets", args.outputDir)
	require.Equal(t, 1, args.numValidators)
	require.Equal(t, string(hd.EthSecp256k1Type), args.algo)
	require.Equal(t, "tcp://0.0.0.0:26657", args.rpcAddress)
	require.Equal(t, "tcp://0.0.0.0:1317", args.apiAddress)
	// Bound to config.Default* rather than literals, so a bump to the upstream
	// default does not need a matching edit here.
	require.NotEmpty(t, args.grpcAddress)
	require.NotEmpty(t, args.jsonrpcAddress)
	require.False(t, args.enableLogging)
	require.False(t, args.printMnemonic)
}

// TestAssignFlagSurfacesUnregisteredFlag is the behavioural core of the fix.
// Reading a name that was never registered must return an error carrying that
// name, so a renamed or deleted registration fails loudly at the call site
// instead of travelling on as a zero value.
func TestAssignFlagSurfacesUnregisteredFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "probe"}

	const missing = "definitely-not-registered"

	var s string
	err := assignStringFlag(&s, cmd, missing)
	require.Error(t, err)
	require.Contains(t, err.Error(), missing, "the error must name the offending flag")

	var i int
	require.Error(t, assignIntFlag(&i, cmd, missing))

	var b bool
	require.Error(t, assignBoolFlag(&b, cmd, missing))
}

// TestAssignFlagReadsRegisteredFlag is the counter-sentinel: the helpers must
// actually copy values through, not merely fail on everything. It also pins
// that a flag changed after registration is picked up.
func TestAssignFlagReadsRegisteredFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "probe"}
	cmd.Flags().String("name", "default", "")
	cmd.Flags().Int("count", 7, "")
	cmd.Flags().Bool("on", true, "")

	var s string
	require.NoError(t, assignStringFlag(&s, cmd, "name"))
	require.Equal(t, "default", s)

	require.NoError(t, cmd.Flags().Set("name", "changed"))
	require.NoError(t, assignStringFlag(&s, cmd, "name"))
	require.Equal(t, "changed", s)

	var i int
	require.NoError(t, assignIntFlag(&i, cmd, "count"))
	require.Equal(t, 7, i)

	var b bool
	require.NoError(t, assignBoolFlag(&b, cmd, "on"))
	require.True(t, b)
}

// TestTestnetDoesNotDiscardFlagErrors is a static guard so the `, _ =` form
// cannot come back for a subset of the flags (which would be easy to miss, since
// the extracted readers would keep the rest honest). The production file does
// not contain this literal anywhere, comments included.
func TestTestnetDoesNotDiscardFlagErrors(t *testing.T) {
	src, err := os.ReadFile("testnet.go")
	require.NoError(t, err)

	body := string(src)
	require.NotContains(t, body, ", _ = cmd.Flags()")
	require.NotContains(t, body, "_, _ = cmd.Flags()")

	// Indirect guard: the reader functions must remain the only way the two
	// commands obtain their arguments, so the count of literal flag lookups in
	// the file stays confined to the assign* helpers.
	require.Equal(t, 3, strings.Count(body, "cmd.Flags().Get"),
		"flag lookups must live only in the three assign* helpers")
}
