package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetNftDatasToleratesShortSavedSlice pins the boundary guard that
// x/cw721 has always had and x/erc721 did not.
//
// getNftDatas walks the caller-supplied ids and pairs them by index with the
// saved mappings. Today both production callers happen to build nftSaveds with
// exactly the same length as nftPairOrgs (one entry appended per input id), so
// the access is in range -- but that is a property of the callers, not of this
// function, and the twin module guards it explicitly. A short slice must
// degrade to "no saved value" (the fallback getNftData already implements for
// an empty string) instead of panicking the node.
//
// This is the erc721 half of a twin-module asymmetry; the cw721 twin of this
// behaviour is covered by the same guard in x/cw721/keeper/nft_data.go.
func TestGetNftDatasToleratesShortSavedSlice(t *testing.T) {
	rets, err := getNftDatas(nil, []string{"nft1", "nft2", "nft3"}, []string{"0xaa"}, 2)
	require.NoError(t, err)
	require.Len(t, rets, 3)

	// The covered element uses the saved value...
	require.Equal(t, "0xaa", rets[0])
	// ...and the uncovered ones derive a value instead of panicking.
	require.NotEmpty(t, rets[1])
	require.NotEmpty(t, rets[2])
}

// TestGetNftDatasToleratesNilSavedSlice is the same contract for the callers
// that legitimately pass no saved state at all (EVM -> Cosmos conversion).
func TestGetNftDatasToleratesNilSavedSlice(t *testing.T) {
	rets, err := getNftDatas(nil, []string{"nft1", "nft2"}, nil, 2)
	require.NoError(t, err)
	require.Len(t, rets, 2)
	require.NotEmpty(t, rets[0])
	require.NotEmpty(t, rets[1])
}
