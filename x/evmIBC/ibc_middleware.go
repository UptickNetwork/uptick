package evmIBC

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"strings"

	"github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"

	"github.com/UptickNetwork/uptick/ibc"
	"github.com/UptickNetwork/uptick/x/evmIBC/keeper"

	erc721Types "github.com/UptickNetwork/evm-nft-convert/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

var _ porttypes.Middleware = &IBCMiddleware{}

const convertERC721 = "erc721"
const convertCW721 = "cw721"

// IBCMiddleware implements the ICS26 callbacks for the transfer middleware given
// the claim keeper and the underlying application.
type IBCMiddleware struct {
	*ibc.Module
	keeper keeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
func NewIBCMiddleware(k keeper.Keeper, app porttypes.IBCModule) IBCMiddleware {
	return IBCMiddleware{
		Module: ibc.NewModule(app),
		keeper: k,
	}
}

type PackageMemo struct {
	ConvertTo string `protobuf:"bytes,1,opt,name=convert_to,proto3" json:"convert_to,omitempty"`
}

// OnRecvPacket implements the IBCModule interface.
// If fees are not enabled, this callback will default to the ibc-core packet callback.
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {

	ackResult := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		ackResult = channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "cannot unmarshal ICS-721 nft-transfer packet data"),
		)
		return ackResult
	}

	var packageMemo PackageMemo
	err := json.Unmarshal([]byte(data.Memo), &packageMemo)
	if err != nil {
		return im.Module.OnRecvPacket(ctx, packet, relayer)
	}

	if strings.ToLower(packageMemo.ConvertTo) == convertERC721 {

		newPackage, dstReceiver := PackageToModuleAccount(packet)
		if !common.IsHexAddress(dstReceiver) {
			ackResult = channeltypes.NewErrorAcknowledgement(
				sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "receiver address format error"),
			)
			return ackResult
		}
		ack := im.Module.OnRecvPacket(ctx, newPackage, relayer)
		// return if the acknowledgement is an error ACK
		if !ack.Success() {
			return ack
		}
		return im.keeper.OnRecvPacket(ctx, newPackage, dstReceiver, 0)

	} else if strings.ToLower(packageMemo.ConvertTo) == convertCW721 {

		newPackage, dstReceiver := PackageToModuleAccount(packet)
		ack := im.Module.OnRecvPacket(ctx, newPackage, relayer)
		// return if the acknowledgement is an error ACK
		if !ack.Success() {
			return ack
		}
		// im.keeper.
		return im.keeper.OnRecvPacket(ctx, newPackage, dstReceiver, 1)
	} else {
		return im.Module.OnRecvPacket(ctx, packet, relayer)
	}

}

func PackageToModuleAccount(packet channeltypes.Packet) (channeltypes.Packet, string) {

	//
	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return channeltypes.Packet{}, ""
	}
	dstReceiver := data.Receiver
	data.Receiver = erc721Types.AccModuleAddress.String()
	packet.Data = types.ModuleCdc.MustMarshalJSON(&data)

	return packet, dstReceiver
}

// OnAcknowledgementPacket implements the IBCModule interface
// If fees are not enabled, this callback will default to the ibc-core packet callback.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {

	// decode the data
	var ack channeltypes.Acknowledgement
	if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest,
			"cannot unmarshal ICS-721 transfer packet acknowledgement: %v", err)
	}

	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest,
			"cannot unmarshal ICS-721 transfer packet data: %s", err.Error())
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, data, ack); err != nil {
		return err
	}

	if err := im.Module.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer); err != nil {
		return err
	}

	return nil
}

func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64, data []byte) (sequence uint64, err error) {
	return 0, nil
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	return nil
}

func (im IBCMiddleware) GetAppVersion(
	ctx sdk.Context,
	portID,
	channelID string,
) (string, bool) {
	return "", false
}

// OnTimeoutPacket implements the IBCModule interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal ICS-721 transfer packet data: %s", err.Error())
	}
	// refund tokens
	if err := im.keeper.OnTimeoutPacket(ctx, packet, data); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeTimeout,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(types.AttributeKeySender, data.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
			sdk.NewAttribute(types.AttributeKeyClassID, data.ClassId),
			sdk.NewAttribute(types.AttributeKeyTokenIDs, strings.Join(data.TokenIds, ",")),
		),
	)

	return nil
}
