package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

// GetAllTokenPairs - get all registered token tokenPairs
func (k Keeper) GetAllTokenPairs(ctx sdk.Context) []types.TokenPair {
	tokenPairs := []types.TokenPair{}

	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.KeyPrefixTokenPair)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var tokenPair types.TokenPair
		if err := k.cdc.Unmarshal(iterator.Value(), &tokenPair); err != nil {
			k.Logger(ctx).Error("failed to unmarshal token pair", "error", err.Error())
			continue
		}

		tokenPairs = append(tokenPairs, tokenPair)
	}

	return tokenPairs
}

// GetTokenPairID returns the pair id from either of the registered tokens.
func (k Keeper) GetTokenPairID(ctx sdk.Context, token string) []byte {
	if common.IsHexAddress(token) {
		addr := common.HexToAddress(token)
		return k.GetERC20Map(ctx, addr)
	}
	return k.GetDenomMap(ctx, token)
}

// GetTokenPair - get registered token pair from the identifier
func (k Keeper) GetTokenPair(ctx sdk.Context, id []byte) (types.TokenPair, bool) {
	if id == nil {
		return types.TokenPair{}, false
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	var tokenPair types.TokenPair
	bz := store.Get(id)
	if len(bz) == 0 {
		return types.TokenPair{}, false
	}

	if err := k.cdc.Unmarshal(bz, &tokenPair); err != nil {
		k.Logger(ctx).Error("failed to unmarshal token pair", "error", err.Error())
		return types.TokenPair{}, false
	}
	return tokenPair, true
}

// SetTokenPair stores a token pair
func (k Keeper) SetTokenPair(ctx sdk.Context, tokenPair types.TokenPair) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	key := tokenPair.GetID()
	bz := k.cdc.MustMarshal(&tokenPair)
	store.Set(key, bz)
}

// DeleteTokenPair removes a token pair.
func (k Keeper) DeleteTokenPair(ctx sdk.Context, tokenPair types.TokenPair) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	key := tokenPair.GetID()
	store.Delete(key)

	// Clean up reverse mappings to prevent stale data
	k.DeleteDenomMap(ctx, tokenPair.Denom)
	k.DeleteERC20Map(ctx, tokenPair.GetERC20Contract())
}

// GetERC20Map returns the token pair id for the given address
func (k Keeper) GetERC20Map(ctx sdk.Context, erc20 common.Address) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC20)
	return store.Get(erc20.Bytes())
}

// GetDenomMap returns the token pair id for the given denomination
func (k Keeper) GetDenomMap(ctx sdk.Context, denom string) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByDenom)
	return store.Get([]byte(denom))
}

// SetERC20Map sets the token pair id for the given address
func (k Keeper) SetERC20Map(ctx sdk.Context, erc20 common.Address, id []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC20)
	store.Set(erc20.Bytes(), id)
}

// DeleteERC20Map deletes the token pair id for the given address
func (k Keeper) DeleteERC20Map(ctx sdk.Context, erc20 common.Address) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC20)
	store.Delete(erc20.Bytes())
}

// SetDenomMap sets the token pair id for the denomination
func (k Keeper) SetDenomMap(ctx sdk.Context, denom string, id []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByDenom)
	store.Set([]byte(denom), id)
}

// DeleteDenomMap deletes the token pair id for the given denom
func (k Keeper) DeleteDenomMap(ctx sdk.Context, denom string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByDenom)
	store.Delete([]byte(denom))
}

// IsTokenPairRegistered - check if registered token tokenPair is registered
func (k Keeper) IsTokenPairRegistered(ctx sdk.Context, id []byte) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	return store.Has(id)
}

// IsERC20Registered check if registered ERC20 token is registered
func (k Keeper) IsERC20Registered(ctx sdk.Context, erc20 common.Address) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC20)
	return store.Has(erc20.Bytes())
}

// IsDenomRegistered check if registered coin denom is registered
func (k Keeper) IsDenomRegistered(ctx sdk.Context, denom string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByDenom)
	return store.Has([]byte(denom))
}
