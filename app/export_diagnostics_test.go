package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	nftkeeper "cosmossdk.io/x/nft/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	nfttypes "github.com/UptickNetwork/uptick/x/collection/types"
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
	require.NoError(t, app.removeStaleExportDiagnosticsReport())
	require.NoFileExists(t, path)

	// Removing an absent report is a no-op, not a failure.
	require.NoError(t, app.removeStaleExportDiagnosticsReport())
}

// TestCollectExportDiagnosticsReportsSupplyMismatch pins the D-G1 fix: a class
// whose stored total-supply counter disagrees with the number of NFTs it
// actually holds must reach <home>/export-issues.json, not just the node log.
//
// Before the fix the collection half of collectExportDiagnostics used
// ExportIssues, which reads class records alone and cannot see supply_mismatch.
// That mismatch is the only check in the repository that can detect a diverged
// supply counter, so leaving it log-only meant it vanished with the process.
//
// The seeded class is healthy in every other respect (metadata present, its NFT
// list readable), so the ONLY diagnostic it can produce is the supply mismatch.
func TestCollectExportDiagnosticsReportsSupplyMismatch(t *testing.T) {
	app, ctx := sharedTestApp(t)

	// Isolate the writes in a cache layer on top of the shared application's
	// store: the shared app is a process-wide singleton, and a keeper mutation
	// performed here must not be visible to the next test. The cache is never
	// written back, so the mutation is discarded at the end of the test.
	cacheMS := ctx.MultiStore().CacheMultiStore()
	ctx = ctx.WithMultiStore(cacheMS)

	const classID = "mismatchclass"
	// 20-byte addresses so the bech32 round-trips the checks perform are valid.
	creator := sdk.AccAddress([]byte("mismatch-creator-01"))
	owner := sdk.AccAddress([]byte("mismatch-owner0001"))

	// Healthy collection + one NFT: metadata present (no class_metadata_*
	// issue) and stored supply == held count == 1.
	require.NoError(t, app.NFTKeeper.SaveDenom(
		ctx, classID, "Mismatch", "", "MM", creator, false, false, "", "", "", ""))
	require.NoError(t, app.NFTKeeper.SaveNFT(
		ctx, classID, "1", "One", "ipfs://1", "", "", owner))

	// Diverge the counter. The supply is maintained by the upstream nft
	// keeper's mint/burn and every write path in this repository goes through
	// them, so the only way to reproduce a historical bypass is to write the
	// counter key directly: 0x05 || classID
	// (cosmossdk.io/x/nft/keeper.ClassTotalSupply).
	store := ctx.KVStore(app.GetKey(nfttypes.StoreKey))
	supplyKey := append(append([]byte{}, nftkeeper.ClassTotalSupply...), []byte(classID)...)
	store.Set(supplyKey, sdk.Uint64ToBigEndian(99))

	diags := app.collectExportDiagnostics(ctx)

	var mismatch *ExportDiagnostic
	for i := range diags {
		if diags[i].Module == nfttypes.ModuleName &&
			diags[i].Key == classID &&
			diags[i].Kind == string(collectionkeeper.ExportIssueSupplyMismatch) {
			mismatch = &diags[i]
			break
		}
	}
	require.NotNil(t, mismatch,
		"supply_mismatch must reach the diagnostics collector; got %+v", diags)
	require.Contains(t, mismatch.Detail, "stored total supply counter is 99")

	// The degradation must survive into the durable sidecar -- the whole point
	// of D-G1. A fresh writer keeps the file in a temp dir, isolated from the
	// shared application's home.
	writer := &Uptick{homeDir: t.TempDir()}
	path, err := writer.writeExportDiagnosticsReport(ctx.BlockHeight(), diags)
	require.NoError(t, err)

	bz, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(bz), string(collectionkeeper.ExportIssueSupplyMismatch))
	require.Contains(t, string(bz), classID)
}

// ---------------------------------------------------------------------------
// The exit-code contract of `uptickd export`.
//
// ExportDiagnosticsStatus is the CLI's decision input, and its predicates are
// pure (no file system, no subprocess, no clock), so the contract is pinned
// here rather than only through a built binary. The values below are the
// externally observable interface: scripts gate on the code, operators gate on
// the message.
//
//	0  the export is fully accounted for (clean, or degraded with the report on
//	   disk)
//	3  the genesis was written but the degradations could not be recorded
//	1  reserved: the export itself failed (cobra / main.go's error path) --
//	   this mapping must never return it
// ---------------------------------------------------------------------------

