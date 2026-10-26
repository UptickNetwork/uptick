package keeper

import (
	"encoding/hex"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func (k Keeper) SetParams(ctx sdk.Context, p types.Params) error {
	if err := p.Validate(); err != nil {
		return err
	}
	bz := k.cdc.MustMarshal(&p)
	k.store(ctx).Set(types.KeyPrefixParams, bz)
	return nil
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	bz := k.store(ctx).Get(types.KeyPrefixParams)
	if bz == nil {
		return types.DefaultParams()
	}
	var p types.Params
	k.cdc.MustUnmarshal(bz, &p)
	if p.PauseState == nil {
		p.PauseState = &types.PauseState{}
	}
	return p
}

func (k Keeper) SetApplication(ctx sdk.Context, app types.Application) {
	k.store(ctx).Set(types.ApplicationKey(app.AppId), k.cdc.MustMarshal(&app))
}

func (k Keeper) GetApplication(ctx sdk.Context, appID []byte) (types.Application, bool) {
	bz := k.store(ctx).Get(types.ApplicationKey(appID))
	if bz == nil {
		return types.Application{}, false
	}
	var app types.Application
	k.cdc.MustUnmarshal(bz, &app)
	return app, true
}

func (k Keeper) StoreEpochBudget(ctx sdk.Context, b types.EpochBudget) {
	k.store(ctx).Set(types.EpochBudgetKey(b.Epoch), k.cdc.MustMarshal(&b))
}

func (k Keeper) GetEpochBudget(ctx sdk.Context, epoch uint64) (types.EpochBudget, bool) {
	bz := k.store(ctx).Get(types.EpochBudgetKey(epoch))
	if bz == nil {
		return types.EpochBudget{}, false
	}
	var b types.EpochBudget
	k.cdc.MustUnmarshal(bz, &b)
	return b, true
}

func (k Keeper) SetAttestation(ctx sdk.Context, a types.AttestationCommitment) {
	k.store(ctx).Set(types.AttestationKey(a.AppId, a.Epoch), k.cdc.MustMarshal(&a))
}

func (k Keeper) GetAttestation(ctx sdk.Context, appID []byte, epoch uint64) (types.AttestationCommitment, bool) {
	bz := k.store(ctx).Get(types.AttestationKey(appID, epoch))
	if bz == nil {
		return types.AttestationCommitment{}, false
	}
	var a types.AttestationCommitment
	k.cdc.MustUnmarshal(bz, &a)
	return a, true
}

func (k Keeper) SetChallenge(ctx sdk.Context, c types.Challenge) {
	k.store(ctx).Set(types.ChallengeKey(c.ChallengeId), k.cdc.MustMarshal(&c))
}

func (k Keeper) GetChallenge(ctx sdk.Context, id uint64) (types.Challenge, bool) {
	bz := k.store(ctx).Get(types.ChallengeKey(id))
	if bz == nil {
		return types.Challenge{}, false
	}
	var c types.Challenge
	k.cdc.MustUnmarshal(bz, &c)
	return c, true
}

func (k Keeper) SetSettlement(ctx sdk.Context, s types.Settlement) {
	k.store(ctx).Set(types.SettlementKey(s.AppId, s.Epoch), k.cdc.MustMarshal(&s))
}

func (k Keeper) GetSettlement(ctx sdk.Context, appID []byte, epoch uint64) (types.Settlement, bool) {
	bz := k.store(ctx).Get(types.SettlementKey(appID, epoch))
	if bz == nil {
		return types.Settlement{}, false
	}
	var s types.Settlement
	k.cdc.MustUnmarshal(bz, &s)
	return s, true
}

func (k Keeper) SetGasCredit(ctx sdk.Context, a types.GasCreditAccount) {
	k.store(ctx).Set(types.GasCreditKey(a.AppId), k.cdc.MustMarshal(&a))
}

func (k Keeper) GetGasCredit(ctx sdk.Context, appID []byte) (types.GasCreditAccount, bool) {
	bz := k.store(ctx).Get(types.GasCreditKey(appID))
	if bz == nil {
		return types.GasCreditAccount{}, false
	}
	var a types.GasCreditAccount
	k.cdc.MustUnmarshal(bz, &a)
	return a, true
}

func (k Keeper) GetOrCreateGasCredit(ctx sdk.Context, appID []byte) types.GasCreditAccount {
	if a, ok := k.GetGasCredit(ctx, appID); ok {
		return a
	}
	return types.GasCreditAccount{
		AppId:         appID,
		Balance:       "0",
		TotalIssued:   "0",
		TotalConsumed: "0",
	}
}

func (k Keeper) HasGasCreditConsumption(ctx sdk.Context, appID, sourceTx []byte) bool {
	return k.store(ctx).Has(types.GasCreditConsumptionKey(appID, sourceTx))
}

func (k Keeper) SetGasCreditConsumption(ctx sdk.Context, c types.GasCreditConsumption) {
	k.store(ctx).Set(types.GasCreditConsumptionKey(c.AppId, c.SourceTxHash), k.cdc.MustMarshal(&c))
}

func (k Keeper) GetNextChallengeID(ctx sdk.Context) uint64 {
	bz := k.store(ctx).Get(types.KeyPrefixNextChallengeID)
	if bz == nil {
		return 1
	}
	return binaryUint64(bz)
}

func (k Keeper) SetNextChallengeID(ctx sdk.Context, id uint64) {
	k.store(ctx).Set(types.KeyPrefixNextChallengeID, types.Uint64Bytes(id))
}

func (k Keeper) SetOpenChallengeIndex(ctx sdk.Context, appID []byte, epoch, challengeID uint64) {
	k.store(ctx).Set(types.OpenChallengeIndexKey(appID, epoch, challengeID), []byte{1})
}

func (k Keeper) DeleteOpenChallengeIndex(ctx sdk.Context, appID []byte, epoch, challengeID uint64) {
	k.store(ctx).Delete(types.OpenChallengeIndexKey(appID, epoch, challengeID))
}

func (k Keeper) HasOpenChallenge(ctx sdk.Context, appID []byte, epoch uint64) bool {
	store := k.store(ctx)
	prefix := types.OpenChallengeIndexPrefix(appID, epoch)
	it := storetypes.KVStorePrefixIterator(store, prefix)
	defer it.Close()
	return it.Valid()
}

func binaryUint64(bz []byte) uint64 {
	if len(bz) < 8 {
		return 1
	}
	var v uint64
	for _, b := range bz[:8] {
		v = (v << 8) | uint64(b)
	}
	return v
}

// eventWithAppID builds an event that always carries app_id + app_id_display (Q-02).
func eventWithAppID(etype string, appID []byte, extra ...sdk.Attribute) sdk.Event {
	attrs := []sdk.Attribute{
		sdk.NewAttribute(types.AttributeKeyAppID, hex.EncodeToString(appID)),
		sdk.NewAttribute(types.AttributeKeyAppIDDisplay, types.AppIDDisplay(appID)),
	}
	attrs = append(attrs, extra...)
	return sdk.NewEvent(etype, attrs...)
}

func hexBytes(b []byte) string {
	return hex.EncodeToString(b)
}

// ListOpenChallengeIDs returns open challenge ids for (app, epoch).
func (k Keeper) ListOpenChallengeIDs(ctx sdk.Context, appID []byte, epoch uint64) []uint64 {
	store := k.store(ctx)
	prefix := types.OpenChallengeIndexPrefix(appID, epoch)
	it := storetypes.KVStorePrefixIterator(store, prefix)
	defer it.Close()
	var ids []uint64
	for ; it.Valid(); it.Next() {
		key := it.Key()
		if len(key) < 8 {
			continue
		}
		ids = append(ids, binaryUint64(key[len(key)-8:]))
	}
	return ids
}

// --- V1 entitlement flow storage ---

func (k Keeper) SetEntitlementPool(ctx sdk.Context, p types.Pool) {
	k.store(ctx).Set(types.EntitlementPoolKey(p.PoolId), k.cdc.MustMarshal(&p))
}

func (k Keeper) GetEntitlementPool(ctx sdk.Context, poolID string) (types.Pool, bool) {
	bz := k.store(ctx).Get(types.EntitlementPoolKey(poolID))
	if bz == nil {
		return types.Pool{}, false
	}
	var p types.Pool
	k.cdc.MustUnmarshal(bz, &p)
	return p, true
}

func (k Keeper) SetCandidate(ctx sdk.Context, c types.Candidate) {
	k.store(ctx).Set(types.CandidateKey(c.EntitlementId), k.cdc.MustMarshal(&c))
}

func (k Keeper) GetCandidate(ctx sdk.Context, entitlementID string) (types.Candidate, bool) {
	bz := k.store(ctx).Get(types.CandidateKey(entitlementID))
	if bz == nil {
		return types.Candidate{}, false
	}
	var c types.Candidate
	k.cdc.MustUnmarshal(bz, &c)
	return c, true
}

func (k Keeper) SetEntitlement(ctx sdk.Context, e types.Entitlement) {
	k.store(ctx).Set(types.EntitlementKey(e.EntitlementId), k.cdc.MustMarshal(&e))
}

func (k Keeper) GetEntitlement(ctx sdk.Context, entitlementID string) (types.Entitlement, bool) {
	bz := k.store(ctx).Get(types.EntitlementKey(entitlementID))
	if bz == nil {
		return types.Entitlement{}, false
	}
	var e types.Entitlement
	k.cdc.MustUnmarshal(bz, &e)
	return e, true
}

func (k Keeper) SetEntitlementChallenge(ctx sdk.Context, c types.EntitlementChallenge) {
	k.store(ctx).Set(types.EntitlementChallengeKey(c.ChallengeId), k.cdc.MustMarshal(&c))
}

func (k Keeper) GetEntitlementChallenge(ctx sdk.Context, challengeID string) (types.EntitlementChallenge, bool) {
	bz := k.store(ctx).Get(types.EntitlementChallengeKey(challengeID))
	if bz == nil {
		return types.EntitlementChallenge{}, false
	}
	var c types.EntitlementChallenge
	k.cdc.MustUnmarshal(bz, &c)
	return c, true
}

func (k Keeper) SetOpenEntitlementChallenge(ctx sdk.Context, entitlementID, challenger, challengeID string) {
	k.store(ctx).Set(types.OpenEntitlementChallengeKey(entitlementID, challenger), []byte(challengeID))
}

func (k Keeper) GetOpenEntitlementChallenge(ctx sdk.Context, entitlementID, challenger string) (string, bool) {
	bz := k.store(ctx).Get(types.OpenEntitlementChallengeKey(entitlementID, challenger))
	if bz == nil {
		return "", false
	}
	return string(bz), true
}

func (k Keeper) DeleteOpenEntitlementChallenge(ctx sdk.Context, entitlementID, challenger string) {
	k.store(ctx).Delete(types.OpenEntitlementChallengeKey(entitlementID, challenger))
}

// HasOpenOrUpheldEntitlementChallenge reports whether any challenge for the
// candidate is OPEN or UPHELD (blocks approval). DISMISSED does not block.
func (k Keeper) HasOpenOrUpheldEntitlementChallenge(ctx sdk.Context, entitlementID string) bool {
	store := k.store(ctx)
	it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlementChallenge)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var c types.EntitlementChallenge
		k.cdc.MustUnmarshal(it.Value(), &c)
		if c.CandidateId != entitlementID {
			continue
		}
		if c.Status == types.EntitlementChallengeStatusOpen || c.Status == types.EntitlementChallengeStatusUpheld {
			return true
		}
	}
	return false
}

