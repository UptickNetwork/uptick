package app

// Regression tests for the two coupled defects this batch fixes:
//
//	O-02  <home>/export-issues.json was written non-atomically and BEFORE the
//	      export had produced anything, so a failed export could leave behind a
//	      report that described an export which never happened, and an
//	      interrupted write could leave a truncated one.
//	G-01  a sidecar that could not be written was swallowed into a log line, so
//	      a degraded export was indistinguishable from a clean one to every
//	      consumer that was not reading the node log.
//
// Every test here has a documented reverse control (see the batch report): the
// assertion is chosen so that reverting the production change turns it red.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
)

// exportTestHome points the shared application at a throwaway node home for the
// duration of the test, and hands the previous one back afterwards: the shared
// app is a process-wide singleton (see sharedTestApp), so the home must not leak
// into the next test.
func exportTestHome(t *testing.T, app *Uptick, home string) {
	t.Helper()
	prev := app.homeDir
	app.homeDir = home
	t.Cleanup(func() { app.homeDir = prev })
}

// plantDegradation installs exactly one degradation the genesis export can
// report: a reverse-only NFT UID index entry (nftUID -> tokenUID) with no
// matching forward entry. x/erc721's exporter reports it as
// nft_uid_index_reverse_without_forward and drops the binding; the export then
// succeeds but is degraded, which is the state all the sidecar tests need.
//
// The key is deleted again on cleanup, so the shared application is left exactly
// as it was found and no test depends on execution order.
func plantDegradation(t *testing.T, nftUID string) {
	t.Helper()
	app, ctx := sharedTestApp(t)

	app.Erc721Keeper.SetNFTUIDPairByNFTUID(ctx, nftUID, "token-uid-"+nftUID)
	t.Cleanup(func() {
		app.Erc721Keeper.DeleteNFTUIDPairByNFTUID(sharedTestAppCtx, nftUID)
	})
}

// shortWriteLimit makes every write larger than 64 bytes fail with EFBIG, and
// ignores SIGXFSZ so the kernel returns the error instead of killing the test
// process. This is the same failure class as a full disk or a quota, and unlike
// a read-only directory it cannot be bypassed by running as root. The returned
// function restores the previous limit (a cleanup is registered as a safety
// net).
func shortWriteLimit(t *testing.T) func() {
	t.Helper()
	signal.Ignore(syscall.SIGXFSZ)

	var prev syscall.Rlimit
	require.NoError(t, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &prev))
	require.NoError(t, syscall.Setrlimit(syscall.RLIMIT_FSIZE,
		&syscall.Rlimit{Cur: 64, Max: prev.Max}))

	restored := false
	restore := func() {
		if restored {
			return
		}
		restored = true
		_ = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &prev)
	}
	t.Cleanup(restore)
	return restore
}

// exportDirEntries lists a directory, so a test can assert that a failed write
// left nothing at all behind (not even a temporary file).
func exportDirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// captureExportNotices swaps the machine-readable notice stream for a buffer.
func captureExportNotices(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := exportDiagnosticsStderr
	exportDiagnosticsStderr = buf
	t.Cleanup(func() { exportDiagnosticsStderr = prev })
	return buf
}

