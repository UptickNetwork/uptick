package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtcli "github.com/cometbft/cometbft/libs/cli"
	flags "github.com/cosmos/cosmos-sdk/client/flags"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// newCLIRoot builds the command tree the way a user actually gets it, which is
// not what NewRootCmd returns on its own: main() hands the root to
// svrcmd.Execute, which adds the three logging flags, and to CometBFT's
// PrepareBaseCmd, which adds --home / --trace and prepends the viper pre-run
// step. Both matter here -- --home is how a test points the CLI at a poisoned
// config.toml, and the logging flags are *legitimate* config.toml collisions,
// so a guard that cannot see them would be checking an unrealistically small
// surface.
func newCLIRoot(t *testing.T, home string) *cobra.Command {
	t.Helper()

	root := NewRootCmd()

	// Mirrors svrcmd.Execute (cosmos-sdk server/cmd/execute.go:28-31), which
	// adds exactly these three and offers no way to add them separately.
	root.PersistentFlags().String(flags.FlagLogLevel, "info", "The logging level")
	root.PersistentFlags().String(flags.FlagLogFormat, "plain", "The logging format (json|plain)")
	root.PersistentFlags().Bool(flags.FlagLogNoColor, false, "Disable colored logs")

	// Adds --home / --trace and chains bindFlagsLoadViper into PersistentPreRunE.
	// The returned Executor is discarded on purpose: its Execute() calls
	// os.Exit, and a test needs the error instead.
	_ = cmtcli.PrepareBaseCmd(root, "uptickd", home)

	return root
}

// nodeConfigTopLevelKeys returns every name a node's configuration contributes
// to the single *flat* viper namespace the SDK's pre-run hook reads
// (cosmos-sdk v0.53 server/util.go:86-94). That namespace is the union of
// config.toml and app.toml, and it collides with a cobra flag whenever a
// top-level key of either file has the same name.
//
// Both files are produced exactly the way server.interceptConfigs produces them
// for a fresh home, so the list cannot drift from what a node actually carries.
func nodeConfigTopLevelKeys(t *testing.T) map[string]bool {
	t.Helper()

	dir := t.TempDir()

	// config.toml is written from initTendermintConfig(); app.toml from the
	// initAppConfig template. Both are the real sources, not copies of one.
	cmtPath := filepath.Join(dir, "config.toml")
	cmtcfg.WriteConfigFile(cmtPath, initTendermintConfig())

	appPath := filepath.Join(dir, "app.toml")
	template, appCfg := initAppConfig()
	serverconfig.SetConfigTemplate(template)
	serverconfig.WriteConfigFile(appPath, appCfg)

	keys := map[string]bool{}
	for _, path := range []string{cmtPath, appPath} {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)

		var top map[string]any
		require.NoErrorf(t, toml.Unmarshal(raw, &top), "%s is not valid TOML", path)
		for k := range top {
			keys[k] = true
		}
	}

	require.NotEmpty(t, keys, "no configuration keys were parsed; the guard would pass vacuously")
	return keys
}

// configOwnedFlagNames returns the flag names the SDK is *supposed* to seed from
// config.toml / app.toml.
//
// That set is the config file's own surface: the flags `start` and `prune`
// declare, plus the root's global flags (--log_level, --log_format,
// --log_no_color). Those commands read their values back out of viper, so
// seeding them is the intended behaviour, not a leak. A flag declared by a
// *module* subcommand is never in this set -- which is what makes the assertion
// below meaningful.
func configOwnedFlagNames(t *testing.T, root *cobra.Command) map[string]bool {
	t.Helper()

	owned := map[string]bool{}

	collect := func(cmd *cobra.Command) {
		require.NotNil(t, cmd)
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) { owned[f.Name] = true })
	}

	// The root's own (global) flags.
	collect(root)

	for _, name := range []string{"start", "prune"} {
		var found *cobra.Command
		for _, sub := range root.Commands() {
			if sub.Name() == name {
				found = sub
				break
			}
		}
		require.NotNilf(t, found, "the %q command must exist: it is part of the config file's flag surface", name)
		collect(found)
	}

	require.NotEmpty(t, owned)
	return owned
}

