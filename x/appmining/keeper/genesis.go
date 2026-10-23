package keeper

import (
	"strings"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// InitGenesis imports genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := types.ValidateGenesis(gs); err != nil {
		panic(err)
	}
	// Ensure segregated fund accounts exist (genesis and upgrade InitGenesis).
	if acc := k.ak.GetModuleAccount(ctx, types.RewardPoolName); acc == nil {
		panic("the appmining_reward_pool module account has not been set")
	}
	if acc := k.ak.GetModuleAccount(ctx, types.ChallengeEscrowName); acc == nil {
		panic("the appmining_challenge_escrow module account has not been set")
	}
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(err)
	}
	for _, app := range gs.Applications {
		k.SetApplication(ctx, app)
		for _, payer := range app.GasPayers {
			if err := k.SetGasPayerIndex(ctx, payer, app.AppId); err != nil {
				panic(err)
			}
		}
	}
	for _, b := range gs.EpochBudgets {
		k.StoreEpochBudget(ctx, b)
	}
	for _, a := range gs.Attestations {
		k.SetAttestation(ctx, a)
	}
	for _, c := range gs.Challenges {
		k.SetChallenge(ctx, c)
		if c.Status == types.ChallengeStatusOpen {
			k.SetOpenChallengeIndex(ctx, c.AppId, c.Epoch, c.ChallengeId)
		}
	}
	for _, s := range gs.Settlements {
		k.SetSettlement(ctx, s)
	}
	for _, a := range gs.GasCreditAccounts {
		k.SetGasCredit(ctx, a)
	}
	for _, c := range gs.GasCreditConsumptions {
		k.SetGasCreditConsumption(ctx, c)
	}
	next := gs.NextChallengeId
	if next == 0 {
		next = 1
	}
	k.SetNextChallengeID(ctx, next)

	// --- V1 entitlement flow ---
	for _, p := range gs.EntitlementPools {
		k.SetEntitlementPool(ctx, p)
	}
	for _, rk := range gs.RoleKeys {
		k.SetRoleKey(ctx, rk)
	}
	for _, c := range gs.Candidates {
		k.SetCandidate(ctx, c)
	}
	for _, c := range gs.EntitlementChallenges {
		k.SetEntitlementChallenge(ctx, c)
	}
	for _, e := range gs.Entitlements {
		k.SetEntitlement(ctx, e)
	}
	for _, n := range gs.UsedAuthNonces {
		parts := strings.SplitN(n, "/", 3)
		if len(parts) != 3 {
			continue // validated by types.ValidateGenesis
		}
		role, ok := types.RoleFromWire(parts[0])
		if !ok {
			continue
		}
		k.SetAuthNonce(ctx, role, parts[1], parts[2])
	}
}

// ExportGenesis exports full module state (BLOCK-2 fix).
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	gs := types.DefaultGenesisState()
	gs.Params = k.GetParams(ctx)
	gs.NextChallengeId = k.GetNextChallengeID(ctx)

	store := k.store(ctx)

	it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixApplication)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var app types.Application
		k.cdc.MustUnmarshal(it.Value(), &app)
		gs.Applications = append(gs.Applications, app)
	}

	it2 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEpochBudget)
	defer it2.Close()
	for ; it2.Valid(); it2.Next() {
		var b types.EpochBudget
		k.cdc.MustUnmarshal(it2.Value(), &b)
		gs.EpochBudgets = append(gs.EpochBudgets, b)
	}

	it3 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixAttestation)
	defer it3.Close()
	for ; it3.Valid(); it3.Next() {
		var a types.AttestationCommitment
		k.cdc.MustUnmarshal(it3.Value(), &a)
		gs.Attestations = append(gs.Attestations, a)
	}

	it4 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixChallenge)
	defer it4.Close()
	for ; it4.Valid(); it4.Next() {
		var c types.Challenge
		k.cdc.MustUnmarshal(it4.Value(), &c)
		gs.Challenges = append(gs.Challenges, c)
	}

	it5 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixSettlement)
	defer it5.Close()
	for ; it5.Valid(); it5.Next() {
		var s types.Settlement
		k.cdc.MustUnmarshal(it5.Value(), &s)
		gs.Settlements = append(gs.Settlements, s)
	}

	it6 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixGasCredit)
	defer it6.Close()
	for ; it6.Valid(); it6.Next() {
		var a types.GasCreditAccount
		k.cdc.MustUnmarshal(it6.Value(), &a)
		gs.GasCreditAccounts = append(gs.GasCreditAccounts, a)
	}

	it7 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixGasCreditConsumption)
	defer it7.Close()
	for ; it7.Valid(); it7.Next() {
		var c types.GasCreditConsumption
		k.cdc.MustUnmarshal(it7.Value(), &c)
		gs.GasCreditConsumptions = append(gs.GasCreditConsumptions, c)
	}

	// --- V1 entitlement flow ---
	it8 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlementPool)
	defer it8.Close()
	for ; it8.Valid(); it8.Next() {
		var p types.Pool
		k.cdc.MustUnmarshal(it8.Value(), &p)
		gs.EntitlementPools = append(gs.EntitlementPools, p)
	}

	for _, role := range []types.EntitlementRole{
		types.EntitlementRolePublisher,
		types.EntitlementRoleApprover,
		types.EntitlementRoleReviewer,
		types.EntitlementRoleRisk,
		types.EntitlementRoleAudit,
	} {
		k.IterateRoleKeys(ctx, role, func(rk types.RoleKey) bool {
			gs.RoleKeys = append(gs.RoleKeys, rk)
			return true
		})
	}

	it9 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixCandidate)
	defer it9.Close()
	for ; it9.Valid(); it9.Next() {
		var c types.Candidate
		k.cdc.MustUnmarshal(it9.Value(), &c)
		gs.Candidates = append(gs.Candidates, c)
	}

	it10 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlementChallenge)
	defer it10.Close()
	for ; it10.Valid(); it10.Next() {
		var c types.EntitlementChallenge
		k.cdc.MustUnmarshal(it10.Value(), &c)
		gs.EntitlementChallenges = append(gs.EntitlementChallenges, c)
	}

	it11 := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlement)
	defer it11.Close()
	for ; it11.Valid(); it11.Next() {
		var e types.Entitlement
		k.cdc.MustUnmarshal(it11.Value(), &e)
		gs.Entitlements = append(gs.Entitlements, e)
	}

	k.IterateAuthNonces(ctx, func(n string) bool {
		gs.UsedAuthNonces = append(gs.UsedAuthNonces, n)
		return true
	})

	return gs
}
