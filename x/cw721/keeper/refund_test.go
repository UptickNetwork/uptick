package keeper

import (
	"testing"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
)

// refundAttr flattens an event's attributes for stable lookups.
func refundAttr(event sdk.Event) map[string]string {
	out := make(map[string]string, len(event.Attributes))
	for _, attr := range event.Attributes {
		out[attr.Key] = attr.Value
	}
	return out
}

func findEvent(events sdk.Events, eventType string) *sdk.Event {
	for i := range events {
		if events[i].Type == eventType {
			return &events[i]
		}
	}
	return nil
}

// moveNFTToModule transfers the native NFT into the cw721 module account so
// RefundPacketToken is authorised to burn it.
func moveNFTToModule(t *testing.T, k Keeper, ctx sdk.Context, from sdk.AccAddress, classID, nftID string) {
	t.Helper()
	nft, err := k.nftKeeper.GetNFT(ctx, classID, nftID)
	require.NoError(t, err)

	_, err = k.nftKeeper.TransferNFT(ctx, &collectiontypes.MsgTransferNFT{
		DenomId:   classID,
		Id:        nftID,
		Name:      nft.GetName(),
		URI:       nft.GetURI(),
		Data:      nft.GetData(),
		UriHash:   nft.GetURIHash(),
		Sender:    from.String(),
		Recipient: cw721types.AccModuleAddress.String(),
	})
	require.NoError(t, err)
}

func TestRefundPacketToken_MissingPair(t *testing.T) {
	k, ctx := setupKeeper(t)
	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.ErrorIs(t, err, cw721types.ErrTokenPairNotFound)
}

func TestRefundPacketToken_MissingReceiver(t *testing.T) {
	k, ctx, _, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	// The module still escrows the CW721, so the refund path is taken and it
	// requires a receiver recorded at send time.
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.ErrorIs(t, err, errortypes.ErrInvalidAddress)
}

// When the module account no longer owns the CW721 the refund must be skipped
// instead of returning an error (an error would strand the IBC packet).
func TestRefundPacketToken_SkipsWhenModuleDoesNotOwnToken(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	// The CW721 is held by somebody else: already refunded, never escrowed, or
	// moved on after the failed transfer.
	wasm.setOwner(contract, "1", owner.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "a non-escrowed token must be skipped, not abort the IBC callback")

	// A skip event explains why nothing was transferred.
	skip := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip)
	require.NotNil(t, skip, "expected a %s event", cw721types.EventTypeRefundPacketTokenSkip)
	require.Equal(t, "owner_is_not_module_account", refundAttr(*skip)["reason"])
	require.Equal(t, "1", refundAttr(*skip)[cw721types.AttributeKeyCW721TokenID])

	// No refund event was emitted.
	require.Nil(t, findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketToken))

	// The token must not have moved to an arbitrary receiver.
	require.Equal(t, owner.String(), wasm.ownerOf(contract, "1"))

	// Cleanup still happens: the NFT is burned and the mappings released.
	require.Empty(t, k.GetNFTPairByContractTokenID(ctx, contract, "1"))
	require.Empty(t, k.GetTokenUIDPairByNFTUID(ctx, cw721types.CreateNFTUID("kitty", "nft1")))
	_, err = k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.Error(t, err, "the escrowed NFT must be burned even when no CW721 is refunded")
}

func TestRefundPacketToken_RefundsWhenModuleOwnsToken(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err)

	// The CW721 went back to the original owner.
	require.Equal(t, owner.String(), wasm.ownerOf(contract, "1"))

	// The event reports the CW721 token id (not the NFT id) and the receiver.
	ev := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketToken)
	require.NotNil(t, ev)
	attrs := refundAttr(*ev)
	require.Equal(t, "1", attrs[cw721types.AttributeKeyCW721TokenID])
	require.Equal(t, "nft1", attrs[cw721types.AttributeKeyNFTID])
	require.Equal(t, contract, attrs[cw721types.AttributeKeyCW721Token])
	require.Equal(t, owner.String(), attrs[cw721types.AttributeKeyReceiver])

	// State is cleaned up.
	require.Empty(t, k.GetNFTPairByContractTokenID(ctx, contract, "1"))
	require.Empty(t, k.GetCwAddressByContractTokenId(ctx, contract, "1"))
	_, err = k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.Error(t, err)
}

// A token the CW721 contract does not know (never escrowed for the module, or
// minted straight to a user) must be skipped rather than abort the refund.
func TestRefundPacketToken_SkipsWhenOwnerUnknown(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	// No owner recorded: as far as the contract is concerned the token is absent.
	wasm.setOwner(contract, "1", "")
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "an unknown owner must skip, not abort the IBC callback")

	require.NotNil(t, findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip))
	require.Empty(t, wasm.ownerOf(contract, "1"))
	require.Empty(t, k.GetNFTPairByContractTokenID(ctx, contract, "1"))
}

