package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// RegisterInvariants registers appmining ledger invariants.
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "epoch-spent-le-limit", EpochSpentInvariant(k))
	ir.RegisterRoute(types.ModuleName, "one-settlement-per-app-epoch", OneSettlementInvariant(k))
}

// EpochSpentInvariant checks spent <= limit for every epoch budget.
func EpochSpentInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		store := k.store(ctx)
		it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEpochBudget)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var b types.EpochBudget
			k.cdc.MustUnmarshal(it.Value(), &b)
			cmp, err := types.CompareAmountStrings(b.Spent, b.Limit)
			if err != nil || cmp > 0 {
				return sdk.FormatInvariant(types.ModuleName, "epoch-spent-le-limit",
					fmt.Sprintf("epoch %d spent %s > limit %s", b.Epoch, b.Spent, b.Limit)), true
			}
		}
		return sdk.FormatInvariant(types.ModuleName, "epoch-spent-le-limit", "ok"), false
	}
}

// OneSettlementInvariant is a structural check: settlement keys are unique by construction.
func OneSettlementInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		seen := map[string]struct{}{}
		store := k.store(ctx)
		it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixSettlement)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var s types.Settlement
			k.cdc.MustUnmarshal(it.Value(), &s)
			key := string(s.AppId) + "/" + fmt.Sprint(s.Epoch)
			if _, ok := seen[key]; ok {
				return sdk.FormatInvariant(types.ModuleName, "one-settlement-per-app-epoch",
					fmt.Sprintf("duplicate settlement %s", key)), true
			}
			seen[key] = struct{}{}
		}
		return sdk.FormatInvariant(types.ModuleName, "one-settlement-per-app-epoch", "ok"), false
	}
}
