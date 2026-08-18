package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// GetParams returns the total set of cw721 parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)

	key := types.KeyPrefixParams
	if !store.Has(key) {
		return types.DefaultParams()
	}

	bz := store.Get(key)
	_ = k.cdc.Unmarshal(bz, &params)
	if params.WasmCodeId == 0 {
		if id, err := k.GetWasmCodeID(ctx); err == nil {
			params.WasmCodeId = id
		}
	}
	return params
}

// SetParams sets the cw721 parameters to the store.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)

	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(types.KeyPrefixParams, bz)
	if params.WasmCodeId != 0 {
		k.SetWasmCode(ctx, types.ModuleName, params.WasmCodeId)
	}
	return nil
}

// GetEnableCw721 returns the EnableCw721 param
func (k Keeper) GetEnableCw721(ctx sdk.Context) bool {
	return k.GetParams(ctx).EnableCw721
}

// GetEnableEVMHook returns the EnableEVMHook param
func (k Keeper) GetEnableEVMHook(ctx sdk.Context) bool {
	return k.GetParams(ctx).EnableEVMHook
}
