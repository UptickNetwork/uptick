package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
)

// ExportDiagnosticsFileName is the diagnostics sidecar written into the node
// home directory when a genesis export had to degrade.
//
// AppModule.ExportGenesis has no error channel, so a module cannot hand a
// structured report back to the CLI; persisting it next to the exported genesis
// is what keeps a degraded export auditable after the process exits (the same
// information also goes to the node log, but a log stream is not durable).
//
// CONSUMER CONTRACT -- READ THIS BEFORE ADDING A GATE
// --------------------------------------------------
// This sidecar is for an OPERATOR to read after a real `uptickd export`; it is
// deliberately NOT a CI gate input. Nothing under scripts/, .github/ or the
// Makefile reads it and it is not committed -- it lives in the node home of the
// machine that ran the export, so a non-empty report blocks no pipeline. Its
// value is making a degraded export *visible* to the operator instead of leaving
// the degradation in a log stream that scrolls away. A future gate must first
// define where the file comes from in CI (it never exists in a fresh checkout),
// or it would always pass.
const ExportDiagnosticsFileName = "export-issues.json"

// ExportDiagnosticsStatusDegraded is the status carried by every committed
// sidecar. A clean export removes the file instead of writing a report, so the
// only possible status of a file that exists is "degraded" -- and, combined with
// export_id/finished_at below, that is enough for an operator to tell whether
// the file on disk describes the export they are looking at.
const ExportDiagnosticsStatusDegraded = "degraded"

// ExportDiagnosticsDegradedExitCode is the process exit code `uptickd export`
// terminates with when the genesis was exported successfully but the
// diagnostics sidecar could not be committed -- i.e. a degraded export whose
// degradation is only recorded in a log stream. It is deliberately distinct
// from 0 (fully accounted export) and from 1 (the export itself failed), so a
// pipeline can gate on it. The export output itself is always complete: see the
// "Never fail the export over the sidecar" invariant in app/export.go.
const ExportDiagnosticsDegradedExitCode = 3

// Events carried by the machine-readable stderr notice (see
// ExportDiagnosticsNotice). They are stable strings: scripts match on them.
const (
	// exportDiagnosticsEventNotPersisted: the export degraded and its report
	// could not be written.
	exportDiagnosticsEventNotPersisted = "export_diagnostics_not_persisted"
	// exportDiagnosticsEventStaleNotRemoved: the export was clean but the
	// previous export's report could not be deleted, so the file on disk no
	// longer describes the most recent export.
	exportDiagnosticsEventStaleNotRemoved = "export_diagnostics_stale_not_removed"
)

// ExportDiagnostic is one record that a genesis export could not represent
// faithfully.
type ExportDiagnostic struct {
	// Module that degraded, e.g. "erc721".
	Module string `json:"module"`
	// Kind classifies the damage, e.g. "token_pair_corrupt".
	Kind string `json:"kind"`
	// Key is the store key or class id the problem belongs to.
	Key string `json:"key"`
	// Detail is the human-readable explanation.
	Detail string `json:"detail"`
}

// ExportDiagnosticsReport is the document written to
// <home>/export-issues.json. It is written only when at least one degradation
// happened; a clean export removes a stale report instead, so the presence of
// the file always means "the most recent export was degraded".
//
// ExportID / Status / FinishedAt exist so a report can never be mistaken for
// the current export's: they identify WHICH export the file describes. This
// matters because the file is only best-effort (a degraded export must never be
// failed by a read-only home), so in principle an older report can survive a
// newer, degraded export. The failure path quarantines such a file, and these
// fields make any survivor self-describing for an operator or a script that
// compares them against the export_id printed on stderr / the process log.
type ExportDiagnosticsReport struct {
	// ExportID uniquely identifies the export run that wrote this report.
	ExportID string `json:"export_id"`
	// Status is always ExportDiagnosticsStatusDegraded for a committed report.
	Status string `json:"status"`
	// FinishedAt is when the report was committed (RFC 3339, UTC).
	FinishedAt string `json:"finished_at"`
	// Height is the height the exported genesis starts at.
	Height int64 `json:"height"`
	// Total is the number of degradation records in Issues.
	Total int `json:"total"`
	// Issues lists the records that could not be represented faithfully.
	Issues []ExportDiagnostic `json:"issues"`
}

