package evmibc

import (
	"encoding/json"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/ibc"
	cw721Types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721Types "github.com/UptickNetwork/uptick/x/erc721/types"
	"github.com/UptickNetwork/uptick/x/evmibc/keeper"
	evmibctypes "github.com/UptickNetwork/uptick/x/evmibc/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

var _ porttypes.Middleware = &IBCMiddleware{}

const convertERC721 = "erc721"
const convertCW721 = "cw721"
const maxMemoLength = 1024

// IBCMiddleware implements the ICS26 callbacks for the transfer middleware given
// the claim keeper and the underlying application.
type IBCMiddleware struct {
	*ibc.Module
	keeper keeper.Keeper
	ics4   porttypes.ICS4Wrapper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
func NewIBCMiddleware(k keeper.Keeper, app porttypes.IBCModule, ics4 porttypes.ICS4Wrapper) IBCMiddleware {
	return IBCMiddleware{
		Module: ibc.NewModule(app),
		keeper: k,
		ics4:   ics4,
	}
}

type PackageMemo struct {
	ConvertTo string `protobuf:"bytes,1,opt,name=convert_to,proto3" json:"convert_to,omitempty"`
}

// OnRecvPacket implements the IBCModule interface.
// If fees are not enabled, this callback will default to the ibc-core packet callback.
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {

	var ackResult exported.Acknowledgement
	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		ackResult = channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrap(errortypes.ErrInvalidType, "cannot unmarshal ICS-721 nft-transfer packet data"),
		)
		return ackResult
	}

	if len(data.Memo) > maxMemoLength {
		im.keeper.Logger(ctx).Debug(
			"memo exceeds max length, skipping convert",
			"memo_length", len(data.Memo),
			"max_memo_length", maxMemoLength,
		)
		return im.Module.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	var packageMemo PackageMemo
	err := json.Unmarshal([]byte(data.Memo), &packageMemo)
	if err != nil {
		return im.Module.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	switch strings.ToLower(packageMemo.ConvertTo) {
	case convertERC721:
		newPackage, dstReceiver := PackageToModuleAccount(packet, erc721Types.AccModuleAddress)
		if !common.IsHexAddress(dstReceiver) || common.HexToAddress(dstReceiver) == (common.Address{}) {
			ackResult = channeltypes.NewErrorAcknowledgement(
				sdkerrors.Wrap(errortypes.ErrInvalidType, "receiver address format error"),
			)
			return ackResult
		}
		return im.recvAndConvert(ctx, channelVersion, newPackage, relayer, dstReceiver, 0)

	case convertCW721:
		newPackage, dstReceiver := PackageToModuleAccount(packet, cw721Types.AccModuleAddress)
		if _, err := sdk.AccAddressFromBech32(dstReceiver); err != nil {
			ackResult = channeltypes.NewErrorAcknowledgement(
				sdkerrors.Wrap(errortypes.ErrInvalidType, "receiver address format error"),
			)
			return ackResult
		}
		return im.recvAndConvert(ctx, channelVersion, newPackage, relayer, dstReceiver, 1)
	}
	return im.Module.OnRecvPacket(ctx, channelVersion, packet, relayer)

}

// recvAndConvert runs nft-transfer mint and ERC721/CW721 conversion on one
// cache context. write() is called only if both succeed, so a convert failure
// rolls back the voucher mint instead of leaving it on the module account
// while returning an error ACK (which would also refund on the source chain).
func (im IBCMiddleware) recvAndConvert(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	dstReceiver string,
	convertType uint,
) exported.Acknowledgement {
	cctx, write := ctx.CacheContext()
	ack := im.Module.OnRecvPacket(cctx, channelVersion, packet, relayer)
	if !ack.Success() {
		return ack
	}
	convertAck := im.keeper.OnRecvPacket(cctx, packet, dstReceiver, convertType)
	if !convertAck.Success() {
		return convertAck
	}
	write()
	return convertAck
}

func PackageToModuleAccount(packet channeltypes.Packet, moduleAddr sdk.AccAddress) (channeltypes.Packet, string) {
	// Rewrites the packet receiver to the conversion module account
	// and returns the original destination receiver for later conversion.
	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return channeltypes.Packet{}, ""
	}
	dstReceiver := data.Receiver
	data.Receiver = moduleAddr.String()
	packet.Data = types.ModuleCdc.MustMarshalJSON(&data)

	return packet, dstReceiver
}

// OnAcknowledgementPacket implements the IBCModule interface
// If fees are not enabled, this callback will default to the ibc-core packet callback.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {

	// decode the data
	var ack channeltypes.Acknowledgement
	if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return sdkerrors.Wrapf(errortypes.ErrUnknownRequest,
			"cannot unmarshal ICS-721 transfer packet acknowledgement: %v", err)
	}

	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(errortypes.ErrUnknownRequest,
			"cannot unmarshal ICS-721 transfer packet data: %s", err.Error())
	}

	// On error ack for convert packets the evmIBC keeper handles both the
	// ERC721/CW721 refund and the NFT-side refund (routed to module address).
	// Skip the nft-transfer module to prevent double refund of the NFT.
	// Use a cache context to ensure atomicity: if the NFT-side refund fails
	// after the ERC721/CW721 refund, the entire operation is rolled back.
	if _, isError := ack.Response.(*channeltypes.Acknowledgement_Error); isError && evmibctypes.IsOutboundConvertPacket(data) {
		cctx, commit := ctx.CacheContext()
		if err := im.keeper.OnAcknowledgementPacket(cctx, packet, data, ack); err != nil {
			return err
		}
		commit()
		return nil
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, data, ack); err != nil {
		return err
	}

	return im.Module.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

// SendPacket is a no-op stub — the EVM IBC middleware intercepts on the
// receiving side (OnRecvPacket), not the sending side.
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64, data []byte) (sequence uint64, err error) {
	if im.ics4 == nil {
		return 0, sdkerrors.Wrap(errortypes.ErrLogic, "ics4 wrapper is not set")
	}
	return im.ics4.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	if im.ics4 == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "ics4 wrapper is not set")
	}
	return im.ics4.WriteAcknowledgement(ctx, packet, ack)
}

func (im IBCMiddleware) GetAppVersion(
	ctx sdk.Context,
	portID,
	channelID string,
) (string, bool) {
	if im.ics4 == nil {
		return "", false
	}
	return im.ics4.GetAppVersion(ctx, portID, channelID)
}

// OnTimeoutPacket implements the IBCModule interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(errortypes.ErrUnknownRequest, "cannot unmarshal ICS-721 transfer packet data: %s", err.Error())
	}

	// For convert packets the keeper handles both ERC721/CW721 and NFT sides.
	// Skip the nft-transfer module to prevent double refund.
	// Cache the convert refund so a later NFT-side failure rolls back the ERC721/CW721 refund.
	if evmibctypes.IsOutboundConvertPacket(data) {
		cctx, commit := ctx.CacheContext()
		if err := im.keeper.OnTimeoutPacket(cctx, packet, data); err != nil {
			return err
		}
		commit()
	} else {
		if err := im.Module.OnTimeoutPacket(ctx, channelVersion, packet, relayer); err != nil {
			return err
		}
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
