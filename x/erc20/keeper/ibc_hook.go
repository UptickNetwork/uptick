package keeper

import (
	"cosmossdk.io/math"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	"github.com/UptickNetwork/uptick/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
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
		if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
			k.Logger(ctx).Error("failed to emit IBC event", "status", event.Status, "error", err.Error())
		}
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(errortypes.ErrInvalidType, "cannot unmarshal ICS-20 transfer packet data"),
		)
	}
	transferAmount, ok := math.NewIntFromString(data.Amount)
	if !ok {
		event.Status = types.STATUS_FAILED
		event.Message = "Change data.Amount type to int error"
		if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
			k.Logger(ctx).Error("failed to emit IBC event", "status", event.Status, "error", err.Error())
		}
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(transfertypes.ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", data.Amount),
		)
	}
	receiver, err := sdk.AccAddressFromBech32(data.Receiver)
	if err != nil {
		event.Status = types.STATUS_FAILED
		event.Message = fmt.Sprintf("invalid receiver address: %s", data.Receiver)
		if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
			k.Logger(ctx).Error("failed to emit IBC event", "status", event.Status, "error", err.Error())
		}
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid receiver address: %s", data.Receiver),
		)
	}
	denom, err = types.IBCDenom(packet.GetDestPort(), packet.GetDestChannel(), data.Denom)
	if err != nil {
		event.Status = types.STATUS_FAILED
		event.Message = err.Error()
		if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
			k.Logger(ctx).Error("failed to emit IBC event", "status", event.Status, "error", err.Error())
		}
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "invalid IBC denom: %s", err.Error()),
		)
	}

	if !k.IsDenomRegistered(ctx, denom) {
		event.Status = types.STATUS_FAILED
		event.Message = fmt.Sprintf("denom %s not registered", denom)
		if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
			k.Logger(ctx).Error("failed to emit IBC event", "status", event.Status, "error", err.Error())
		}
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(types.ErrTokenPairNotFound, "denom %s not registered", denom),
		)
	}
	msg := types.NewMsgConvertCoin(
		sdk.NewCoin(denom, transferAmount),
		common.BytesToAddress(receiver.Bytes()),
		receiver,
	)
	// use cctx to ConvertCoin
	goCtx := sdk.WrapSDKContext(cctx)
	_, err = k.ConvertCoin(goCtx, msg)
	if err != nil {
		event.Status = types.STATUS_FAILED
		event.Message = err.Error()
		if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
			k.Logger(ctx).Error("failed to emit IBC event", "status", event.Status, "error", err.Error())
		}
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "failed to convert coin: %s", err.Error()),
		)
	}

	write()
	ctx.EventManager().EmitEvents(cctx.EventManager().Events())
	event.Status = types.STATUS_SUCCESS
	if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
		k.Logger(ctx).Error("failed to emit IBC event", "status", event.Status, "error", err.Error())
	}
	return channeltypes.NewResultAcknowledgement([]byte{byte(1)})
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
		return k.refundPacketToken(ctx, packet, data)
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		k.DeleteIBCTransferProvenance(ctx, packet, data)
		return nil
	}
}
