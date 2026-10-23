package keeper_test

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestE1_TamperedSignatureRejected(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))

	msg := approveMsg(t, e, aprPriv, c, "901")
	sig, err := base64.StdEncoding.DecodeString(msg.Proof.SignatureBase64)
	require.NoError(t, err)
	sig[0] ^= 0xff
	msg.Proof.SignatureBase64 = base64.StdEncoding.EncodeToString(sig)

	_, err = e.server.ApproveEntitlement(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrUnauthorized)
}

func TestE2_CrossChainIDReplayRejected(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))

	msg := approveMsg(t, e, aprPriv, c, "902")
	msg.ChainId = "other-chain-9"
	_, err := e.server.ApproveEntitlement(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrChainIDMismatch)
}

func TestE2_CrossPoolIDReplayRejected(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))

	msg := approveMsg(t, e, aprPriv, c, "903")
	msg.PoolId = "other-pool"
	_, err := e.server.ApproveEntitlement(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrPoolMismatch)
}

func TestVector_DISMISSED_CHALLENGE_ALLOWS_APPROVAL(t *testing.T) {
	e, _, revPriv, aprPriv, c := setupP2(t)

	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	require.NoError(t, err)

	resolveRec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
		"nonce": "920", "expires_at": "1893456000",
		"challenge_id": openRes.ChallengeId, "candidate_id": c.EntitlementID,
		"outcome": "DISMISSED", "rationale_digest": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: openRes.ChallengeId, Outcome: "DISMISSED",
		RationaleDigest: resolveRec["rationale_digest"],
		Proof:           signProof(t, revPriv, resolveRec),
	})
	require.NoError(t, err)

	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "921"))
	require.NoError(t, err)
	ent, ok := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.True(t, ok)
	require.Equal(t, types.EntitlementStateClaimable, ent.State)
}

func TestE5_ExportImportGenesisRoundTrip(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "951"))
	require.NoError(t, err)

	exported := e.k.ExportGenesis(e.ctx)
	require.NoError(t, types.ValidateGenesis(*exported))
	require.NotEmpty(t, exported.Candidates)
	require.NotEmpty(t, exported.Entitlements)
	require.NotEmpty(t, exported.EntitlementPools)

	e2 := setupTest(t)
	e2.ctx = e2.ctx.WithChainID(testChainID)
	require.NotPanics(t, func() { e2.k.InitGenesis(e2.ctx, *exported) })
	got, ok := e2.k.GetEntitlement(e2.ctx, c.EntitlementID)
	require.True(t, ok)
	require.Equal(t, types.EntitlementStateClaimable, got.State)
}
