package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func TestVector_OPEN_OR_UPHELD_CHALLENGE_BLOCKS_APPROVAL(t *testing.T) {
	e, _, revPriv, aprPriv, c := setupP2(t)

	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	require.NoError(t, err)

	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1201"))
	require.ErrorIs(t, err, types.ErrOpenOrUpheldChallenge)

	resolveRec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
		"nonce": "1202", "expires_at": "1893456000",
		"challenge_id": openRes.ChallengeId, "candidate_id": c.EntitlementID,
		"outcome": "UPHELD", "rationale_digest": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	}
	_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: openRes.ChallengeId, Outcome: "UPHELD",
		RationaleDigest: resolveRec["rationale_digest"],
		Proof:           signProof(t, revPriv, resolveRec),
	})
	require.NoError(t, err)

	cand, _ = e.k.GetCandidate(e.ctx, c.EntitlementID)
	require.Equal(t, types.CandidateStatusRejected, cand.Status)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds+1, 0))
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1203"))
	require.Error(t, err) // not published and/or upheld challenge
	require.True(t,
		errors.Is(err, types.ErrCandidateNotPublished) || errors.Is(err, types.ErrOpenOrUpheldChallenge),
		"got %v", err)

	pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.Equal(t, "0", pool.Reserved)
	require.NoError(t, types.ValidatePoolInvariant(pool))
}

func TestVector_UNAUTHORIZED_AMOUNT_AND_DUPLICATE_APPROVAL(t *testing.T) {
	e, pubPriv, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))

	// Non-approver key_id → role key miss / unauthorized
	rec := map[string]string{
		"protocol_version": "1", "operation": "APPROVE", "role": "approver",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "1101", "expires_at": "1893456000",
		"entitlement_id": c.EntitlementID, "app_id": c.AppID,
		"candidate_hash":            cand.CandidateHash,
		"challenge_resolution_hash": "",
		"amount":                    c.Amount, "denom": c.Denom, "rule_version": c.RuleVersion,
	}
	_, err := e.server.ApproveEntitlement(e.ctx, &types.MsgApproveEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId:           c.EntitlementID,
		ChallengeResolutionHash: rec["challenge_resolution_hash"],
		Proof:                   signProof(t, pubPriv, rec),
	})
	require.ErrorIs(t, err, types.ErrRoleKeyNotFound)

	// Signed amount ≠ candidate amount → wire verify fails (UNAUTHORIZED_AMOUNT)
	badAmt := map[string]string{
		"protocol_version": "1", "operation": "APPROVE", "role": "approver",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "apr-1",
		"nonce": "1102", "expires_at": "1893456000",
		"entitlement_id": c.EntitlementID, "app_id": c.AppID,
		"candidate_hash":            cand.CandidateHash,
		"challenge_resolution_hash": "",
		"amount":                    "1", "denom": c.Denom, "rule_version": c.RuleVersion,
	}
	_, err = e.server.ApproveEntitlement(e.ctx, &types.MsgApproveEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId:           c.EntitlementID,
		ChallengeResolutionHash: badAmt["challenge_resolution_hash"],
		Proof:                   signProof(t, aprPriv, badAmt),
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1103"))
	require.NoError(t, err)

	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1104"))
	require.ErrorIs(t, err, types.ErrEntitlementExists)
}

func TestVector_INVALID_EVIDENCE_CANNOT_FREEZE(t *testing.T) {
	e, riskPriv, _, _, c := setupP3(t)
	now := e.ctx.BlockTime().Unix()
	msg := freezeMsg(t, e, riskPriv, c, now+3600, "1110")
	msg.EvidenceType = types.EvidenceTypeUnspecified
	_, err := e.server.FreezeEntitlement(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidEvidenceType)

	ent, _ := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.Equal(t, types.EntitlementStateClaimable, ent.State)
	pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.Equal(t, "31500", pool.Reserved)
}

