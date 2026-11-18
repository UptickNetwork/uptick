package keeper

import (
	"bytes"
	"encoding/binary"
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

// SetEntitlementChallenge is the ONLY write path for V1 challenge records.
// Besides the record itself it maintains two derived structures so that
// per-candidate reads never scan the whole history:
//
//   - candidate index: candidate_id | 0x00 | challenge_id -> challenge_id
//   - timeout bucket:  be(resolution_deadline) | challenge_id -> challenge_id
//
// Keeping maintenance here (rather than in each handler) means open, resolve,
// expire and genesis cannot drift apart. Both writes are idempotent.
func (k Keeper) SetEntitlementChallenge(ctx sdk.Context, c types.EntitlementChallenge) {
	store := k.store(ctx)

	// The previous revision may carry a different deadline (or none). Drop its
	// bucket before writing the new one so the sweeper never revisits it.
	if prev, ok := k.GetEntitlementChallenge(ctx, c.ChallengeId); ok {
		k.deleteChallengeDeadlineBucket(ctx, prev)
	}

	store.Set(types.EntitlementChallengeKey(c.ChallengeId), k.cdc.MustMarshal(&c))

	// candidate_id is the entitlement_id (CandidateKey is keyed by it).
	if c.CandidateId != "" {
		store.Set(
			types.EntitlementChallengeByCandidateKey(c.CandidateId, c.ChallengeId),
			[]byte(c.ChallengeId),
		)
	}

	// Only OPEN challenges are sweepable; settled ones must not occupy the
	// timeout index forever.
	if c.Status == types.EntitlementChallengeStatusOpen && c.ResolutionDeadline > 0 {
		store.Set(
			types.ChallengeResolutionDeadlineKey(c.ResolutionDeadline, c.ChallengeId),
			[]byte(c.ChallengeId),
		)
	}
}

// deleteChallengeDeadlineBucket removes the timeout bucket of one revision.
func (k Keeper) deleteChallengeDeadlineBucket(ctx sdk.Context, c types.EntitlementChallenge) {
	if c.ResolutionDeadline <= 0 {
		return
	}
	k.store(ctx).Delete(types.ChallengeResolutionDeadlineKey(c.ResolutionDeadline, c.ChallengeId))
}

// BackfillChallengeIndexes rebuilds both derived structures from the record
// table. It is idempotent and is run once from Migrate1to2 so chains that
// already hold V1 challenges before this change keep correct per-candidate
// reads. Returns the number of records visited.
func (k Keeper) BackfillChallengeIndexes(ctx sdk.Context) (int, error) {
	var records []types.EntitlementChallenge
	it := storetypes.KVStorePrefixIterator(k.store(ctx), types.KeyPrefixEntitlementChallenge)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var c types.EntitlementChallenge
		if err := k.cdc.Unmarshal(it.Value(), &c); err != nil {
			return 0, err
		}
		records = append(records, c)
	}
	grace := k.GetParams(ctx).EffectiveChallengeResolutionGraceSeconds()
	for _, c := range records {
		// Records written before resolution_deadline existed carry 0. Seal them
		// now (with the migration-time grace) so the sweeper can reach them;
		// otherwise an old OPEN challenge would never time out automatically.
		if c.Status == types.EntitlementChallengeStatusOpen && c.ResolutionDeadline <= 0 {
			if cand, ok := k.GetCandidate(ctx, c.CandidateId); ok {
				c.ResolutionDeadline = types.EffectiveResolutionDeadline(cand, c, grace)
			}
		}
		k.SetEntitlementChallenge(ctx, c)
	}
	return len(records), nil
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

// iterateChallengesOfCandidate walks only the challenges of one candidate via
// the candidate index. Cost is O(challenges of that candidate), independent of
// how many challenges the chain has recorded in total.
//
// The index stores challenge ids; a dangling id (impossible unless the store is
// corrupted) is skipped rather than panicking inside a tx.
func (k Keeper) iterateChallengesOfCandidate(ctx sdk.Context, candidateID string, fn func(types.EntitlementChallenge) bool) {
	it := storetypes.KVStorePrefixIterator(k.store(ctx), types.EntitlementChallengeByCandidatePrefix(candidateID))
	defer it.Close()
	for ; it.Valid(); it.Next() {
		c, ok := k.GetEntitlementChallenge(ctx, string(it.Value()))
		if !ok {
			continue
		}
		if !fn(c) {
			return
		}
	}
}

// IterateEntitlementChallengesByCandidate is the exported per-candidate walk.
func (k Keeper) IterateEntitlementChallengesByCandidate(ctx sdk.Context, candidateID string, fn func(types.EntitlementChallenge) bool) {
	k.iterateChallengesOfCandidate(ctx, candidateID, fn)
}

// HasOpenOrUpheldEntitlementChallenge reports whether any challenge for the
// candidate is OPEN or UPHELD (blocks approval). DISMISSED/EXPIRED do not.
func (k Keeper) HasOpenOrUpheldEntitlementChallenge(ctx sdk.Context, entitlementID string) bool {
	found := false
	k.iterateChallengesOfCandidate(ctx, entitlementID, func(c types.EntitlementChallenge) bool {
		if c.Status == types.EntitlementChallengeStatusOpen || c.Status == types.EntitlementChallengeStatusUpheld {
			found = true
			return false
		}
		return true
	})
	return found
}

// HasAnyEntitlementChallenge reports whether the candidate has any stored
// challenge record (any status). Used by the no-challenge approve path
// (UGE DEC-01 scheme C): empty challenge_resolution_hash requires zero challenges.
func (k Keeper) HasAnyEntitlementChallenge(ctx sdk.Context, entitlementID string) bool {
	it := storetypes.KVStorePrefixIterator(k.store(ctx), types.EntitlementChallengeByCandidatePrefix(entitlementID))
	defer it.Close()
	return it.Valid()
}

// HasSettledEntitlementChallengeResolution reports whether the candidate has a
// settled challenge (DISMISSED by a reviewer, or EXPIRED by timeout) whose
// stored ResolutionDigest equals digest. Both are "no allegation proven" states
// that unblock approval; OPEN and UPHELD are not accepted here.
func (k Keeper) HasSettledEntitlementChallengeResolution(ctx sdk.Context, entitlementID, digest string) bool {
	if digest == "" {
		return false
	}
	found := false
	k.iterateChallengesOfCandidate(ctx, entitlementID, func(c types.EntitlementChallenge) bool {
		if c.ResolutionDigest != digest {
			return true
		}
		if c.Status == types.EntitlementChallengeStatusDismissed || c.Status == types.EntitlementChallengeStatusExpired {
			found = true
			return false
		}
		return true
	})
	return found
}

// IterateEntitlementChallenges walks every entitlement challenge record.
//
// NOT for tx paths: use IterateEntitlementChallengesByCandidate there. This is
// the O(history) walk kept for ExportGenesis, the 1→2 backfill and crisis
// invariants.
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

// SetAuthNonce consumes the wire-v1 nonce. The value is the proof's
// expires_at as 8-byte big-endian, used only for pruning; the genesis string
// ("role_wire/key_id/nonce") is reconstructed from the key, which carries the
// full identity. Legacy entries written before the expiry format keep their
// old string values -- they are never pruned and export identically.
func (k Keeper) SetAuthNonce(ctx sdk.Context, role types.EntitlementRole, keyID, nonce string, expiresAt int64) {
	k.pruneAuthNonces(ctx, role, keyID, expiresAt)
	k.store(ctx).Set(types.AuthNonceKey(role, keyID, nonce), types.Uint64Bytes(uint64(expiresAt)))
}

// pruneAuthNonces deletes consumed nonces of one (role, key_id) whose expiry
// passed params.nonce_retention_seconds. Safe because replaying a pruned nonce
// requires resubmitting its original signed record, whose expires_at has
// already expired (now > exp fails verifyWireAuth). Entries with unparseable
// values (legacy pre-expiry format) are kept.
func (k Keeper) pruneAuthNonces(ctx sdk.Context, role types.EntitlementRole, keyID string, now int64) {
	cutoff := now - k.GetParams(ctx).EffectiveNonceRetentionSeconds()
	prefix := append(append([]byte{}, types.KeyPrefixAuthNonce...), byte(role))
	prefix = append(prefix, []byte(keyID)...)
	prefix = append(prefix, 0)
	store := k.store(ctx)
	var toDelete [][]byte
	it := storetypes.KVStorePrefixIterator(store, prefix)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		if v := it.Value(); len(v) == 8 {
			if int64(binary.BigEndian.Uint64(v)) < cutoff {
				toDelete = append(toDelete, it.Key())
			}
		}
	}
	for _, key := range toDelete {
		store.Delete(key)
	}
}

