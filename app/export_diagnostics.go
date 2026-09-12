package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
type ExportDiagnosticsReport struct {
	Height int64              `json:"height"`
	Total  int                `json:"total"`
	Issues []ExportDiagnostic `json:"issues"`
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

// writeExportDiagnosticsReport persists the report into the node home
// directory and returns the path it wrote.
func (app *Uptick) writeExportDiagnosticsReport(height int64, diags []ExportDiagnostic) (string, error) {
	if app.homeDir == "" {
		return "", errors.New("node home directory is not configured")
	}

	bz, err := json.MarshalIndent(ExportDiagnosticsReport{
		Height: height,
		Total:  len(diags),
		Issues: diags,
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal export diagnostics: %w", err)
	}
	bz = append(bz, '\n')

	path := filepath.Join(app.homeDir, ExportDiagnosticsFileName)
	if err := os.WriteFile(path, bz, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// removeStaleExportDiagnosticsReport clears the sidecar after a clean export (see
// ExportDiagnosticsReport for the presence invariant this maintains).
func (app *Uptick) removeStaleExportDiagnosticsReport() {
	if app.homeDir == "" {
		return
	}

	path := filepath.Join(app.homeDir, ExportDiagnosticsFileName)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Best effort: the export itself already succeeded, and failing it
		// over a leftover diagnostics file would be a worse trade.
		fmt.Fprintf(os.Stderr, "warning: could not remove stale %s: %v\n", path, err)
	}
}
