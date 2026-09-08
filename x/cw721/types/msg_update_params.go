package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

const TypeMsgUpdateParams = "cw721/MsgUpdateParams"

var (
	_ sdk.Msg              = &MsgUpdateParams{}
	_ sdk.HasValidateBasic = &MsgUpdateParams{}
	_ proto.Message        = &MsgUpdateParams{}
)

// The MsgUpdateParams struct and its (de)serialization live in the generated
// tx.pb.go. This file only carries the sdk.Msg glue that protoc does not emit.

func (m *MsgUpdateParams) Route() string { return RouterKey }
func (m *MsgUpdateParams) Type() string  { return TypeMsgUpdateParams }

func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return sdkerrors.Wrap(err, "invalid authority")
	}
	return m.Params.Validate()
}
