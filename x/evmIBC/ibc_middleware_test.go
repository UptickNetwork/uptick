package evmIBC

import (
	"testing"

	erc721Types "github.com/UptickNetwork/evm-nft-convert/types"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

func TestPackageToModuleAccount(t *testing.T) {
	packetData := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "class-1",
		TokenIds: []string{"1", "2"},
		Sender:   "cosmos1sender",
		Receiver: "0x1111111111111111111111111111111111111111",
	}
	packet := channeltypes.Packet{
		Data: nfttransfertypes.ModuleCdc.MustMarshalJSON(&packetData),
	}

	newPacket, dstReceiver := PackageToModuleAccount(packet)
	require.Equal(t, packetData.Receiver, dstReceiver)

	var decoded nfttransfertypes.NonFungibleTokenPacketData
	err := nfttransfertypes.ModuleCdc.UnmarshalJSON(newPacket.Data, &decoded)
	require.NoError(t, err)
	require.Equal(t, erc721Types.AccModuleAddress.String(), decoded.Receiver)
	require.Equal(t, packetData.ClassId, decoded.ClassId)
	require.Equal(t, packetData.TokenIds, decoded.TokenIds)
}

func TestPackageToModuleAccount_InvalidPacketData(t *testing.T) {
	packet := channeltypes.Packet{Data: []byte("invalid-json")}
	newPacket, dstReceiver := PackageToModuleAccount(packet)
	require.Equal(t, "", dstReceiver)
	require.Equal(t, channeltypes.Packet{}, newPacket)
}

