package keeper

import (
	"context"

	cw721keep "github.com/UptickNetwork/uptick/x/cw721/keeper"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

// ERC721Converter is the x/erc721 surface used by inbound convert and outbound refunds.
type ERC721Converter interface {
	ConvertNFT(ctx context.Context, msg *erc721types.MsgConvertNFT) (*erc721types.MsgConvertNFTResponse, error)
	RefundPacketToken(ctx sdk.Context, data nfttransfertypes.NonFungibleTokenPacketData) error
}

// CW721Converter is the x/cw721 surface used by inbound convert and outbound refunds.
type CW721Converter interface {
	ConvertNFT(ctx context.Context, msg *cw721types.MsgConvertNFT) (*cw721types.MsgConvertNFTResponse, error)
	RefundPacketToken(ctx sdk.Context, data nfttransfertypes.NonFungibleTokenPacketData) error
}

// ICS721Keeper is the nft-transfer surface used to refund the Cosmos NFT side
// of an outbound convert packet without going through the IBC middleware twice.
type ICS721Keeper interface {
	OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data nfttransfertypes.NonFungibleTokenPacketData, ack channeltypes.Acknowledgement) error
	OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, data nfttransfertypes.NonFungibleTokenPacketData) error
	// GetVoucherClassID resolves a class id — which may be a multi-hop trace
	// path on a back-to-origin receive — to the canonical local voucher/native
	// id, matching nft-transfer's keeper logic (HasClass check + ibc/<hash>).
	GetVoucherClassID(ctx sdk.Context, classID string) (string, error)
}

var (
	_ ERC721Converter = erc721keeper.Keeper{}
	_ CW721Converter  = cw721keep.Keeper{}
	_ ICS721Keeper    = ibcnfttransferkeeper.Keeper{}
)
