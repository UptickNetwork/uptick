package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenesisEntryPointsBootstrapDenomBeforeDefaults guards the *ordering* of
// the default-denom bootstrap, which is the actual defect being fixed: both
// genesis entry points must call app.PrepareDefaultGenesisDenom before the
// BasicManager materialises any module default genesis, because mint's
// MintDenom, crisis' ConstantFee and gov's deposit minimums are copied out of
// sdk.DefaultBondDenom at that moment and the SDK zero value is "stake".
//
// The assertion is made on the source rather than by running the commands on
// purpose. initGenFiles is unexported, a second app cannot be built in this
// test binary (see app/shared_testapp_test.go), and a full `testnet init-files`
// run needs a keyring and a writable node home -- all for a defect whose
// failure mode is a silently wrong genesis rather than an error. The
// behavioural half of this guard lives in
// app.TestPrepareDefaultGenesisDenomDrivesDerivedDefaults.
func TestGenesisEntryPointsBootstrapDenomBeforeDefaults(t *testing.T) {
	const (
		bootstrapCall = "PrepareDefaultGenesisDenom("
		defaultsCall  = ".DefaultGenesis("
	)

	for _, file := range []string{"init.go", "testnet.go"} {
		t.Run(file, func(t *testing.T) {
			raw, err := os.ReadFile(file)
			require.NoError(t, err)

			lines := codeLines(string(raw))

			bootstrapLine := firstMatch(lines, bootstrapCall)
			require.NotEqual(t, -1, bootstrapLine,
				"%s never calls %s, so a genesis built through this entry point keeps the SDK zero denom", file, bootstrapCall)

			defaultsLine := firstMatch(lines, defaultsCall)
			require.NotEqual(t, -1, defaultsLine,
				"%s no longer materialises a default genesis; delete this guard if the entry point changed shape", file)

			require.Less(t, bootstrapLine, defaultsLine,
				"%s calls %s on line %d, after the first %s on line %d: the module defaults derived from sdk.DefaultBondDenom (mint MintDenom, crisis ConstantFee) would be built against the SDK zero denom",
				file, bootstrapCall, bootstrapLine+1, defaultsCall, defaultsLine+1)
		})
	}
}

// codeLines returns the source with line comments stripped, so a marker that
// only appears in prose cannot satisfy (or defeat) the ordering assertion.
func codeLines(src string) []string {
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx != -1 {
			lines[i] = line[:idx]
		}
	}
	return lines
}

// firstMatch returns the index of the first line containing needle, or -1.
func firstMatch(lines []string, needle string) int {
	for i, line := range lines {
		if strings.Contains(line, needle) {
			return i
		}
	}
	return -1
}
