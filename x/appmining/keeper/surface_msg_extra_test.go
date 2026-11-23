package keeper_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// Second pass over the message surface: the branches that need a deliberately
// malformed piece of chain state or a guard that only fires after another
// message has already run. Kept separate from surface_msg_test.go so the first
// file stays readable as "one gate per test".

func resolveEntitlementMsg(t *testing.T, e testEnv, revPriv ed25519.PrivateKey, candidateID, challengeID, outcome, digest, nonce string) *types.MsgResolveEntitlementChallenge {
	t.Helper()
	rec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
		"nonce": nonce, "expires_at": wireExp(e.ctx),
		"challenge_id": challengeID, "candidate_id": candidateID,
		"outcome": outcome, "rationale_digest": digest,
	}
	return &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: challengeID, Outcome: outcome, RationaleDigest: digest,
		Proof: signProof(t, revPriv, rec),
	}
}

// TestSurface_ClaimStateGates covers the entitlement states that are not
// CLAIMABLE plus the pool/app switches that can block a legitimate claim.
func TestSurface_ClaimStateGates(t *testing.T) {
	t.Run("revoked entitlement", func(t *testing.T) {
		e, riskPriv, auditPriv, _, c := setupP3(t)
		_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, e.ctx.BlockTime().Unix()+3600, "8101"))
		require.NoError(t, err)
		_, err = e.server.RevokeEntitlement(e.ctx, revokeMsg(t, e, riskPriv, auditPriv, c, "8102", "8103"))
		require.NoError(t, err)

		_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.ErrorIs(t, err, types.ErrNotClaimable)
	})

	t.Run("suspended application", func(t *testing.T) {
		e, _, _, _, c := setupP3(t)
		appID, err := hexDecode(c.AppID)
		require.NoError(t, err)
		_, err = e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
			Authority: e.auth.String(), AppId: appID, Status: types.ApplicationStatusSuspended,
		})
		require.NoError(t, err)
		_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.ErrorIs(t, err, types.ErrApplicationNotActive)
	})

	t.Run("pool paused", func(t *testing.T) {
		e, _, _, _, c := setupP3(t)
		_, err := e.server.SetEntitlementPoolPaused(e.ctx, &types.MsgSetEntitlementPoolPaused{
			Authority: e.auth.String(), PoolId: testPoolID, Paused: true,
		})
		require.NoError(t, err)
		_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.ErrorIs(t, err, types.ErrPoolPaused)
	})
}

// TestSurface_ApproveGuards covers the approve guards that depend on governance
// params or on the pool ledger rather than on the candidate itself.
func TestSurface_ApproveGuards(t *testing.T) {
	t.Run("rule version not allowed", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.DefaultChallengeWindowSeconds, 0))
		params := e.k.GetParams(e.ctx)
		params.EntitlementAllowedRuleVersions = nil // default-deny
		require.NoError(t, e.k.SetParams(e.ctx, params))
		_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "8201"))
		require.ErrorIs(t, err, types.ErrRuleVersionNotAllowed)
	})

	t.Run("insufficient available", func(t *testing.T) {
		e, pubPriv, _, aprPriv, _ := setupP2(t)
		big := sampleCandidateFields()
		big.Amount = "60000" // pool available is 51000
		big.EntitlementID = wirev1.EntitlementIDOf(big)
		_, err := e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, big, "8202", wireExp(e.ctx), false))
		require.NoError(t, err)
		cand, _ := e.k.GetCandidate(e.ctx, big.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.DefaultChallengeWindowSeconds, 0))
		_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, big, "8203"))
		require.ErrorIs(t, err, types.ErrInsufficientAvailable)
	})

	t.Run("application suspended", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.DefaultChallengeWindowSeconds, 0))
		appID, err := hexDecode(c.AppID)
		require.NoError(t, err)
		_, err = e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
			Authority: e.auth.String(), AppId: appID, Status: types.ApplicationStatusSuspended,
		})
		require.NoError(t, err)
		_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "8204"))
		require.ErrorIs(t, err, types.ErrApplicationNotActive)
	})
}

