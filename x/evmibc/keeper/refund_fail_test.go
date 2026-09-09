package keeper

import (
	"errors"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
)

// TestOnAcknowledgementPacket_RefundFailurePropagatesButReleaseRanFirst pins
// that the "release IBC escrow BEFORE refund-burn" order is a semantic
// requirement, not a stylistic one. When RefundPacketToken fails, the IBC
// release (ibcKeeper.OnAcknowledgementPacket) has ALREADY run — the escrow
// was opened to the module account first, then the (failed) burn ran — and
// the failure propagates up. The release is never skipped, and the refund
// never runs before the release.
//
// This guards against a regression that "simplifies" the order: running
// RefundPacketToken first would burn against an escrow the module does not
// yet own, fail, and roll back the whole cache context — stranding both the
// ERC721 and the NFT (the original refund deadlock the order was added to
// prevent).
func TestOnAcknowledgementPacket_RefundFailurePropagatesButReleaseRanFirst(t *testing.T) {
	order := &[]string{}
	k := NewKeeper(&mockICS721{order: order})
	k.SetErc721Keeper(&mockERC721{order: order, refundErr: errors.New("refund burn failed")})
	k.SetCw721Keeper(&mockCW721{order: order})

	packet := channeltypes.Packet{SourcePort: "nft-transfer", SourceChannel: "channel-0"}
	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
		Sender:   erc721types.AccModuleAddress.String(),
		Memo:     erc721types.TransferERC721Memo,
	}
	ack := channeltypes.NewErrorAcknowledgement(sdkerrors.Wrap(errortypes.ErrInvalidRequest, "boom"))

	err := k.OnAcknowledgementPacket(sdk.Context{}, packet, data, ack)
	require.Error(t, err)
	require.ErrorContains(t, err, "refund burn failed")
	// Release ran first, then the (failing) refund — order is fixed even on
	// the failure path.
	require.Equal(t, []string{"ibc:ack", "erc:refund"}, *order)
}

// TestOnTimeoutPacket_RefundFailurePropagatesButReleaseRanFirst mirrors the
// ack path for timeouts: the IBC release runs first, then the refund-burn,
// and a refund failure propagates without skipping the release.
func TestOnTimeoutPacket_RefundFailurePropagatesButReleaseRanFirst(t *testing.T) {
	order := &[]string{}
	k := NewKeeper(&mockICS721{order: order})
	k.SetErc721Keeper(&mockERC721{order: order, refundErr: errors.New("refund burn failed")})
	k.SetCw721Keeper(&mockCW721{order: order})

	packet := channeltypes.Packet{SourcePort: "nft-transfer", SourceChannel: "channel-0"}
	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "kitty",
		TokenIds: []string{"nft1"},
		Sender:   erc721types.AccModuleAddress.String(),
		Memo:     erc721types.TransferERC721Memo,
	}

	err := k.OnTimeoutPacket(sdk.Context{}, packet, data)
	require.Error(t, err)
	require.ErrorContains(t, err, "refund burn failed")
	require.Equal(t, []string{"ibc:timeout", "erc:refund"}, *order)
}
