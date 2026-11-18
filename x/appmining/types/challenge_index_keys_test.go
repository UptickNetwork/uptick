package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEntitlementChallengeByCandidateKey_IsolatesCandidateIDs(t *testing.T) {
	// A prefix scan for "aa" must not reach "aab": the 0x00 separator keeps the
	// id boundary explicit instead of relying on fixed-length ids.
	prefix := EntitlementChallengeByCandidatePrefix("aa")
	own := EntitlementChallengeByCandidateKey("aa", "challenge-1")
	neighbour := EntitlementChallengeByCandidateKey("aab", "challenge-2")

	require.True(t, bytes.HasPrefix(own, prefix))
	require.False(t, bytes.HasPrefix(neighbour, prefix),
		"candidate id boundary must not be a bare prefix match")

	require.Equal(t, []byte("challenge-1"), own[len(prefix):])
}

func TestChallengeResolutionDeadlineKey_RoundTripAndOrdering(t *testing.T) {
	early := ChallengeResolutionDeadlineKey(1_700_000_000, "a")
	late := ChallengeResolutionDeadlineKey(1_700_000_100, "b")

	require.Equal(t, 1_700_000_000, int(DeadlineFromDeadlineKey(early)))
	require.Equal(t, 1_700_000_100, int(DeadlineFromDeadlineKey(late)))
	require.Equal(t, "a", ChallengeIDFromDeadlineKey(early))
	require.Equal(t,
		bytes.Compare(early, late) < 0, true,
		"big-endian deadlines must sort lexicographically so the sweeper can stop early",
	)
}

func TestDeadlineKey_DefensiveOnShortKeys(t *testing.T) {
	require.EqualValues(t, 0, DeadlineFromDeadlineKey(nil))
	require.EqualValues(t, 0, DeadlineFromDeadlineKey([]byte{1, 2, 3}))
	require.Equal(t, "", ChallengeIDFromDeadlineKey([]byte{1, 2, 3}))
}

func TestEffectiveResolutionDeadline_SnapshotWins(t *testing.T) {
	cand := Candidate{PublishedAt: 100, ChallengeDeadline: 700} // sealed window 600
	chal := EntitlementChallenge{OpenedAt: 1000, ResolutionDeadline: 1660}

	// Governance doubled the grace (600 -> 1200): the snapshot still holds.
	require.EqualValues(t, 1660, EffectiveResolutionDeadline(cand, chal, 1200))
	// Governance dropped it to 0: the snapshot still holds.
	require.EqualValues(t, 1660, EffectiveResolutionDeadline(cand, chal, 0))

	// Records written before the field existed (0) keep the dynamic semantics.
	legacy := EntitlementChallenge{OpenedAt: 1000}
	require.EqualValues(t, 1000+600+60, EffectiveResolutionDeadline(cand, legacy, 60))
}
