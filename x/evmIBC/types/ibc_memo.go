package types

import (
	"strings"

	erc721types "github.com/UptickNetwork/evm-nft-convert/types"
	cw721types "github.com/UptickNetwork/wasm-nft-convert/types"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
)

type ConvertKind string

const (
	ConvertKindNone   ConvertKind = ""
	ConvertKindERC721 ConvertKind = "erc721"
	ConvertKindCW721  ConvertKind = "cw721"
)

// OutboundConvertKind classifies packets created by the local NFT conversion
// modules. User memo text alone must not select refund logic.
func OutboundConvertKind(data nfttransfertypes.NonFungibleTokenPacketData) ConvertKind {
	switch {
	case data.Sender == erc721types.AccModuleAddress.String() &&
		strings.HasSuffix(data.Memo, erc721types.TransferERC721Memo):
		return ConvertKindERC721
	case data.Sender == cw721types.AccModuleAddress.String() &&
		strings.HasSuffix(data.Memo, cw721types.TransferCW721Memo):
		return ConvertKindCW721
	default:
		return ConvertKindNone
	}
}

func IsOutboundConvertPacket(data nfttransfertypes.NonFungibleTokenPacketData) bool {
	return OutboundConvertKind(data) != ConvertKindNone
}
