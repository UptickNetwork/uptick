package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/keeper"
	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// legacyOpenChallenge writes a challenge the way a v1 binary would have: the
// record exists and an index-less chain has no candidate entry, no timeout
// bucket and no sealed deadline. The escrow side effect is whatever the caller
// already created; this helper only shapes the record.
func legacyOpenChallenge(t *testing.T, e testEnv, candidateID, challengeID string, openedAt int64) types.EntitlementChallenge {
	t.Helper()
	c := types.EntitlementChallenge{
		ChallengeId:    challengeID,
		CandidateId:    candidateID,
		Challenger:     e.payer.String(),
		Status:         types.EntitlementChallengeStatusOpen,
		OpenedAt:       openedAt,
		EvidenceDigest: "1111111111111111111111111111111111111111111111111111111111111111",
	}
	e.k.SetEntitlementChallenge(e.ctx, c)      // writes the derived entries ...
	e.k.StripChallengeIndexesForTest(e.ctx, c) // ... then remove them again
	return c
}

// TestBackfill_LegacyChallengeStillBlocksApprove is the regression that made the
// rebuild necessary: a chain upgraded by a bare binary swap holds challenges
// with no index entry. If approval trusted the index, an OPEN challenge would be
// invisible and the empty-hash path would release the candidate's funds.
func TestBackfill_LegacyChallengeStillBlocksApprove(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, ok := e.k.GetCandidate(e.ctx, c.EntitlementID)
	require.True(t, ok)

	legacyOpenChallenge(t, e, c.EntitlementID, fmt.Sprintf("%064d", 1), cand.PublishedAt)
	require.False(t, e.k.ChallengeIndexesReady(e.ctx), "a fresh chain has no rebuild record")

	// Predicates must fall back to the record table and find it.
	require.True(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, c.EntitlementID),
		"pre-backfill reads must not miss a legacy OPEN challenge")
	require.True(t, e.k.HasAnyEntitlementChallenge(e.ctx, c.EntitlementID))

	// End to end: the approve gate still rejects after the window elapsed.
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.ChallengeDeadline, 0))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "710001"))
	require.ErrorIs(t, err, types.ErrOpenOrUpheldChallenge,
		"an index-less OPEN challenge must still block approval")
}

// TestBackfill_ChunkedConvergence pins the block budget: a backlog converts over
// several EndBlocker passes, the cursor resumes, and "ready" flips only when the
// whole table has been walked.
func TestBackfill_ChunkedConvergence(t *testing.T) {
	e, _, _, _, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)

	const extra = 5
	total := keeper.MaxChallengeIndexBackfillPerBlock + extra
	for i := 0; i < total; i++ {
		legacyOpenChallenge(t, e, c.EntitlementID, fmt.Sprintf("%064d", i), cand.PublishedAt)
	}
	require.False(t, e.k.ChallengeIndexesReady(e.ctx))

	require.Equal(t, keeper.MaxChallengeIndexBackfillPerBlock, e.k.EnsureChallengeIndexes(e.ctx),
		"one pass must not exceed the per-block budget")
	st := e.k.GetChallengeIndexState(e.ctx)
	require.False(t, st.Done)
	require.NotEmpty(t, st.Cursor, "an interrupted rebuild must persist its cursor")

	// Still unsafe to trust the index while records remain unconverted.
	require.True(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, c.EntitlementID))

	require.Equal(t, extra, e.k.EnsureChallengeIndexes(e.ctx), "resumes from the cursor")
	require.True(t, e.k.ChallengeIndexesReady(e.ctx))
	require.Equal(t, 0, e.k.EnsureChallengeIndexes(e.ctx), "steady state is a single state read")

	// Every record now has a candidate index entry.
	seen := 0
	e.k.IterateEntitlementChallengesByCandidate(e.ctx, c.EntitlementID, func(types.EntitlementChallenge) bool {
		seen++
		return true
	})
	require.Equal(t, total, seen, "the candidate index must reach every converted record")
	msg, broken := keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.False(t, broken, msg)
}