// IterateAuthNonces walks consumed wire-v1 nonces ("role/key_id/nonce"),
// reconstructing the identity from the store key so the value format stays
// free to evolve.
func (k Keeper) IterateAuthNonces(ctx sdk.Context, fn func(string) bool) {
	it := storetypes.KVStorePrefixIterator(k.store(ctx), types.KeyPrefixAuthNonce)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		if n, ok := authNonceIdentityFromKey(it.Key()); ok {
			if !fn(n) {
				return
			}
		}
	}
}

// authNonceIdentityFromKey parses "prefix | role | key_id | 0x00 | nonce"
// back into "role_wire/key_id/nonce". key_id is validated by ValidateWireID
// ([A-Za-z0-9._:-]), so 0x00 is an unambiguous separator.
func authNonceIdentityFromKey(key []byte) (string, bool) {
	rest, ok := bytes.CutPrefix(key, types.KeyPrefixAuthNonce)
	if !ok || len(rest) < 3 {
		return "", false
	}
	roleWire, ok := types.RoleWire(types.EntitlementRole(rest[0]))
	if !ok {
		return "", false
	}
	sep := bytes.IndexByte(rest[1:], 0)
	if sep < 0 {
		return "", false
	}
	keyID := string(rest[1 : 1+sep])
	nonce := string(rest[2+sep:])
	if keyID == "" || nonce == "" {
		return "", false
	}
	return roleWire + "/" + keyID + "/" + nonce, true
}