// TestModuleFlagsDoNotInheritNodeConfig is the structural half of the guard for
// flagsShieldedFromConfig.
//
// server.bindFlags copies config.toml / app.toml into every flag the user left
// alone, keyed by name, so any module flag that shares a name with a top-level
// config key silently runs on the node's value instead of its own. That defect
// reached a testnet once: `tx interchain-accounts controller register
// --version` inherited config.toml's `version = "0.38.19"` and every
// registration failed with "cannot unmarshal ICS-27 interchain accounts
// metadata" -- a symptom that points at ICA rather than at the flag.
//
// The invariant: a flag declared outside the server's own command surface must
// either be shielded, or not collide at all. It is checked against the *real*
// command tree and the *real* config templates, so a new collision cannot slip
// in through an SDK or ibc-go bump without failing here.
func TestModuleFlagsDoNotInheritNodeConfig(t *testing.T) {
	configKeys := nodeConfigTopLevelKeys(t)
	root := newCLIRoot(t, t.TempDir())
	configOwned := configOwnedFlagNames(t, root)

	shielded := map[string]bool{}
	for _, name := range flagsShieldedFromConfig {
		shielded[name] = true
	}

	var leaks []string
	collidingPaths := map[string][]string{}

	var walk func(cmd *cobra.Command, path string)
	walk = func(cmd *cobra.Command, path string) {
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if !configKeys[f.Name] {
				return
			}
			collidingPaths[f.Name] = append(collidingPaths[f.Name], path)

			if configOwned[f.Name] || shielded[f.Name] {
				return
			}
			leaks = append(leaks, fmt.Sprintf("%s --%s", path, f.Name))
		})

		for _, sub := range cmd.Commands() {
			walk(sub, path+" "+sub.Name())
		}
	}
	walk(root, root.Name())

	sort.Strings(leaks)
	require.Emptyf(t, leaks,
		"these flags share a name with a top-level config.toml / app.toml key, so the SDK "+
			"overwrites them from the node's config file unless the user passes them explicitly "+
			"(server.bindFlags). Shield them via flagsShieldedFromConfig, or prove they are part "+
			"of the server's own config surface:\n  %s", joinLines(leaks))

	// A shielded name that no longer collides is dead weight that hides the next
	// real collision behind a stale entry.
	for _, name := range flagsShieldedFromConfig {
		require.NotEmptyf(t, collidingPaths[name],
			"flagsShieldedFromConfig lists %q, but no flag of that name collides with a "+
				"top-level config key any more; drop the entry", name)
	}
}

func joinLines(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += "\n  "
		}
		out += x
	}
	return out
}

// captureStdIO runs fn with the process's stdout and stderr redirected, and
// returns what each received.
//
// The generated transaction reaches the process stdout rather than cobra's
// output writer: client.Context.PrintBytes falls back to os.Stdout when the
// context carries no writer, and nothing in the SDK binds one for a
// --generate-only tx. Redirecting cobra alone would capture nothing at all,
// which is exactly how this test first failed.
func captureStdIO(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	origStdout, origStderr := os.Stdout, os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	stderrFile, err := os.CreateTemp(t.TempDir(), "stderr")
	require.NoError(t, err)

	os.Stdout, os.Stderr = stdoutW, stderrFile
	defer func() { os.Stdout, os.Stderr = origStdout, origStderr }()

	fn()

	// Close before reading: a full pipe buffer would otherwise block the read.
	require.NoError(t, stdoutW.Close())
	out, err := io.ReadAll(stdoutR)
	require.NoError(t, err)
	require.NoError(t, stdoutR.Close())

	require.NoError(t, stderrFile.Close())
	errBytes, err := os.ReadFile(stderrFile.Name())
	require.NoError(t, err)

	return string(out), string(errBytes)
}

// TestICARegisterFlagIsNotSeededFromCometVersion is the behavioural half: it
// runs the real command tree against a home whose config.toml carries a
// CometBFT `version`, and reads Msg.version back out of the generated tx.
//
// This is the observable contract the structural test above protects. Under the
// defect, an unmentioned --version produced "9.9.9" (whatever the config file
// said) and the registration was rejected on chain.
func TestICARegisterFlagIsNotSeededFromCometVersion(t *testing.T) {
	const poisoned = "9.9.9"

	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(home, "config", "config.toml"),
		[]byte("version = \""+poisoned+"\"\n"),
		0o600,
	))

	// The address only has to parse under whatever prefix the global SDK config
	// carries in this process; --generate-only never touches the keyring.
	from, err := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()).
		BytesToString(bytes.Repeat([]byte{7}, 20))
	require.NoError(t, err)

	msgVersion := func(t *testing.T, extra ...string) string {
		t.Helper()

		// A fresh tree per run: pflag keeps `Changed` across Execute calls, so a
		// reused command would report the previous run's flags as user-supplied.
		cmd := newCLIRoot(t, home)

		args := []string{
			"tx", "interchain-accounts", "controller", "register", "connection-0",
			"--from", from,
			"--keyring-backend", "test",
			"--keyring-dir", filepath.Join(home, "keyring"),
			"--generate-only", "--offline",
			"--account-number", "0", "--sequence", "0",
			"-o", "json",
		}
		cmd.SetArgs(append(args, extra...))

		stdout, stderr := captureStdIO(t, func() {
			require.NoError(t, cmd.Execute())
		})

		var tx struct {
			Body struct {
				Messages []struct {
					Version string `json:"version"`
				} `json:"messages"`
			} `json:"body"`
		}
		require.NoErrorf(t, json.Unmarshal([]byte(stdout), &tx),
			"stdout was empty or not JSON.\nstdout: %q\nstderr: %q", stdout, stderr)
		require.Len(t, tx.Body.Messages, 1)
		return tx.Body.Messages[0].Version
	}

	require.Equalf(t, "", msgVersion(t),
		"Msg.version inherited the node's config.toml: the caller never passed --version, "+
			"but the registration would be sent with %q and rejected by the controller as "+
			"\"cannot unmarshal ICS-27 interchain accounts metadata\"", poisoned)

	require.Equal(t, "channel-1", msgVersion(t, "--version=channel-1"),
		"an explicitly passed --version must survive the pre-run hook untouched")
}
