package evmibc

import (
	"testing"

	cw721Types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721Types "github.com/UptickNetwork/uptick/x/erc721/types"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

func nftPacket(receiver, memo string) channeltypes.Packet {
	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  "class-1",
		TokenIds: []string{"1"},
		Sender:   "cosmos1sender",
		Receiver: receiver,
		Memo:     memo,
	}
	return channeltypes.Packet{Data: nfttransfertypes.ModuleCdc.MustMarshalJSON(&data)}
}

func TestOnRecvPacket_UnmarshalError(t *testing.T) {
	im := IBCMiddleware{}
	ack := im.OnRecvPacket(sdk.Context{}, "", channeltypes.Packet{Data: []byte("not-json")}, nil)
	require.False(t, ack.Success())
}

func TestOnRecvPacket_InvalidConvertReceiver(t *testing.T) {
	im := IBCMiddleware{}

	t.Run("erc721", func(t *testing.T) {
		ack := im.OnRecvPacket(sdk.Context{}, "", nftPacket("not-a-hex-address", `{"convert_to":"erc721"}`), nil)
		require.False(t, ack.Success())
	})

	t.Run("cw721", func(t *testing.T) {
		ack := im.OnRecvPacket(sdk.Context{}, "", nftPacket("not-a-bech32", `{"convert_to":"cw721"}`), nil)
		require.False(t, ack.Success())
	})
}

func TestOnTimeoutPacket_UnmarshalError(t *testing.T) {
	im := IBCMiddleware{}
	err := im.OnTimeoutPacket(sdk.Context{}, "", channeltypes.Packet{Data: []byte("not-json")}, nil)
	require.Error(t, err)
}

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

	t.Run("erc721 module account", func(t *testing.T) {
		newPacket, dstReceiver := PackageToModuleAccount(packet, erc721Types.AccModuleAddress)
		require.Equal(t, packetData.Receiver, dstReceiver)

		var decoded nfttransfertypes.NonFungibleTokenPacketData
		err := nfttransfertypes.ModuleCdc.UnmarshalJSON(newPacket.Data, &decoded)
		require.NoError(t, err)
		require.Equal(t, erc721Types.AccModuleAddress.String(), decoded.Receiver)
		require.Equal(t, packetData.ClassId, decoded.ClassId)
		require.Equal(t, packetData.TokenIds, decoded.TokenIds)
	})

	t.Run("cw721 module account", func(t *testing.T) {
		newPacket, dstReceiver := PackageToModuleAccount(packet, cw721Types.AccModuleAddress)
		require.Equal(t, packetData.Receiver, dstReceiver)

		var decoded nfttransfertypes.NonFungibleTokenPacketData
		err := nfttransfertypes.ModuleCdc.UnmarshalJSON(newPacket.Data, &decoded)
		require.NoError(t, err)
		require.Equal(t, cw721Types.AccModuleAddress.String(), decoded.Receiver)
		require.NotEqual(t, erc721Types.AccModuleAddress.String(), decoded.Receiver)
	})
}

func TestPackageToModuleAccount_InvalidPacketData(t *testing.T) {
	packet := channeltypes.Packet{Data: []byte("invalid-json")}
	newPacket, dstReceiver := PackageToModuleAccount(packet, erc721Types.AccModuleAddress)
	require.Equal(t, "", dstReceiver)
	require.Equal(t, channeltypes.Packet{}, newPacket)
}