// TestBackfill_SealsAndSettlesInSamePass is the origin_1170-3 scenario: an OPEN
// challenge whose window closed before the upgrade, with no bucket to find it.
// One EndBlocker pass must seal it and settle it without any transaction.
func TestBackfill_SealsAndSettlesInSamePass(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9401",
		"2222222222222222222222222222222222222222222222222222222222222222")

	deadline := resolutionDeadline(t, e, cid)
	chal, _ := e.k.GetEntitlementChallenge(e.ctx, cid)

	// Rewind to the pre-upgrade shape: no sealed deadline, no derived entries.
	chal.ResolutionDeadline = 0
	e.k.SetEntitlementChallenge(e.ctx, chal)
	e.k.StripChallengeIndexesForTest(e.ctx, chal)
	require.False(t, e.k.ChallengeIndexesReady(e.ctx))

	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline+1, 0))
	before := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)

	require.Equal(t, 1, e.k.EnsureChallengeIndexes(e.ctx))
	require.True(t, e.k.ChallengeIndexesReady(e.ctx))
	require.Equal(t, 1, e.k.SweepExpiredChallenges(e.ctx),
		"a sealed, already-expired challenge must be swept in the same block")

	chal, _ = e.k.GetEntitlementChallenge(e.ctx, cid)
	require.Equal(t, types.EntitlementChallengeStatusExpired, chal.Status)
	require.Equal(t, types.ExpiredResolutionDigest(cid), chal.ResolutionDigest)
	require.NotZero(t, chal.ResolutionDeadline, "the rebuild seals the deadline")

	after := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)
	require.Equal(t, before.Amount.Int64()+int64(500), after.Amount.Int64())

	msg, broken := keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.False(t, broken, msg)
}

// TestBackfill_InvariantSkipsUntilReady keeps x/crisis quiet on a chain that is
// legitimately mid-rebuild, and strict afterwards.
func TestBackfill_InvariantSkipsUntilReady(t *testing.T) {
	e, _, _, _, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)

	legacyOpenChallenge(t, e, c.EntitlementID, fmt.Sprintf("%064d", 7), cand.PublishedAt)
	msg, broken := keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.False(t, broken, "pre-rebuild state is expected, not a fault: %s", msg)

	require.Equal(t, 1, e.k.EnsureChallengeIndexes(e.ctx))
	require.True(t, e.k.ChallengeIndexesReady(e.ctx))
	msg, broken = keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.False(t, broken, msg)

	// A real drift after the rebuild is still caught.
	c2 := legacyOpenChallenge(t, e, c.EntitlementID, fmt.Sprintf("%064d", 8), cand.PublishedAt)
	msg, broken = keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.True(t, broken, "post-rebuild drift must be reported")
	_ = c2
}

// TestBackfill_MigrationMarksReady makes sure the plan-based path and the
// EndBlocker path converge: after the 1->2 backfill the rebuild is a no-op.
func TestBackfill_MigrationMarksReady(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9402",
		"3333333333333333333333333333333333333333333333333333333333333333")
	chal, _ := e.k.GetEntitlementChallenge(e.ctx, cid)
	e.k.StripChallengeIndexesForTest(e.ctx, chal)
	require.False(t, e.k.ChallengeIndexesReady(e.ctx))

	err := keeper.NewMigrator(e.k).Migrate1to2(e.ctx)
	require.NoError(t, err)
	require.True(t, e.k.ChallengeIndexesReady(e.ctx), "the migration must declare the indexes complete")
	require.Equal(t, 0, e.k.EnsureChallengeIndexes(e.ctx), "no work left for the EndBlocker rebuild")

	msg, broken := keeper.ChallengeIndexInvariant(e.k)(e.ctx)
	require.False(t, broken, msg)
}