// ExportDiagnosticsNotice is the single-line JSON document written to stderr
// when a degraded export could not durably record its degradations, or when a
// stale report could not be cleared. One line, stable keys: it is meant to be
// consumed by a wrapper script (`uptickd export ... 2> >(grep ...)`), while the
// human-readable rendering is printed by the CLI.
type ExportDiagnosticsNotice struct {
	// Event is one of the exportDiagnosticsEvent* constants.
	Event string `json:"event"`
	// ExportID identifies the export run this notice is about.
	ExportID string `json:"export_id"`
	// Height is the height the exported genesis starts at.
	Height int64 `json:"height"`
	// Status is "degraded" for a degraded export, "clean" for the stale-report
	// case.
	Status string `json:"status"`
	// Issues is the number of degradations that could not be recorded.
	Issues int `json:"issues,omitempty"`
	// Path is the report location the notice is about.
	Path string `json:"path"`
	// Error is why the report could not be committed.
	Error string `json:"error"`
	// StaleReportRemovalError is set when, on top of the failure above, the
	// report left by an earlier export could not be removed either.
	StaleReportRemovalError string `json:"stale_report_removal_error,omitempty"`
}

// ExportDiagnosticsStatus is the verdict of the diagnostics commit step of one
// export. It is what the CLI (cmd/uptickd) renders and what a test can assert
// on without a file system.
type ExportDiagnosticsStatus struct {
	// ExportID identifies the export run.
	ExportID string
	// Height is the height the exported genesis starts at.
	Height int64
	// Issues is the number of degradations the export produced.
	Issues int
	// Degraded is true when the genesis does not faithfully represent the chain
	// state.
	Degraded bool
	// ReportPath is where the report was (or would have been) written.
	ReportPath string
	// ReportWritten is true when the sidecar was durably committed.
	ReportWritten bool
	// WriteErr is why the sidecar was not committed, if it was not.
	WriteErr error
	// StaleCleanupErr is why a report left by an earlier export could not be
	// removed (either after a clean export, or after this export failed to
	// commit its own report).
	StaleCleanupErr error
}

// LostDiagnostics reports whether the export degraded but no durable record of
// that degradation exists. This is the case the CLI turns into
// ExportDiagnosticsDegradedExitCode.
func (s ExportDiagnosticsStatus) LostDiagnostics() bool {
	return s.Degraded && !s.ReportWritten
}

// NeedsAttention reports whether anything about the diagnostics state must be
// surfaced to the operator.
func (s ExportDiagnosticsStatus) NeedsAttention() bool {
	return s.Degraded || s.StaleCleanupErr != nil
}

// CLIExitCode is the process exit code `uptickd export` must finish with.
// Only LostDiagnostics is fatal-for-pipelines: a degraded export whose report
// WAS written is fully accounted for, and a stale report on a clean export is
// noise an operator must read but that does not make the genesis wrong.
func (s ExportDiagnosticsStatus) CLIExitCode() int {
	if s.LostDiagnostics() {
		return ExportDiagnosticsDegradedExitCode
	}
	return 0
}

// CLIMessage is the single operator-facing line for the export CLI. It is empty
// when the export needs no remark at all.
func (s ExportDiagnosticsStatus) CLIMessage() string {
	switch {
	case s.LostDiagnostics():
		return fmt.Sprintf(
			"uptickd export: DEGRADED: the genesis was written, but its diagnostics report could not be persisted "+
				"(export_id=%s, affected_records=%d, path=%s, err=%v); treat this export as NOT fully accounted for",
			s.ExportID, s.Issues, s.ReportPath, s.WriteErr)
	case s.Degraded:
		return fmt.Sprintf(
			"uptickd export: DEGRADED: the genesis was written with %d record(s) that could not be represented; "+
				"see %s (export_id=%s)",
			s.Issues, s.ReportPath, s.ExportID)
	case s.StaleCleanupErr != nil:
		return fmt.Sprintf(
			"uptickd export: WARNING: the genesis is clean, but the diagnostics report of an earlier export could not be removed "+
				"from %s (err=%v); do not trust that file",
			s.ReportPath, s.StaleCleanupErr)
	}
	return ""
}

// exportDiagnosticsStderr is where the machine-readable notices go. It is a
// variable rather than a direct os.Stderr reference so tests can capture it.
var exportDiagnosticsStderr io.Writer = os.Stderr

// lastExportDiagnostics holds the verdict of the most recent export performed in
// this process. ExportAppStateAndValidators has no channel to return it (its
// signature is fixed by servertypes.AppExporter), and the export command runs
// once per process, so a package-level slot is the only place the CLI can read
// it from.
var (
	lastExportDiagnosticsMu sync.Mutex
	lastExportDiagnostics   ExportDiagnosticsStatus
)

// LastExportDiagnosticsStatus returns the diagnostics verdict of the most recent
// ExportAppStateAndValidators call in this process.
func LastExportDiagnosticsStatus() ExportDiagnosticsStatus {
	lastExportDiagnosticsMu.Lock()
	defer lastExportDiagnosticsMu.Unlock()
	return lastExportDiagnostics
}

