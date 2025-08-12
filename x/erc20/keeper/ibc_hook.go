package keeper

import (
	"cosmossdk.io/math"
	"fmt"

	"github.com/UptickNetwork/uptick/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/ethereum/go-ethereum/common"
)

// OnRecvPacket will get the denom name from ibc ,generate by port/channel/denom
func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ack exported.Acknowledgement,
) exported.Acknowledgement {
	event := &types.EventIBCERC20{
		Status:             types.STATUS_UNKNOWN,
		Message:            "",
		Sequence:           packet.Sequence,
		SourceChannel:      packet.SourceChannel,
		DestinationChannel: packet.DestinationChannel,
	}
	cctx, write := ctx.CacheContext()

	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		event.Status = types.STATUS_FAILED
		event.Message = err.Error()
		_ = ctx.EventManager().EmitTypedEvent(event)
		return nil
	}
	transferAmount, ok := math.NewIntFromString(data.Amount)
	if !ok {
		event.Status = types.STATUS_FAILED
		event.Message = "Change data.Amount type to int error"
		_ = ctx.EventManager().EmitTypedEvent(event)
		return nil
	}
	receiver, _ := sdk.AccAddressFromBech32(data.Receiver)
	denom, err := types.IBCDenom(packet.GetDestPort(), packet.GetDestChannel(), data.Denom)
	if err != nil {
		event.Status = types.STATUS_FAILED
		event.Message = err.Error()
		_ = ctx.EventManager().EmitTypedEvent(event)
		return nil
	}

	if !k.IsDenomRegistered(ctx, denom) {
		event.Status = types.STATUS_FAILED
		event.Message = fmt.Sprintf("denom %s not registered", denom)
		_ = ctx.EventManager().EmitTypedEvent(event)
		return nil
	}
	msg := types.NewMsgConvertCoin(
		sdk.NewCoin(denom, transferAmount),
		common.BytesToAddress(receiver.Bytes()),
		receiver,
	)
	// use cctx to ConvertCoin
	context := sdk.WrapSDKContext(cctx)
	_, err = k.ConvertCoin(context, msg)
	if err != nil {
		event.Status = types.STATUS_FAILED
		event.Message = err.Error()
		_ = ctx.EventManager().EmitTypedEvent(event)
		return nil
	}

	write()
	ctx.EventManager().EmitEvents(cctx.EventManager().Events())
	event.Status = types.STATUS_SUCCESS
	_ = ctx.EventManager().EmitTypedEvent(event)
	return nil
}

func (k Keeper) SendPacket(
	ctx sdk.Context, channelCap *capabilitytypes.Capability,
	sourcePort string, sourceChannel string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte,
) (uint64, error) {
	return k.ics4Wrapper.SendPacket(ctx, channelCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (k Keeper) WriteAcknowledgement(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet exported.PacketI, ack exported.Acknowledgement) error {
	return k.ics4Wrapper.WriteAcknowledgement(ctx, channelCap, packet, ack)
}

// GetAppVersion returns the underlying application version.
func (k Keeper) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return k.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// OnAcknowledgementPacket responds to the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement
// was a success then nothing occurs. If the acknowledgement failed, then
// the sender is refunded their tokens using the refundPacketToken function.
func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data transfertypes.FungibleTokenPacketData, ack channeltypes.Acknowledgement) error {

	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		k.refundPacketToken(ctx, packet, data)
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
	}
	return nil
}