// ---------------------------------------------------------------------------
// G-01: a degraded export whose report cannot be persisted must still succeed,
// must say so on stderr in a machine-readable way, must carry a CLI marker, and
// must exit with a code that is neither 0 nor 1.
//
// Reverse control: restore the pre-fix body of the sidecar step in export.go
// (write, and on failure only ctx.Logger().Error(...)). The status is then never
// populated and every assertion below fails.
// ---------------------------------------------------------------------------
func TestExportDiagnosticsDegradedExportSurvivesLostReport(t *testing.T) {
	const nftUID = "export-test-lost-nft"

	app, ctx := sharedTestApp(t)
	plantDegradation(t, nftUID)

	// Precondition: the plant really does degrade the export. Without this the
	// whole experiment would be vacuous.
	require.NotEmpty(t, app.collectExportDiagnostics(ctx),
		"the planted reverse-only index entry must degrade the export")

	home := t.TempDir()
	exportTestHome(t, app, home)
	notices := captureExportNotices(t)

	restore := shortWriteLimit(t)
	exp, err := app.ExportAppStateAndValidators(false, nil, nil)
	restore()

	require.NoError(t, err, "the export must succeed even when its report is lost")
	require.NotEmpty(t, exp.AppState, "a valid genesis must still be handed back")
	require.Equal(t, app.LastBlockHeight()+1, exp.Height)

	status := LastExportDiagnosticsStatus()
	require.True(t, status.Degraded, "a degraded export must record that it degraded")
	require.False(t, status.ReportWritten, "the report could not be written")
	require.Error(t, status.WriteErr)
	require.True(t, status.LostDiagnostics())

	// The CLI contract: a distinct exit code, and a standing marker.
	require.Equal(t, ExportDiagnosticsDegradedExitCode, status.CLIExitCode())
	require.Equal(t, 3, status.CLIExitCode(),
		"the degraded code must not collide with 0 (clean) or 1 (export failed)")
	msg := status.CLIMessage()
	require.Contains(t, msg, "DEGRADED")
	require.Contains(t, msg, status.ExportID)

	// The machine-readable channel: exactly one line of JSON naming this export.
	lines := strings.Split(strings.TrimRight(notices.String(), "\n"), "\n")
	require.Len(t, lines, 1, "expected exactly one notice line, got %q", notices.String())
	require.True(t, json.Valid([]byte(lines[0])), "the notice must be one line of JSON: %q", lines[0])

	var notice ExportDiagnosticsNotice
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &notice))
	require.Equal(t, exportDiagnosticsEventNotPersisted, notice.Event)
	require.Equal(t, status.ExportID, notice.ExportID)
	require.Equal(t, status.Issues, notice.Issues)
	require.NotZero(t, notice.Issues)
	require.NotEmpty(t, notice.Error)
	require.Empty(t, notice.StaleReportRemovalError, "there was no stale report to remove here")

	require.NoFileExists(t, filepath.Join(home, ExportDiagnosticsFileName))
}

// ---------------------------------------------------------------------------
// O-02 (atomicity): an interrupted write must leave NOTHING behind.
//
// Reverse control: change writeFileAtomic to a plain os.WriteFile. The partial
// bytes are then already on disk when the write returns EFBIG, so both the
// "no report" and the "no leftovers" assertions fail.
// ---------------------------------------------------------------------------
func TestExportDiagnosticsFailedAtomicWriteLeavesNothingBehind(t *testing.T) {
	const nftUID = "export-test-atomic-nft"

	app, ctx := sharedTestApp(t)
	plantDegradation(t, nftUID)
	require.NotEmpty(t, app.collectExportDiagnostics(ctx))

	home := t.TempDir()
	exportTestHome(t, app, home)

	restore := shortWriteLimit(t)
	_, err := app.ExportAppStateAndValidators(false, nil, nil)
	restore()

	require.NoError(t, err)

	// The failure must be a real write failure, otherwise the two assertions
	// below would be satisfied by the clean-export branch instead.
	status := LastExportDiagnosticsStatus()
	require.True(t, status.Degraded)
	require.Error(t, status.WriteErr)

	// ...and it must be a failure of a SIBLING TEMPORARY file, not of the
	// report path itself: writing the report path in place is exactly what made
	// a short write visible to a reader.
	require.Contains(t, status.WriteErr.Error(), "."+ExportDiagnosticsFileName+".tmp-",
		"the bytes must go to a sibling temp file, never to the report path directly: %v", status.WriteErr)

	require.NoFileExists(t, filepath.Join(home, ExportDiagnosticsFileName))
	require.Empty(t, exportDirEntries(t, home),
		"an interrupted atomic write must not leave a temporary file behind")
}

