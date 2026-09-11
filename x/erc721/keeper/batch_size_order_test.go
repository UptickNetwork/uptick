package keeper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ConvertNFT used to bound the batch size only inside convertCosmos2Evm, which
// runs at the very end — after RegisterNFT → GetContractAddressAndTokenIds has
// already deployed an ERC721 contract for an unregistered class. The SDK rolls
// the handler back, so nothing corrupt is committed, but a batch that can never
// succeed still burns a whole contract deployment's worth of gas and store
// writes first. x/cw721 validates before deploying; this pins the same order on
// the erc721 side.
//
// Asserted on the source because reaching the deployment path in a test means
// standing up the EVM keeper, a token pair and a real deployment, while the
// property being guarded — call order — is static.
func TestConvertNFTValidatesBatchSizeBeforeDeploying(t *testing.T) {
	body := functionSource(t, "msg_server.go", "func (k Keeper) ConvertNFT(")
	require.NotEmpty(t, body)

	batchAt := firstLineContaining(body, "maxERC721BatchSize")
	require.NotEqual(t, -1, batchAt,
		"ConvertNFT must bound the batch size (its final check lives in convertCosmos2Evm)")

	deployAt := firstLineContaining(body, "GetContractAddressAndTokenIds")
	require.NotEqual(t, -1, deployAt,
		"ConvertNFT is expected to resolve the contract through GetContractAddressAndTokenIds")

	require.Less(t, batchAt, deployAt,
		"the batch-size check (function line %d) must run before contract resolution (function line %d), which deploys for an unregistered class",
		batchAt+1, deployAt+1)
}

// firstLineContaining returns the index within body of the first line that
// mentions needle, or -1. Line comments are stripped so prose cannot satisfy
// (or defeat) an ordering assertion.
func firstLineContaining(body []string, needle string) int {
	for i, line := range body {
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		if strings.Contains(line, needle) {
			return i
		}
	}
	return -1
}
