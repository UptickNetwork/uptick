package main_test

import (
	"bytes"
	"testing"

	uptickd "github.com/UptickNetwork/uptick/cmd/uptickd"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/stretchr/testify/require"
)

// The two tests here cover the half of `precheck-collection-migration` that the
// existing precheck_collection_migration_test.go cannot see: the cobra command
// itself. That file exercises only runCollectionPrecheck, the pure core, so the
// registration in root.go and the RunE body -- the code that opens the
// application database, builds the app and resolves the collection store key --
// had zero coverage. A file existing, a command being registered and the pure
// core being green is exactly the "looks tested, never runs" shape this
// repository keeps finding.

// TestPrecheckCollectionMigrationCmdIsRegistered pins that the offline precheck
// is actually wired into the root command. Without this a deleted AddCommand
// line would remove the operator-facing entry point while every other test
// stayed green.
func TestPrecheckCollectionMigrationCmdIsRegistered(t *testing.T) {
	rootCmd := uptickd.NewRootCmd()

	names := make([]string, 0, len(rootCmd.Commands()))
	for _, c := range rootCmd.Commands() {
		names = append(names, c.Name())
	}
	require.Contains(t, names, "precheck-collection-migration")
}

// TestPrecheckCollectionMigrationCmdRunsCleanOnFreshHome executes the command
// end to end against a fresh temporary home and pins the outcome observed on an
// empty store: RunE opens <home>/data, builds the app with loadLatest=true,
// scans the (empty) legacy collection store and exits 0 having printed the
// clean confirmation.
//
// The assertion is on the printed confirmation, not only on the returned error:
// a RunE that early-returns without opening the database would also return nil,
// but it would print nothing. Asserting the side effect is what makes the W3
// mutation self-check fail (see the report) and keeps this test from being
// satisfied by a stub.
func TestPrecheckCollectionMigrationCmdRunsCleanOnFreshHome(t *testing.T) {
	homeDir := t.TempDir()

	rootCmd := uptickd.NewRootCmd()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{
		"precheck-collection-migration",
		"--home=" + homeDir,
	})

	err := svrcmd.Execute(rootCmd, "uptick", homeDir)
	require.NoError(t, err, "the command must complete on an empty store")

	require.Contains(t, out.String(),
		"clean - no record would abort the v1->v2 migration",
		"the command body must actually run the scan and print its verdict")
}
