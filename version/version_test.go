package version

import (
	"runtime"
	"strings"
	"testing"
)

// The release build injects AppVersion/GitCommit via -ldflags (see the Makefile
// and .goreleaser.yml). Without injection the package must fall back to a
// recognisable "dev" marker rather than an empty string, so an unversioned
// binary can never be mistaken for a release build.
func TestVersion_DefaultsToDevWhenNotInjected(t *testing.T) {
	t.Parallel()

	t.Logf("AppVersion=%q GitCommit=%q BuildDate=%q", AppVersion, GitCommit, BuildDate)

	if AppVersion == "" {
		t.Fatal("AppVersion must never be empty; init() falls back to \"dev\"")
	}
	if AppVersion == "dev" {
		// Not a release build: this is what an unversioned `go build` produces.
		requireDevMarker(t)
		return
	}
	// Injected by the release build: it must look like a version, not a leftover.
	if strings.HasPrefix(AppVersion, "-") || strings.TrimSpace(AppVersion) != AppVersion {
		t.Fatalf("injected AppVersion %q is malformed", AppVersion)
	}
}

func requireDevMarker(t *testing.T) {
	t.Helper()
	if got := Version(); !strings.Contains(got, "dev") {
		t.Fatalf("Version() = %q, want it to contain the dev marker", got)
	}
}

func TestVersion_ReportsToolchain(t *testing.T) {
	t.Parallel()

	if GoVersion != runtime.Version() {
		t.Fatalf("GoVersion = %q, want %q", GoVersion, runtime.Version())
	}
	if GoArch != runtime.GOARCH {
		t.Fatalf("GoArch = %q, want %q", GoArch, runtime.GOARCH)
	}
}

func TestVersion_String(t *testing.T) {
	t.Parallel()

	got := Version()
	for _, want := range []string{"Version ", AppVersion, GoVersion, GoArch} {
		if !strings.Contains(got, want) {
			t.Fatalf("Version() = %q, want it to contain %q", got, want)
		}
	}
}
