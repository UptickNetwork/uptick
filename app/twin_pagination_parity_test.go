package app

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The three page-request shapers are deliberately separate files in three
// keeper packages -- x/erc721 and x/cw721 are twin modules that this repository
// keeps as independent copies by convention, and x/collection is the reference
// implementation they were brought in line with. Deliberate duplication only
// stays safe while it is actually duplicated, so this guard compares the code
// (comments and formatting normalised away) rather than trusting a reviewer to
// remember.
//
// A legitimate divergence -- say cw721 deciding a different limit is correct --
// has to update this guard and say why in the commit, which is the point: the
// failure mode it prevents is a client call being rejected by one module and
// silently scanned by another, which is exactly how TokenPairs behaved before
// both twins were aligned with x/collection.
func normalizedPaginationSource(t *testing.T, path string) string {
	t.Helper()

	bz, err := os.ReadFile(path)
	require.NoError(t, err)

	var lines []string
	for _, line := range strings.Split(string(bz), "\n") {
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		// Collapse indentation so gofmt-level differences do not matter.
		line = strings.Join(strings.Fields(line), " ")
		if line == "" || strings.HasPrefix(line, "package ") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func TestTwinPaginationShapersStayInSync(t *testing.T) {
	reference := normalizedPaginationSource(t, "../x/collection/keeper/pagination.go")
	require.Contains(t, reference, "shapePageRequest",
		"the reference shaper moved; point this guard at its new home instead of deleting it")

	for _, path := range []string{
		"../x/erc721/keeper/pagination.go",
		"../x/cw721/keeper/pagination.go",
	} {
		got := normalizedPaginationSource(t, path)
		require.Contains(t, got, "shapePageRequest", "%s no longer shapes page requests", path)
		require.Equal(t, reference, got,
			"%s drifted from x/collection's page-request contract: the same client call must be "+
				"rejected or accepted identically on every module", path)
	}
}