func TestE3_ClaimAndRevokeAtMostOneWins(t *testing.T) {
	t.Run("claim_then_revoke", func(t *testing.T) {
		e, riskPriv, auditPriv, _, c := setupP3(t)
		_, err := e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.NoError(t, err)

		now := e.ctx.BlockTime().Unix()
		_, err = e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+3600, "1121"))
		require.ErrorIs(t, err, types.ErrAlreadyClaimed)
		_, err = e.server.RevokeEntitlement(e.ctx, revokeMsg(t, e, riskPriv, auditPriv, c, "1122", "1123"))
		require.ErrorIs(t, err, types.ErrAlreadyClaimed)

		ent, _ := e.k.GetEntitlement(e.ctx, c.EntitlementID)
		require.Equal(t, types.EntitlementStateClaimed, ent.State)
		pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
		require.NoError(t, types.ValidatePoolInvariant(pool))
		require.Equal(t, "31500", pool.Paid)
		require.Equal(t, "0", pool.Reserved)
	})

	t.Run("revoke_then_claim", func(t *testing.T) {
		e, riskPriv, auditPriv, _, c := setupP3(t)
		now := e.ctx.BlockTime().Unix()
		_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+7200, "1131"))
		require.NoError(t, err)

		_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.ErrorIs(t, err, types.ErrNotClaimable)

		_, err = e.server.RevokeEntitlement(e.ctx, revokeMsg(t, e, riskPriv, auditPriv, c, "1132", "1133"))
		require.NoError(t, err)

		_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
			Owner: e.owner.String(), EntitlementId: c.EntitlementID,
		})
		require.ErrorIs(t, err, types.ErrNotClaimable)

		ent, _ := e.k.GetEntitlement(e.ctx, c.EntitlementID)
		require.Equal(t, types.EntitlementStateRevoked, ent.State)
		pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
		require.NoError(t, types.ValidatePoolInvariant(pool))
		require.Equal(t, "0", pool.Paid)
		require.Equal(t, "0", pool.Reserved)
	})
}

func TestE4_ModuleBalanceReconcilesWithLedger(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))

	assertReconcile := func(t *testing.T) {
		t.Helper()
		q, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: testPoolID})
		require.NoError(t, err)
		require.NoError(t, types.ValidatePoolInvariant(q.Pool))
		// Unpaid liability still sits in the module account.
		wantBal, err := types.AddAmountStrings(q.Pool.Available, q.Pool.Reserved)
		require.NoError(t, err)
		require.Equal(t, wantBal, q.ModuleBalance)
	}

	assertReconcile(t)

	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1141"))
	require.NoError(t, err)
	assertReconcile(t)

	_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.owner.String(), EntitlementId: c.EntitlementID,
	})
	require.NoError(t, err)
	assertReconcile(t)

	q, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: testPoolID})
	require.NoError(t, err)
	require.Equal(t, "31500", q.Pool.Paid)
	require.Equal(t, "0", q.Pool.Reserved)
}

func TestE8_UnfundedPoolRejectsEntitlementMsgs(t *testing.T) {
	e := withP1Context(setupTest(t))
	_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: testPoolID, Denom: types.DefaultChallengeBondDenom, Version: 1,
	})
	require.NoError(t, err)

	c := sampleCandidateFields()
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = 0x01
	}
	pubPriv := ed25519.NewKeyFromSeed(seed)
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "pub-1", Pubkey: pubPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	require.NoError(t, err)

	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "1150", "expires_at": "1893456000",
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
	require.ErrorIs(t, err, types.ErrPoolNotEnabled)

	q, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: testPoolID})
	require.NoError(t, err)
	require.False(t, q.EntitlementFlowEnabled)
}

// F-1 (workorder 13 §3): MsgApproveEntitlement must reject candidates whose
// rule_version is not on the governance allowlist; empty allowlist denies all.
func TestGapF1_RuleVersionAllowlist(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))

	// default-deny: empty allowlist rejects the approve
	params := e.k.GetParams(e.ctx)
	params.EntitlementAllowedRuleVersions = nil
	require.NoError(t, e.k.SetParams(e.ctx, params))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1401"))
	require.ErrorIs(t, err, types.ErrRuleVersionNotAllowed)

	// listed version → approve succeeds (positive control)
	params.EntitlementAllowedRuleVersions = []string{"other-1.0", c.RuleVersion}
	require.NoError(t, e.k.SetParams(e.ctx, params))
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1402"))
	require.NoError(t, err)
}

