package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// skipEventFor returns the refund_packet_token_skip event for one NFT id.
func skipEventFor(t *testing.T, ctx sdk.Context, nftID string) *sdk.Event {
	t.Helper()
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != types.EventTypeRefundPacketTokenSkip {
			continue
		}
		if attributeMap(ev)[types.AttributeKeyNFTID] == nftID {
			found := ev
			return &found
		}
	}
	t.Fatalf("no refund_packet_token_skip event for %s", nftID)
	return nil
}

// TestRefundPacketToken_NFTAlreadyGoneKeepsPairsAndFlagsTheEvent pins the
// attribute that makes the skip event self-describing.
//
// x/erc721 and x/cw721 publish the same event type with the same
// `nft_already_gone` reason for opposite situations: here the pair mappings are
// kept because the ERC721 may still be escrowed, so a human has to look;
// x/cw721 deletes them because its refund side is already settled, so nothing
// needs doing. Telling the two apart required knowing that asymmetry, which
// meant reading both modules — so the event now says it itself, and x/cw721
// omits the attribute (absence means false).
//
// The mapping assertion is the other half: if a later change starts cleaning up
// on this path, an escrowed ERC721 becomes undiscoverable, and this is the only
// test that would notice.
func TestRefundPacketToken_NFTAlreadyGoneKeepsPairsAndFlagsTheEvent(t *testing.T) {
	k, ctx, owner := setupConvertKeeper(t)

	// A registered pair, a recorded refund receiver, and no native NFT — the
	// exact state this branch exists for. nft9 is never minted.
	require.NoError(t, k.SetNFTPairs(ctx, registerTestContract, "9", "kitty", "nft9"))
	k.SetEvmRefundReceiver(ctx, registerTestContract, []string{"nft9"}, []string{"9"},
		common.BytesToAddress(owner.Bytes()).Hex())
	require.False(t, k.nftKeeper.HasNFT(ctx, "kitty", "nft9"))

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft9"},
	})
	require.NoError(t, err)

	require.Equal(t, "nft_already_gone", refundSkipReasons(ctx)["nft9"])

	skip := skipEventFor(t, ctx, "nft9")
	require.Equal(t, "true", attributeMap(*skip)[types.AttributeKeyPairsRetained],
		"erc721 must flag the skip so an operator can triage without cross-module knowledge")

	require.NotEmpty(t, k.GetTokenUIDPairByNFTUID(ctx, types.CreateNFTUID("kitty", "nft9")),
		"the pair mapping must survive: the ERC721 may still be escrowed, and without the mapping it is undiscoverable")
	require.False(t, hasRefundEvent(ctx), "nothing was refunded, so no refund event may be emitted")
}
