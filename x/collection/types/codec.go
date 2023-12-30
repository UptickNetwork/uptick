package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	"github.com/UptickNetwork/uptick/x/collection/exported"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	proto "github.com/cosmos/gogoproto/proto"
)

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}

// RegisterLegacyAminoCodec concrete types on codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgIssueDenom{}, "uptick/collection/v1/MsgIssueDenom", nil)
	cdc.RegisterConcrete(&MsgTransferNFT{}, "uptick/collection/v1/MsgTransferNFT", nil)
	cdc.RegisterConcrete(&MsgEditNFT{}, "uptick/collection/v1/MsgEditNFT", nil)
	cdc.RegisterConcrete(&MsgMintNFT{}, "uptick/collection/v1/MsgMintNFT", nil)
	cdc.RegisterConcrete(&MsgBurnNFT{}, "uptick/collection/v1/MsgBurnNFT", nil)
	cdc.RegisterConcrete(&MsgTransferDenom{}, "uptick/collection/v1/MsgTransferDenom", nil)

	cdc.RegisterInterface((*exported.NFT)(nil), nil)
	cdc.RegisterConcrete(&BaseNFT{}, "uptick/collection/v1/BaseNFT", nil)
}

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

	registry.RegisterImplementations(
		(*proto.Message)(nil),
		&DenomMetadata{},
		&NFTMetadata{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
