package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	// govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// ModuleCdc references the global erc20 module codec. Note, the codec should
// ONLY be used in certain instances of tests and for JSON encoding.
//
// The actual codec used for serialization should be provided to modules/erc20 and
// defined at the application level.
// var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

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
	cdc.RegisterConcrete(&MsgConvertCoin{}, "uptick/erc20/v1/MsgConvertCoin", nil)
	cdc.RegisterConcrete(&MsgConvertERC20{}, "uptick/erc20/v1/MsgConvertERC20", nil)

	//cdc.RegisterConcrete(&RegisterCoinProposal{}, "uptick/erc20/RegisterCoinProposal", nil)
	//cdc.RegisterConcrete(&RegisterERC20Proposal{}, "uptick/erc20/RegisterERC20Proposal", nil)
	//cdc.RegisterConcrete(&ToggleTokenRelayProposal{}, "uptick/erc20/ToggleTokenRelayProposal", nil)
	//cdc.RegisterConcrete(&UpdateTokenPairERC20Proposal{}, "uptick/erc20/UpdateTokenPairERC20Proposal", nil)
}

// RegisterInterfaces register implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {

	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgConvertCoin{},
		&MsgConvertERC20{},
	)

	registry.RegisterImplementations(
		(*govv1beta1.Content)(nil),
		&RegisterCoinProposal{},
		&RegisterERC20Proposal{},
		&ToggleTokenRelayProposal{},
		&UpdateTokenPairERC20Proposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
