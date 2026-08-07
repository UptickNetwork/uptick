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
	"github.com/UptickNetwork/uptick/x/evmibc/keeper"
	evmibctypes "github.com/UptickNetwork/uptick/x/evmibc/types"

	erc721Types "github.com/UptickNetwork/evm-nft-convert/types"
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
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {

	ackResult := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	var data types.NonFungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		ackResult = channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(errortypes.ErrInvalidType, "cannot unmarshal ICS-721 nft-transfer packet data"),
		)
		return ackResult
	}

	if len(data.Memo) > maxMemoLength {
		return im.Module.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	var packageMemo PackageMemo
	err := json.Unmarshal([]byte(data.Memo), &packageMemo)
	if err != nil {
		return im.Module.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	if strings.ToLower(packageMemo.ConvertTo) == convertERC721 {

		newPackage, dstReceiver := PackageToModuleAccount(packet)
		if !common.IsHexAddress(dstReceiver) {
			ackResult = channeltypes.NewErrorAcknowledgement(
				sdkerrors.Wrapf(errortypes.ErrInvalidType, "receiver address format error"),
			)
			return ackResult
		}
		ack := im.Module.OnRecvPacket(ctx, channelVersion, newPackage, relayer)
		// return if the acknowledgement is an error ACK
		if !ack.Success() {
			return ack
		}
		return im.keeper.OnRecvPacket(ctx, newPackage, dstReceiver, 0)

	} else if strings.ToLower(packageMemo.ConvertTo) == convertCW721 {

		newPackage, dstReceiver := PackageToModuleAccount(packet)
		ack := im.Module.OnRecvPacket(ctx, channelVersion, newPackage, relayer)
		// return if the acknowledgement is an error ACK
		if !ack.Success() {
			return ack
		}
		// im.keeper.
		return im.keeper.OnRecvPacket(ctx, newPackage, dstReceiver, 1)
	} else {
		return im.Module.OnRecvPacket(ctx, channelVersion, packet, relayer)
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
		ctx.EventManager().EmitEvents(cctx.EventManager().Events())
		return nil
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, data, ack); err != nil {
		return err
	}

	return im.Module.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64, data []byte) (sequence uint64, err error) {
	return 0, nil
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
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
	// For non-convert packets delegate to the nft-transfer module directly.
	if evmibctypes.IsOutboundConvertPacket(data) {
		if err := im.keeper.OnTimeoutPacket(ctx, packet, data); err != nil {
			return err
		}
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
