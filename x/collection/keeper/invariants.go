package keeper

// DONTCOVER

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// RegisterInvariants registers all supply invariants
//
// NOTHING CALLS THIS AT RUNTIME. It satisfies module.HasInvariants
// (x/collection/module/module.go), but the only thing that would invoke it --
// module.Manager.RegisterInvariants -- is a deliberate no-op in cosmos-sdk
// v0.53.6 (types/module/module.go:454-457). app/app.go's call therefore
// registers zero routes and every crisis entry point asserts an empty set;
// app/app.go lists the sites.
//
// The supply check is observed on the export path instead
// (GetCollectionsWithReport -> export log). Wiring it into x/crisis is a
// decision, not a cleanup: see app/invariants_wiring_test.go.
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "supply", SupplyInvariant(k))
}

// SupplyInvariant checks that each class' stored total-supply counter matches the
// number of NFTs it holds.
//
// It reports the ExportIssueSupplyMismatch records produced by the export walk
// rather than re-deriving the comparison, so the invariant and the export
// diagnostic can never disagree about what a broken supply is. That walk is the
// module's only costly operation, so the invariant is not run periodically.
func SupplyInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		// GetCollectionsWithReport rather than GetCollections: the latter logs
		// each degradation at Error level, which is right for an export and
		// wrong for a check that is itself reporting them.
		_, issues := k.GetCollectionsWithReport(ctx)

		var msg string
		count := 0
		for _, issue := range issues {
			if issue.Kind != ExportIssueSupplyMismatch {
				continue
			}
			count++
			msg += fmt.Sprintf(
				"total %s NFTs supply invariance:\n"+
					"\t%s\n",
				issue.ClassID, issue.Detail,
			)
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
