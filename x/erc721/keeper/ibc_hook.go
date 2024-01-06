package keeper

import (
	"fmt"
	erc20Types "github.com/UptickNetwork/uptick/x/erc20/types"
	erc721Types "github.com/UptickNetwork/uptick/x/erc721/types"
	"github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"strings"
)

// OnRecvPacket processes a cross chain fungible token transfer. If the
// sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// unescrowed and sent to the receiving address.
func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	receiver string) exported.Acknowledgement {

	fmt.Printf("xxl -- 0000 OnRecvPacket \n")
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

	fmt.Printf("xxl -- 0001 OnRecvPacket %v-%v-%v \n", packet.GetDestPort(), packet.GetDestChannel(), data.ClassId)
	voucherClassID := k.GetVoucherClassID(packet.GetDestPort(), packet.GetDestChannel(), data.ClassId)
	fmt.Printf("xxl -- 0001.5 voucherClassID %s \n", voucherClassID)

	msg := erc721Types.MsgConvertNFT{
		EvmContractAddress: "",
		EvmTokenIds:        nil,
		ClassId:            voucherClassID,
		CosmosTokenIds:     data.TokenIds,
		CosmosSender:       erc721Types.AccModuleAddress.String(),
		EvmReceiver:        receiver,
	}

	fmt.Printf("xxl -- 0002 msg %v \n", msg)
	// use cctx to ConvertCoin
	context := sdk.WrapSDKContext(cctx)
	_, err := k.ConvertNFT(context, &msg)
	if err != nil {
		event.Status = erc20Types.STATUS_FAILED
		event.Message = err.Error()
		_ = ctx.EventManager().EmitTypedEvent(event)

		fmt.Printf("xxl 0003 err %v \n", err)
		return nil
	}

	write()
	ctx.EventManager().EmitEvents(cctx.EventManager().Events())
	event.Status = erc20Types.STATUS_SUCCESS
	_ = ctx.EventManager().EmitTypedEvent(event)

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
			k.RefundPacketToken(ctx, data)
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
		k.RefundPacketToken(ctx, data)
	}
	return nil
}
