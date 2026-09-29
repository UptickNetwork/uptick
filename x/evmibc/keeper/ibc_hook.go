package keeper

import (
	"fmt"
	"strings"

	cw721Types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	evmibctypes "github.com/UptickNetwork/uptick/x/evmibc/types"

	"github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)


// OnAcknowledgementPacket responds to the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement
// was a success then nothing occurs. If the acknowledgement failed, then
// the sender is refunded their tokens using the refundPacketToken function.
func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData, ack channeltypes.Acknowledgement) error {

	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		switch evmibctypes.OutboundConvertKind(data) {
		case evmibctypes.ConvertKindERC721:
			localClassID, err := k.getRefundClassId(ctx, packet, data)
			if err != nil {
				return err
			}
			// Redirect the NFT refund to the module address so the sender
			// does not receive both the ERC721 and the NFT (double refund).
			// nftData keeps the original (full-path) ClassId so the IBC
			// nft-transfer refund preserves its IsAwayFromOrigin mint/unescrow
			// decision; the erc721 refund uses the local voucher id.
			nftData := data
			nftData.Sender = erc721types.AccModuleAddress.String()
			// Release the IBC-escrowed NFT to the module account FIRST:
			// RefundPacketToken burns the native NFT, which requires the
			// module account to own it. Reversed order would roll back the
			// whole cache context and strand both assets. Mirrors the CW721
			// branch below.
			if err := k.ibcKeeper.OnAcknowledgementPacket(ctx, packet, nftData, ack); err != nil {
				return err
			}
			data.ClassId = localClassID
			return k.erc721keeper.RefundPacketToken(ctx, data)
		case evmibctypes.ConvertKindCW721:
			localClassID, err := k.getRefundClassId(ctx, packet, data)
			if err != nil {
				return err
			}
			nftData := data
			nftData.Sender = cw721Types.AccModuleAddress.String()
			if err := k.ibcKeeper.OnAcknowledgementPacket(ctx, packet, nftData, ack); err != nil {
				return err
			}
			data.ClassId = localClassID
			return k.cw721Keeper.RefundPacketToken(ctx, data)
		}
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
	}
	return nil
}

// OnTimeoutPacket refunds the sender since the original packet sent was
// never received and has been timed out.
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData) error {

	switch evmibctypes.OutboundConvertKind(data) {
	case evmibctypes.ConvertKindERC721:
		localClassID, err := k.getRefundClassId(ctx, packet, data)
		if err != nil {
			return err
		}
		// Redirect the NFT refund to the module address so the sender
		// does not receive both the ERC721 and the NFT (double refund).
		// nftData keeps the original (full-path) ClassId so the IBC
		// nft-transfer refund preserves its mint/unescrow decision.
		nftData := data
		nftData.Sender = erc721types.AccModuleAddress.String()
		// Release the IBC-escrowed NFT first (see OnAcknowledgementPacket).
		if err := k.ibcKeeper.OnTimeoutPacket(ctx, packet, nftData); err != nil {
			return err
		}
		data.ClassId = localClassID
		return k.erc721keeper.RefundPacketToken(ctx, data)
	case evmibctypes.ConvertKindCW721:
		localClassID, err := k.getRefundClassId(ctx, packet, data)
		if err != nil {
			return err
		}
		nftData := data
		nftData.Sender = cw721Types.AccModuleAddress.String()
		if err := k.ibcKeeper.OnTimeoutPacket(ctx, packet, nftData); err != nil {
			return err
		}
		data.ClassId = localClassID
		return k.cw721Keeper.RefundPacketToken(ctx, data)
	}
	return nil
}

// getRefundClassId resolves the local (voucher) class id to feed into the
// erc721/cw721 module-side refund on acknowledgement-error / timeout. The
// caller keeps the original `data.ClassId` for the IBC nft-transfer refund,
// which must see the full path to preserve its IsAwayFromOrigin mint/unescrow
// decision (feeding it the local id flips the decision and strands the asset):
//
//  1. Bare class id (no "/"): original NFT is native to this chain — return as-is.
//  2. Voucher matching this packet's (port, channel) prefix: strip and return
//     the canonical ibc/<hash> form (single-hop).
//  3. Anything else: ask the ICS-721 keeper, which resolves the id the same way
//     nft-transfer does on the receive path (`HasClass(id) ? id : ibc/<hash>`).
//     That covers the multi-hop case (different channel or unrelated port
//     prefix) without guessing from the string: a class id that already exists
//     locally IS the local class, even when it contains "/" — idString allows
//     the slash, so a natively issued `sub/collection` is a legal class on this
//     chain, and rewriting it to ibc/<hash> would hand the module-side refund a
//     class that does not exist. Emits `cross_channel_refund` for observability.
func (k Keeper) getRefundClassId(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData) (string, error) {
	// Shape 1: bare class id (no "/" anywhere). Nothing to rewrite.
	if !strings.Contains(data.ClassId, "/") {
		return data.ClassId, nil
	}

	expectedPrefix := types.GetClassPrefix(packet.GetSourcePort(), packet.GetSourceChannel())

	// Shape 2: voucher prefix matches this packet's (port, channel).
	// Strip the prefix and return the canonical ibc/<hash> voucher.
	if strings.HasPrefix(data.ClassId, expectedPrefix) {
		orgClass, err := types.RemoveClassPrefix(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId)
		if err != nil {
			return "", err
		}
		return k.GetVoucherClassID(packet.GetSourcePort(), packet.GetSourceChannel(), orgClass), nil
	}

	// Shape 3 / 4: voucher prefix does not match this packet's (port,
	// channel) — multi-hop ICS-721, or a local class id that merely contains
	// slashes. Emit for observability and let the ICS-721 keeper decide.
	k.Logger(ctx).Info(
		"getRefundClassId: cross-channel or non-matching voucher prefix, resolving local class id",
		"class_id", data.ClassId,
		"packet_source_port", packet.GetSourcePort(),
		"packet_source_channel", packet.GetSourceChannel(),
		"sequence", packet.Sequence,
	)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"cross_channel_refund",
			sdk.NewAttribute("class_id", data.ClassId),
			sdk.NewAttribute("source_port", packet.GetSourcePort()),
			sdk.NewAttribute("source_channel", packet.GetSourceChannel()),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
		),
	)
	// GetVoucherClassID is the ICS-721 keeper's own resolver: it returns the id
	// unchanged when a class with exactly that id exists locally, and the
	// canonical ibc/<hash> otherwise. Deriving the id from the string shape
	// alone (ParseClassTrace) cannot express the first case, and the two paths
	// must agree or the refund and the receive disagree about which class an
	// NFT belongs to.
	return k.ibcKeeper.GetVoucherClassID(ctx, data.ClassId)
}
