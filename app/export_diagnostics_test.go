package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The diagnostics sidecar is the durable half of the "degrade, but never
// silently" export policy: these tests pin that the report is actually written
// where an operator can find it, and that a clean export removes a stale one so
// the file always describes the LAST export.
func TestWriteExportDiagnosticsReport(t *testing.T) {
	home := t.TempDir()
	app := &Uptick{homeDir: home}

	diags := []ExportDiagnostic{
		{Module: "erc721", Kind: "token_pair_corrupt", Key: "0xdeadbeef", Detail: "boom"},
		{Module: "collection", Kind: "class_metadata_missing", Key: "ibc/ABC", Detail: "no metadata blob"},
	}

	path, err := app.writeExportDiagnosticsReport(4242, diags)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ExportDiagnosticsFileName), path)

	bz, err := os.ReadFile(path)
	require.NoError(t, err)

	var got ExportDiagnosticsReport
	require.NoError(t, json.Unmarshal(bz, &got))
	require.Equal(t, int64(4242), got.Height)
	require.Equal(t, 2, got.Total)
	require.Equal(t, diags, got.Issues)
}

// A report must never be written without a destination, but the caller must be
// able to tell the difference between "no issues" and "could not report".
func TestWriteExportDiagnosticsReportWithoutHome(t *testing.T) {
	app := &Uptick{}

	_, err := app.writeExportDiagnosticsReport(1, []ExportDiagnostic{{Module: "erc721"}})
	require.Error(t, err)
}

func TestRemoveStaleExportDiagnosticsReport(t *testing.T) {
	home := t.TempDir()
	app := &Uptick{homeDir: home}

	path, err := app.writeExportDiagnosticsReport(1, []ExportDiagnostic{{Module: "erc721", Kind: "k"}})
	require.NoError(t, err)
	require.FileExists(t, path)

	// A clean export must not leave the previous run's report behind.
	app.removeStaleExportDiagnosticsReport()
	require.NoFileExists(t, path)

	// Removing an absent report is a no-op, not a failure.
	require.NotPanics(t, app.removeStaleExportDiagnosticsReport)
}
