package keeper

import (
	"context"
	cw721Types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc20Types "github.com/UptickNetwork/uptick/x/erc20/types"
	erc721Types "github.com/UptickNetwork/uptick/x/erc721/types"

	"github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"strings"
)

// OnRecvPacket processes a cross chain fungible token transfer. If the
// convertType 0:erc721 1:cw721
func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	receiver string,
	convertType uint) exported.Acknowledgement {

	k.Logger(ctx).Info("OnRecvPacket ", "convertType", convertType)
	event := &erc20Types.EventIBCERC20{
		Status:             erc20Types.STATUS_UNKNOWN,
		Message:            "",
		Sequence:           packet.Sequence,
		SourceChannel:      packet.SourceChannel,
		DestinationChannel: packet.DestinationChannel,
	}
	cctx, write := ctx.CacheContext()

	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		event.Status = erc20Types.STATUS_FAILED
		event.Message = err.Error()
		_ = ctx.EventManager().EmitTypedEvent(event)
		return nil
	}

	// add the prefix class check for the case of class id
	var voucherClassID string
	if types.IsAwayFromOrigin(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId) {
		voucherClassID = k.GetVoucherClassID(packet.GetDestPort(), packet.GetDestChannel(), data.ClassId)
	} else {
		voucherClassID, _ = types.RemoveClassPrefix(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId)
	}

	k.Logger(ctx).Info("OnRecvPacket ", "voucherClassID", voucherClassID)
	// use cctx to ConvertCoin
	context := sdk.WrapSDKContext(cctx)
	var err error
	if convertType == 0 {
		err = k.ConvertNFTFromErc721(context, voucherClassID, data.TokenIds, receiver)
	} else if convertType == 1 {
		err = k.ConvertNFTFromCw721(context, voucherClassID, data.TokenIds, receiver)
	}
	if err != nil {
		event.Status = erc20Types.STATUS_FAILED
		event.Message = err.Error()
		k.Logger(ctx).Error("OnRecvPacket ", "err ", err.Error())
		_ = ctx.EventManager().EmitTypedEvent(event)
		return nil
	}

	write()
	ctx.EventManager().EmitEvents(cctx.EventManager().Events())
	event.Status = erc20Types.STATUS_SUCCESS
	_ = ctx.EventManager().EmitTypedEvent(event)

	k.Logger(ctx).Info("OnRecvPacket ", "finish OK")

	return nil

}

func (k Keeper) ConvertNFTFromErc721(context context.Context, voucherClassID string, tokenIds []string, receiver string) error {

	msg := erc721Types.MsgConvertNFT{
		EvmContractAddress: "",
		EvmTokenIds:        nil,
		ClassId:            voucherClassID,
		CosmosTokenIds:     tokenIds,
		CosmosSender:       erc721Types.AccModuleAddress.String(),
		EvmReceiver:        receiver,
	}
	_, err := k.ConvertNFT(context, &msg)
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
		Sender:          erc721Types.AccModuleAddress.String(),
		ContractAddress: "",
		TokenIds:        nil,
	}
	_, err := k.cw721Keep.ConvertNFT(context, &msg)
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
		if strings.Contains(data.Memo, erc721Types.TransferERC721Memo) {
			data.ClassId = k.getRefundClassId(packet, data)
			k.RefundPacketToken(ctx, data)
		} else if strings.Contains(data.Memo, cw721Types.TransferCW721Memo) {
			data.ClassId = k.getRefundClassId(packet, data)
			k.cw721Keep.RefundPacketToken(ctx, data)
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

	if strings.Contains(data.Memo, erc721Types.TransferERC721Memo) {
		data.ClassId = k.getRefundClassId(packet, data)
		k.RefundPacketToken(ctx, data)
	} else if strings.Contains(data.Memo, cw721Types.TransferCW721Memo) {
		data.ClassId = k.getRefundClassId(packet, data)
		k.cw721Keep.RefundPacketToken(ctx, data)
	}
	return nil
}

func (k Keeper) getRefundClassId(packet channeltypes.Packet, data types.NonFungibleTokenPacketData) string {
	var voucherClassID string

	if strings.Contains(data.ClassId, "nft-transfer/") {
		// if types.IsAwayFromOrigin(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId) {
		orgClass, _ := types.RemoveClassPrefix(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId)
		voucherClassID = k.GetVoucherClassID(packet.GetSourcePort(), packet.GetSourceChannel(), orgClass)

	} else {
		voucherClassID = data.ClassId
	}

	return voucherClassID
}
