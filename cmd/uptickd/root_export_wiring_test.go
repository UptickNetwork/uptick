package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// wrapExportCommand is the last line of defence for a degraded export whose
// diagnostics report could not be written. Everything upstream of it is
// deliberately tolerant -- app/export.go and app/export_diagnostics.go never
// fail an export over the sidecar (a full disk must not block disaster
// recovery) -- so the process exit code this wrapper installs
// (app.ExportDiagnosticsDegradedExitCode) is the only channel left that a
// pipeline can gate on.
//
// If wrapExportCommand quietly stops returning true, that channel disappears and
// the CLI is back to the pre-fix behaviour: a degraded export that could not
// record its degradations looks exactly like a clean one to every consumer that
// does not read the node log. The failure is invisible at run time and in every
// other test, which is why the true case is asserted here against the REAL
// command tree instead of being trusted.
//
// The wrapper's absence also makes NewRootCmd panic (root.go:149-151). That
// panic is not unguarded -- TestInitCmd and the precheck tests already call
// NewRootCmd, so CI would catch it -- but it surfaces there under a name that
// sends the reader towards init/testnet instead of towards the export wiring.
// This file is where the defect is named properly.
//
// NOTE: wrapExportCommand is NOT idempotent. It wraps whatever RunE the command
// currently carries, so a second call nests composeExportRunE inside itself and
// the export prints its marker twice (the exit code is then derived from the
// same published status twice, which is harmless but pointless). NewRootCmd
// calls it exactly once, on a freshly built tree, and that is the only supported
// usage -- do not reuse this as a generic "make sure it is wired" helper.
func TestWrapExportCommandRejectsARootWithoutAnExportCommand(t *testing.T) {
	// cobra's Find falls back to returning the command itself when the argument
	// names no child, so it is the NAME guard that refuses this one -- and it has
	// to, or the root's own RunE would be wrapped.
	require.False(t, wrapExportCommand(&cobra.Command{Use: "uptickd"}),
		"a root without an export child must not be reported as wrapped")

	// A child that is named "export" but carries no RunE would make
	// composeExportRunE wrap nil, so the RunE guard is the clause under test.
	exportWithoutRunE := &cobra.Command{Use: "uptickd"}
	exportWithoutRunE.AddCommand(&cobra.Command{Use: "export"})
	require.False(t, wrapExportCommand(exportWithoutRunE),
		"an export command with no RunE must not be reported as wrapped")
}

func TestWrapExportCommandFindsTheRealExportCommand(t *testing.T) {
	var root *cobra.Command
	require.NotPanics(t, func() { root = NewRootCmd() },
		"NewRootCmd must build a root: it panics if wrapExportCommand cannot find the export command")

	require.True(t, wrapExportCommand(root),
		"the SDK export command must be found and wrapped, so the degraded-export "+
			"exit code is installed; this pins all three conditions (the command is "+
			"named \"export\", its RunE is set, and Find reaches it) against the real "+
			"tree, which no other test does")
}
