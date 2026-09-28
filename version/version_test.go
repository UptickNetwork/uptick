package version

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	web3 "github.com/cosmos/evm/rpc/namespaces/ethereum/web3"
	evmversion "github.com/cosmos/evm/version"
)

// The release build injects AppVersion/GitCommit/BuildDate via -ldflags (see the
// Makefile and .goreleaser.yml). Without injection the package must fall back to
// a recognisable "dev" marker rather than an empty string, so an unversioned
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

// BuildDate used to be empty in every build, which is what made the endpoint
// print "Compiled at  using Go ..." with nothing between "at" and "using".
func TestVersion_BuildDateIsNeverEmpty(t *testing.T) {
	t.Parallel()

	if BuildDate == "" {
		t.Fatal("BuildDate must never be empty: an injected date, or the explicit \"unknown\" marker")
	}
}

func TestVersion_ReportsToolchain(t *testing.T) {
	t.Parallel()

	// The toolchain is reported by the cosmos/evm renderer this package
	// delegates to; asserting it here is what keeps the two from drifting apart.
	got := Version()
	if !strings.Contains(got, runtime.Version()) {
		t.Fatalf("Version() = %q, want it to report the toolchain %q", got, runtime.Version())
	}
	if !strings.Contains(got, runtime.GOARCH) {
		t.Fatalf("Version() = %q, want it to report the architecture %q", got, runtime.GOARCH)
	}
}

func TestVersion_String(t *testing.T) {
	t.Parallel()

	got := Version()
	for _, want := range []string{"Version ", AppVersion, runtime.Version(), runtime.GOARCH} {
		if !strings.Contains(got, want) {
			t.Fatalf("Version() = %q, want it to contain %q", got, want)
		}
	}
}

// Sync is the bridge between the build metadata injected into this package and
// the JSON-RPC endpoint `web3_clientVersion`, which renders the cosmos/evm
// version package instead. It is asserted against the endpoint's own service
// method rather than against the variables, so a change in which package the
// endpoint formats is caught here instead of on a deployed node.
//
// Not parallel: it mutates the package variables the other tests read.
func TestSync_PublishesTheBuildMetadataToWeb3ClientVersion(t *testing.T) {
	appVersion, gitCommit, buildDate := AppVersion, GitCommit, BuildDate
	evmAppVersion, evmGitCommit, evmBuildDate := evmversion.AppVersion, evmversion.GitCommit, evmversion.BuildDate
	t.Cleanup(func() {
		AppVersion, GitCommit, BuildDate = appVersion, gitCommit, buildDate
		evmversion.AppVersion, evmversion.GitCommit, evmversion.BuildDate = evmAppVersion, evmGitCommit, evmBuildDate
	})

	AppVersion = "v0.0.0-sync-test"
	GitCommit = "0123456789abcdef0123456789abcdef01234567"
	BuildDate = "2026-09-22T00:00:00Z"
	// The state every build of this repository used to produce: nothing was ever
	// injected into the package the endpoint reads.
	evmversion.AppVersion, evmversion.GitCommit, evmversion.BuildDate = "dev", "", ""

	Sync()

	got := web3.NewPublicAPI().ClientVersion()
	for _, want := range []string{AppVersion, GitCommit, BuildDate} {
		if !strings.Contains(got, want) {
			t.Fatalf("web3_clientVersion = %q, want it to contain the injected %q", got, want)
		}
	}
	if strings.Contains(got, "Version dev") {
		t.Fatalf("web3_clientVersion = %q still reports the un-injected fallback", got)
	}
}

// The VCS stamp is the only thing identifying a binary built without the
// release pipeline, so an injection must never be overwritten by it.
//
// Not parallel: it mutates the package variables the other tests read.
func TestFillFromBuildInfoNeverOverridesAnInjection(t *testing.T) {
	appVersion, gitCommit, buildDate := AppVersion, GitCommit, BuildDate
	t.Cleanup(func() {
		AppVersion, GitCommit, BuildDate = appVersion, gitCommit, buildDate
	})

	GitCommit = "injected-by-ldflags"
	BuildDate = "2026-09-22T00:00:00Z"
	fillFromBuildInfo()

	if GitCommit != "injected-by-ldflags" {
		t.Fatalf("GitCommit = %q, an injected commit must survive fillFromBuildInfo", GitCommit)
	}
	if BuildDate != "2026-09-22T00:00:00Z" {
		t.Fatalf("BuildDate = %q, an injected date must survive fillFromBuildInfo", BuildDate)
	}
}

func TestRevisionFromBuildSettings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		settings  []debug.BuildSetting
		wantRev   string
		wantDirty bool
	}{
		{
			name: "clean checkout",
			settings: []debug.BuildSetting{
				{Key: "-compiler", Value: "gc"},
				{Key: "vcs.revision", Value: "abcdef0123456789abcdef0123456789abcdef01"},
				{Key: "vcs.modified", Value: "false"},
			},
			wantRev: "abcdef0123456789abcdef0123456789abcdef01",
		},
		{
			name: "dirty checkout",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef0123456789abcdef0123456789abcdef01"},
				{Key: "vcs.modified", Value: "true"},
			},
			wantRev:   "abcdef0123456789abcdef0123456789abcdef01",
			wantDirty: true,
		},
		{
			// Built outside a checkout (a source tarball): vcs.modified is absent,
			// so dirtiness must default to false rather than to a missing value
			// being read as truthy.
			name: "revision without the modified flag",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef0123456789abcdef0123456789abcdef01"},
			},
			wantRev: "abcdef0123456789abcdef0123456789abcdef01",
		},
		{
			name:     "no vcs stamp at all",
			settings: []debug.BuildSetting{{Key: "-compiler", Value: "gc"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotRev, gotDirty := revisionFromBuildSettings(tc.settings)
			if gotRev != tc.wantRev || gotDirty != tc.wantDirty {
				t.Fatalf("revisionFromBuildSettings() = (%q, %v), want (%q, %v)",
					gotRev, gotDirty, tc.wantRev, tc.wantDirty)
			}
		})
	}
}
