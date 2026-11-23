package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/keeper"
	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// TestChallengeIndex_CandidateIsolation pins the candidate index boundary.
//
// Candidate ids are hashes but the index must not rely on their length: a bare
// prefix scan would let candidate "aa" match candidate "aab" and make one
// candidate's OPEN challenge block (or unblock) another candidate's approval.
func TestChallengeIndex_CandidateIsolation(t *testing.T) {
	e := setupTest(t)
	mk := func(candidateID, challengeID string) {
		e.k.SetEntitlementChallenge(e.ctx, types.EntitlementChallenge{
			ChallengeId:        challengeID,
			CandidateId:        candidateID,
			Status:             types.EntitlementChallengeStatusOpen,
			OpenedAt:           1,
			ResolutionDeadline: 100,
		})
	}
	mk("aa", "challenge-1")
	mk("aab", "challenge-2")

	require.True(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, "aa"))
	require.True(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, "aab"))

	require.False(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, "a"),
		"a shorter candidate id must not inherit a neighbour's challenge")
	require.False(t, e.k.HasAnyEntitlementChallenge(e.ctx, "a"),
		"existence check must respect the id boundary too")

	seen := 0
	e.k.IterateEntitlementChallengesByCandidate(e.ctx, "aa", func(types.EntitlementChallenge) bool {
		seen++
		return true
	})
	require.Equal(t, 1, seen)
}

// TestChallengeIndex_SettledStatusGatesApproval checks that the index-backed
// predicates keep their per-status meaning: only OPEN/UPHELD block approval, and
// only DISMISSED/EXPIRED count as a settled resolution.
func TestChallengeIndex_SettledStatusGatesApproval(t *testing.T) {
	e := setupTest(t)
	const cand = "candidate-x"
	put := func(id string, st types.EntitlementChallengeStatus, digest string) {
		e.k.SetEntitlementChallenge(e.ctx, types.EntitlementChallenge{
			ChallengeId: id, CandidateId: cand, Status: st,
			OpenedAt: 1, ResolutionDeadline: 100, ResolutionDigest: digest,
		})
	}

	put("c1", types.EntitlementChallengeStatusDismissed, "digest-1")
	require.False(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, cand),
		"DISMISSED must not block approval")
	require.True(t, e.k.HasAnyEntitlementChallenge(e.ctx, cand))
	require.True(t, e.k.HasSettledEntitlementChallengeResolution(e.ctx, cand, "digest-1"))
	require.False(t, e.k.HasSettledEntitlementChallengeResolution(e.ctx, cand, "digest-2"))

	put("c2", types.EntitlementChallengeStatusUpheld, "digest-2")
	require.True(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, cand),
		"UPHELD must block approval")
	require.False(t, e.k.HasSettledEntitlementChallengeResolution(e.ctx, cand, "digest-2"),
		"UPHELD is not a settled-and-dismissed resolution")

	put("c3", types.EntitlementChallengeStatusExpired, "digest-3")
	require.True(t, e.k.HasSettledEntitlementChallengeResolution(e.ctx, cand, "digest-3"))
}

// TestSweep_SettlesAtDeadlineWithoutAnyTx is the core of the liveness fix: the
// timeout outcome must happen because the block time passed, not because
// somebody submitted MsgExpireEntitlementChallenge.
func TestSweep_SettlesAtDeadlineWithoutAnyTx(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9301",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	chal, found := e.k.GetEntitlementChallenge(e.ctx, cid)
	require.True(t, found)
	require.NotZero(t, chal.ResolutionDeadline, "deadline must be sealed at open time")

	deadline := resolutionDeadline(t, e, cid)
	require.EqualValues(t, chal.ResolutionDeadline, deadline,
		"the sealed snapshot must equal the dynamic computation at open time")

	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline-1, 0))
	require.Equal(t, 0, e.k.SweepExpiredChallenges(e.ctx), "one second early: nothing to settle")
	chal, _ = e.k.GetEntitlementChallenge(e.ctx, cid)
	require.Equal(t, types.EntitlementChallengeStatusOpen, chal.Status)

	before := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline, 0))
	require.Equal(t, 1, e.k.SweepExpiredChallenges(e.ctx))

	chal, _ = e.k.GetEntitlementChallenge(e.ctx, cid)
	require.Equal(t, types.EntitlementChallengeStatusExpired, chal.Status)
	require.Equal(t, types.ExpiredResolutionDigest(cid), chal.ResolutionDigest)
	require.Empty(t, chal.Reviewer, "the window decided, nobody signed")
	require.EqualValues(t, deadline, chal.ResolvedAt)

	cand, _ := e.k.GetCandidate(e.ctx, chal.CandidateId)
	require.Equal(t, types.CandidateStatusPublished, cand.Status)

	after := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)
	require.Equal(t, before.Amount.Int64()+int64(500), after.Amount.Int64(),
		"deposit must be fully refunded")

	_, open := e.k.GetOpenEntitlementChallenge(e.ctx, chal.CandidateId, chal.Challenger)
	require.False(t, open)
	require.False(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, chal.CandidateId))

	// Idempotent: the bucket is gone, so the same block range settles nothing.
	require.Equal(t, 0, e.k.SweepExpiredChallenges(e.ctx))
}

