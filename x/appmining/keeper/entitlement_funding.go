package keeper

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// sumUnpaidForDenom returns Σ(available + reserved) over every entitlement
// pool that uses denom. Shared module account appmining_entitlement_pool must
// cover this aggregate (UGE doc15 / requirements 1.0).
func (k Keeper) sumUnpaidForDenom(ctx sdk.Context, denom string) (math.Int, error) {
	total := math.ZeroInt()
	store := k.store(ctx)
	it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlementPool)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var p types.Pool
		k.cdc.MustUnmarshal(it.Value(), &p)
		if p.Denom != denom {
			continue
		}
		unpaidStr, err := types.UnpaidLiability(p)
		if err != nil {
			return math.Int{}, err
		}
		unpaid, err := types.ParseAmount(unpaidStr)
		if err != nil {
			return math.Int{}, err
		}
		total = total.Add(unpaid)
	}
	return total, nil
}

// IsEntitlementFlowEnabled reports whether V1 business msgs may run for pool:
// per-pool pause/min-funded gates, plus denom-aggregated
// available+reserved <= module balance (covers multi-pool / genesis skew).
func (k Keeper) IsEntitlementFlowEnabled(ctx sdk.Context, pool types.Pool) bool {
	params := k.GetParams(ctx)
	modAddr := k.ak.GetModuleAddress(types.EntitlementPoolName)
	bal := k.bank.GetBalance(ctx, modAddr, pool.Denom).Amount
	// Per-pool unpaid is a fast reject; aggregate is authoritative for shared accounts.
	if !types.EntitlementFlowEnabled(pool, params, types.AmountString(bal)) {
		return false
	}
	unpaid, err := k.sumUnpaidForDenom(ctx, pool.Denom)
	if err != nil {
		return false
	}
	return !bal.LT(unpaid)
}
