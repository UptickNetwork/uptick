package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// GetPair checks that:
//   - the global parameter for cw721 conversion is enabled
//   - minting is enabled for the given (cw721,nft) token pair
func (k Keeper) GetPair(
	ctx sdk.Context,
	token string,
) (types.TokenPair, error) {

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

	return pair, nil
}
