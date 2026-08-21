package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// ModuleCdc references the global cw721 module codec. Note, the codec should
// ONLY be used in certain instances of tests and for JSON encoding.
//
// The actual codec used for serialization should be provided to modules/cw721 and
// defined at the application level.
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// RegisterInterfaces register implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgConvertNFT{},
		&MsgConvertCW721{},
		&MsgTransferCW721{},
		&MsgUpdateParams{},
	)
	registry.RegisterImplementations(
		(*gov.Content)(nil),
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec registers the necessary x/cw721 interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgConvertNFT{}, "cw721/ConvertNFT", nil)
	cdc.RegisterConcrete(&MsgConvertCW721{}, "cw721/ConvertCW721", nil)
	cdc.RegisterConcrete(&MsgTransferCW721{}, "cw721/TransferCW721", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "cw721/UpdateParams", nil)
}
