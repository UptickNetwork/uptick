package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// GetParams returns the total set of erc721 parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixParams)
	if len(bz) == 0 {
		return types.DefaultParams()
	}
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		k.Logger(ctx).Error("failed to unmarshal erc721 params", "error", err)
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the erc721 parameters to the store.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to marshal erc721 params")
	}
	store.Set(types.KeyPrefixParams, bz)
	return nil
}

// GetEnableErc721 returns whether ERC721 conversion is enabled.
func (k Keeper) GetEnableErc721(ctx sdk.Context) bool {
	return k.GetParams(ctx).EnableErc721
}
