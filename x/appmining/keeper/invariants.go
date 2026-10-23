package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// RegisterInvariants registers appmining ledger invariants.
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "epoch-spent-le-limit", EpochSpentInvariant(k))
	ir.RegisterRoute(types.ModuleName, "one-settlement-per-app-epoch", OneSettlementInvariant(k))
	ir.RegisterRoute(types.ModuleName, "gas-credit-conservation", GasCreditConservationInvariant(k))
	ir.RegisterRoute(types.ModuleName, "open-challenge-escrow", OpenChallengeEscrowInvariant(k))
	ir.RegisterRoute(types.ModuleName, "settlement-attestation-consistent", SettlementAttestationInvariant(k))
	ir.RegisterRoute(types.ModuleName, "open-budget-pool-covered", OpenBudgetPoolInvariant(k))
	ir.RegisterRoute(types.ModuleName, "entitlement-pool-invariant", EntitlementPoolInvariant(k))
}

// EntitlementPoolInvariant checks funded = available + reserved + paid and
// reserved <= module account balance for every V1 entitlement pool.
func EntitlementPoolInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		store := k.store(ctx)
		it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlementPool)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var p types.Pool
			k.cdc.MustUnmarshal(it.Value(), &p)
			if err := types.ValidatePoolInvariant(p); err != nil {
				return sdk.FormatInvariant(types.ModuleName, "entitlement-pool-invariant",
					fmt.Sprintf("pool %q: %v", p.PoolId, err)), true
			}
			modAddr := k.ak.GetModuleAddress(types.EntitlementPoolName)
			reserved, err := types.ParseAmount(p.Reserved)
			if err != nil {
				return sdk.FormatInvariant(types.ModuleName, "entitlement-pool-invariant",
					fmt.Sprintf("pool %q reserved parse: %v", p.PoolId, err)), true
			}
			bal := k.bank.GetBalance(ctx, modAddr, p.Denom).Amount
			if bal.LT(reserved) {
				return sdk.FormatInvariant(types.ModuleName, "entitlement-pool-invariant",
					fmt.Sprintf("pool %q module balance %s < reserved %s (%s)",
						p.PoolId, bal, p.Reserved, p.Denom)), true
			}
		}
		return sdk.FormatInvariant(types.ModuleName, "entitlement-pool-invariant", "ok"), false
	}
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

// GasCreditConservationInvariant checks balance + total_consumed == total_issued.
func GasCreditConservationInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		store := k.store(ctx)
		it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixGasCredit)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var a types.GasCreditAccount
			k.cdc.MustUnmarshal(it.Value(), &a)
			sum, err := types.AddAmountStrings(a.Balance, a.TotalConsumed)
			if err != nil {
				return sdk.FormatInvariant(types.ModuleName, "gas-credit-conservation",
					fmt.Sprintf("app %x parse error: %v", a.AppId, err)), true
			}
			cmp, err := types.CompareAmountStrings(sum, a.TotalIssued)
			if err != nil || cmp != 0 {
				return sdk.FormatInvariant(types.ModuleName, "gas-credit-conservation",
					fmt.Sprintf("app %x balance(%s)+consumed(%s) != issued(%s)",
						a.AppId, a.Balance, a.TotalConsumed, a.TotalIssued)), true
			}
		}
		return sdk.FormatInvariant(types.ModuleName, "gas-credit-conservation", "ok"), false
	}
}

// OpenChallengeEscrowInvariant checks challenge escrow balance covers sum of OPEN bonds per denom.
func OpenChallengeEscrowInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		needed := map[string]math.Int{}
		store := k.store(ctx)
		it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixChallenge)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var c types.Challenge
			k.cdc.MustUnmarshal(it.Value(), &c)
			if c.Status != types.ChallengeStatusOpen {
				continue
			}
			amt, err := types.ParseAmount(c.BondAmount)
			if err != nil {
				return sdk.FormatInvariant(types.ModuleName, "open-challenge-escrow",
					fmt.Sprintf("challenge %d bad bond: %v", c.ChallengeId, err)), true
			}
			cur, ok := needed[c.BondDenom]
			if !ok {
				cur = math.ZeroInt()
			}
			needed[c.BondDenom] = cur.Add(amt)
		}
		escrow := k.ak.GetModuleAddress(types.ChallengeEscrowName)
		for denom, want := range needed {
			have := k.bank.GetBalance(ctx, escrow, denom).Amount
			if have.LT(want) {
				return sdk.FormatInvariant(types.ModuleName, "open-challenge-escrow",
					fmt.Sprintf("escrow %s has %s < open bonds %s", denom, have, want)), true
			}
		}
		return sdk.FormatInvariant(types.ModuleName, "open-challenge-escrow", "ok"), false
	}
}

// OpenBudgetPoolInvariant checks sum(limit-spent) over open budgets per denom
// does not exceed the reward pool balance (vector: released/commitment vs pool).
func OpenBudgetPoolInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		needed := map[string]math.Int{}
		store := k.store(ctx)
		it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEpochBudget)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var b types.EpochBudget
			k.cdc.MustUnmarshal(it.Value(), &b)
			if b.Closed {
				continue
			}
			rem, err := remainingBudget(b)
			if err != nil {
				return sdk.FormatInvariant(types.ModuleName, "open-budget-pool-covered",
					fmt.Sprintf("epoch %d remaining: %v", b.Epoch, err)), true
			}
			cur, ok := needed[b.Denom]
			if !ok {
				cur = math.ZeroInt()
			}
			needed[b.Denom] = cur.Add(rem)
		}
		pool := k.ak.GetModuleAddress(types.RewardPoolName)
		for denom, want := range needed {
			have := k.bank.GetBalance(ctx, pool, denom).Amount
			if have.LT(want) {
				return sdk.FormatInvariant(types.ModuleName, "open-budget-pool-covered",
					fmt.Sprintf("pool %s has %s < open budget remaining %s", denom, have, want)), true
			}
		}
		return sdk.FormatInvariant(types.ModuleName, "open-budget-pool-covered", "ok"), false
	}
}

// SettlementAttestationInvariant checks each settlement references a FINALIZED attestation.
func SettlementAttestationInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		store := k.store(ctx)
		it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixSettlement)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var s types.Settlement
			k.cdc.MustUnmarshal(it.Value(), &s)
			att, ok := k.GetAttestation(ctx, s.AppId, s.Epoch)
			if !ok {
				return sdk.FormatInvariant(types.ModuleName, "settlement-attestation-consistent",
					fmt.Sprintf("settlement %x/%d missing attestation", s.AppId, s.Epoch)), true
			}
			if att.Status != types.AttestationStatusFinalized {
				return sdk.FormatInvariant(types.ModuleName, "settlement-attestation-consistent",
					fmt.Sprintf("settlement %x/%d attestation status %s", s.AppId, s.Epoch, att.Status)), true
			}
			if _, ok := k.GetApplication(ctx, s.AppId); !ok {
				return sdk.FormatInvariant(types.ModuleName, "settlement-attestation-consistent",
					fmt.Sprintf("settlement %x/%d missing application", s.AppId, s.Epoch)), true
			}
		}
		return sdk.FormatInvariant(types.ModuleName, "settlement-attestation-consistent", "ok"), false
	}
}
