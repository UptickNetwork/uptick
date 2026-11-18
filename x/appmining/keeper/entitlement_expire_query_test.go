package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestQueryEntitlementChallenges_FiltersAndStatus(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9301", "4444444444444444444444444444444444444444444444444444444444444444")
	candID := mustCandidateID(t, e, cid)

	// OPEN filter finds it, EXPIRED does not (yet).
	open, err := e.k.EntitlementChallenges(e.ctx, &types.QueryEntitlementChallengesRequest{Status: "OPEN"})
	require.NoError(t, err)
	require.Len(t, open.Challenges, 1)
	require.Equal(t, cid, open.Challenges[0].ChallengeId)

	expired, err := e.k.EntitlementChallenges(e.ctx, &types.QueryEntitlementChallengesRequest{Status: "expired"})
	require.NoError(t, err)
	require.Empty(t, expired.Challenges)

	byCand, err := e.k.EntitlementChallenges(e.ctx, &types.QueryEntitlementChallengesRequest{CandidateId: candID})
	require.NoError(t, err)
	require.Len(t, byCand.Challenges, 1)

	other, err := e.k.EntitlementChallenges(e.ctx, &types.QueryEntitlementChallengesRequest{
		CandidateId: "5555555555555555555555555555555555555555555555555555555555555555",
	})
	require.NoError(t, err)
	require.Empty(t, other.Challenges)

	// Timeout settlement flips the status filter buckets.
	deadline := resolutionDeadline(t, e, cid)
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline, 0))
	_, err = expireChallenge(e, cid)
	require.NoError(t, err)

	open, err = e.k.EntitlementChallenges(e.ctx, &types.QueryEntitlementChallengesRequest{Status: "OPEN"})
	require.NoError(t, err)
	require.Empty(t, open.Challenges, "an expired challenge must leave the OPEN worklist")

	expired, err = e.k.EntitlementChallenges(e.ctx, &types.QueryEntitlementChallengesRequest{Status: "4"})
	require.NoError(t, err)
	require.Len(t, expired.Challenges, 1)
	require.Equal(t, types.EntitlementChallengeStatusExpired, expired.Challenges[0].Status)

	// Unknown status values are rejected instead of silently matching nothing.
	_, err = e.k.EntitlementChallenges(e.ctx, &types.QueryEntitlementChallengesRequest{Status: "NOPE"})
	require.Error(t, err)
	_, err = e.k.EntitlementChallenges(e.ctx, &types.QueryEntitlementChallengesRequest{Status: "99"})
	require.Error(t, err)
}
