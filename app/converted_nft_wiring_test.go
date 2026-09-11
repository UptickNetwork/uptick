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

// TestInternftBurnGuardIsWired pins the ICS-721 half of the same wiring.
//
// app/keepers builds the ICS-721 adapter before the checker can exist, so the
// adapter holds a *copy* of the collection keeper and reaches the guard through
// it. If someone turned Keeper.convertedNFTs back into a plain field, that copy
// would keep a nil checker and InterNftKeeper.Burn would go back to destroying
// bound NFTs -- silently, with every other test still green. Only the app can
// observe the real wiring, so it is pinned here.
func TestInternftBurnGuardIsWired(t *testing.T) {
	app, ctx := sharedTestApp(t)

	const (
		classID  = "wired-ics721-class"
		nftID    = "wired-ics721-nft"
		contract = "0x00000000000000000000000000000000000000B2"
	)

	require.False(t, app.InterNftKeeper.IsConvertedNFT(ctx, classID, nftID),
		"nothing is bound yet")

	require.NoError(t, app.Erc721Keeper.SetNFTPairs(
		ctx, common.HexToAddress(contract).Hex(), "9", classID, nftID))

	require.True(t, app.InterNftKeeper.IsConvertedNFT(ctx, classID, nftID),
		"the ICS-721 adapter must see the pair store through the collection keeper it copied")

	// Negative control: the same predicate is not a blanket true.
	require.False(t, app.InterNftKeeper.IsConvertedNFT(ctx, classID, "sibling-nft"))
}