// ---------------------------------------------------------------------------
// O-02 (atomicity, symlink judgement): the publish step must be a rename of a
// sibling file, never an in-place write through whatever occupies the path. The
// symlink criterion is the crisp one: rename replaces the link, an O_TRUNC write
// punches through it and destroys the target.
//
// Reverse control: replace writeFileAtomic with os.WriteFile. The link is then
// followed, the target is overwritten, and the link is still a link.
// ---------------------------------------------------------------------------
func TestExportDiagnosticsWriteIsAtomicThroughSymlink(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "innocent-target.json")
	require.NoError(t, os.WriteFile(target, []byte("ORIGINAL-CONTENT"), 0o600))

	link := filepath.Join(home, ExportDiagnosticsFileName)
	require.NoError(t, os.Symlink(target, link))

	writer := &Uptick{homeDir: home}
	path, err := writer.writeExportDiagnosticsReport(7, []ExportDiagnostic{
		{Module: "erc721", Kind: "token_pair_corrupt", Key: "0xdead", Detail: "boom"},
	})
	require.NoError(t, err)
	require.Equal(t, link, path)

	tbz, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "ORIGINAL-CONTENT", string(tbz),
		"an atomic rename must not touch whatever the symlink pointed at")

	fi, err := os.Lstat(link)
	require.NoError(t, err)
	require.Zero(t, fi.Mode()&os.ModeSymlink,
		"the symlink must have been replaced by the newly written regular file")
	require.Equal(t, os.FileMode(0o600), fi.Mode().Perm())

	bz, err := os.ReadFile(link)
	require.NoError(t, err)
	require.True(t, json.Valid(bz), "the published report must be complete JSON: %s", string(bz))
}

// ---------------------------------------------------------------------------
// O-02, second half (staleness): when a degraded export cannot write its report,
// the report left by an EARLIER export must not survive, or an operator reads it
// as if it described the current export.
//
// Reverse control: drop the removeStaleExportDiagnosticsReport() call from the
// write-failure branch. The stale file then stays byte-identical on disk.
// ---------------------------------------------------------------------------
func TestExportDiagnosticsStaleReportIsClearedWhenReportCannotBeWritten(t *testing.T) {
	const nftUID = "export-test-stale-nft"

	app, ctx := sharedTestApp(t)
	plantDegradation(t, nftUID)
	require.NotEmpty(t, app.collectExportDiagnostics(ctx))

	home := t.TempDir()
	stale := filepath.Join(home, ExportDiagnosticsFileName)
	staleBody := []byte(`{"height":111,"total":1,"issues":[{"module":"erc721",` +
		`"kind":"token_pair_corrupt","key":"0xfrom-the-past","detail":"a previous export"}]}` + "\n")
	require.NoError(t, os.WriteFile(stale, staleBody, 0o600))
	exportTestHome(t, app, home)
	notices := captureExportNotices(t)

	restore := shortWriteLimit(t)
	exp, err := app.ExportAppStateAndValidators(false, nil, nil)
	restore()

	require.NoError(t, err)
	require.NotEmpty(t, exp.AppState)

	status := LastExportDiagnosticsStatus()
	require.True(t, status.Degraded)
	require.False(t, status.ReportWritten)
	require.Error(t, status.WriteErr)
	require.NoError(t, status.StaleCleanupErr,
		"the stale report was removable, so the export must have removed it")

	// The consequence that matters: the previous export's report is gone, so
	// there is nothing left that could be misread as this export's report.
	require.NoFileExists(t, stale)
	require.Empty(t, exportDirEntries(t, home))

	var notice ExportDiagnosticsNotice
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(notices.Bytes()), &notice))
	require.Equal(t, exportDiagnosticsEventNotPersisted, notice.Event)
	require.Empty(t, notice.StaleReportRemovalError)
	require.Equal(t, status.ExportID, notice.ExportID)
}