func publishExportDiagnosticsStatus(s ExportDiagnosticsStatus) {
	lastExportDiagnosticsMu.Lock()
	lastExportDiagnostics = s
	lastExportDiagnosticsMu.Unlock()
}

var exportIDSeq atomic.Uint64

// newExportID returns an opaque identifier for one export run. It only has to
// be unique within a node's lifetime and sortable-ish for log correlation,
// hence timestamp + pid + counter; it is never persisted anywhere but the
// report and the notices.
func newExportID() string {
	return fmt.Sprintf("%s-%d-%d",
		time.Now().UTC().Format("20060102T150405.000000000"), os.Getpid(), exportIDSeq.Add(1))
}

// collectExportDiagnostics asks every genesis-exporting module for the records
// it could not represent faithfully. It runs after the module manager has
// exported and uses the modules' issue scanners rather than re-exporting: the
// export path itself only logs what it drops, and re-exporting would be a second
// full export.
//
// For collection it uses ExportIssuesWithReport rather than the class-only
// ExportIssues: supply_mismatch and nft_list_failed can only be observed while
// walking a class' NFT list, so the class-only scan would silently drop them. The
// walk matches the one the module's ExportGenesis performs, so the sidecar and
// the export agree by construction; the second traversal happens only on this
// operator-initiated path, never in consensus. erc721/cw721 already report their
// full issue sets here.
func (app *Uptick) collectExportDiagnostics(ctx sdk.Context) []ExportDiagnostic {
	var diags []ExportDiagnostic

	for _, issue := range app.NFTKeeper.ExportIssuesWithReport(ctx) {
		diags = append(diags, ExportDiagnostic{
			Module: collectiontypes.ModuleName,
			Kind:   string(issue.Kind),
			Key:    issue.ClassID,
			Detail: issue.Detail,
		})
	}

	for _, issue := range app.Erc721Keeper.ExportIssues(ctx) {
		diags = append(diags, ExportDiagnostic{
			Module: erc721types.ModuleName,
			Kind:   string(issue.Kind),
			Key:    issue.Key,
			Detail: issue.Detail,
		})
	}

	for _, issue := range app.Cw721Keeper.ExportIssues(ctx) {
		diags = append(diags, ExportDiagnostic{
			Module: cw721types.ModuleName,
			Kind:   string(issue.Kind),
			Key:    issue.Key,
			Detail: issue.Detail,
		})
	}

	return diags
}

// finalizeExportDiagnostics commits the diagnostics sidecar for an export that
// has ALREADY produced its genesis. The caller (app/export.go) must only invoke
// it once MarshalIndent and WriteValidators have both succeeded, so that a
// report can never describe an export that never happened.
//
// It never returns an error and never panics: the export must not fail because
// its sidecar could not be written (see app/export.go). Everything that went
// wrong is published through LastExportDiagnosticsStatus, mirrored to stderr as
// one line of JSON, and logged.
//
// The diagnostics are passed in rather than re-collected so the commit step can
// be driven directly by a test with a synthetic degradation, without corrupting
// the process-wide application singleton.
func (app *Uptick) finalizeExportDiagnostics(ctx sdk.Context, height int64, diags []ExportDiagnostic) {
	st := ExportDiagnosticsStatus{
		ExportID:   newExportID(),
		Height:     height,
		Issues:     len(diags),
		Degraded:   len(diags) > 0,
		ReportPath: app.exportDiagnosticsReportPath(),
	}
	defer func() { publishExportDiagnosticsStatus(st) }()

	if !st.Degraded {
		// A clean export must not leave an earlier (degraded) export's report
		// behind: the presence of the file means "the most recent export was
		// degraded".
		if err := app.removeStaleExportDiagnosticsReport(); err != nil {
			st.StaleCleanupErr = err
			ctx.Logger().Error("clean genesis export could not remove a stale diagnostics report",
				"path", st.ReportPath, "err", err, "export_id", st.ExportID)
			emitExportDiagnosticsNotice(exportDiagnosticsStderr, ExportDiagnosticsNotice{
				Event:    exportDiagnosticsEventStaleNotRemoved,
				ExportID: st.ExportID,
				Height:   st.Height,
				Status:   "clean",
				Path:     st.ReportPath,
				Error:    err.Error(),
			})
		}
		return
	}

	path, writeErr := app.writeExportDiagnosticsReportFor(st.ExportID, height, diags)
	if writeErr == nil {
		st.ReportWritten = true
		st.ReportPath = path
		ctx.Logger().Error(
			"genesis export was degraded; the full list of affected records was written to the diagnostics report",
			"path", path,
			"affected_records", len(diags),
			"export_id", st.ExportID,
		)
		return
	}

	// Never fail the export over the sidecar: the degradations are already
	// logged by each module, and the genesis itself is valid.
	ctx.Logger().Error("failed to write the export diagnostics report",
		"err", writeErr, "export_id", st.ExportID, "affected_records", len(diags))
	st.WriteErr = writeErr

	// A failed commit must not leave the PREVIOUS export's report on disk: an
	// operator would read it as if it described this export. Clear it, and
	// report if even that fails.
	st.StaleCleanupErr = app.removeStaleExportDiagnosticsReport()

	notice := ExportDiagnosticsNotice{
		Event:    exportDiagnosticsEventNotPersisted,
		ExportID: st.ExportID,
		Height:   st.Height,
		Status:   ExportDiagnosticsStatusDegraded,
		Issues:   len(diags),
		Path:     st.ReportPath,
		Error:    writeErr.Error(),
	}
	if st.StaleCleanupErr != nil {
		notice.StaleReportRemovalError = st.StaleCleanupErr.Error()
		ctx.Logger().Error("could not remove the stale diagnostics report either",
			"path", st.ReportPath, "err", st.StaleCleanupErr)
	}
	emitExportDiagnosticsNotice(exportDiagnosticsStderr, notice)
}

