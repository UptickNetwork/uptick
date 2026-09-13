package main

// The exit code of `uptickd export` is the only channel that survives the
// command's own success -- by the time it is computed the genesis is already on
// stdout (or in --output-document) -- so it is part of the CLI's contract and is
// pinned here rather than only by a throwaway probe.
//
// os.Exit cannot be observed inside the process that calls it, so the reporting
// step is exercised in a CHILD PROCESS THAT RE-USES THIS TEST BINARY (the
// standard TestHelperProcess idiom). That keeps the check deterministic and
// dependency-free: no network, no live chain, no pre-built binary, and nothing
// written to the working tree. It deliberately does NOT go through a real
// `export` invocation: a node home that never ran InitChain aborts inside the
// SDK modules' ExportGenesis, which is unrelated to this contract.

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/app"
)

const (
	// exportExitHelperEnv selects the child's scenario. Unset means "this is the
	// ordinary test run", and the helper does nothing.
	exportExitHelperEnv = "UPTICK_EXPORT_EXIT_HELPER"

	exportExitHelperClean    = "clean"
	exportExitHelperRecorded = "degraded-recorded"
	exportExitHelperLost     = "degraded-lost"
	// exportExitHelperUnknown makes the helper fail loudly. Without it, a broken
	// env passthrough or a stale -test.run pattern would make the child exit 0
	// and the "clean" assertion below would pass for the wrong reason.
	exportExitHelperUnknown     = "unknown-mode"
	exportExitHelperUnknownCode = 99
)

// TestExportExitCodeHelperProcess is the child half of the checks below. Run as
// part of the ordinary suite it is a no-op, because no scenario is selected.
func TestExportExitCodeHelperProcess(t *testing.T) {
	mode := os.Getenv(exportExitHelperEnv)
	if mode == "" {
		return
	}

	switch mode {
	case exportExitHelperClean:
		// A fully accounted-for export: nothing to report, no special code.
		_ = finishExport(os.Stdout, app.ExportDiagnosticsStatus{
			ExportID: "helper-clean",
			Height:   2,
		})

	case exportExitHelperRecorded:
		// Degraded, but the degradations reached the disk: loud, not fatal.
		_ = finishExport(os.Stdout, app.ExportDiagnosticsStatus{
			ExportID:      "helper-recorded",
			Height:        2,
			Issues:        3,
			Degraded:      true,
			ReportPath:    "/node/export-issues.json",
			ReportWritten: true,
		})

	case exportExitHelperLost:
		// Degraded and the report could not be written: the one case that must
		// change the process exit code.
		_ = finishExport(os.Stdout, app.ExportDiagnosticsStatus{
			ExportID:   "helper-lost",
			Height:     2,
			Issues:     3,
			Degraded:   true,
			ReportPath: "/node/export-issues.json",
			WriteErr:   errors.New("file too large"),
		})

	default:
		// finishExport is deliberately NOT called: this mode only proves the
		// plumbing reached the helper.
		_, _ = os.Stdout.WriteString("helper: unrecognised mode " + mode + "\n")
		os.Exit(exportExitHelperUnknownCode)
	}

	// Reached only when finishExport did not terminate the process, i.e. when
	// the export needs no pipeline-visible signal.
	os.Exit(0)
}

// runExportHelper runs the helper half in a child process and returns its exit
// code together with its combined output.
func runExportHelper(t *testing.T, mode string) (int, string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestExportExitCodeHelperProcess$")
	cmd.Env = append(os.Environ(), exportExitHelperEnv+"="+mode)

	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(out)
	}

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr,
		"the helper child failed to start or was killed; output: %s", out)
	return exitErr.ExitCode(), string(out)
}

// The plumbing first: an unrecognised mode must come back with the helper's own
// code. If the child never actually ran the helper (typo in -test.run, env not
// passed through, binary not rebuilt) this returns 0 and fails here, so the
// contract tests below cannot pass vacuously.
func TestExportExitCodeHelperIsWired(t *testing.T) {
	code, out := runExportHelper(t, exportExitHelperUnknown)
	require.Equal(t, exportExitHelperUnknownCode, code, "child output: %s", out)
	require.Contains(t, out, "unrecognised mode")
}

// The contract, end to end through a real process exit.
func TestExportExitCodeContract(t *testing.T) {
	cases := []struct {
		name         string
		mode         string
		wantCode     int
		wantMarker   string
		wantMentions string
	}{
		{
			name:     "clean_export_is_silent_and_succeeds",
			mode:     exportExitHelperClean,
			wantCode: 0,
		},
		{
			name:         "degraded_with_report_warns_but_still_succeeds",
			mode:         exportExitHelperRecorded,
			wantCode:     0,
			wantMarker:   "DEGRADED",
			wantMentions: "helper-recorded",
		},
		{
			name:         "degraded_without_report_exits_with_the_degraded_code",
			mode:         exportExitHelperLost,
			wantCode:     app.ExportDiagnosticsDegradedExitCode,
			wantMarker:   "DEGRADED",
			wantMentions: "helper-lost",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out := runExportHelper(t, tc.mode)
			require.Equal(t, tc.wantCode, code, "child output: %s", out)

			if tc.wantMarker == "" {
				require.NotContains(t, out, "DEGRADED")
				require.NotContains(t, out, "WARNING")
				return
			}
			require.Contains(t, out, tc.wantMarker)
			require.Contains(t, out, tc.wantMentions,
				"the marker must name the export it is about, so the operator can correlate it")
		})
	}
}

// The degraded code has to be usable as a gate, which means it must not be
// confusable with "the export itself failed" (cobra / main.go exit 1).
func TestExportExitCodeIsDistinguishableFromExportFailure(t *testing.T) {
	code, out := runExportHelper(t, exportExitHelperLost)

	require.Equal(t, 3, code, "child output: %s", out)
	require.NotEqual(t, 1, code,
		"reusing 1 would make a usable genesis look like a failed export")
	require.NotZero(t, code, "0 means the export is fully accounted for")
	require.Equal(t, app.ExportDiagnosticsDegradedExitCode, code,
		"the value is quoted by runbooks and scripts; it must stay stable")
}
