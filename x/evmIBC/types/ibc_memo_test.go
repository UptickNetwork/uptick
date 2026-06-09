package types

import (
	"testing"

	erc721types "github.com/UptickNetwork/evm-nft-convert/types"
	cw721types "github.com/UptickNetwork/wasm-nft-convert/types"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/stretchr/testify/require"
)

func TestOutboundConvertKindRequiresModuleSenderAndSuffix(t *testing.T) {
	tests := []struct {
		name string
		data nfttransfertypes.NonFungibleTokenPacketData
		want ConvertKind
	}{
		{
			name: "erc721 conversion packet",
			data: nfttransfertypes.NonFungibleTokenPacketData{
				Sender: erc721types.AccModuleAddress.String(),
				Memo:   "user memo" + cw721types.TransferCW721Memo + erc721types.TransferERC721Memo,
			},
			want: ConvertKindERC721,
		},
		{
			name: "cw721 conversion packet",
			data: nfttransfertypes.NonFungibleTokenPacketData{
				Sender: cw721types.AccModuleAddress.String(),
				Memo:   "user memo" + erc721types.TransferERC721Memo + cw721types.TransferCW721Memo,
			},
			want: ConvertKindCW721,
		},
		{
			name: "marker suffix from user sender is ignored",
			data: nfttransfertypes.NonFungibleTokenPacketData{
				Sender: "uptick1user",
				Memo:   "user memo" + erc721types.TransferERC721Memo,
			},
			want: ConvertKindNone,
		},
		{
			name: "module sender with marker only in user memo is ignored",
			data: nfttransfertypes.NonFungibleTokenPacketData{
				Sender: erc721types.AccModuleAddress.String(),
				Memo:   erc721types.TransferERC721Memo + " extra text",
			},
			want: ConvertKindNone,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, OutboundConvertKind(tc.data))
			require.Equal(t, tc.want != ConvertKindNone, IsOutboundConvertPacket(tc.data))
		})
	}
}