// emitExportDiagnosticsNotice writes one line of machine-readable JSON. It must
// never be the reason an export fails, so encoding problems degrade to a plain
// warning.
func emitExportDiagnosticsNotice(w io.Writer, notice ExportDiagnosticsNotice) {
	if w == nil {
		return
	}
	bz, err := json.Marshal(notice)
	if err != nil {
		fmt.Fprintf(w, "warning: could not encode the export diagnostics notice: %v\n", err)
		return
	}
	fmt.Fprintf(w, "%s\n", bz)
}

// exportDiagnosticsReportPath is where the sidecar lives, or "" when the node
// home is not configured (a bare app in a unit test).
func (app *Uptick) exportDiagnosticsReportPath() string {
	if app.homeDir == "" {
		return ""
	}
	return filepath.Join(app.homeDir, ExportDiagnosticsFileName)
}

// writeExportDiagnosticsReport persists the report into the node home
// directory and returns the path it wrote.
func (app *Uptick) writeExportDiagnosticsReport(height int64, diags []ExportDiagnostic) (string, error) {
	return app.writeExportDiagnosticsReportFor(newExportID(), height, diags)
}

// writeExportDiagnosticsReportFor persists the report under a caller-supplied
// export id.
//
// The write is atomic: the bytes go to a temporary file in the SAME directory
// (so the final rename cannot cross a device boundary), are flushed to stable
// storage, and only then published with rename(2). A short write, a full disk
// or a crash therefore cannot leave behind a truncated report that both claims
// "the most recent export degraded" and fails to describe it.
func (app *Uptick) writeExportDiagnosticsReportFor(exportID string, height int64, diags []ExportDiagnostic) (string, error) {
	if app.homeDir == "" {
		return "", errors.New("node home directory is not configured")
	}

	bz, err := json.MarshalIndent(ExportDiagnosticsReport{
		ExportID:   exportID,
		Status:     ExportDiagnosticsStatusDegraded,
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
		Height:     height,
		Total:      len(diags),
		Issues:     diags,
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal export diagnostics: %w", err)
	}
	bz = append(bz, '\n')

	path := filepath.Join(app.homeDir, ExportDiagnosticsFileName)
	if err := writeFileAtomic(path, bz); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// writeFileAtomic publishes data at path by writing a sibling temporary file and
// renaming it into place. The temporary file is created with 0o600 (the mode the
// report must have) and is removed if the commit does not complete, so a failed
// write leaves the directory exactly as it found it.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)

	// The temp file has to be a sibling: rename(2) is only atomic within one
	// filesystem, and the report must never be observed half-written.
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tmp := f.Name()

	committed := false
	defer func() {
		if !committed {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()

	if _, err := f.Write(data); err != nil {
		return err
	}
	// Sync before the rename: a crash between the two must not be able to
	// surface an empty or partially flushed report under the final name.
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}

	committed = true
	return nil
}

// removeStaleExportDiagnosticsReport clears the sidecar after a clean export (see
// ExportDiagnosticsReport for the presence invariant this maintains). It returns
// the failure instead of only printing it, so the caller can make it visible to
// the operator; an absent report is not a failure.
func (app *Uptick) removeStaleExportDiagnosticsReport() error {
	path := app.exportDiagnosticsReportPath()
	if path == "" {
		return nil
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale %s: %w", path, err)
	}
	return nil
}
