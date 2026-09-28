package main

import (
	"strings"
	"testing"

	web3 "github.com/cosmos/evm/rpc/namespaces/ethereum/web3"
	evmversion "github.com/cosmos/evm/version"
	"github.com/stretchr/testify/require"

	uptickversion "github.com/UptickNetwork/uptick/version"
)

// The JSON-RPC endpoint `web3_clientVersion` is served by the cosmos/evm `web3`
// namespace, which formats github.com/cosmos/evm/version -- a package no build in
// this repository injects into. Every node therefore answered with
// "Version dev ()\nCompiled at  using Go ...": the version number, the commit and
// the build time were all missing, and two binaries built from different commits
// were indistinguishable over JSON-RPC, which is exactly the surface an operator
// reaches for when debugging a deployed chain.
//
// The bridge is uptickversion.Sync(). The package's init() already calls it, so
// what this test is built to catch is the *explicit* call in NewRootCmd
// disappearing -- that one survives an import being tidied away, and it is the
// reason the defect cannot come back silently. The cosmos/evm variables are
// therefore reset to their un-injected state first: after that, anything that
// fills them in has to have come from the command tree.
func TestRootCmdPublishesBuildMetadataToWeb3ClientVersion(t *testing.T) {
	appVersion, gitCommit, buildDate := uptickversion.AppVersion, uptickversion.GitCommit, uptickversion.BuildDate
	evmAppVersion, evmGitCommit, evmBuildDate := evmversion.AppVersion, evmversion.GitCommit, evmversion.BuildDate
	t.Cleanup(func() {
		// Restore both packages together: restoring only one would leave the
		// other reporting this test's values to the next test in the package.
		uptickversion.AppVersion, uptickversion.GitCommit, uptickversion.BuildDate = appVersion, gitCommit, buildDate
		evmversion.AppVersion, evmversion.GitCommit, evmversion.BuildDate = evmAppVersion, evmGitCommit, evmBuildDate
	})

	uptickversion.AppVersion = "v0.0.0-wiring-test"
	uptickversion.GitCommit = "0123456789abcdef0123456789abcdef01234567"
	uptickversion.BuildDate = "2026-09-22T00:00:00Z"
	evmversion.AppVersion, evmversion.GitCommit, evmversion.BuildDate = "dev", "", ""

	// NewRootCmd returns no error and cannot return nil (it panics when the
	// command tree it is built from is incomplete), so a panic is the failure
	// mode to catch here.
	require.NotPanics(t, func() { NewRootCmd() },
		"NewRootCmd must build a root: the build metadata is published from it")

	got := web3.NewPublicAPI().ClientVersion()
	for _, want := range []string{
		"v0.0.0-wiring-test",
		"0123456789abcdef0123456789abcdef01234567",
		"2026-09-22T00:00:00Z",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("web3_clientVersion = %q, want NewRootCmd to publish %q to it", got, want)
		}
	}
	if strings.Contains(got, "Version dev") {
		t.Fatalf("web3_clientVersion = %q still reports the un-injected fallback: "+
			"NewRootCmd no longer calls uptickversion.Sync()", got)
	}
}
