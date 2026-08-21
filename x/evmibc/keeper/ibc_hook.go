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
	cctx, write := ctx.CacheContext()

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
		voucherClassID = classID
	}

	k.Logger(ctx).Info("OnRecvPacket ", "voucherClassID", voucherClassID)
	// use cctx to ConvertCoin
	context := sdk.WrapSDKContext(cctx)
	var err error
	if convertType == 0 {
		err = k.ConvertNFTFromErc721(context, voucherClassID, data.TokenIds, receiver)
	} else if convertType == 1 {
		err = k.ConvertNFTFromCw721(context, voucherClassID, data.TokenIds, receiver)
	} else {
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

	write()
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
			classID, err := k.getRefundClassId(packet, data)
			if err != nil {
				return err
			}
			data.ClassId = classID
			if err := k.erc721keeper.RefundPacketToken(ctx, data); err != nil {
				return err
			}
			// Redirect the NFT refund to the module address so the sender
			// does not receive both the ERC721 and the NFT (double refund).
			nftData := data
			nftData.Sender = erc721types.AccModuleAddress.String()
			return k.ibcKeeper.OnAcknowledgementPacket(ctx, packet, nftData, ack)
		case evmibctypes.ConvertKindCW721:
			classID, err := k.getRefundClassId(packet, data)
			if err != nil {
				return err
			}
			data.ClassId = classID
			nftData := data
			nftData.Sender = cw721Types.AccModuleAddress.String()
			if err := k.ibcKeeper.OnAcknowledgementPacket(ctx, packet, nftData, ack); err != nil {
				return err
			}
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
		classID, err := k.getRefundClassId(packet, data)
		if err != nil {
			return err
		}
		data.ClassId = classID
		if err := k.erc721keeper.RefundPacketToken(ctx, data); err != nil {
			return err
		}
		// Redirect the NFT refund to the module address so the sender
		// does not receive both the ERC721 and the NFT (double refund).
		nftData := data
		nftData.Sender = erc721types.AccModuleAddress.String()
		return k.ibcKeeper.OnTimeoutPacket(ctx, packet, nftData)
	case evmibctypes.ConvertKindCW721:
		classID, err := k.getRefundClassId(packet, data)
		if err != nil {
			return err
		}
		data.ClassId = classID
		nftData := data
		nftData.Sender = cw721Types.AccModuleAddress.String()
		if err := k.ibcKeeper.OnTimeoutPacket(ctx, packet, nftData); err != nil {
			return err
		}
		return k.cw721Keeper.RefundPacketToken(ctx, data)
	}
	return nil
}

func (k Keeper) getRefundClassId(packet channeltypes.Packet, data types.NonFungibleTokenPacketData) (string, error) {
	if !strings.HasPrefix(data.ClassId, packet.GetSourcePort()+"/") {
		return data.ClassId, nil
	}

	orgClass, err := types.RemoveClassPrefix(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId)
	if err != nil {
		return "", err
	}
	return k.GetVoucherClassID(packet.GetSourcePort(), packet.GetSourceChannel(), orgClass), nil
}
