package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// MintingEnabled checks that:
//  - the global parameter for erc721 conversion is enabled
//  - minting is enabled for the given (erc721,nft) token pair
func (k Keeper) MintingEnabled(
	ctx sdk.Context,
	sender sdk.AccAddress,
	receiver sdk.AccAddress,
	token string,
) (types.TokenPair, error) {

	params := k.GetParams(ctx)
	if !params.EnableErc721 {
		return types.TokenPair{}, sdkerrors.Wrap(
			types.ErrERC721Disabled, "module is currently disabled by governance",
		)
	}

	id := k.GetTokenPairID(ctx, token)
	if len(id) == 0 {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered by id", token,
		)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered", token,
		)
	}

	if !pair.Enabled {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrERC721TokenPairDisabled, "minting token '%s' is not enabled by governance", token,
		)
	}

	return pair, nil
}
