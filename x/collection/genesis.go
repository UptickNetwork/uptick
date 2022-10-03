package collection

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/keeper"
	"github.com/UptickNetwork/uptick/x/collection/types"
)

// InitGenesis stores the collection genesis.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, data types.GenesisState) {

	fmt.Println("xxl InitGenesis collection :::")

	if err := types.ValidateGenesis(data); err != nil {
		panic(err.Error())
	}

	for _, c := range data.Collections {
		if err := k.SetCollection(ctx, c); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	collections, err := k.GetCollections(ctx)
	if err != nil {
		panic(err)
	}
	return types.NewGenesisState(collections)
}

// DefaultGenesisState returns a default genesis state
func DefaultGenesisState() *types.GenesisState {
	return types.NewGenesisState([]types.Collection{})
}
