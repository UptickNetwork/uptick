package keeper_test

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// The V1 challenge window used to be a compile-time constant, so the only way to
// change it was a new binary. It is a Params field now (workorder 13 §3), and the
// whole point of making it governable is that governance can shorten it. That
// makes "does a param change reach back into candidates that are already in
// flight?" a correctness question, not a cosmetic one: silently rewriting a
// published deadline would cut a challenger's reaction time retroactively.
//
// These tests pin the two halves of that contract: the deadline is sealed at
// publish, and approval still waits for the sealed deadline rather than the
// current param.

// publishCandidateEpoch publishes a copy of the sample candidate with a different
// epoch (and therefore a different entitlement_id), signed by the publisher key.
func publishCandidateEpoch(t *testing.T, e testEnv, pubPriv ed25519.PrivateKey, epoch, nonce string) wirev1.Candidate {
	t.Helper()
	c := sampleCandidateFields()
	c.Epoch = epoch
	c.EntitlementID = wirev1.EntitlementIDOf(c)
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": nonce, "expires_at": wireExp(e.ctx),
		"entitlement_id": c.EntitlementID, "app_id": c.AppID,
		"candidate_hash": hash, "rule_version": c.RuleVersion,
	}
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch, Amount: c.Amount,
		Denom: c.Denom, GasUsed: c.GasUsed, QualifiedTxCount: c.QualifiedTxCount,
		PavId: c.PavID, EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
		RuleVersion: c.RuleVersion, Proof: signProof(t, pubPriv, rec),
	})
	require.NoError(t, err)
	return c
}

func TestParams_ChallengeWindowChangeAppliesOnlyToNewCandidates(t *testing.T) {
	e, pubPriv, _, _, first := setupP2(t)

	// setupP2 publishes one candidate under the default 7-day window.
	before, ok := e.k.GetCandidate(e.ctx, first.EntitlementID)
	require.True(t, ok)
	require.Equal(t, types.CandidateStatusPublished, before.Status)
	require.Equal(t, int64(types.DefaultChallengeWindowSeconds), before.ChallengeDeadline-before.PublishedAt,
		"a fresh candidate must use the default window")

	// Governance shortens the window to 2h.
	params := e.k.GetParams(e.ctx)
	params.ChallengeWindowSeconds = 7200
	require.NoError(t, e.k.SetParams(e.ctx, params))

	// The in-flight candidate keeps the deadline it was published with.
	after, ok := e.k.GetCandidate(e.ctx, first.EntitlementID)
	require.True(t, ok)
	require.Equal(t, before.ChallengeDeadline, after.ChallengeDeadline,
		"SetParams must not rewrite a deadline that was already sealed at publish")

	// The next candidate picks up the new window.
	e.ctx = e.ctx.WithBlockTime(time.Unix(before.PublishedAt+60, 0))
	second := publishCandidateEpoch(t, e, pubPriv, "2", "9101")
	got, ok := e.k.GetCandidate(e.ctx, second.EntitlementID)
	require.True(t, ok)
	require.Equal(t, int64(7200), got.ChallengeDeadline-got.PublishedAt,
		"a candidate published after the change must use the new window")
	require.Less(t, got.ChallengeDeadline, before.ChallengeDeadline,
		"the shortened window has to actually move the new deadline in")
}

func TestParams_ShortenedWindowDoesNotUnblockInFlightCandidate(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, ok := e.k.GetCandidate(e.ctx, c.EntitlementID)
	require.True(t, ok)

	// Shorten the window after publication. The sealed deadline is 7 days out;
	// if approval read the live param it would open up within 2h instead.
	params := e.k.GetParams(e.ctx)
	params.ChallengeWindowSeconds = 7200
	require.NoError(t, e.k.SetParams(e.ctx, params))

	approve := func(nonce string) error {
		rec := map[string]string{
			"protocol_version": "1", "operation": "APPROVE", "role": "approver",
			"chain_id": testChainID, "pool_id": testPoolID, "key_id": "apr-1",
			"nonce": nonce, "expires_at": wireExp(e.ctx),
			"entitlement_id": c.EntitlementID, "app_id": c.AppID,
			"candidate_hash":            cand.CandidateHash,
			"challenge_resolution_hash": "",
			"amount":                    c.Amount, "denom": c.Denom, "rule_version": c.RuleVersion,
		}
		_, err := e.server.ApproveEntitlement(e.ctx, &types.MsgApproveEntitlement{
			Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
			EntitlementId:           c.EntitlementID,
			ChallengeResolutionHash: rec["challenge_resolution_hash"],
			Proof:                   signProof(t, aprPriv, rec),
		})
		return err
	}

	// Past the new 2h window but before the sealed 7-day deadline: still closed.
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+7200+60, 0))
	require.ErrorIs(t, approve("9201"), types.ErrChallengeWindow,
		"shortening the param must not let an in-flight candidate be approved early")

	// At the sealed deadline approval is allowed, which also proves the assertion
	// above fails for the right reason rather than because approval never works.
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.ChallengeDeadline, 0))
	require.NoError(t, approve("9202"))
}
