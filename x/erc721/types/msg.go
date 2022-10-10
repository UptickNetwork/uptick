package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	_ sdk.Msg = &MsgConvertNFT{}
	_ sdk.Msg = &MsgConvertERC721{}
)

const (
	TypeMsgConvertNFT    = "convert_nft"
	TypeMsgConvertERC721 = "convert_ERC721"
)

// NewMsgConvertNFT creates a new instance of MsgConvertNFT
func NewMsgConvertNFT(classID string, nftID string, receiver common.Address, sender sdk.AccAddress) *MsgConvertNFT { // nolint: interfacer
	return &MsgConvertNFT{
		ClassId:  classID,
		NftId:    nftID,
		Receiver: receiver.Hex(),
		Sender:   sender.String(),
	}
}

// Route should return the name of the module
func (msg MsgConvertNFT) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertNFT) Type() string { return TypeMsgConvertNFT }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertNFT) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "invalid sender address")
	}
	if !common.IsHexAddress(msg.Receiver) {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid receiver hex address %s", msg.Receiver)
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

// NewMsgConvertERC721 creates a new instance of MsgConvertERC721
func NewMsgConvertERC721(tokenID string, receiver sdk.AccAddress, contract, sender common.Address) *MsgConvertERC721 { // nolint: interfacer
	return &MsgConvertERC721{
		ContractAddress: contract.String(),
		TokenId:         tokenID,
		Receiver:        receiver.String(),
		Sender:          sender.Hex(),
	}
}

// Route should return the name of the module
func (msg MsgConvertERC721) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertERC721) Type() string { return TypeMsgConvertERC721 }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertERC721) ValidateBasic() error {
	if !common.IsHexAddress(msg.ContractAddress) {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid contract hex address '%s'", msg.ContractAddress)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Receiver); err != nil {
		return sdkerrors.Wrap(err, "invalid reciver address")
	}
	if !common.IsHexAddress(msg.Sender) {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender hex address %s", msg.Sender)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertERC721) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertERC721) GetSigners() []sdk.AccAddress {
	addr := common.HexToAddress(msg.Sender)
	return []sdk.AccAddress{addr.Bytes()}
}
