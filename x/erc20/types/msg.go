package types

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/ethereum/go-ethereum/common"
)

var (
	_ sdk.Msg = &MsgConvertCoin{}
	_ sdk.Msg = &MsgConvertERC20{}
	_ sdk.Msg = &MsgTransferERC20{}
)

const (
	TypeMsgConvertCoin   = "convert_coin"
	TypeMsgConvertERC20  = "convert_ERC20"
	TypeMsgTransferERC20 = "transfer_ERC20"
)

// NewMsgConvertCoin creates a new instance of MsgConvertCoin
func NewMsgConvertCoin(coin sdk.Coin, receiver common.Address, sender sdk.AccAddress) *MsgConvertCoin { // nolint: interfacer
	return &MsgConvertCoin{
		Coin:     coin,
		Receiver: receiver.Hex(),
		Sender:   sender.String(),
	}
}

// Route should return the name of the module
func (msg MsgConvertCoin) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertCoin) Type() string { return TypeMsgConvertCoin }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertCoin) ValidateBasic() error {
	if err := ValidateErc20Denom(msg.Coin.Denom); err != nil {
		if err := ibctransfertypes.ValidateIBCDenom(msg.Coin.Denom); err != nil {
			return err
		}
	}

	if !msg.Coin.Amount.IsPositive() {
		return sdkerrors.Wrapf(errortypes.ErrInvalidCoins, "cannot mint a non-positive amount")
	}
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid sender address")
	}
	if !common.IsHexAddress(msg.Receiver) {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid receiver hex address %s", msg.Receiver)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertCoin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertCoin) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil
	}

	return []sdk.AccAddress{addr}
}

// NewMsgConvertERC20 creates a new instance of MsgConvertERC20
func NewMsgConvertERC20(amount math.Int, receiver sdk.AccAddress, contract, sender common.Address) *MsgConvertERC20 { // nolint: interfacer
	return &MsgConvertERC20{
		ContractAddress: contract.String(),
		Amount:          amount,
		Receiver:        receiver.String(),
		Sender:          sender.Hex(),
	}
}

// Route should return the name of the module
func (msg MsgConvertERC20) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertERC20) Type() string { return TypeMsgConvertERC20 }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertERC20) ValidateBasic() error {
	if !common.IsHexAddress(msg.ContractAddress) {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid contract hex address '%s'", msg.ContractAddress)
	}
	if !msg.Amount.IsPositive() {
		return sdkerrors.Wrapf(errortypes.ErrInvalidCoins, "cannot mint a non-positive amount")
	}
	_, err := sdk.AccAddressFromBech32(msg.Receiver)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid reciver address")
	}
	if !common.IsHexAddress(msg.Sender) {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid sender hex address %s", msg.Sender)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertERC20) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertERC20) GetSigners() []sdk.AccAddress {
	addr := common.HexToAddress(msg.Sender)
	return []sdk.AccAddress{addr.Bytes()}
}

// ----------- MsgTransferERC20 --------------------

// Route should return the name of the module
func (msg MsgTransferERC20) Route() string { return RouterKey }

// Type should return the action
func (msg MsgTransferERC20) Type() string { return TypeMsgTransferERC20 }

// ValidateBasic runs stateless checks on the message
func (msg MsgTransferERC20) ValidateBasic() error {
	if !common.IsHexAddress(msg.EvmContractAddress) {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid contract hex address '%s'", msg.EvmContractAddress)
	}
	if !common.IsHexAddress(msg.EvmSender) {
		return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid sender hex address %s", msg.EvmSender)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgTransferERC20) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgTransferERC20) GetSigners() []sdk.AccAddress {
	addr := common.HexToAddress(msg.EvmSender)
	return []sdk.AccAddress{addr.Bytes()}
}
