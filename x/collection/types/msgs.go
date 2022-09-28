package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// constant used to indicate that some field should not be updated
const (
	TypeMsgIssueDenom    = "issue_denom"
	TypeMsgTransferNFT   = "transfer_nft"
	TypeMsgEditNFT       = "edit_nft"
	TypeMsgMintNFT       = "mint_nft"
	TypeMsgBurnNFT       = "burn_nft"
	TypeMsgTransferDenom = "transfer_denom"
)

var (
	_ sdk.Msg = &MsgIssueDenom{}
	_ sdk.Msg = &MsgTransferNFT{}
	_ sdk.Msg = &MsgEditNFT{}
	_ sdk.Msg = &MsgMintNFT{}
	_ sdk.Msg = &MsgBurnNFT{}
	_ sdk.Msg = &MsgTransferDenom{}
)

// NewMsgIssueDenom is a constructor function for MsgSetName
func NewMsgIssueDenom(
	denomID string,
	denomName string,
	schema string,
	sender string,
	symbol string,
	mintRestricted bool,
	updateRestricted bool,
) *MsgIssueDenom {
	return &MsgIssueDenom{
		Sender:           sender,
		ID:               denomID,
		Name:             denomName,
		Schema:           schema,
		Symbol:           symbol,
		MintRestricted:   mintRestricted,
		UpdateRestricted: updateRestricted,
	}
}

// ValidateBasic Implements Msg.
func (msg MsgIssueDenom) ValidateBasic() error {
	if err := ValidateDenomID(msg.ID); err != nil {
		return err
	}

	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	return ValidateKeywords(msg.ID)
}

// GetSigners Implements Msg.
func (msg MsgIssueDenom) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}

// NewMsgTransferNFT is a constructor function for MsgSetName
func NewMsgTransferNFT(
	tokenID, denomID, tokenName, tokenURI, tokenData, sender, recipient string,
) *MsgTransferNFT {
	return &MsgTransferNFT{
		ID:        tokenID,
		DenomID:   denomID,
		Name:      tokenName,
		URI:       tokenURI,
		Data:      tokenData,
		Sender:    sender,
		Recipient: recipient,
	}
}

// ValidateBasic Implements Msg.
func (msg MsgTransferNFT) ValidateBasic() error {
	if err := ValidateDenomID(msg.DenomID); err != nil {
		return err
	}

	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid recipient address (%s)", err)
	}
	return ValidateTokenID(msg.ID)
}

// GetSigners Implements Msg.
func (msg MsgTransferNFT) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}

// NewMsgEditNFT is a constructor function for MsgSetName
func NewMsgEditNFT(
	tokenID, denomID, tokenName, tokenURI, tokenData, sender string,
) *MsgEditNFT {
	return &MsgEditNFT{
		ID:      tokenID,
		DenomID: denomID,
		Name:    tokenName,
		URI:     tokenURI,
		Data:    tokenData,
		Sender:  sender,
	}
}

// ValidateBasic Implements Msg.
func (msg MsgEditNFT) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if err := ValidateDenomID(msg.DenomID); err != nil {
		return err
	}

	if err := ValidateTokenURI(msg.URI); err != nil {
		return err
	}
	return ValidateTokenID(msg.ID)
}

// GetSigners Implements Msg.
func (msg MsgEditNFT) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}

// NewMsgMintNFT is a constructor function for MsgMintNFT
func NewMsgMintNFT(
	tokenID, denomID, tokenName, tokenURI, tokenData, sender, recipient string,
) *MsgMintNFT {
	return &MsgMintNFT{
		ID:        tokenID,
		DenomID:   denomID,
		Name:      tokenName,
		URI:       tokenURI,
		Data:      tokenData,
		Sender:    sender,
		Recipient: recipient,
	}
}

// ValidateBasic Implements Msg.
func (msg MsgMintNFT) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid receipt address (%s)", err)
	}
	if err := ValidateDenomID(msg.DenomID); err != nil {
		return err
	}
	if err := ValidateKeywords(msg.DenomID); err != nil {
		return err
	}
	if err := ValidateTokenURI(msg.URI); err != nil {
		return err
	}
	return ValidateTokenID(msg.ID)
}

// GetSigners Implements Msg.
func (msg MsgMintNFT) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}

// NewMsgBurnNFT is a constructor function for MsgBurnNFT
func NewMsgBurnNFT(sender, tokenID, denomID string) *MsgBurnNFT {
	return &MsgBurnNFT{
		Sender:  sender,
		ID:      tokenID,
		DenomID: denomID,
	}
}

// ValidateBasic Implements Msg.
func (msg MsgBurnNFT) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if err := ValidateDenomID(msg.DenomID); err != nil {
		return err
	}
	return ValidateTokenID(msg.ID)
}

// GetSigners Implements Msg.
func (msg MsgBurnNFT) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}

// NewMsgTransferDenom is a constructor function for msgTransferDenom
func NewMsgTransferDenom(denomID, sender, recipient string) *MsgTransferDenom {
	return &MsgTransferDenom{
		ID:        denomID,
		Sender:    sender,
		Recipient: recipient,
	}
}

// ValidateBasic Implements Msg.
func (msg MsgTransferDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid recipient address (%s)", err)
	}
	if err := ValidateDenomID(msg.ID); err != nil {
		return err
	}
	return nil
}

// GetSigners Implements Msg.
func (msg MsgTransferDenom) GetSigners() []sdk.AccAddress {
	from, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{from}
}