// TestSweep_ResolvedChallengeLeavesNoBucket makes sure a reviewer decision
// takes the challenge out of the timeout queue, so a later sweep cannot
// overwrite UPHELD/DISMISSED with EXPIRED.
func TestSweep_ResolvedChallengeLeavesNoBucket(t *testing.T) {
	e, pubPriv, revPriv, _ := expireEnv(t)
	c := publishForDeposit(t, e, pubPriv, "9302")
	res, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger:     e.payer.String(),
		EntitlementId:  c.EntitlementID,
		EvidenceType:   types.EvidenceTypeTxMissingOrUnfinalized,
		EvidenceDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	require.NoError(t, err)

	rationale := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	rec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
		"nonce": "9303", "expires_at": wireExp(e.ctx),
		"challenge_id": res.ChallengeId, "candidate_id": c.EntitlementID,
		"outcome": "UPHELD", "rationale_digest": rationale,
	}
	_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: res.ChallengeId, Outcome: "UPHELD", RationaleDigest: rationale,
		Proof: signProof(t, revPriv, rec),
	})
	require.NoError(t, err)

	deadline := resolutionDeadline(t, e, res.ChallengeId)
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline+600, 0))
	require.Equal(t, 0, e.k.SweepExpiredChallenges(e.ctx),
		"a resolved challenge must not be swept into EXPIRED")

	chal, _ := e.k.GetEntitlementChallenge(e.ctx, res.ChallengeId)
	require.Equal(t, types.EntitlementChallengeStatusUpheld, chal.Status)
}

// TestSweep_GraceChangeDoesNotMoveInFlightChallenge pins the sealed-window
// property: governance may change grace, but a challenge already opened keeps
// the deadline it was created with (in either direction).
func TestSweep_GraceChangeDoesNotMoveInFlightChallenge(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9304",
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")

	chal, _ := e.k.GetEntitlementChallenge(e.ctx, cid)
	cand, _ := e.k.GetCandidate(e.ctx, chal.CandidateId)
	sealed := chal.ResolutionDeadline
	require.NotZero(t, sealed)

	// Reviewer grace grows tenfold after the challenge opened.
	params := e.k.GetParams(e.ctx)
	params.ChallengeResolutionGraceSeconds = expireGraceSeconds * 60
	require.NoError(t, e.k.SetParams(e.ctx, params))

	require.Equal(t, sealed, types.EffectiveResolutionDeadline(cand, chal, params.EffectiveChallengeResolutionGraceSeconds()),
		"the snapshot must win over the new parameter")

	// At the sealed deadline the sweep still settles, i.e. the change did not
	// silently extend the challenger's money lock-up.
	e.ctx = e.ctx.WithBlockTime(time.Unix(sealed, 0))
	require.Equal(t, 1, e.k.SweepExpiredChallenges(e.ctx))
	require.True(t, e.k.HasSettledEntitlementChallengeResolution(e.ctx, chal.CandidateId, types.ExpiredResolutionDigest(cid)))
}

// TestSweep_BoundedPerBlock caps how much one EndBlocker pass does, so a backlog
// degrades into a few blocks instead of one oversized block.
func TestSweep_BoundedPerBlock(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	c := publishForDeposit(t, e, pubPriv, "9305")

	past := e.ctx.BlockTime().Unix() - 1
	total := keeper.MaxExpiredChallengesPerBlock + 6
	for i := 0; i < total; i++ {
		e.k.SetEntitlementChallenge(e.ctx, types.EntitlementChallenge{
			ChallengeId:        fmt.Sprintf("%064d", i),
			CandidateId:        c.EntitlementID,
			Status:             types.EntitlementChallengeStatusOpen,
			OpenedAt:           1,
			ResolutionDeadline: past,
		})
	}

	require.Equal(t, keeper.MaxExpiredChallengesPerBlock, e.k.SweepExpiredChallenges(e.ctx))
	require.Equal(t, total-keeper.MaxExpiredChallengesPerBlock, e.k.SweepExpiredChallenges(e.ctx))
	require.Equal(t, 0, e.k.SweepExpiredChallenges(e.ctx))
}

// TestBackfillChallengeIndexes_Idempotent covers the migration entry point: it
// walks every record, rebuilds both derived structures and can be re-run.
func TestBackfillChallengeIndexes_Idempotent(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9306",
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")

	n1, err := e.k.BackfillChallengeIndexes(e.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n1)

	n2, err := e.k.BackfillChallengeIndexes(e.ctx)
	require.NoError(t, err)
	require.Equal(t, n1, n2, "backfill must not duplicate work")

	require.True(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, mustCandidateID(t, e, cid)))
	chal, _ := e.k.GetEntitlementChallenge(e.ctx, cid)
	require.NotZero(t, chal.ResolutionDeadline, "an unsealed OPEN record gets a snapshot")

	msg, broken := keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.False(t, broken, msg)
}

// TestChallengeIndexInvariant_CleanAfterLifecycle guards the invariant itself:
// open -> expire must leave no dangling candidate entry and no leftover bucket.
func TestChallengeIndexInvariant_CleanAfterLifecycle(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9307",
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	// The invariant is a no-op until the rebuild has run; declare the indexes
	// complete so this test checks the strict mode, not the skip.
	e.k.MarkChallengeIndexesReady(e.ctx)

	msg, broken := keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.False(t, broken, "OPEN record must have both index and bucket: %s", msg)

	deadline := resolutionDeadline(t, e, cid)
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline, 0))
	require.Equal(t, 1, e.k.SweepExpiredChallenges(e.ctx))

	msg, broken = keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.False(t, broken, "EXPIRED record must keep the index but drop the bucket: %s", msg)
}
