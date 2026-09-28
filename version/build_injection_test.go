package version

import (
	"os"
	"strings"
	"testing"
)

// The three symbols every build configuration has to keep injecting. They are
// matched by their full -X target, because that is the literal text a
// configuration must contain: -X silently ignores an unknown symbol, so a rename
// or a typo removes the value without failing any build.
var injectedSymbols = []string{
	"github.com/UptickNetwork/uptick/version.AppVersion",
	"github.com/UptickNetwork/uptick/version.GitCommit",
	"github.com/UptickNetwork/uptick/version.BuildDate",
}

// missingSymbols reports which of want a configuration no longer sets.
func missingSymbols(config string, want []string) []string {
	var missing []string
	for _, s := range want {
		if !strings.Contains(config, s) {
			missing = append(missing, s)
		}
	}
	return missing
}

// Both build configurations must inject all three values. The Makefile covers
// `make build`/`make install`, and therefore the binary CI's e2e job runs;
// .goreleaser.yml covers the released binaries, and that path is NOT exercised
// by anything else -- goreleaser only runs on a tag push -- so without this check
// a dropped symbol would go unnoticed until a release answered
// `web3_clientVersion` with "Version dev ()" again.
//
// A configuration can be read here without living in this package's own
// directory because `go test` runs every test with the package directory as the
// working directory; the same technique is used by app/export_guard_test.go and
// cmd/uptickd/testnet_flag_test.go.
func TestBuildConfigurationsInjectTheVersionSymbols(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"../Makefile", "../.goreleaser.yml"} {
		config, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if missing := missingSymbols(string(config), injectedSymbols); len(missing) > 0 {
			t.Fatalf("%s no longer injects %v: the JSON-RPC web3_clientVersion endpoint "+
				"would report the un-injected fallback for those fields", path, missing)
		}
	}
}

// The check above is only worth running if it can fail, and the way it fails is
// by naming what a configuration dropped -- so that is asserted directly rather
// than assumed.
func TestMissingSymbolsNamesWhatABuildNoLongerInjects(t *testing.T) {
	t.Parallel()

	complete := strings.Join(injectedSymbols, " -X ")
	if got := missingSymbols(complete, injectedSymbols); len(got) != 0 {
		t.Fatalf("a complete configuration was reported as missing %v", got)
	}

	// A configuration that dropped the build date: i.e. the pre-fix state, where
	// the endpoint printed "Compiled at  using Go ...".
	incomplete := strings.Replace(complete, injectedSymbols[2], "", 1)
	got := missingSymbols(incomplete, injectedSymbols)
	if len(got) != 1 || got[0] != injectedSymbols[2] {
		t.Fatalf("missingSymbols() = %v, want exactly [%s]", got, injectedSymbols[2])
	}
}
