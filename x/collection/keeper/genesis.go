package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// InitGenesis stores the NFT genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	if err := types.ValidateGenesis(data); err != nil {
		panic(err.Error())
	}

	for _, c := range data.Collections {
		creator, err := sdk.AccAddressFromBech32(c.Denom.Creator)
		if err != nil {
			panic(err)
		}
		if err := k.SaveDenom(ctx,
			c.Denom.Id,
			c.Denom.Name,
			c.Denom.Schema,
			c.Denom.Symbol,
			creator,
			c.Denom.MintRestricted,
			c.Denom.UpdateRestricted,
			c.Denom.Description,
			c.Denom.Uri,
			c.Denom.UriHash,
			c.Denom.Data,
		); err != nil {
			panic(err)
		}

		if err := k.SaveCollection(ctx, c); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
//
// Degradations reported by GetCollectionsWithReport are logged at Error level
// and summarized: the export still succeeds (no class or NFT is dropped), but
// an operator must be able to see that some metadata fields were lost.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	collections, issues := k.GetCollectionsWithReport(ctx)
	for _, issue := range issues {
		k.Logger(ctx).Error("ExportGenesis: export degraded", "issue", issue.String())
	}
	if len(issues) > 0 {
		k.Logger(ctx).Error(
			"ExportGenesis: exported genesis is partially degraded; listed classes lost metadata fields",
			"degraded_classes", len(issues),
			"exported_classes", len(collections),
		)
	}
	return types.NewGenesisState(collections)
}
