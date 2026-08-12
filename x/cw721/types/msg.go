package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgConvertNFT{}
	_ sdk.Msg = &MsgConvertCW721{}
	_ sdk.Msg = &MsgTransferCW721{}
)

const (
	TypeMsgConvertNFT    = "convert_nft"
	TypeMsgConvertCW721  = "convert_CW721"
	TypeMsgTransferCW721 = "transfer_CW721"
)

// Route should return the name of the module
func (msg MsgConvertNFT) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertNFT) Type() string { return TypeMsgConvertNFT }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertNFT) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "invalid sender address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertNFT) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertNFT) GetSigners() []sdk.AccAddress {

	addr := sdk.MustAccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

// Route should return the name of the module
func (msg MsgConvertCW721) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertCW721) Type() string { return TypeMsgConvertCW721 }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertCW721) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Receiver); err != nil {
		return sdkerrors.Wrap(err, "invalid reciver address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertCW721) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertCW721) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

// ----------- MsgTransferCW721 --------------------

// Route should return the name of the module
func (msg MsgTransferCW721) Route() string { return RouterKey }

// Type should return the action
func (msg MsgTransferCW721) Type() string { return TypeMsgTransferCW721 }

// ValidateBasic runs stateless checks on the message
func (msg MsgTransferCW721) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.CwSender); err != nil {
		return sdkerrors.Wrap(err, "invalid sender address")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgTransferCW721) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgTransferCW721) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(msg.CwSender)
	return []sdk.AccAddress{addr}
}
