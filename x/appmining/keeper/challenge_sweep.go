package keeper

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// MaxExpiredChallengesPerBlock bounds how many challenges a single EndBlocker
// pass settles. A large backlog therefore degrades into "a few more blocks"
// instead of one oversized block; the rest keep their (earlier) buckets and are
// picked up on the next block. Normal operation settles zero per block.
const MaxExpiredChallengesPerBlock = 64

// dueChallenge is one entry of the timeout bucket, carried together with its own
// key so a stale entry can be deleted without rescanning the index.
type dueChallenge struct {
	challengeID string
	bucket      []byte
}

// SweepExpiredChallenges settles every OPEN challenge whose sealed resolution
// deadline has elapsed. It is the automatic counterpart of
// MsgExpireEntitlementChallenge: same settlement helper, so the outcome (EXPIRED
// + full deposit refund, candidate stays PUBLISHED) is identical either way.
//
// Design notes:
//
//   - Bucket order is big-endian deadline, so the walk stops at the first
//     deadline still in the future. Cost is O(entries actually due), normally 0,
//     and never scales with settled history.
//   - It never returns an error and never panics: an EndBlocker failure halts
//     the chain, so per-entry problems are logged and skipped.
//   - Only block time and module-local state are read, so it is deterministic
//     across validators.
//   - It deliberately does not gate on requireEntitlementFlow: releasing a
//     stranded deposit must keep working while the pool is paused. No pool
//     money moves — the deposit only returns to the challenger that paid it.
func (k Keeper) SweepExpiredChallenges(ctx sdk.Context) int {
	now := ctx.BlockTime().Unix()
	store := k.store(ctx)

	// Collect first, then mutate: writing while an iterator is open on the same
	// store is undefined behaviour.
	due := make([]dueChallenge, 0, 4)
	it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixChallengeResolutionDeadline)
	for ; it.Valid(); it.Next() {
		if types.DeadlineFromDeadlineKey(it.Key()) > now {
			break
		}
		due = append(due, dueChallenge{
			challengeID: string(it.Value()),
			bucket:      append([]byte{}, it.Key()...),
		})
		if len(due) >= MaxExpiredChallengesPerBlock {
			break
		}
	}
	it.Close()

	if len(due) == 0 {
		return 0 // steady state: one empty ordered iteration per block
	}

	grace := k.GetParams(ctx).EffectiveChallengeResolutionGraceSeconds()
	settled := 0
	for _, d := range due {
		chal, found := k.GetEntitlementChallenge(ctx, d.challengeID)
		if !found {
			// Dangling index entry: drop the bucket, otherwise it pins the head
			// of the ordered walk forever.
			store.Delete(d.bucket)
			continue
		}
		if chal.Status != types.EntitlementChallengeStatusOpen {
			// Settled between write and sweep; SetEntitlementChallenge already
			// drops the bucket in that case, so this is only a safety net.
			store.Delete(d.bucket)
			continue
		}
		cand, found := k.GetCandidate(ctx, chal.CandidateId)
		if !found {
			// Candidate vanished: nothing can be settled. Drop the bucket so the
			// sweep is not blocked; the OPEN record still blocks approval and
			// needs an operator.
			store.Delete(d.bucket)
			ctx.Logger().Error("appmining: OPEN challenge without candidate; timeout sweep skipped",
				"challenge_id", d.challengeID, "candidate_id", chal.CandidateId)
			continue
		}

		// Defence in depth: the record's own deadline must also be due. A bucket
		// written from a stale revision is repaired instead of settled early.
		deadline := types.EffectiveResolutionDeadline(cand, chal, grace)
		if now < deadline {
			store.Delete(d.bucket)
			chal.ResolutionDeadline = deadline
			k.SetEntitlementChallenge(ctx, chal)
			continue
		}

		at := observedTime(now, cand.MaxObservedTime)
		if _, err := k.settleExpiredChallenge(ctx, chal, cand, at, SettlementTriggerAutomatic); err != nil {
			ctx.Logger().Error("appmining: automatic challenge expiry failed; will retry",
				"challenge_id", d.challengeID, "err", err)
			continue
		}
		settled++
	}
	return settled
}
