package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// GetPair looks up a token pair by class id first, then by CW721 contract.
// Class ids and contract addresses are distinct namespaces: a class-path
// caller (ConvertNFT) must use GetPairByClass so a denom whose id equals a
// contract address cannot resolve to that contract's pair.
func (k Keeper) GetPair(
	ctx sdk.Context,
	token string,
) (types.TokenPair, error) {
	if id := k.GetClassMap(ctx, token); len(id) != 0 {
		return k.pairFromID(ctx, id, token)
	}
	return k.pairFromID(ctx, k.GetCW721Map(ctx, token), token)
}

// GetPairByClass resolves a pair exclusively from the class-id map.
func (k Keeper) GetPairByClass(ctx sdk.Context, classID string) (types.TokenPair, error) {
	return k.pairFromID(ctx, k.GetClassMap(ctx, classID), classID)
}

// GetPairByCW721 resolves a pair exclusively from the CW721 contract map.
func (k Keeper) GetPairByCW721(ctx sdk.Context, contract string) (types.TokenPair, error) {
	return k.pairFromID(ctx, k.GetCW721Map(ctx, contract), contract)
}

func (k Keeper) pairFromID(ctx sdk.Context, id []byte, token string) (types.TokenPair, error) {
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