// F-2 registration-side (§1.5): the same pubkey must not be ACTIVE under two
// different roles.
func TestGapF2_SamePubkeyCrossRoleRegistrationRejected(t *testing.T) {
	e := setupTest(t)
	seed := bytes.Repeat([]byte{0x07}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	_, err := e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleRisk,
		KeyId: "risk-shared", Pubkey: pub, Version: 1,
	})
	require.NoError(t, err)

	// same pubkey, different key_id, different role → rejected
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleAudit,
		KeyId: "audit-shared", Pubkey: pub, Version: 1,
	})
	require.ErrorIs(t, err, types.ErrRoleKeyConflict)

	// same pubkey under the SAME role with a new key_id is fine (positive control)
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleRisk,
		KeyId: "risk-shared-2", Pubkey: pub, Version: 1,
	})
	require.NoError(t, err)

	// revoked keys in the other role no longer block reuse: revoke BOTH risk
	// keys sharing the pubkey, then the audit registration succeeds.
	for _, keyID := range []string{"risk-shared", "risk-shared-2"} {
		rk, ok := e.k.GetRoleKey(e.ctx, types.EntitlementRoleRisk, keyID)
		require.True(t, ok)
		rk.Status = types.RoleKeyStatusRevoked
		e.k.SetRoleKey(e.ctx, rk)
	}
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleAudit,
		KeyId: "audit-shared", Pubkey: pub, Version: 1,
	})
	require.NoError(t, err)
}

// F-2 enforcement-side (§1.5): a revoke whose risk and audit proofs resolve to
// the SAME pubkey bytes is rejected even when the key_ids differ (covers
// genesis-imported state that predates the registration check).
func TestGapF2_RevokeWithSamePubkeyBytesRejected(t *testing.T) {
	e, riskPriv, auditPriv, _, c := setupP3(t)
	now := e.ctx.BlockTime().Unix()
	_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+7200, "1410"))
	require.NoError(t, err)

	// forge genesis-style state: audit key registered under a different
	// key_id but with the risk key's pubkey bytes.
	riskRK, ok := e.k.GetRoleKey(e.ctx, types.EntitlementRoleRisk, "risk-1")
	require.True(t, ok)
	auditRK, ok := e.k.GetRoleKey(e.ctx, types.EntitlementRoleAudit, "audit-1")
	require.True(t, ok)
	auditRK.Pubkey = append([]byte(nil), riskRK.Pubkey...)
	e.k.SetRoleKey(e.ctx, auditRK)

	msg := revokeMsg(t, e, riskPriv, auditPriv, c, "1411", "1412")
	_, err = e.server.RevokeEntitlement(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrDualAuthRequired)

	// restore the honest audit key: the same revoke now succeeds — the forged
	// attempt failed before any nonce consumption, so nonces 1411/1412 are
	// still fresh (positive control).
	auditRK.Pubkey = auditPriv.Public().(ed25519.PublicKey)
	e.k.SetRoleKey(e.ctx, auditRK)
	_, err = e.server.RevokeEntitlement(e.ctx, msg)
	require.NoError(t, err)
}