// HasDismissedEntitlementChallengeResolution reports whether the candidate has
// a DISMISSED challenge whose stored ResolutionDigest equals digest. Approvals
// must reference such a resolution so challenge_resolution_hash cannot be
// fabricated (see ApproveEntitlement).
func (k Keeper) HasDismissedEntitlementChallengeResolution(ctx sdk.Context, entitlementID, digest string) bool {
	store := k.store(ctx)
	it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEntitlementChallenge)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var c types.EntitlementChallenge
		k.cdc.MustUnmarshal(it.Value(), &c)
		if c.CandidateId != entitlementID {
			continue
		}
		if c.Status == types.EntitlementChallengeStatusDismissed && c.ResolutionDigest == digest {
			return true
		}
	}
	return false
}

// IterateEntitlementChallenges walks every entitlement challenge record.
func (k Keeper) IterateEntitlementChallenges(ctx sdk.Context, fn func(types.EntitlementChallenge) bool) {
	it := storetypes.KVStorePrefixIterator(k.store(ctx), types.KeyPrefixEntitlementChallenge)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var c types.EntitlementChallenge
		k.cdc.MustUnmarshal(it.Value(), &c)
		if !fn(c) {
			return
		}
	}
}

func (k Keeper) SetRoleKey(ctx sdk.Context, rk types.RoleKey) {
	k.store(ctx).Set(types.RoleKeyStoreKey(rk.Role, rk.KeyId), k.cdc.MustMarshal(&rk))
}

