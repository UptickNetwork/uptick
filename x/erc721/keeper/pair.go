package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetPair looks up a token pair by contract address first, then by class id.
// NOTE: this combined lookup routes by string shape (see GetTokenPairID) and
// is therefore ambiguous when a class id equals a contract address. Handlers
// must use the semantics-specific lookups below instead.
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

// GetPairByClass resolves a pair exclusively from the class-id map. Conversion
// handlers that operate on a native class (ConvertNFT) must use this so a
// hex-address-shaped denom can never resolve to the contract-address map of a
// different, victim pair.
func (k Keeper) GetPairByClass(
	ctx sdk.Context,
	classID string,
) (types.TokenPair, error) {
	id := k.GetClassMap(ctx, classID)
	if len(id) == 0 {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered by id", classID,
		)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered", classID,
		)
	}

	return pair, nil
}

// GetPairByEVM resolves a pair exclusively from the contract-address map.
// Handlers that operate on an ERC721 contract (ConvertERC721) must use this so
// the lookup cannot be hijacked through the class-id namespace.
func (k Keeper) GetPairByEVM(
	ctx sdk.Context,
	contract string,
) (types.TokenPair, error) {
	id := k.GetERC721Map(ctx, common.HexToAddress(contract))
	if len(id) == 0 {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered by id", contract,
		)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered", contract,
		)
	}

	return pair, nil
}
