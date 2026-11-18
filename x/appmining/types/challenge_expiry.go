package types

import (
	"crypto/sha256"
	"encoding/hex"
)

// expiredResolutionDomain separates timeout digests from reviewer-supplied
// rationale digests. A reviewer therefore cannot pre-commit to the EXPIRED
// outcome by choosing a rationale that happens to equal the timeout digest.
const expiredResolutionDomain = "appmining/entitlement-challenge-expired/v1|"

// ExpiredResolutionDigest returns the digest stored on a challenge settled by
// timeout (ENTITLEMENT_CHALLENGE_STATUS_EXPIRED). It depends only on
// challenge_id, so any client can recompute and verify it, and
// MsgApproveEntitlement can bind to it exactly like a DISMISSED resolution
// digest (UGE DEC-01 scheme C: non-empty hash must bind a settled resolution).
func ExpiredResolutionDigest(challengeID string) string {
	sum := sha256.Sum256([]byte(expiredResolutionDomain + challengeID))
	return hex.EncodeToString(sum[:])
}

// ChallengeResolutionDeadline returns the unix second after which an OPEN
// challenge becomes expirable by anyone.
//
// The resolution window is sealed at publish time
// (Candidate.challenge_deadline - Candidate.published_at) and starts at
// Challenge.opened_at, so a challenge opened late in the window still gets a
// full window, and later governance changes cannot shorten an in-flight one.
// graceSeconds normally comes from Params.EffectiveChallengeResolutionGraceSeconds.
func ChallengeResolutionDeadline(cand Candidate, chal EntitlementChallenge, graceSeconds int64) int64 {
	sealedWindow := cand.ChallengeDeadline - cand.PublishedAt
	if sealedWindow < 0 {
		// Defensive: state written before the deadline field existed.
		sealedWindow = 0
	}
	if graceSeconds < 0 {
		graceSeconds = 0
	}
	return chal.OpenedAt + sealedWindow + graceSeconds
}

// EffectiveResolutionDeadline is the single read-side accessor for "when may
// this challenge be settled by timeout".
//
// Open-time snapshot wins. It was sealed when the challenge was created, so a
// governance change to challenge_resolution_grace_seconds cannot move an
// in-flight challenge in either direction. Records written before the
// resolution_deadline field existed (0) fall back to the dynamic computation so
// pre-upgrade state keeps its original semantics.
func EffectiveResolutionDeadline(cand Candidate, chal EntitlementChallenge, graceSeconds int64) int64 {
	if chal.ResolutionDeadline > 0 {
		return chal.ResolutionDeadline
	}
	return ChallengeResolutionDeadline(cand, chal, graceSeconds)
}
