package app

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// The collection module's burn guard is opt-in wiring: x/collection cannot see
// the erc721/cw721 pair stores and may not import those modules (they are built
// from the collection keeper), so app/keepers/keepers.go injects the check
// after they exist.
//
// An unwired checker fails silently rather than loudly -- RemoveNFT simply goes
// back to burning the native half of a converted NFT and stranding the escrowed
// contract half -- so the wiring itself is what gets tested here.
func TestConvertedNFTCheckerIsWired(t *testing.T) {
	app, ctx := sharedTestApp(t)

	const (
		classID  = "wiredclass"
		nftID    = "wired-nft"
		contract = "0x00000000000000000000000000000000000000A1"
	)

	// Nothing is bound yet.
	require.False(t, app.NFTKeeper.IsConvertedNFT(ctx, classID, nftID))

	require.NoError(t, app.Erc721Keeper.SetNFTPairs(
		ctx, common.HexToAddress(contract).Hex(), "7", classID, nftID))

	require.True(t, app.NFTKeeper.IsConvertedNFT(ctx, classID, nftID),
		"the collection keeper must observe the erc721 pair store through the injected checker")

	// A sibling token of the same class has no binding of its own.
	require.False(t, app.NFTKeeper.IsConvertedNFT(ctx, classID, "sibling-nft"))
}
