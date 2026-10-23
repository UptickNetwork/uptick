package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func setupP3(t *testing.T) (testEnv, ed25519.PrivateKey, ed25519.PrivateKey, ed25519.PrivateKey, wirev1.Candidate) {
	t.Helper()
	e, _, _, aprPriv, c := setupP2(t)

	riskPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x04}, ed25519.SeedSize))
	auditPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x05}, ed25519.SeedSize))
	_, err := e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleRisk,
		KeyId: "risk-1", Pubkey: riskPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	require.NoError(t, err)
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleAudit,
		KeyId: "audit-1", Pubkey: auditPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	require.NoError(t, err)

	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "601"))
	require.NoError(t, err)

	return e, riskPriv, auditPriv, aprPriv, c
}

func freezeMsg(t *testing.T, e testEnv, riskPriv ed25519.PrivateKey, c wirev1.Candidate, until int64, nonce string) *types.MsgFreezeEntitlement {
	t.Helper()
	untilStr := strconv.FormatInt(until, 10)
	rec := map[string]string{
		"protocol_version": "1", "operation": "FREEZE", "role": "risk",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "risk-1",
		"nonce": nonce, "expires_at": "1893456000",
		"entitlement_id":  c.EntitlementID,
		"evidence_type":   "CONFIRMED_REFUND_OR_REVERSAL",
		"evidence_digest": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"freeze_until":    untilStr,
	}
	return &types.MsgFreezeEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId:  c.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: rec["evidence_digest"],
		FreezeUntil:    untilStr,
		Proof:          signProof(t, riskPriv, rec),
	}
}

func revokeMsg(t *testing.T, e testEnv, riskPriv, auditPriv ed25519.PrivateKey, c wirev1.Candidate, riskNonce, auditNonce string) *types.MsgRevokeEntitlement {
	t.Helper()
	base := map[string]string{
		"protocol_version": "1", "operation": "REVOKE",
		"chain_id": testChainID, "pool_id": testPoolID,
		"entitlement_id":   c.EntitlementID,
		"evidence_type":    "CONFIRMED_REFUND_OR_REVERSAL",
		"evidence_digest":  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"rationale_digest": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	riskRec := map[string]string{}
	for k, v := range base {
		riskRec[k] = v
	}
	riskRec["role"] = "risk"
	riskRec["key_id"] = "risk-1"
	riskRec["nonce"] = riskNonce
	riskRec["expires_at"] = "1893456000"
	auditRec := map[string]string{}
	for k, v := range base {
		auditRec[k] = v
	}
	auditRec["role"] = "audit"
	auditRec["key_id"] = "audit-1"
	auditRec["nonce"] = auditNonce
	auditRec["expires_at"] = "1893456000"
	return &types.MsgRevokeEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId:   c.EntitlementID,
		EvidenceType:    types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest:  base["evidence_digest"],
		RationaleDigest: base["rationale_digest"],
		RiskProof:       signProof(t, riskPriv, riskRec),
		AuditProof:      signProof(t, auditPriv, auditRec),
	}
}

func TestVector_FREEZE_PRESERVES_LIABILITY_AND_BLOCKS_CLAIM(t *testing.T) {
	e, riskPriv, _, _, c := setupP3(t)
	now := e.ctx.BlockTime().Unix()
	_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+3600, "701"))
	require.NoError(t, err)

	pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.Equal(t, "31500", pool.Reserved) // unchanged
	require.Equal(t, "0", pool.Paid)

	_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.owner.String(), EntitlementId: c.EntitlementID,
	})
	require.ErrorIs(t, err, types.ErrNotClaimable)
}

func TestVector_FREEZE_MAX_72_HOURS_AND_EXPIRY(t *testing.T) {
	e, riskPriv, _, _, c := setupP3(t)
	now := e.ctx.BlockTime().Unix()

	_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+types.MaxFreezeSeconds+1, "702"))
	require.ErrorIs(t, err, types.ErrFreezeDuration)

	until := now + types.MaxFreezeSeconds
	_, err = e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, until, "703"))
	require.NoError(t, err)

	// after freeze_until, claim allowed without admin
	e.ctx = e.ctx.WithBlockTime(time.Unix(until, 0))
	_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.owner.String(), EntitlementId: c.EntitlementID,
	})
	require.NoError(t, err)
	ent, _ := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.Equal(t, types.EntitlementStateClaimed, ent.State)
}

func TestVector_REVOKE_REQUIRES_INDEPENDENT_DUAL_AUTHORITY(t *testing.T) {
	e, riskPriv, auditPriv, _, c := setupP3(t)
	now := e.ctx.BlockTime().Unix()
	_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+7200, "801"))
	require.NoError(t, err)

	// only risk proof duplicated as audit with same key_id → ValidateBasic / dual auth
	msg := revokeMsg(t, e, riskPriv, auditPriv, c, "802", "803")
	msg.AuditProof = msg.RiskProof // same key_id
	_, err = e.server.RevokeEntitlement(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrDualAuthRequired)

	_, err = e.server.RevokeEntitlement(e.ctx, revokeMsg(t, e, riskPriv, auditPriv, c, "804", "805"))
	require.NoError(t, err)

	ent, _ := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.Equal(t, types.EntitlementStateRevoked, ent.State)
	pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.Equal(t, "0", pool.Reserved)
	require.Equal(t, "0", pool.Paid)

	// second revoke rejected
	_, err = e.server.RevokeEntitlement(e.ctx, revokeMsg(t, e, riskPriv, auditPriv, c, "806", "807"))
	require.Error(t, err)
}

func TestVector_CLAIM_PRECEDES_REVOKE_AT_MOST_ONE_WINS(t *testing.T) {
	e, riskPriv, auditPriv, _, c := setupP3(t)
	_, err := e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.owner.String(), EntitlementId: c.EntitlementID,
	})
	require.NoError(t, err)

	now := e.ctx.BlockTime().Unix()
	_, err = e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+3600, "901"))
	require.ErrorIs(t, err, types.ErrAlreadyClaimed)

	_, err = e.server.RevokeEntitlement(e.ctx, revokeMsg(t, e, riskPriv, auditPriv, c, "902", "903"))
	require.ErrorIs(t, err, types.ErrAlreadyClaimed)
}

func TestMsgFreeze_InvalidEvidenceType(t *testing.T) {
	msg := &types.MsgFreezeEntitlement{
		Submitter: "uptick1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqcrc5kq",
		ChainId:   testChainID, PoolId: testPoolID,
		EntitlementId:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EvidenceType:   types.EvidenceTypeUnspecified,
		EvidenceDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		FreezeUntil:    "1",
		Proof:          types.WireProof{KeyId: "k", Nonce: "1", ExpiresAt: "9", SignatureBase64: "YQ=="},
	}
	// fix address for ValidateBasic - use a valid bech32 from setup
	e := setupTest(t)
	msg.Submitter = e.sub.String()
	require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidEvidenceType)
}
