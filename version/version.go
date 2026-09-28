// Package version holds the build metadata that identifies an uptickd binary.
//
// It exists in this module, rather than relying on the SDK's package alone,
// because the JSON-RPC endpoint `web3_clientVersion` does not read the SDK's
// variables. That endpoint is served by the cosmos/evm `web3` namespace, which
// formats github.com/cosmos/evm/version.Version() -- and nothing in this
// repository injected anything into that package: not the Makefile, not
// .goreleaser.yml, not the rbuilder image. Every release node therefore answered
// `web3_clientVersion` with "Version dev ()\nCompiled at  using Go <version>",
// so the version number, the commit and the build time were all missing, and two
// nodes built from different commits could not be told apart by a client.
//
// The fix has two halves and both are load-bearing:
//
//   - the build injects AppVersion/GitCommit/BuildDate here (Makefile ldflags,
//     .goreleaser.yml), and
//   - Sync publishes them to the cosmos/evm package, i.e. to the variables the
//     endpoint actually renders. One injected package plus one bridge is
//     deliberate: injecting the cosmos/evm symbols directly would spread the
//     same three symbols over every build configuration instead.
//
// A build that bypasses the release pipeline (`go build ./cmd/uptickd`,
// `go run`) is still identifiable: Go stamps the VCS revision into the binary,
// and fillFromBuildInfo reports it as the commit.
package version

import (
	"runtime/debug"

	evmversion "github.com/cosmos/evm/version"
)

// Set by the build with -ldflags -X; see the Makefile and .goreleaser.yml.
// The names are part of the build contract -- renaming one silently disables
// its injection, because -X on an unknown symbol is not an error.
var (
	AppVersion = ""
	GitCommit  = ""
	BuildDate  = ""
)

const (
	// devVersion marks a binary that no release build produced. It is
	// deliberately not the empty string: an unversioned binary has to be
	// recognisable as exactly that, and never look like a release.
	devVersion = "dev"

	// unknownBuildDate is what a build that injected no date reports. Go records
	// the *commit* time in its build info, never the build time, and printing a
	// commit time as "Compiled at" would be a false statement about the binary
	// -- so the gap is named instead of filled with something that looks
	// authoritative.
	unknownBuildDate = "unknown"
)

func init() {
	if AppVersion == "" {
		AppVersion = devVersion
	}
	fillFromBuildInfo()
	Sync()
}

// fillFromBuildInfo identifies builds that bypass the release pipeline. Both
// fields are only filled when the build injected nothing, so an explicit
// injection always wins.
func fillFromBuildInfo() {
	if GitCommit == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info != nil {
			if rev, dirty := revisionFromBuildSettings(info.Settings); rev != "" {
				GitCommit = rev
				if dirty {
					GitCommit += "-dirty"
				}
			}
		}
	}
	if BuildDate == "" {
		BuildDate = unknownBuildDate
	}
}

// revisionFromBuildSettings reads the VCS stamp Go adds to a binary built from a
// checkout (`vcs.revision`, `vcs.modified`; on by default since Go 1.18). It is
// split out from the reader so the parsing can be exercised without a real build
// info.
func revisionFromBuildSettings(settings []debug.BuildSetting) (rev string, dirty bool) {
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	return rev, dirty
}

// Sync publishes this package's build metadata to the cosmos/evm version
// package, which is what the JSON-RPC endpoint `web3_clientVersion` renders.
//
// It runs from init() and is called again explicitly from the CLI entry point
// (cmd/uptickd/root.go): init() only executes while this package is linked, so
// the explicit call is what keeps the bridge from disappearing together with an
// import someone tidies away. Sync is idempotent, and a field is only copied
// when this package has a value for it, so an injection straight into the
// cosmos/evm variables still wins when this package is not the injected one.
func Sync() {
	if AppVersion != "" {
		evmversion.AppVersion = AppVersion
	}
	if GitCommit != "" {
		evmversion.GitCommit = GitCommit
	}
	if BuildDate != "" {
		evmversion.BuildDate = BuildDate
	}
}

// Version returns the string that identifies this binary, and it is the same
// string the JSON-RPC endpoint `web3_clientVersion` serves. It deliberately
// delegates to the cosmos/evm renderer instead of defining a second format here:
// two renderers would eventually disagree, and the endpoint is the surface
// operators actually read.
func Version() string {
	return evmversion.Version()
}