func (k Keeper) SetAssessmentSigner(ctx sdk.Context, s types.AssessmentSigner) {
	k.store(ctx).Set(types.AssessmentSignerKey(s.KeyId), k.cdc.MustMarshal(&s))
}

func (k Keeper) GetAssessmentSigner(ctx sdk.Context, keyID string) (types.AssessmentSigner, bool) {
	bz := k.store(ctx).Get(types.AssessmentSignerKey(keyID))
	if bz == nil {
		return types.AssessmentSigner{}, false
	}
	var s types.AssessmentSigner
	k.cdc.MustUnmarshal(bz, &s)
	return s, true
}

func (k Keeper) IterateAssessmentSigners(ctx sdk.Context, fn func(types.AssessmentSigner) bool) {
	it := storetypes.KVStorePrefixIterator(k.store(ctx), types.KeyPrefixAssessmentSigner)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var s types.AssessmentSigner
		k.cdc.MustUnmarshal(it.Value(), &s)
		if !fn(s) {
			return
		}
	}
}

// HasCredentialID reports whether credential_id was consumed. Index entries are
// never deleted — including after candidate REJECTED (D-A permanent one-shot).
func (k Keeper) HasCredentialID(ctx sdk.Context, credentialID string) bool {
	return k.store(ctx).Has(types.CredentialIndexKey(credentialID))
}

func (k Keeper) SetCredentialIndex(ctx sdk.Context, credentialID, entitlementID string) {
	k.store(ctx).Set(types.CredentialIndexKey(credentialID), []byte(entitlementID))
}

func (k Keeper) GetCredentialEntitlement(ctx sdk.Context, credentialID string) (string, bool) {
	bz := k.store(ctx).Get(types.CredentialIndexKey(credentialID))
	if bz == nil {
		return "", false
	}
	return string(bz), true
}
