package keeper

// DONTCOVER

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// RegisterInvariants registers all supply invariants
//
// NOTHING CALLS THIS AT RUNTIME. It satisfies module.HasInvariants, which
// x/collection/module/module.go implements; the only caller of that interface
// method is module.Manager.RegisterInvariants, and cosmos-sdk v0.53.6 ships it
// as a deliberate no-op (types/module/module.go:454-457). The call in
// app/app.go therefore registers zero routes, and every crisis entry point that
// asserts invariants then asserts an empty set.
//
// The supply check is observed through the export path instead
// (GetCollectionsWithReport -> export log). Wiring it into x/crisis is a
// decision, not a cleanup: see app/invariants_wiring_test.go.
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "supply", SupplyInvariant(k))
}

// SupplyInvariant checks that the total amount of NFTs on collections matches
// the total amount owned by addresses.
//
// It reports the ExportIssueSupplyMismatch records produced by the export walk
// rather than re-deriving the comparison, so the invariant and the export
// diagnostic can never disagree about what a broken supply is. The walk is the
// only thing in this module that costs anything, which is why the invariant is
// not run periodically anywhere.
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
