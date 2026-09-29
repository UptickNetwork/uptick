package keeper

import (
	cw721keep "github.com/UptickNetwork/uptick/x/cw721/keeper"
	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

// ERC721Converter is the x/erc721 surface used by outbound refunds.
//
// NOTE (R1-C): the inbound ConvertNFT path was removed — convert memos are now
// settled natively, so no token pair is written and the burn guard never locks
// a bridged voucher out of its origin chain. Only the refund side remains.
type ERC721Converter interface {
	RefundPacketToken(ctx sdk.Context, data nfttransfertypes.NonFungibleTokenPacketData) error
}

// CW721Converter is the x/cw721 surface used by outbound refunds.
//
// NOTE (R1-C): same as ERC721Converter — inbound ConvertNFT was removed.
type CW721Converter interface {
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
