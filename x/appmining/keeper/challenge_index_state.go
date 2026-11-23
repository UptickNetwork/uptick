package keeper

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// MaxChallengeIndexBackfillPerBlock bounds how many challenge records one
// EndBlocker pass converts into index entries. A chain whose history predates
// the indexes therefore converges over a few blocks instead of one slow block.
//
// This is a wall-clock bound, not a gas bound: Begin/EndBlock run with an
// infinite block gas meter (baseapp sets NewInfiniteGasMeter before them), so a
// large chunk would not be rejected, it would only delay the round.
const MaxChallengeIndexBackfillPerBlock = 256

// ChallengeIndexState is the persisted progress of the one-time index rebuild.
// A single record is enough: the rebuild is a linear walk over the challenge
// table, so "done" plus "where we stopped" fully describes it.
type ChallengeIndexState struct {
	Done   bool
	Cursor string // last challenge_id converted; empty when unstarted or done
}

// GetChallengeIndexState reads the rebuild state. A missing record means the
// indexes were never built, which is exactly the pre-upgrade situation.
func (k Keeper) GetChallengeIndexState(ctx sdk.Context) ChallengeIndexState {
	bz := k.store(ctx).Get(types.ChallengeIndexStateKey())
	if len(bz) == 0 {
		return ChallengeIndexState{}
	}
	return ChallengeIndexState{Done: bz[0] == 1, Cursor: string(bz[1:])}
}

func (k Keeper) setChallengeIndexState(ctx sdk.Context, st ChallengeIndexState) {
	bz := make([]byte, 0, 1+len(st.Cursor))
	if st.Done {
		bz = append(bz, 1)
	} else {
		bz = append(bz, 0)
	}
	bz = append(bz, []byte(st.Cursor)...)
	k.store(ctx).Set(types.ChallengeIndexStateKey(), bz)
}

// ChallengeIndexesReady reports whether every stored challenge record is
// guaranteed to be reachable through the candidate index and the timeout
// buckets. Until it is true, per-candidate reads must fall back to the record
// table: a partially rebuilt index is worse than no index, because a candidate
// with two legacy challenges would look "indexed" after only one of them was
// converted.
func (k Keeper) ChallengeIndexesReady(ctx sdk.Context) bool {
	return k.GetChallengeIndexState(ctx).Done
}

// MarkChallengeIndexesReady declares the indexes complete. Used by the 1→2
// migration after its full backfill, so the EndBlocker rebuild becomes a no-op.
func (k Keeper) MarkChallengeIndexesReady(ctx sdk.Context) {
	k.setChallengeIndexState(ctx, ChallengeIndexState{Done: true})
}

// EnsureChallengeIndexes advances the index rebuild by at most
// MaxChallengeIndexBackfillPerBlock records and returns how many it visited.
//
// It exists so a binary swap that does NOT run a store migration still ends up
// with correct per-candidate reads. Without it, challenges written by a v1
// binary have neither a candidate index entry nor a timeout bucket, which makes
// an OPEN challenge invisible to approve (an approval could then release funds
// that an unresolved challenge should still block) and makes the sweeper never
// see it.
//
// Properties:
//   - Deterministic: reads only module state and params, walks the table in
//     key order, so every validator performs byte-identical writes.
//   - Idempotent and resumable: the cursor is persisted; re-running after a
//     restart continues where it stopped, and re-processing a record rewrites
//     the same bytes.
//   - Cheap in steady state: one key read per block once Done is set.
//   - Writes only records with an actual gap (missing index entry, or an OPEN
//     record still missing its sealed deadline), so the cost tracks real
//     backlog rather than total history.
func (k Keeper) EnsureChallengeIndexes(ctx sdk.Context) int {
	st := k.GetChallengeIndexState(ctx)
	if st.Done {
		return 0
	}

	start := types.KeyPrefixEntitlementChallenge
	if st.Cursor != "" {
		// Resume strictly after the last converted record. Ids are hex strings
		// and never contain 0x00, so appending it cannot skip a sibling id.
		start = append(types.EntitlementChallengeKey(st.Cursor), 0x00)
	}
	end := storetypes.PrefixEndBytes(types.KeyPrefixEntitlementChallenge)

	// Collect first, then write: mutating the store while an iterator over it
	// is open is undefined behaviour.
	batch := make([]types.EntitlementChallenge, 0, 8)
	it := k.store(ctx).Iterator(start, end)
	for ; it.Valid(); it.Next() {
		var c types.EntitlementChallenge
		k.cdc.MustUnmarshal(it.Value(), &c)
		batch = append(batch, c)
		if len(batch) >= MaxChallengeIndexBackfillPerBlock {
			break
		}
	}
	it.Close()

	if len(batch) == 0 {
		k.setChallengeIndexState(ctx, ChallengeIndexState{Done: true})
		return 0
	}

	grace := k.GetParams(ctx).EffectiveChallengeResolutionGraceSeconds()
	for _, c := range batch {
		need := false
		if c.CandidateId == "" ||
			!k.store(ctx).Has(types.EntitlementChallengeByCandidateKey(c.CandidateId, c.ChallengeId)) {
			need = true
		}
		// Pre-index records also predate the sealed deadline. Seal it now so the
		// sweeper can reach the challenge; the value is the same dynamic
		// computation readers already fall back to, frozen into the record.
		if c.Status == types.EntitlementChallengeStatusOpen && c.ResolutionDeadline <= 0 {
			if cand, ok := k.GetCandidate(ctx, c.CandidateId); ok {
				c.ResolutionDeadline = types.EffectiveResolutionDeadline(cand, c, grace)
				need = true
			}
		}
		if c.Status == types.EntitlementChallengeStatusOpen && c.CandidateId != "" && c.Challenger != "" {
			// The per-challenger open index is a derived structure too; legacy
			// records are missing it, which would let the same challenger open a
			// second challenge while the first is still OPEN.
			k.SetOpenEntitlementChallenge(ctx, c.CandidateId, c.Challenger, c.ChallengeId)
		}
		if need {
			k.SetEntitlementChallenge(ctx, c)
		}
	}

	if len(batch) < MaxChallengeIndexBackfillPerBlock {
		// The walk reached the end of the table: the indexes are now complete.
		k.setChallengeIndexState(ctx, ChallengeIndexState{Done: true})
	} else {
		k.setChallengeIndexState(ctx, ChallengeIndexState{
			Cursor: batch[len(batch)-1].ChallengeId,
		})
	}
	return len(batch)
}
