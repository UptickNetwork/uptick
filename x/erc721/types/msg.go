package types

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

var (
	_ sdk.Msg = &MsgConvertNFT{}
	_ sdk.Msg = &MsgConvertERC721{}
	_ sdk.Msg = &MsgTransferERC721{}
)

const (
	TypeMsgConvertNFT     = "convert_nft"
	TypeMsgConvertERC721  = "convert_ERC721"
	TypeMsgTransferERC721 = "transfer_ERC721"
)

//----------- TypeMsgConvertNFT --------------------

// Route should return the name of the module
func (msg MsgConvertNFT) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertNFT) Type() string { return TypeMsgConvertNFT }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertNFT) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.CosmosSender); err != nil {
		return sdkerrors.Wrap(err, "invalid sender address")
	}
	if !common.IsHexAddress(msg.EvmReceiver) {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid receiver hex address %s", msg.EvmReceiver)
	}
	if strings.TrimSpace(msg.ClassId) == "" {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "class id cannot be empty")
	}
	if len(msg.CosmosTokenIds) == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "cosmos token ids cannot be empty")
	}
	for _, id := range msg.CosmosTokenIds {
		if id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "cosmos token id cannot be empty")
		}
	}
	for _, tokenID := range msg.EvmTokenIds {
		if err := ValidateEVMTokenID(tokenID); err != nil {
			return err
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
	addr, err := sdk.AccAddressFromBech32(msg.CosmosSender)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

// ----------- MsgConvertERC721 --------------------

// Route should return the name of the module
func (msg MsgConvertERC721) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertERC721) Type() string { return TypeMsgConvertERC721 }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertERC721) ValidateBasic() error {
	if !common.IsHexAddress(msg.EvmContractAddress) {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid contract hex address '%s'", msg.EvmContractAddress)
	}
	if _, err := sdk.AccAddressFromBech32(msg.CosmosReceiver); err != nil {
		return sdkerrors.Wrap(err, "invalid reciver address")
	}

	if _, err := sdk.AccAddressFromBech32(msg.CosmosSender); err != nil {
		return sdkerrors.Wrap(err, "invalid sender address")
	}
	if len(msg.EvmTokenIds) == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "evm token ids cannot be empty")
	}
	for _, tokenID := range msg.EvmTokenIds {
		if err := ValidateEVMTokenID(tokenID); err != nil {
			return err
		}
	}
	for _, id := range msg.CosmosTokenIds {
		if id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "cosmos token id cannot be empty")
		}
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertERC721) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertERC721) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.CosmosSender)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

// ----------- MsgTransferERC721 --------------------

// Route should return the name of the module
func (msg MsgTransferERC721) Route() string { return RouterKey }

// Type should return the action
func (msg MsgTransferERC721) Type() string { return TypeMsgTransferERC721 }

// ValidateBasic runs stateless checks on the message
func (msg MsgTransferERC721) ValidateBasic() error {
	if !common.IsHexAddress(msg.EvmContractAddress) {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid contract hex address '%s'", msg.EvmContractAddress)
	}

	if _, err := sdk.AccAddressFromBech32(msg.CosmosSender); err != nil {
		return sdkerrors.Wrap(err, "invalid sender address")
	}
	if len(msg.EvmTokenIds) == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "evm token ids cannot be empty")
	}
	for _, tokenID := range msg.EvmTokenIds {
		if err := ValidateEVMTokenID(tokenID); err != nil {
			return err
		}
	}
	for _, id := range msg.CosmosTokenIds {
		if id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "cosmos token id cannot be empty")
		}
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgTransferERC721) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgTransferERC721) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.CosmosSender)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}