// ---------------------------------------------------------------------------
// G-01 / O-02 (ordering): the sidecar is the record of an export, so an export
// that fails in its LAST step (staking.WriteValidators) must neither publish a
// new report nor destroy the previous one.
//
// Reverse control: move the sidecar step back above MarshalIndent/WriteValidators.
// A clean run then deletes the pre-placed report and a degraded run overwrites
// it; either way the byte-for-byte assertion fails.
// ---------------------------------------------------------------------------
func TestExportDiagnosticsNotPublishedWhenExportFailsLate(t *testing.T) {
	app, ctx := sharedTestApp(t)

	home := t.TempDir()
	stale := filepath.Join(home, ExportDiagnosticsFileName)
	staleBody := []byte(`{"height":111,"total":1,"issues":[{"module":"erc721",` +
		`"kind":"token_pair_corrupt","key":"0xfrom-the-past","detail":"a previous export"}]}` + "\n")
	require.NoError(t, os.WriteFile(stale, staleBody, 0o600))
	exportTestHome(t, app, home)

	// Break the last step: a bonded-validator index entry whose operator has no
	// validator record makes staking.WriteValidators fail while every earlier
	// step of the export succeeds. The value is the wire encoding of
	// gogotypes.Int64Value{Value: 1}, which staking's own ExportGenesis reads
	// without error, so the export is not aborted earlier than intended.
	orphan := sdk.ValAddress(bytes.Repeat([]byte{0xAB}, 20))
	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	orphanKey := stakingtypes.GetLastValidatorPowerKey(orphan)
	store.Set(orphanKey, []byte{0x08, 0x01})
	t.Cleanup(func() { store.Delete(orphanKey) })

	// Pin the injection to WriteValidators, so the assertion below cannot be
	// satisfied by an export that failed somewhere harmless.
	_, verr := staking.WriteValidators(ctx, app.StakingKeeper)
	require.Error(t, verr, "the orphan last-validator entry must break WriteValidators")

	exp, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.Error(t, err, "the orphan last-validator entry must fail the export")
	require.Empty(t, exp.AppState)
	require.Zero(t, exp.Height)

	got, rerr := os.ReadFile(stale)
	require.NoError(t, rerr)
	require.Equal(t, string(staleBody), string(got),
		"a failed export must not rewrite or delete the report of a previous one")
}

// ---------------------------------------------------------------------------
// A clean export that cannot clear a stale report is a warning an operator has
// to see, not a silent stderr line -- but it is not fatal for a pipeline, since
// the genesis itself is clean.
//
// Reverse control: make removeStaleExportDiagnosticsReport swallow the error
// again (log only). StaleCleanupErr is then nil and every assertion below fails.
// ---------------------------------------------------------------------------
func TestExportDiagnosticsStaleCleanupFailureIsVisible(t *testing.T) {
	app, _ := sharedTestApp(t)

	home := t.TempDir()
	// A non-empty directory where the sidecar belongs makes removal fail
	// deterministically and independently of the process UID.
	blocked := filepath.Join(home, ExportDiagnosticsFileName)
	require.NoError(t, os.MkdirAll(filepath.Join(blocked, "keepme"), 0o755))
	exportTestHome(t, app, home)
	notices := captureExportNotices(t)

	// The clean branch, driven directly so the assertion does not depend on the
	// shared application carrying no degradation from some other test.
	app.finalizeExportDiagnostics(sharedTestAppCtx, app.LastBlockHeight()+1, nil)

	status := LastExportDiagnosticsStatus()
	require.False(t, status.Degraded)
	require.False(t, status.ReportWritten)
	require.Error(t, status.StaleCleanupErr,
		"a stale report that cannot be removed must be reported, not merely printed")
	require.True(t, status.NeedsAttention())

	// Loud, but not a pipeline failure: the genesis is clean.
	require.Zero(t, status.CLIExitCode())
	msg := status.CLIMessage()
	require.Contains(t, msg, "WARNING")
	require.Contains(t, msg, blocked)

	var notice ExportDiagnosticsNotice
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(notices.Bytes()), &notice))
	require.Equal(t, exportDiagnosticsEventStaleNotRemoved, notice.Event)
	require.Equal(t, "clean", notice.Status)
	require.NotEmpty(t, notice.Error)

	// And the export as a whole still succeeds: the sidecar never fails one.
	exp, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	require.NotEmpty(t, exp.AppState)
	_, serr := os.Stat(filepath.Join(blocked, "keepme"))
	require.NoError(t, serr, "the blocking directory must have been left untouched")
}

