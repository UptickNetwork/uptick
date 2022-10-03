package inter_nft

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgIssueClass{}
	_ sdk.Msg = &MsgMintNFT{}
)

// ValidateBasic implements sdk.Msg
func (msg MsgIssueClass) ValidateBasic() error {
	if strings.TrimSpace(msg.Issuer) == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "missing sender address")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Issuer); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "failed to parse address: %s", msg.Issuer)
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgIssueClass) GetSigners() []sdk.AccAddress {
	accAddr, err := sdk.AccAddressFromBech32(msg.Issuer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{accAddr}
}

// GetSigners implements sdk.Msg
func (msg MsgMintNFT) GetSigners() []sdk.AccAddress {
	accAddr, err := sdk.AccAddressFromBech32(msg.Minter)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{accAddr}
}

// ValidateBasic implements sdk.Msg
func (msg MsgMintNFT) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Minter)
	if err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "invalid minter address")
	}

	return nil
}