// lostStatus is the one state that changes the process exit code: degraded,
// with no durable record of the degradation.
func lostStatus() ExportDiagnosticsStatus {
	return ExportDiagnosticsStatus{
		ExportID:   "status-test-lost",
		Height:     2,
		Issues:     3,
		Degraded:   true,
		ReportPath: "/node/export-issues.json",
		WriteErr:   errors.New("file too large"),
	}
}

func TestExportDiagnosticsStatusExitCodeTable(t *testing.T) {
	recorded := ExportDiagnosticsStatus{
		ExportID:      "status-test-recorded",
		Height:        2,
		Issues:        3,
		Degraded:      true,
		ReportPath:    "/node/export-issues.json",
		ReportWritten: true,
	}
	staleNotRemoved := ExportDiagnosticsStatus{
		ExportID:        "status-test-stale",
		Height:          2,
		ReportPath:      "/node/export-issues.json",
		StaleCleanupErr: errors.New("remove stale /node/export-issues.json: directory not empty"),
	}

	cases := []struct {
		name          string
		status        ExportDiagnosticsStatus
		wantCode      int
		wantLost      bool
		wantAttention bool
		wantMarker    string // substring the operator must see, "" for nothing
	}{
		{
			name:     "clean",
			status:   ExportDiagnosticsStatus{ExportID: "status-test-clean", Height: 2},
			wantCode: 0,
		},
		{
			name:          "degraded_report_on_disk",
			status:        recorded,
			wantCode:      0,
			wantAttention: true,
			wantMarker:    "DEGRADED",
		},
		{
			name:          "degraded_report_lost",
			status:        lostStatus(),
			wantCode:      ExportDiagnosticsDegradedExitCode,
			wantLost:      true,
			wantAttention: true,
			wantMarker:    "DEGRADED",
		},
		{
			name:          "clean_but_stale_report_not_removed",
			status:        staleNotRemoved,
			wantCode:      0,
			wantAttention: true,
			wantMarker:    "WARNING",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantLost, tc.status.LostDiagnostics(),
				"LostDiagnostics is the sole input of the exit code; getting it wrong is a silent gate failure")
			require.Equal(t, tc.wantAttention, tc.status.NeedsAttention())
			require.Equal(t, tc.wantCode, tc.status.CLIExitCode())

			// 1 is cobra's "the export itself failed"; conflating the two would
			// make a usable genesis look like a failed export.
			require.NotEqual(t, 1, tc.status.CLIExitCode())

			msg := tc.status.CLIMessage()
			if tc.wantMarker == "" {
				require.Empty(t, msg, "a fully accounted-for export must stay silent")
				return
			}
			require.Contains(t, msg, tc.wantMarker)

			// The marker has to say WHICH thing is wrong. A degraded export is
			// identified by its export id (correlates with the stderr JSON and
			// the report it could not write); the stale-report warning is
			// identified by the path that must not be trusted.
			if tc.wantLost || tc.status.Degraded {
				require.Contains(t, msg, tc.status.ExportID)
			} else {
				require.Contains(t, msg, tc.status.ReportPath)
			}

			if tc.status.WriteErr != nil {
				require.Contains(t, msg, tc.status.WriteErr.Error())
			}
		})
	}
}

// The two codes an operator can act on must not collide, and 3 must be the
// documented value (a release note, a runbook and a shell script all quote it).
func TestExportDiagnosticsStatusExitCodeValues(t *testing.T) {
	require.Equal(t, 3, ExportDiagnosticsDegradedExitCode)
	require.NotZero(t, ExportDiagnosticsDegradedExitCode, "0 means success")
	require.NotEqual(t, 1, ExportDiagnosticsDegradedExitCode,
		"1 is cobra's export-failed path; reusing it would erase the distinction")

	// A degraded export with an on-disk report is NOT a failure for a pipeline.
	require.Zero(t, ExportDiagnosticsStatus{Degraded: true, ReportWritten: true}.CLIExitCode())

	// A non-degraded export can never be "lost", whatever else is wrong: the
	// stale-cleanup failure is an operator warning, not a lost degradation.
	require.False(t, ExportDiagnosticsStatus{StaleCleanupErr: errors.New("x")}.LostDiagnostics())
	require.Zero(t, ExportDiagnosticsStatus{StaleCleanupErr: errors.New("x")}.CLIExitCode())
}