// TestGapF3_ApproveDualPathBinding covers UGE DEC-01 scheme C negatives:
// non-empty unbound → 78; empty + any challenge → 71; empty + no challenge → OK;
// non-empty bound to DISMISSED → OK with resolution_bound=true.
func TestGapF3_ApproveDualPathBinding(t *testing.T) {
	const dismissedDigest = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	const unboundDigest = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

	t.Run("nonempty_no_challenge_rejected_78", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
		_, err := e.server.ApproveEntitlement(e.ctx, approveMsgWithResolution(t, e, aprPriv, c, "1502", unboundDigest))
		require.ErrorIs(t, err, types.ErrResolutionUnbound)
	})

	t.Run("empty_after_dismissed_rejected_71", func(t *testing.T) {
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
			"nonce": "1501", "expires_at": "1893456000",
			"challenge_id": openRes.ChallengeId, "candidate_id": c.EntitlementID,
			"outcome": "DISMISSED", "rationale_digest": dismissedDigest,
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
		_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1505"))
		require.ErrorIs(t, err, types.ErrOpenOrUpheldChallenge)
	})

	t.Run("empty_no_challenge_ok", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP2(t)
		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
		_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1506"))
		require.NoError(t, err)
		ev, ok := findEvent(e.ctx.EventManager().Events(), types.EventTypeEntitlementApproved)
		require.True(t, ok)
		require.Equal(t, "false", eventAttr(ev, types.AttributeKeyResolutionBound))
	})

	t.Run("bound_dismissed_ok", func(t *testing.T) {
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
			"nonce": "1510", "expires_at": "1893456000",
			"challenge_id": openRes.ChallengeId, "candidate_id": c.EntitlementID,
			"outcome": "DISMISSED", "rationale_digest": dismissedDigest,
		}
		_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
			Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
			ChallengeId: openRes.ChallengeId, Outcome: "DISMISSED",
			RationaleDigest: resolveRec["rationale_digest"],
			Proof:           signProof(t, revPriv, resolveRec),
		})
		require.NoError(t, err)

		cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
		e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0)).
			WithEventManager(sdk.NewEventManager())
		_, err = e.server.ApproveEntitlement(e.ctx, approveMsgWithResolution(t, e, aprPriv, c, "1503", dismissedDigest))
		require.NoError(t, err)
		ent, ok := e.k.GetEntitlement(e.ctx, c.EntitlementID)
		require.True(t, ok)
		require.Equal(t, types.EntitlementStateClaimable, ent.State)
		ev, ok := findEvent(e.ctx.EventManager().Events(), types.EventTypeEntitlementApproved)
		require.True(t, ok)
		require.Equal(t, "true", eventAttr(ev, types.AttributeKeyResolutionBound))
		require.Equal(t, dismissedDigest, eventAttr(ev, types.AttributeKeyEvidenceOrResolutionDigest))
	})

	// UGE reply 0.1 negative 2: a dismissed resolution exists, but the approve
	// carries a digest that does not equal its stored ResolutionDigest → 78.
	t.Run("bound_wrong_digest_rejected_78", func(t *testing.T) {
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
			"nonce": "1520", "expires_at": "1893456000",
			"challenge_id": openRes.ChallengeId, "candidate_id": c.EntitlementID,
			"outcome": "DISMISSED", "rationale_digest": dismissedDigest,
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
		_, err = e.server.ApproveEntitlement(e.ctx, approveMsgWithResolution(t, e, aprPriv, c, "1521", unboundDigest))
		require.ErrorIs(t, err, types.ErrResolutionUnbound)
	})

	// Empty digest and no challenge, but the challenge window has not yet
	// elapsed: the unbound path requires window closure (ErrChallengeWindow).
	t.Run("empty_before_window_rejected", func(t *testing.T) {
		e, _, _, aprPriv, c := setupP2(t)
		// setupP2 leaves ctx before publishedAt+ChallengeWindowSeconds.
		_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1522"))
		require.ErrorIs(t, err, types.ErrChallengeWindow)
	})

	// An UPHELD resolution rejects the candidate outright: no approve path
	// (bound or unbound) may run afterwards — fail-closed end to end.
	t.Run("upheld_resolution_rejected_fail_closed", func(t *testing.T) {
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
			"nonce": "1523", "expires_at": "1893456000",
			"challenge_id": openRes.ChallengeId, "candidate_id": c.EntitlementID,
			"outcome": "UPHELD", "rationale_digest": unboundDigest,
		}
		_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
			Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
			ChallengeId: openRes.ChallengeId, Outcome: "UPHELD",
			RationaleDigest: resolveRec["rationale_digest"],
			Proof:           signProof(t, revPriv, resolveRec),
		})
		require.NoError(t, err)

		_, err = e.server.ApproveEntitlement(e.ctx, approveMsgWithResolution(t, e, aprPriv, c, "1524", unboundDigest))
		require.ErrorIs(t, err, types.ErrCandidateNotPublished)
	})
}
