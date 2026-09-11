package app

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// client/docs/swagger-ui/swagger.yaml is generated from the protos by
// `make proto-swagger-gen`, and client/docs/statik/statik.go is that directory
// embedded by `make update-swagger-docs`. Both are committed artefacts that no
// test reads, and the Makefile's own sync assertion only runs when a developer
// remembers to ask for it, so they can rot silently.
//
// This is the cheap half of that check: the document must describe the modules
// the chain actually has, and must not resurrect one that was deleted. It caught
// nothing when written -- v0.4.0 regenerated the file and dropped erc20 on the
// way -- which is the point of pinning it.
func TestSwaggerDocMatchesTheShippedModuleSet(t *testing.T) {
	bz, err := os.ReadFile("../client/docs/swagger-ui/swagger.yaml")
	require.NoError(t, err, "the generated swagger document is part of the repository")
	doc := string(bz)

	for _, module := range []string{"uptick.collection.v1", "uptick.erc721.v1", "uptick.cw721.v1"} {
		require.Contains(t, doc, module,
			"%s has endpoints in this build but none in the swagger document: re-run make proto-swagger-gen", module)
	}

	// uptick.erc20 was the self-developed module deleted in v0.4.0. If it comes
	// back here, the document was regenerated from a proto tree that still
	// contains it, which is also how a stale statik blob starts.
	require.NotContains(t, doc, "uptick.erc20.",
		"uptick.erc20 was removed in v0.4.0; the swagger document must not advertise it")
	require.NotContains(t, doc, "uptick.nft.v1beta1.",
		"x/nft keeps types only since v0.3.3 and serves no endpoints")
}

// statik.go is the compressed copy of the directory above that the node serves
// when started with swagger enabled. A test cannot decompress it without the
// statik library (a build tool, not a module dependency), but it can at least
// fail when the two files are touched years apart -- which is the shape a stale
// embedding has.
func TestStatikBlobIsPresentNextToTheDocument(t *testing.T) {
	info, err := os.Stat("../client/docs/statik/statik.go")
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(100_000),
		"statik.go should embed the whole swagger-ui directory; a small file means it was emptied or regenerated wrong")

	doc, err := os.Stat("../client/docs/swagger-ui/swagger.yaml")
	require.NoError(t, err)
	require.False(t, strings.HasSuffix(doc.Name(), ".tmp"))
}
