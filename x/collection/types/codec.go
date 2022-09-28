package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	"github.com/UptickNetwork/uptick/x/collection/exported"
)

var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgIssueDenom{},
		&MsgTransferNFT{},
		&MsgEditNFT{},
		&MsgMintNFT{},
		&MsgBurnNFT{},
		&MsgTransferDenom{},
	)

	registry.RegisterImplementations(
		(*exported.NFT)(nil),
		&BaseNFT{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
