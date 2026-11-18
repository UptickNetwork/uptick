package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestExpiredResolutionDigest_DeterministicAndDomainSeparated(t *testing.T) {
	const cid = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	d := types.ExpiredResolutionDigest(cid)
	require.Len(t, d, 64)
	require.NoError(t, types.ValidateWireHash(d))
	require.Equal(t, d, types.ExpiredResolutionDigest(cid), "digest must be a pure function of challenge_id")
	require.NotEqual(t, d, types.ExpiredResolutionDigest("00"+cid[2:]))
	// A reviewer rationale must never collide with the timeout digest space.
	require.NotEqual(t, cid, d)
}

func TestChallengeResolutionDeadline_UsesSealedWindow(t *testing.T) {
	cand := types.Candidate{PublishedAt: 1_000, ChallengeDeadline: 1_000 + 600} // 600s sealed window
	chal := types.EntitlementChallenge{OpenedAt: 1_200}

	require.EqualValues(t, 1_200+600+60, types.ChallengeResolutionDeadline(cand, chal, 60))

	// A late-open challenge still gets a full sealed window, never less.
	late := types.EntitlementChallenge{OpenedAt: 1_580}
	require.EqualValues(t, 1_580+600+60, types.ChallengeResolutionDeadline(cand, late, 60))

	// Negative grace is clamped instead of shortening the deadline.
	require.EqualValues(t, 1_800, types.ChallengeResolutionDeadline(cand, chal, -100))

	// Pre-deadline state (no sealed window) degrades to opened_at + grace.
	legacy := types.Candidate{PublishedAt: 5_000, ChallengeDeadline: 0}
	require.EqualValues(t, 1_200+0+60, types.ChallengeResolutionDeadline(legacy, chal, 60))
}

func TestParamsValidate_ChallengeResolutionGraceBounds(t *testing.T) {
	p := types.DefaultParams()
	require.NoError(t, p.Validate())
	require.EqualValues(t, types.DefaultChallengeResolutionGraceSeconds, p.EffectiveChallengeResolutionGraceSeconds())

	// 0 means "use the module default" (migration-free for pre-field state).
	p.ChallengeResolutionGraceSeconds = 0
	require.NoError(t, p.Validate())
	require.EqualValues(t, types.DefaultChallengeResolutionGraceSeconds, p.EffectiveChallengeResolutionGraceSeconds())

	p.ChallengeResolutionGraceSeconds = types.MinChallengeResolutionGraceSeconds
	require.NoError(t, p.Validate())

	p.ChallengeResolutionGraceSeconds = types.MinChallengeResolutionGraceSeconds - 1
	require.Error(t, p.Validate())

	p.ChallengeResolutionGraceSeconds = types.MaxChallengeResolutionGraceSeconds + 1
	require.Error(t, p.Validate())
}