func hexDecode(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// TestSurface_OpenChallengeDepositMode covers the deposit-funded challenge mode
// end to end: escrow on open, refund on UPHELD, credit to the pool on DISMISSED.
func TestSurface_OpenChallengeDepositMode(t *testing.T) {
	newDepositEnv := func(t *testing.T) (testEnv, ed25519.PrivateKey, wirev1.Candidate) {
		t.Helper()
		e, _, revPriv, _, c := setupP2(t)
		params := e.k.GetParams(e.ctx)
		params.ChallengeOpenAccess = types.ChallengeOpenAccessDeposit
		params.ChallengeDepositAmount = "10"
		params.ChallengeDepositDenom = types.DefaultChallengeBondDenom
		require.NoError(t, e.k.SetParams(e.ctx, params))
		e.bank.fundAccount(e.owner, types.DefaultChallengeBondDenom, math.NewInt(1000))
		return e, revPriv, c
	}

	t.Run("escrow on open", func(t *testing.T) {
		e, _, c := newDepositEnv(t)
		before := e.bank.GetBalance(e.ctx, sdk.AccAddress(bytes20(0x02)), types.DefaultChallengeBondDenom)

		resp, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.NoError(t, err)

		chal, found := e.k.GetEntitlementChallenge(e.ctx, resp.ChallengeId)
		require.True(t, found)
		require.Equal(t, "10", chal.DepositAmount)
		require.Equal(t, types.DefaultChallengeBondDenom, chal.DepositDenom)
		after := e.bank.GetBalance(e.ctx, sdk.AccAddress(bytes20(0x02)), types.DefaultChallengeBondDenom)
		require.Equal(t, "10", before.Amount.Sub(after.Amount).String())

		// UPHELD refunds the deposit to the challenger.
		_, err = e.server.ResolveEntitlementChallenge(e.ctx, resolveEntitlementMsg(
			t, e, mustReviewerKey(t, e), c.EntitlementID, resp.ChallengeId, "UPHELD", digestOf(0x0d), "8301"))
		require.NoError(t, err)
		require.Equal(t, "0", e.bank.GetBalance(e.ctx, sdk.AccAddress(bytes20(0x02)), types.DefaultChallengeBondDenom).Amount.Sub(before.Amount).String())
	})

	t.Run("dismissed credits pool", func(t *testing.T) {
		e, revPriv, c := newDepositEnv(t)
		resp, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.NoError(t, err)
		before, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)

		_, err = e.server.ResolveEntitlementChallenge(e.ctx, resolveEntitlementMsg(
			t, e, revPriv, c.EntitlementID, resp.ChallengeId, "DISMISSED", digestOf(0x0e), "8302"))
		require.NoError(t, err)

		after, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
		require.Equal(t, "10", mustSub(t, after.Funded, before.Funded))
		require.Equal(t, "10", mustSub(t, after.Available, before.Available))
	})

	t.Run("invalid outcome", func(t *testing.T) {
		e, revPriv, c := newDepositEnv(t)
		resp, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.NoError(t, err)
		_ = revPriv
		// The wire-record validator refuses an unknown outcome, so a reviewer
		// cannot even sign one; the handler's own switch is the second gate.
		// Assert the first one here rather than hand-crafting an unverifiable
		// signature to reach the second.
		_, err = wirev1.AuthorizationBytes(map[string]string{
			"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
			"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
			"nonce": "8303", "expires_at": wireExp(e.ctx),
			"challenge_id": resp.ChallengeId, "candidate_id": c.EntitlementID,
			"outcome": "MAYBE", "rationale_digest": digestOf(0x0f),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid outcome")
	})

	t.Run("resolve while pool paused", func(t *testing.T) {
		e, revPriv, c := newDepositEnv(t)
		resp, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.NoError(t, err)
		_, err = e.server.SetEntitlementPoolPaused(e.ctx, &types.MsgSetEntitlementPoolPaused{
			Authority: e.auth.String(), PoolId: testPoolID, Paused: true,
		})
		require.NoError(t, err)
		_, err = e.server.ResolveEntitlementChallenge(e.ctx, resolveEntitlementMsg(
			t, e, revPriv, c.EntitlementID, resp.ChallengeId, "DISMISSED", digestOf(0x10), "8304"))
		require.ErrorIs(t, err, types.ErrPoolPaused)
	})
}

// TestSurface_OpenChallengeGuards covers the remaining pre-write guards.
func TestSurface_OpenChallengeGuards(t *testing.T) {
	t.Run("invalid evidence type", func(t *testing.T) {
		e, _, _, _, c := setupP2(t)
		_, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceType(0), EvidenceDigest: digestOf(0x0b),
		})
		require.ErrorIs(t, err, types.ErrInvalidEvidenceType)
	})

	t.Run("pool paused", func(t *testing.T) {
		e, _, _, _, c := setupP2(t)
		_, err := e.server.SetEntitlementPoolPaused(e.ctx, &types.MsgSetEntitlementPoolPaused{
			Authority: e.auth.String(), PoolId: testPoolID, Paused: true,
		})
		require.NoError(t, err)
		_, err = e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
			Challenger: e.owner.String(), EntitlementId: c.EntitlementID,
			EvidenceType: types.EvidenceTypeConfirmedRefundOrReversal, EvidenceDigest: digestOf(0x0b),
		})
		require.ErrorIs(t, err, types.ErrPoolPaused)
	})
}

