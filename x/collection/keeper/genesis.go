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
		// An empty creator is a legitimate on-chain state, not corruption:
		// ICS-721 voucher classes are written straight into the underlying nft
		// store by nft-transfer and have no collection-level issuer at all,
		// and ValidateGenesis accepts them. Only a NON-empty value that does
		// not decode is damage, and validation has already rejected that.
		var creator sdk.AccAddress
		if c.Denom.Creator != "" {
			acc, err := sdk.AccAddressFromBech32(c.Denom.Creator)
			if err != nil {
				panic(err)
			}
			creator = acc
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
// an operator must be able to see that some metadata fields were lost. The
// app-level export path additionally writes the same list to
// <home>/export-issues.json (see app/export_diagnostics.go) so the report
// survives the process that produced it.
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
