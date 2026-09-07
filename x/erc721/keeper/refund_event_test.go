package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// attributeMap flattens an event's attributes into a map for stable lookups.
func attributeMap(event sdk.Event) map[string]string {
	out := make(map[string]string, len(event.Attributes))
	for _, attr := range event.Attributes {
		out[attr.Key] = attr.Value
	}
	return out
}

// Regression: the refund event used to publish the Cosmos NFT id in the
// `erc721_token_ids` attribute (a copy of the `nft_ids` value) and to keep only
// the last contract/receiver seen in the loop. Indexers therefore could not
// tell which ERC721 token was returned to whom.
func TestERC721RefundEvents_TokenIDAttributeUsesEvmTokenID(t *testing.T) {
	t.Parallel()

	groups := []erc721RefundGroup{{
		contract: "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		receiver: "0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
		tokenIds: []string{"7", "8"},
		nftIds:   []string{"nft-7", "nft-8"},
	}}

	events := erc721RefundEvents("uptick1sender", "kitty", groups)
	require.Len(t, events, 1)

	attrs := attributeMap(events[0])
	require.Equal(t, "7,8", attrs[types.AttributeKeyERC721TokenID],
		"erc721_token_ids must carry the EVM token id, not the Cosmos NFT id")
	require.Equal(t, "nft-7,nft-8", attrs[types.AttributeKeyNFTID])
	require.Equal(t, "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", attrs[types.AttributeKeyERC721Token])
	require.Equal(t, "0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", attrs[types.AttributeKeyReceiver])
	require.Equal(t, "kitty", attrs[types.AttributeKeyNFTClass])
	require.Equal(t, "uptick1sender", attrs[sdk.AttributeKeySender])
	require.Equal(t, types.EventTypeRefundPacketToken, events[0].Type)
}

func TestERC721RefundEvents_SeparatesContractAndReceiverPairs(t *testing.T) {
	t.Parallel()

	groups := []erc721RefundGroup{
		{contract: "0xAA", receiver: "0x11", tokenIds: []string{"1"}, nftIds: []string{"nft-1"}},
		{contract: "0xBB", receiver: "0x22", tokenIds: []string{"2"}, nftIds: []string{"nft-2"}},
		{contract: "0xAA", receiver: "0x33", tokenIds: []string{"3"}, nftIds: []string{"nft-3"}},
	}

	events := erc721RefundEvents("sender", "kitty", groups)
	require.Len(t, events, 3, "one event per distinct (contract, receiver) pair")

	require.Equal(t, "0xAA", attributeMap(events[0])[types.AttributeKeyERC721Token])
	require.Equal(t, "0x11", attributeMap(events[0])[types.AttributeKeyReceiver])
	require.Equal(t, "0xBB", attributeMap(events[1])[types.AttributeKeyERC721Token])
	require.Equal(t, "0x22", attributeMap(events[1])[types.AttributeKeyReceiver])
	require.Equal(t, "0xAA", attributeMap(events[2])[types.AttributeKeyERC721Token])
	require.Equal(t, "0x33", attributeMap(events[2])[types.AttributeKeyReceiver])
}

func TestERC721RefundEvents_NoGroupsEmitsNothing(t *testing.T) {
	t.Parallel()

	require.Empty(t, erc721RefundEvents("sender", "kitty", nil))
	require.Empty(t, erc721RefundEvents("sender", "kitty", []erc721RefundGroup{}))
}

func TestAppendERC721RefundGroup_MergesSamePair(t *testing.T) {
	t.Parallel()

	groups := appendERC721RefundGroup(nil, "0xAA", "0x11", "1", "nft-1")
	groups = appendERC721RefundGroup(groups, "0xAA", "0x11", "2", "nft-2")

	require.Len(t, groups, 1, "same (contract, receiver) must merge into one group")
	require.Equal(t, []string{"1", "2"}, groups[0].tokenIds)
	require.Equal(t, []string{"nft-1", "nft-2"}, groups[0].nftIds)
}

func TestAppendERC721RefundGroup_SplitsOnReceiverAndContract(t *testing.T) {
	t.Parallel()

	groups := appendERC721RefundGroup(nil, "0xAA", "0x11", "1", "nft-1")
	groups = appendERC721RefundGroup(groups, "0xAA", "0x22", "2", "nft-2")
	groups = appendERC721RefundGroup(groups, "0xBB", "0x11", "3", "nft-3")

	require.Len(t, groups, 3)
}

// Determinism: groups must keep first-seen order so every node emits the events
// in the same sequence (map iteration order would break consensus).
func TestAppendERC721RefundGroup_KeepsFirstSeenOrder(t *testing.T) {
	t.Parallel()

	groups := appendERC721RefundGroup(nil, "0xCC", "0x01", "1", "nft-1")
	groups = appendERC721RefundGroup(groups, "0xAA", "0x02", "2", "nft-2")
	groups = appendERC721RefundGroup(groups, "0xBB", "0x03", "3", "nft-3")
	groups = appendERC721RefundGroup(groups, "0xAA", "0x02", "4", "nft-4")

	require.Equal(t, []string{"0xCC", "0xAA", "0xBB"}, []string{groups[0].contract, groups[1].contract, groups[2].contract})
	require.Equal(t, []string{"2", "4"}, groups[1].tokenIds)

	// Repeated runs must produce identical output.
	for i := 0; i < 20; i++ {
		again := appendERC721RefundGroup(nil, "0xCC", "0x01", "1", "nft-1")
		again = appendERC721RefundGroup(again, "0xAA", "0x02", "2", "nft-2")
		again = appendERC721RefundGroup(again, "0xBB", "0x03", "3", "nft-3")
		again = appendERC721RefundGroup(again, "0xAA", "0x02", "4", "nft-4")
		require.Equal(t, groups, again)
	}
}