// TestSurface_SubmitAttestationGuards covers the epoch/app/profile gates of the
// legacy submission message.
func TestSurface_SubmitAttestationGuards(t *testing.T) {
	t.Run("epoch budget missing", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 42, "10"),
		})
		require.ErrorIs(t, err, types.ErrInvalidEpoch)
	})

	t.Run("epoch budget closed", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		setBudget(t, e, 2, "1000")
		b, _ := e.k.GetEpochBudget(e.ctx, 2)
		b.Closed = true
		e.k.StoreEpochBudget(e.ctx, b)
		_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 2, "10"),
		})
		require.ErrorIs(t, err, types.ErrInvalidEpoch)
	})

	t.Run("application suspended", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		setBudget(t, e, 1, "1000")
		_, err := e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
			Authority: e.auth.String(), AppId: appID, Status: types.ApplicationStatusSuspended,
		})
		require.NoError(t, err)
		_, err = e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
		})
		require.ErrorIs(t, err, types.ErrApplicationNotActive)
	})

	t.Run("profile not allowed", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		setBudget(t, e, 1, "1000")
		att := sampleAttestation(appID, 1, "10")
		att.ProfileVersion = "pav-9.9"
		_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: att,
		})
		require.ErrorIs(t, err, types.ErrUnsupportedProfile)
	})
}

// TestSurface_FinalizeGuards covers the settlement guards that need an existing
// attestation.
func TestSurface_FinalizeGuards(t *testing.T) {
	t.Run("application suspended", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		setBudget(t, e, 1, "1000")
		_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "100"),
		})
		require.NoError(t, err)
		_, err = e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
			Authority: e.auth.String(), AppId: appID, Status: types.ApplicationStatusSuspended,
		})
		require.NoError(t, err)
		e.ctx = e.ctx.WithBlockHeight(types.DefaultChallengeWindowBlocks + 10)
		_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
			Caller: e.auth.String(), AppId: appID, Epoch: 1,
		})
		require.ErrorIs(t, err, types.ErrApplicationNotActive)
	})

	t.Run("open challenge blocks", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		openLegacyChallenge(t, e, appID, 1)
		e.ctx = e.ctx.WithBlockHeight(types.DefaultChallengeWindowBlocks + 10)
		_, err := e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
			Caller: e.auth.String(), AppId: appID, Epoch: 1,
		})
		require.ErrorIs(t, err, types.ErrOpenChallenge)
	})

	t.Run("settlement already exists", func(t *testing.T) {
		e := setupTest(t)
		appID := registerApp(t, e)
		setBudget(t, e, 1, "1000")
		_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
			Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "100"),
		})
		require.NoError(t, err)
		e.ctx = e.ctx.WithBlockHeight(types.DefaultChallengeWindowBlocks + 10)
		_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
			Caller: e.auth.String(), AppId: appID, Epoch: 1,
		})
		require.NoError(t, err)
		_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
			Caller: e.auth.String(), AppId: appID, Epoch: 1,
		})
		require.ErrorIs(t, err, types.ErrSettlementExists)
	})
}

// TestSurface_FreezeDurations covers the freeze_until parsing and bounds.
func TestSurface_FreezeDurations(t *testing.T) {
	e, riskPriv, _, _, c := setupP3(t)

	bad := freezeMsg(t, e, riskPriv, c, 0, "8401")
	bad.FreezeUntil = "not-a-number"
	_, err := e.server.FreezeEntitlement(e.ctx, bad)
	require.ErrorIs(t, err, types.ErrFreezeDuration)

	past := freezeMsg(t, e, riskPriv, c, e.ctx.BlockTime().Unix()-1, "8402")
	_, err = e.server.FreezeEntitlement(e.ctx, past)
	require.ErrorIs(t, err, types.ErrFreezeDuration)

	tooFar := freezeMsg(t, e, riskPriv, c, e.ctx.BlockTime().Unix()+int64(types.DefaultMaxFreezeSeconds)+60, "8403")
	_, err = e.server.FreezeEntitlement(e.ctx, tooFar)
	require.ErrorIs(t, err, types.ErrFreezeDuration)

	badEvidence := freezeMsg(t, e, riskPriv, c, e.ctx.BlockTime().Unix()+3600, "8404")
	badEvidence.EvidenceType = types.EvidenceType(0)
	_, err = e.server.FreezeEntitlement(e.ctx, badEvidence)
	require.ErrorIs(t, err, types.ErrInvalidEvidenceType)

	ok := freezeMsg(t, e, riskPriv, c, e.ctx.BlockTime().Unix()+3600, "8405")
	_, err = e.server.FreezeEntitlement(e.ctx, ok)
	require.NoError(t, err)
}

func mustReviewerKey(t *testing.T, e testEnv) ed25519.PrivateKey {
	t.Helper()
	return e.revPriv
}

func mustSub(t *testing.T, a, b string) string {
	t.Helper()
	out, err := types.SubAmountStrings(a, b)
	require.NoError(t, err)
	return out
}
