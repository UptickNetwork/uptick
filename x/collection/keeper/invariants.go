package keeper

// DONTCOVER

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// RegisterInvariants registers all supply invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "supply", SupplyInvariant(k))
}

// SupplyInvariant checks that the total amount of NFTs on collections matches the total amount owned by addresses
func SupplyInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		ownersCollectionsSupply := make(map[string]uint64)
		var msg string
		count := 0

		// GetCollections cannot fail: degradations are per class and are
		// reported as ExportIssues (and logged), so there is no whole-call
		// error to branch on. The previous "failed to get collections" branch
		// was unreachable because the error was always nil.
		for _, collection := range k.GetCollections(ctx) {
			ownersCollectionsSupply[collection.Denom.Id] = uint64(len(collection.NFTs))
		}

		for denom, supply := range ownersCollectionsSupply {
			totalSupply := k.GetTotalSupply(ctx, denom)
			if supply != totalSupply {
				count++
				msg += fmt.Sprintf(
					"total %s NFTs supply invariance:\n"+
						"\ttotal %s NFTs supply (from store): %d\n"+
						"\tsum of %s NFTs by owner: %d\n",
					denom, denom, totalSupply, denom, supply,
				)
			}
		}
		if count != 0 {
			return sdk.FormatInvariant(
				types.ModuleName, "supply",
				fmt.Sprintf("%d NFT supply invariants found\n%s", count, msg),
			), true
		}

		return "", false
	}
}
