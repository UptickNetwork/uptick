package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Query commands must build their client context with GetClientQueryContext:
// GetClientTxContext does NOT read the height flag, so `--height` is silently
// ignored and a historical query returns the LATEST state. That is worse than
// an error -- an operator reconciling a past block reads today's numbers with
// no warning at all.
//
// x/erc721 and x/cw721 always got this right; x/collection's six query
// commands did not (round 22). The rule is checked across every module rather
// than per file, because "one twin was fixed and the other was not" is this
// repository's most repeated defect.
func TestQueryCommandsUseQueryContext(t *testing.T) {
	// The package dir is x/collection/client/cli; three levels up is x/.
	root := filepath.Join("..", "..", "..")

	checked := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Base(path) != "query.go" {
			return nil
		}
		if !strings.Contains(filepath.ToSlash(path), "/client/cli/") {
			return nil
		}

		src, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		checked++

		require.NotContains(t, string(src), "GetClientTxContext",
			"%s: query commands must use client.GetClientQueryContext, otherwise --height is silently ignored", path)
		return nil
	})
	require.NoError(t, err)

	// Guard the guard: if the walk ever stops finding the module CLI packages,
	// the assertions above would pass vacuously.
	require.GreaterOrEqual(t, checked, 3,
		"expected to check the collection/erc721/cw721 query.go files")
}
