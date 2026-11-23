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
	ir.RegisterRoute(types.ModuleName, "challenge-index-consistent", ChallengeIndexInvariant(k))
}

// ChallengeIndexInvariant checks that every V1 challenge record is reachable
// through the candidate index and that every OPEN record sits in the timeout
// bucket (and settled records do not).
//
// Those two structures are derived state: if they drift, an OPEN challenge
// becomes invisible to approve (which would release funds it must block) or
// stops being swept. Migration 1→2 is the only writer that can legitimately
// leave them stale while it runs, so any break outside an upgrade is a real
// fault. Run with: uptickd query crisis invariant appmining challenge-index-consistent
func ChallengeIndexInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		// Before the one-time rebuild completes, records written by a v1 binary
		// legitimately have no index entry yet, so firing here would report a
		// fault on a consistent chain. Once ready, the check is strict.
		if !k.ChallengeIndexesReady(ctx) {
			return "", false
		}
		store := k.store(ctx)
		broken := 0
		var first string
		it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlementChallenge)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var c types.EntitlementChallenge
			k.cdc.MustUnmarshal(it.Value(), &c)
			report := func(msg string) {
				broken++
				if first == "" {
					first = msg
				}
			}
			if c.CandidateId == "" {
				report(fmt.Sprintf("challenge %q has empty candidate_id", c.ChallengeId))
			} else if !store.Has(types.EntitlementChallengeByCandidateKey(c.CandidateId, c.ChallengeId)) {
				report(fmt.Sprintf("challenge %q missing candidate index for %q", c.ChallengeId, c.CandidateId))
			}
			forOpen := c.Status == types.EntitlementChallengeStatusOpen && c.ResolutionDeadline > 0
			hasBucket := store.Has(types.ChallengeResolutionDeadlineKey(c.ResolutionDeadline, c.ChallengeId))
			if forOpen && !hasBucket {
				report(fmt.Sprintf("open challenge %q missing timeout bucket at %d", c.ChallengeId, c.ResolutionDeadline))
			}
			if !forOpen && c.ResolutionDeadline > 0 && hasBucket {
				report(fmt.Sprintf("settled challenge %q still occupies timeout bucket at %d", c.ChallengeId, c.ResolutionDeadline))
			}
		}
		if broken > 0 {
			return sdk.FormatInvariant(types.ModuleName, "challenge-index-consistent",
				fmt.Sprintf("%d inconsistent challenge index entries; first: %s", broken, first)), true
		}
		return "", false
	}
}

// EntitlementPoolInvariant checks funded = available + reserved + paid per pool
// and, per denom, module balance >= Σ(available + reserved) across all pools
// sharing that denom (UGE doc15: unpaid liability must be fully backed).
func EntitlementPoolInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		unpaidByDenom := map[string]math.Int{}
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
			unpaidStr, err := types.UnpaidLiability(p)
			if err != nil {
				return sdk.FormatInvariant(types.ModuleName, "entitlement-pool-invariant",
					fmt.Sprintf("pool %q unpaid: %v", p.PoolId, err)), true
			}
			unpaid, err := types.ParseAmount(unpaidStr)
			if err != nil {
				return sdk.FormatInvariant(types.ModuleName, "entitlement-pool-invariant",
					fmt.Sprintf("pool %q unpaid parse: %v", p.PoolId, err)), true
			}
			cur, ok := unpaidByDenom[p.Denom]
			if !ok {
				cur = math.ZeroInt()
			}
			unpaidByDenom[p.Denom] = cur.Add(unpaid)
		}
		modAddr := k.ak.GetModuleAddress(types.EntitlementPoolName)
		for denom, want := range unpaidByDenom {
			have := k.bank.GetBalance(ctx, modAddr, denom).Amount
			if have.LT(want) {
				return sdk.FormatInvariant(types.ModuleName, "entitlement-pool-invariant",
					fmt.Sprintf("denom %s module balance %s < unpaid available+reserved %s",
						denom, have, want)), true
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

// OpenChallengeEscrowInvariant checks challenge escrow balance covers sum of
// OPEN attestation bonds plus OPEN V1 entitlement challenge deposits per denom.
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
		eit := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlementChallenge)
		defer eit.Close()
		for ; eit.Valid(); eit.Next() {
			var c types.EntitlementChallenge
			k.cdc.MustUnmarshal(eit.Value(), &c)
			if c.Status != types.EntitlementChallengeStatusOpen || c.DepositAmount == "" {
				continue
			}
			amt, err := types.ParseAmount(c.DepositAmount)
			if err != nil {
				return sdk.FormatInvariant(types.ModuleName, "open-challenge-escrow",
					fmt.Sprintf("entitlement challenge %s bad deposit: %v", c.ChallengeId, err)), true
			}
			cur, ok := needed[c.DepositDenom]
			if !ok {
				cur = math.ZeroInt()
			}
			needed[c.DepositDenom] = cur.Add(amt)
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
