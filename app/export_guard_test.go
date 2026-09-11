package app

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The zero-height export path runs inside a CLI command, never inside a
// consensus handler, so a failure there is always reportable as an error.
// Panicking crashes the node process mid-export and leaves the operator with a
// stack trace instead of a diagnosis. The validator-reinitialisation loop used
// to panic at five separate failure points; it now captures the first failure
// and returns it, matching the errors.Join style the commission and reward
// passes above already use.
//
// The assertion is on the source because every reachable trigger needs corrupt
// validator or distribution state to be planted first. The failure mode is a
// crash, so a future panic anywhere in this file is worth failing the build
// over -- not just in the loop that was fixed.
func TestZeroHeightExportDoesNotPanic(t *testing.T) {
	src, err := os.ReadFile("export.go")
	require.NoError(t, err)

	for i, line := range strings.Split(string(src), "\n") {
		// Strip line comments so prose about panicking cannot trip the guard.
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		require.NotContains(t, line, "panic(",
			"export.go:%d panics; the export path must return an error so the operator can see what failed", i+1)
	}
}
