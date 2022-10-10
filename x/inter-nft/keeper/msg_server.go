package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/nft"

	internft "github.com/UptickNetwork/uptick/x/inter-nft"
)

var _ internft.MsgServer = Keeper{}

func (k Keeper) IssueClass(goCtx context.Context, msg *internft.MsgIssueClass) (*internft.MsgIssueClassResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &internft.MsgIssueClassResponse{}, k.SaveClass(ctx, nft.Class{
		Id:          msg.Id,
		Name:        msg.Name,
		Symbol:      msg.Symbol,
		Description: msg.Description,
		Uri:         msg.Uri,
		UriHash:     msg.Issuer,
	})
}

func (k Keeper) MintNFT(goCtx context.Context, msg *internft.MsgMintNFT) (*internft.MintNFTResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var (
		receiver sdk.AccAddress
		err      error
	)

	receiver, err = sdk.AccAddressFromBech32(msg.Minter)
	if err != nil {
		return nil, err
	}

	if msg.Receiver != "" {
		receiver, err = sdk.AccAddressFromBech32(msg.Receiver)
		if err != nil {
			return nil, err
		}
	}
	return &internft.MintNFTResponse{}, k.Mint(ctx, nft.NFT{
		ClassId: msg.ClassId,
		Id:      msg.Id,
		Uri:     msg.Uri,
		UriHash: msg.UriHash,
	}, receiver)
}