func (k Keeper) GetRoleKey(ctx sdk.Context, role types.EntitlementRole, keyID string) (types.RoleKey, bool) {
	bz := k.store(ctx).Get(types.RoleKeyStoreKey(role, keyID))
	if bz == nil {
		return types.RoleKey{}, false
	}
	var rk types.RoleKey
	k.cdc.MustUnmarshal(bz, &rk)
	return rk, true
}

// IterateRoleKeys walks every registered key of one role (key order = key_id).
func (k Keeper) IterateRoleKeys(ctx sdk.Context, role types.EntitlementRole, fn func(types.RoleKey) bool) {
	prefix := append(append([]byte{}, types.KeyPrefixRoleKey...), byte(role))
	it := storetypes.KVStorePrefixIterator(k.store(ctx), prefix)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var rk types.RoleKey
		k.cdc.MustUnmarshal(it.Value(), &rk)
		if !fn(rk) {
			return
		}
	}
}

// HasAuthNonce reports whether the wire-v1 nonce was already consumed.
func (k Keeper) HasAuthNonce(ctx sdk.Context, role types.EntitlementRole, keyID, nonce string) bool {
	return k.store(ctx).Has(types.AuthNonceKey(role, keyID, nonce))
}

// SetAuthNonce consumes the wire-v1 nonce. The value records
// "role_wire/key_id/nonce" so genesis export can reconstruct UsedAuthNonces.
func (k Keeper) SetAuthNonce(ctx sdk.Context, role types.EntitlementRole, keyID, nonce string) {
	roleWire, _ := types.RoleWire(role)
	k.store(ctx).Set(types.AuthNonceKey(role, keyID, nonce), []byte(roleWire+"/"+keyID+"/"+nonce))
}

// IterateAuthNonces walks consumed wire-v1 nonces ("role/key_id/nonce").
func (k Keeper) IterateAuthNonces(ctx sdk.Context, fn func(string) bool) {
	it := storetypes.KVStorePrefixIterator(k.store(ctx), types.KeyPrefixAuthNonce)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		if !fn(string(it.Value())) {
			return
		}
	}
}
