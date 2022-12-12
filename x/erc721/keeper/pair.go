package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// GetPair checks that:
//  - the global parameter for erc721 conversion is enabled
//  - minting is enabled for the given (erc721,nft) token pair
func (k Keeper) GetPair(
	ctx sdk.Context,
	token string,
) (types.TokenPair, error) {

	fmt.Printf("xxl 01 GetPair 001 start token %v \n", token)
	id := k.GetTokenPairID(ctx, token)
	if len(id) == 0 {

		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered by id", token,
		)
	}

	fmt.Printf("xxl 01 004 ConvertERC721 id %v \n", id)
	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered", token,
		)
	}

	return pair, nil
}
