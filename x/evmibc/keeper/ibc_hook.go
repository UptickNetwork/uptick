package keeper

import (
	"context"
	"fmt"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	cw721Types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	evmibctypes "github.com/UptickNetwork/uptick/x/evmibc/types"

	"github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// OnRecvPacket processes a cross chain fungible token transfer. If the
// convertType 0:erc721 1:cw721
func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	receiver string,
	convertType uint) exported.Acknowledgement {

	k.Logger(ctx).Info("OnRecvPacket ", "convertType", convertType)
	msg := ""

	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		msg = err.Error()
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("ibc_nft_convert",
				sdk.NewAttribute("status", "1"),
				sdk.NewAttribute("message", msg),
				sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
				sdk.NewAttribute("source_channel", packet.SourceChannel),
				sdk.NewAttribute("destination_channel", packet.DestinationChannel),
			),
		)
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(errortypes.ErrInvalidType, "cannot unmarshal NFT transfer packet data"),
		)
	}

	// add the prefix class check for the case of class id
	var voucherClassID string
	if types.IsAwayFromOrigin(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId) {
		voucherClassID = k.GetVoucherClassID(packet.GetDestPort(), packet.GetDestChannel(), data.ClassId)
	} else {
		classID, err := types.RemoveClassPrefix(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId)
		if err != nil {
			msg = err.Error()
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("ibc_nft_convert",
					sdk.NewAttribute("status", "1"),
					sdk.NewAttribute("message", msg),
					sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
					sdk.NewAttribute("source_channel", packet.SourceChannel),
					sdk.NewAttribute("destination_channel", packet.DestinationChannel),
				),
			)
			return channeltypes.NewErrorAcknowledgement(
				sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "invalid class id prefix: %s", err.Error()),
			)
		}
		// Align with nft-transfer's processReceivedPacket: a multi-hop return
		// leaves a trace path in the unprefixed class id, which must be resolved
		// to the canonical local ibc/<hash> (or native) id via the IBC keeper.
		voucherClassID, err = k.ibcKeeper.GetVoucherClassID(ctx, classID)
		if err != nil {
			msg = err.Error()
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("ibc_nft_convert",
					sdk.NewAttribute("status", "1"),
					sdk.NewAttribute("message", msg),
					sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
					sdk.NewAttribute("source_channel", packet.SourceChannel),
					sdk.NewAttribute("destination_channel", packet.DestinationChannel),
				),
			)
			return channeltypes.NewErrorAcknowledgement(
				sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "failed to resolve voucher class id: %s", err.Error()),
			)
		}
	}

	k.Logger(ctx).Info("OnRecvPacket ", "voucherClassID", voucherClassID)
	// Convert on the caller's context directly: the middleware's recvAndConvert
	// already wraps the mint + convert in a single cache context, so an inner
	// cache here would only add a redundant commit layer.
	context := sdk.WrapSDKContext(ctx)
	var err error
	switch convertType {
	case 0:
		err = k.ConvertNFTFromErc721(context, voucherClassID, data.TokenIds, receiver)
	case 1:
		err = k.ConvertNFTFromCw721(context, voucherClassID, data.TokenIds, receiver)
	default:
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "unknown convert type %d", convertType),
		)
	}
	if err != nil {
		msg = err.Error()
		k.Logger(ctx).Error("OnRecvPacket ", "err ", err.Error())
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("ibc_nft_convert",
				sdk.NewAttribute("status", "1"), // FAILED
				sdk.NewAttribute("message", msg),
				sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
				sdk.NewAttribute("source_channel", packet.SourceChannel),
				sdk.NewAttribute("destination_channel", packet.DestinationChannel),
			),
		)
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "failed to convert NFT: %s", err.Error()),
		)
	}

	msg = "ok"
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("ibc_nft_convert",
			sdk.NewAttribute("status", "2"), // SUCCESS
			sdk.NewAttribute("message", msg),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute("source_channel", packet.SourceChannel),
			sdk.NewAttribute("destination_channel", packet.DestinationChannel),
		),
	)

	k.Logger(ctx).Info("OnRecvPacket ", "finish OK")

	return channeltypes.NewResultAcknowledgement([]byte{byte(1)})

}

func (k Keeper) ConvertNFTFromErc721(context context.Context, voucherClassID string, tokenIds []string, receiver string) error {

	msg := erc721types.MsgConvertNFT{
		EvmContractAddress: "",
		EvmTokenIds:        nil,
		ClassId:            voucherClassID,
		CosmosTokenIds:     tokenIds,
		CosmosSender:       erc721types.AccModuleAddress.String(),
		EvmReceiver:        receiver,
	}

	_, err := k.erc721keeper.ConvertNFT(context, &msg)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) ConvertNFTFromCw721(context context.Context, voucherClassID string, tokenIds []string, receiver string) error {
	msg := cw721Types.MsgConvertNFT{
		ClassId:         voucherClassID,
		NftIds:          tokenIds,
		Receiver:        receiver,
		Sender:          cw721Types.AccModuleAddress.String(),
		ContractAddress: "",
		TokenIds:        nil,
	}
	_, err := k.cw721Keeper.ConvertNFT(context, &msg)
	if err != nil {
		return err
	}
	return nil

}

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
//     3/4. Voucher with a different channel or unrelated port prefix: multi-hop
//     ICS-721 (or unrelated string) — emit `cross_channel_refund` and derive
//     the canonical ibc/<hash> form.
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
	// channel) — multi-hop ICS-721 or unrelated string. Emit for
	// observability and derive the canonical local voucher id.
	k.Logger(ctx).Info(
		"getRefundClassId: cross-channel or non-matching voucher prefix, deriving local voucher id",
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
	return types.ParseClassTrace(data.ClassId).IBCClassID(), nil
}
