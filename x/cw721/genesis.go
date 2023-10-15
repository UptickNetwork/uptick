package cw721

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/UptickNetwork/uptick/x/cw721/keeper"
	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	data types.GenesisState,
) {

	k.SetParams(ctx, data.Params)

	// ensure cw721 module account is set on genesis
	if acc := accountKeeper.GetModuleAccount(ctx, types.ModuleName); acc == nil {
		panic("the cw721 module account has not been set")
	}

	for _, pair := range data.TokenPairs {
		id := pair.GetID()
		k.SetTokenPair(ctx, pair)
		k.SetClassMap(ctx, pair.ClassId, id)
		k.SetCW721Map(ctx, pair.GetCw721Address(), id)

	}
}

// ExportGenesis export module status
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:     k.GetParams(ctx),
		TokenPairs: k.GetTokenPairs(ctx),
	}
}