// ---------------------------------------------------------------------------
// The report body has to be able to identify WHICH export it describes, or a
// report that survives a failed commit is indistinguishable from the current
// one.
//
// Reverse control: drop the three fields from ExportDiagnosticsReport. The
// key-presence assertions fail.
// ---------------------------------------------------------------------------
func TestExportDiagnosticsReportCarriesStalenessFields(t *testing.T) {
	home := t.TempDir()
	writer := &Uptick{homeDir: home}

	path, err := writer.writeExportDiagnosticsReport(4242, []ExportDiagnostic{
		{Module: "erc721", Kind: "token_pair_corrupt", Key: "0xdeadbeef", Detail: "boom"},
	})
	require.NoError(t, err)

	bz, err := os.ReadFile(path)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(bz, &raw))
	require.Contains(t, raw, "export_id")
	require.Contains(t, raw, "finished_at")
	require.Contains(t, raw, "status")

	var report ExportDiagnosticsReport
	require.NoError(t, json.Unmarshal(bz, &report))
	require.NotEmpty(t, report.ExportID)
	require.Equal(t, ExportDiagnosticsStatusDegraded, report.Status)
	require.Equal(t, int64(4242), report.Height)
	require.Equal(t, 1, report.Total)

	finishedAt, err := time.Parse(time.RFC3339, report.FinishedAt)
	require.NoError(t, err, "finished_at must be a parseable timestamp")
	require.WithinDuration(t, time.Now().UTC(), finishedAt.UTC(), time.Minute)

	// Two runs must never share an id.
	other, err := writer.writeExportDiagnosticsReport(4242, nil)
	require.NoError(t, err)
	require.Equal(t, path, other)

	otherBz, err := os.ReadFile(other)
	require.NoError(t, err)
	var otherReport ExportDiagnosticsReport
	require.NoError(t, json.Unmarshal(otherBz, &otherReport))
	require.NotEqual(t, report.ExportID, otherReport.ExportID)
}

// ---------------------------------------------------------------------------
// The healthy side of the same contract: a degraded export whose report WAS
// committed is not a pipeline failure, but it still prints a marker, and the
// report on disk names exactly this export.
//
// Reverse control: stop publishing the status (or stop writing export_id into
// the report) and the assertions below fail.
// ---------------------------------------------------------------------------
func TestExportDiagnosticsDegradedExportWithReportIsRecorded(t *testing.T) {
	const nftUID = "export-test-recorded-nft"

	app, ctx := sharedTestApp(t)
	plantDegradation(t, nftUID)
	require.NotEmpty(t, app.collectExportDiagnostics(ctx))

	home := t.TempDir()
	exportTestHome(t, app, home)
	notices := captureExportNotices(t)

	exp, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	require.NotEmpty(t, exp.AppState)

	status := LastExportDiagnosticsStatus()
	require.True(t, status.Degraded)
	require.True(t, status.ReportWritten)
	require.False(t, status.LostDiagnostics())
	require.Zero(t, status.CLIExitCode(), "a recorded degradation is not a pipeline failure")
	require.Contains(t, status.CLIMessage(), "DEGRADED")
	require.Empty(t, notices.String(), "without a lost report there is no machine-readable notice")

	bz, err := os.ReadFile(filepath.Join(home, ExportDiagnosticsFileName))
	require.NoError(t, err)

	var report ExportDiagnosticsReport
	require.NoError(t, json.Unmarshal(bz, &report))
	require.Equal(t, status.ExportID, report.ExportID,
		"the report on disk must identify the export that just ran")
	require.Equal(t, ExportDiagnosticsStatusDegraded, report.Status)
	require.Equal(t, exp.Height, report.Height)
	require.Equal(t, len(report.Issues), report.Total)

	var sawPlanted bool
	for _, issue := range report.Issues {
		if issue.Key == nftUID && issue.Kind == string(erc721keeper.GenesisExportIssueUIDIndexBackward) {
			sawPlanted = true
		}
	}
	require.True(t, sawPlanted, "the report must name the planted degradation; got %+v", report.Issues)
}
