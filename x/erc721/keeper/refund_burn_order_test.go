package keeper

import (
	"testing"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// pairIndexConvertedNFTs is the test twin of app/keepers.convertedNFTChecker:
// it answers x/collection's "is this native NFT bound to a contract token?"
// question from this module's pair index, exactly as the production wiring
// does.
type pairIndexConvertedNFTs struct{ erc721 Keeper }

func (c pairIndexConvertedNFTs) IsConvertedNFT(ctx sdk.Context, classID, nftID string) bool {
	return len(c.erc721.GetNFTPairByClassNFTID(ctx, classID, nftID)) > 0
}

// TestRefundBurnRequiresThePairMappingsGone pins WHY RefundPacketToken deletes
// the pair mappings BEFORE burning the native NFT -- the order a round-29
// report flagged as a risk ("deletes the mapping, then burns; if the burn fails
// the mapping is already lost, so the residue is undiscoverable").
//
// The order is forced, not a choice. x/collection.RemoveNFT refuses to burn an
// NFT that IsConvertedNFT reports as bound to a contract token
// (ErrNFTBoundToContract), and the production checker answers that question
// from THIS module's own pair index. Burning first would therefore fail on
// every single refund -- after the ERC721 had already been returned to the
// user -- so the module account would keep the native NFT forever and the
// packet could never converge. Deleting the binding is what makes the burn
// legal.
//
// The test wires the checker the way the app does, which is the part the other
// refund tests cannot see (their collection keeper has no checker, so a
// burn-first implementation would still pass them). Here it turns the healthy
// path into nft_burn_failed plus a surviving NFT.
func TestRefundBurnRequiresThePairMappingsGone(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)
	setupRefundableToken(t, k, ctx, owner, "kitty", "nft1", "1")

	// Wire the cross-module guard (production does this in app/keepers).
	k.nftKeeper.SetConvertedNFTChecker(pairIndexConvertedNFTs{erc721: k})

	// Sanity check of the premise: while the binding exists, the collection
	// layer refuses the burn. This is the error a burn-first implementation
	// would hit on every refund.
	_, err := k.nftKeeper.BurnNFT(ctx, &collectiontypes.MsgBurnNFT{
		DenomId: "kitty",
		Id:      "nft1",
		Sender:  types.AccModuleAddress.String(),
	})
	require.ErrorIs(t, err, collectiontypes.ErrNFTBoundToContract,
		"a bound NFT must be unbindable only by removing the pair mapping first")
	require.True(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft1"))

	// Now the real refund: ownerOf says the module holds the ERC721 (so the
	// EVM refund runs) and safeTransferFrom succeeds.
	evm := k.evmKeeper.(*fakeEVMKeeper)
	evm.seq = []seqResp{
		{ret: packOwnerOf(t, types.ModuleAddress)},
		{ret: nil},
	}

	require.NoError(t, k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	}))

	require.Empty(t, refundSkipReasons(ctx)["nft1"],
		"a healthy refund must not skip: a skip here means the burn was refused because the mapping was still present")
	require.False(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft1"),
		"the native burn must succeed, which requires the pair mappings to be gone BEFORE BurnNFT")
	require.Empty(t, k.GetTokenUIDPairByNFTUID(ctx, types.CreateNFTUID("kitty", "nft1")))
	require.True(t, hasRefundEvent(ctx), "the refund itself must still be recorded")

	// The mapping really was the only thing blocking the burn: with it gone,
	// the same call the sanity check above refused now has nothing to refuse.
	require.False(t, k.nftKeeper.IsConvertedNFT(ctx, "kitty", "nft1"))
}
