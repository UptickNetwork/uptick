package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgConvertNFT{}
	_ sdk.Msg = &MsgConvertCW721{}
)

const (
	TypeMsgConvertNFT   = "convert_nft"
	TypeMsgConvertCW721 = "convert_CW721"
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
	//if !common.IsHexAddress(msg.Receiver) {
	//	return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid receiver hex address %s", msg.Receiver)
	//}
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
	//if !common.IsHexAddress(msg.ContractAddress) {
	//	return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid contract hex address '%s'", msg.ContractAddress)
	//}
	if _, err := sdk.AccAddressFromBech32(msg.Receiver); err != nil {
		return sdkerrors.Wrap(err, "invalid reciver address")
	}
	//if !common.IsHexAddress(msg.Sender) {
	//	return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender hex address %s", msg.Sender)
	//}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertCW721) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertCW721) GetSigners() []sdk.AccAddress {

	//addr := common.HexToAddress(msg.Sender)
	//return []sdk.AccAddress{addr.Bytes()}
	addr := sdk.MustAccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}