func TestModuleOwnsCW721(t *testing.T) {
	t.Parallel()

	require.True(t, moduleOwnsCW721(cw721types.AccModuleAddress.String()))
	require.False(t, moduleOwnsCW721("uptick1someoneelse"))
	require.False(t, moduleOwnsCW721(""))
}

// Defense-in-depth: if the native NFT is already gone when the refund runs,
// the burn step must be skipped instead of returning an error.
func TestRefundPacketToken_SkipsWhenNativeNFTAlreadyGone(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())

	// Simulate the NFT being gone before the burn step. The cw721 side
	// still thinks the module escrows it; only the native side is empty.
	require.NoError(t, k.nftKeeper.NFTkeeper().Burn(ctx, "kitty", "nft1"))

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "a missing NFT must skip the burn, not abort the IBC callback")

	// The skip event carries the reason so operators can distinguish this
	// case from the cw721-side skip.
	skip := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip)
	require.NotNil(t, skip)
	require.Equal(t, "nft_already_gone", refundAttr(*skip)["reason"])
}

// Defense-in-depth: the native NFT exists but is owned by a
// non-module address (e.g. already refunded on a parallel path). Burn must
// be skipped instead of returning an error from BurnNFT.
func TestRefundPacketToken_SkipsWhenNativeNFTOwnerIsNotModule(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	require.NoError(t, k.SetNFTPairs(ctx, contract, "1", "kitty", "nft1"))
	moveNFTToModule(t, k, ctx, owner, "kitty", "nft1")
	// cw721 side happy; the native NFT moved elsewhere (a parallel refund path).
	wasm.setOwner(contract, "1", cw721types.AccModuleAddress.String())
	k.SetCwAddressByContractTokenId(ctx, contract, "1", owner.String())
	// Re-acquire from module and re-issue to a fresh user via the underlying
	// nft keeper so the collection-level state is updated without going
	// through the message-server path.
	require.NoError(t, k.nftKeeper.NFTkeeper().Transfer(ctx, "kitty", "nft1", owner))

	err := k.RefundPacketToken(ctx, ibcnfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "an NFT not held by module must skip the burn")

	sk := findEvent(ctx.EventManager().Events(), cw721types.EventTypeRefundPacketTokenSkip)
	require.NotNil(t, sk)
	require.Equal(t, "nft_owner_not_module", refundAttr(*sk)["reason"])
	require.Equal(t, owner.String(), refundAttr(*sk)[cw721types.AttributeKeyNFTOwner])

	// The recipient is not double-burned: it still owns the NFT.
	curr, err := k.nftKeeper.GetNFT(ctx, "kitty", "nft1")
	require.NoError(t, err)
	require.NotEmpty(t, curr.GetID())
}

func TestQueryCW721TokenOwner(t *testing.T) {
	k, ctx, owner, contract, wasm := setupConvertKeeper(t)

	wasm.setOwner(contract, "1", owner.String())
	got, err := k.QueryCW721TokenOwner(ctx, contract, "1")
	require.NoError(t, err)
	require.Equal(t, owner.String(), got)

	// An unknown token has no owner recorded; the empty result must not be
	// treated as "owned by the module".
	got, err = k.QueryCW721TokenOwner(ctx, contract, "does-not-exist")
	require.NoError(t, err)
	require.Empty(t, got)
	require.False(t, moduleOwnsCW721(got))
}

func TestAppendToRefundGroups_MergesAndKeepsOrder(t *testing.T) {
	t.Parallel()

	groups := appendToRefundGroups(nil, "c1", "r1", "t1", "nft1")
	groups = appendToRefundGroups(groups, "c2", "r2", "t2", "nft2")
	groups = appendToRefundGroups(groups, "c1", "r1", "t3", "nft3")

	require.Len(t, groups, 2)
	require.Equal(t, []string{"t1", "t3"}, groups[0].tokenIds)
	require.Equal(t, []string{"nft1", "nft3"}, groups[0].nftIds)
	require.Equal(t, []string{"t2"}, groups[1].tokenIds)
}

func TestCW721RefundEvents_GroupsByContractAndReceiver(t *testing.T) {
	t.Parallel()

	groups := []refundGroup{
		{contract: "c1", receiver: "r1", tokenIds: []string{"t1"}, nftIds: []string{"nft1"}},
		{contract: "c2", receiver: "r2", tokenIds: []string{"t2"}, nftIds: []string{"nft2"}},
	}

	events := cw721RefundEvents("sender", "kitty", groups)
	require.Len(t, events, 2)
	require.Equal(t, cw721types.EventTypeRefundPacketToken, events[0].Type)
	require.Equal(t, "c1", refundAttr(events[0])[cw721types.AttributeKeyCW721Token])
	require.Equal(t, "r1", refundAttr(events[0])[cw721types.AttributeKeyReceiver])
	require.Equal(t, "t2", refundAttr(events[1])[cw721types.AttributeKeyCW721TokenID])
}
