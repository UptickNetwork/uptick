package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// ModuleCdc references the global erc721 module codec. Note, the codec should
// ONLY be used in certain instances of tests and for JSON encoding.
//
// The actual codec used for serialization should be provided to modules/erc721 and
// defined at the application level.
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// RegisterInterfaces register implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgConvertNFT{},
		&MsgConvertERC721{},
	)
	registry.RegisterImplementations(
		(*gov.Content)(nil),
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
