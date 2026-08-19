package types

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
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
	if _, err := sdk.AccAddressFromBech32(msg.Receiver); err != nil {
		return sdkerrors.Wrap(err, "invalid receiver address")
	}
	if strings.TrimSpace(msg.ContractAddress) != "" {
		if _, err := sdk.AccAddressFromBech32(msg.ContractAddress); err != nil {
			return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid contract address %s", msg.ContractAddress)
		}
	}
	if strings.TrimSpace(msg.ClassId) == "" {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "class id cannot be empty")
	}
	if len(msg.NftIds) == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "nft ids cannot be empty")
	}
	for _, id := range msg.NftIds {
		if id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "nft id cannot be empty")
		}
	}
	for _, id := range msg.TokenIds {
		if id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "token id cannot be empty")
		}
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertNFT) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertNFT) GetSigners() []sdk.AccAddress {

	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

// Route should return the name of the module
func (msg MsgConvertCW721) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertCW721) Type() string { return TypeMsgConvertCW721 }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertCW721) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "invalid sender address")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Receiver); err != nil {
		return sdkerrors.Wrap(err, "invalid reciver address")
	}
	if _, err := sdk.AccAddressFromBech32(msg.ContractAddress); err != nil {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid contract address %s", msg.ContractAddress)
	}
	if len(msg.TokenIds) == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "token ids cannot be empty")
	}
	for _, id := range msg.TokenIds {
		if id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "token id cannot be empty")
		}
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertCW721) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertCW721) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil
	}
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
	if _, err := sdk.AccAddressFromBech32(msg.CosmosReceiver); err != nil {
		return sdkerrors.Wrap(err, "invalid receiver address")
	}
	if _, err := sdk.AccAddressFromBech32(msg.CwContractAddress); err != nil {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid cw721 contract address %s", msg.CwContractAddress)
	}
	if len(msg.CwTokenIds) == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "cw721 token ids cannot be empty")
	}
	for _, id := range msg.CwTokenIds {
		if id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "cw721 token id cannot be empty")
		}
	}
	if strings.TrimSpace(msg.SourcePort) == "" {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "source port cannot be empty")
	}
	if strings.TrimSpace(msg.SourceChannel) == "" {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "source channel cannot be empty")
	}
	for _, id := range msg.CosmosTokenIds {
		if id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "cosmos token id cannot be empty")
		}
	}
	if msg.TimeoutHeight.IsZero() && msg.TimeoutTimestamp == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "timeout height and timeout timestamp cannot both be zero")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgTransferCW721) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgTransferCW721) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.CwSender)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}
