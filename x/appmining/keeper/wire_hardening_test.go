package keeper_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// Regression pins for the uge/v1.1 audit fixes:
//   - P2-3: WireProof.expires_at is capped at now + 7d (604800s).
//   - P2-1: the PUBLISH idempotent short-circuit requires a valid wire
//     signature; retry-with-same-record semantics are preserved.
//   - P1-2: consumed auth nonces past their retention window are pruned on
//     the next write for the same (role, key_id).
//
// Tests avoid referencing types.MaxWireProofValiditySeconds /
// types.NonceRetentionSeconds literals-free so they compile against both the
// pre-fix and post-fix trees (red/green pinning).

func publishMsgFor(t *testing.T, e testEnv, priv ed25519.PrivateKey, c wirev1.Candidate, nonce, expiresAt string, corruptSig bool) *types.MsgPublishCandidate {
	t.Helper()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": nonce, "expires_at": expiresAt,
		"entitlement_id": c.EntitlementID, "app_id": c.AppID,
		"candidate_hash": hash, "rule_version": c.RuleVersion,
	}
	proof := signProof(t, priv, rec)
	if corruptSig {
		raw, _ := base64.StdEncoding.DecodeString(proof.SignatureBase64)
		raw[0] ^= 0xFF
		proof.SignatureBase64 = base64.StdEncoding.EncodeToString(raw)
	}
	return &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch, Amount: c.Amount,
		Denom: c.Denom, GasUsed: c.GasUsed, QualifiedTxCount: c.QualifiedTxCount,
		PavId: c.PavID, EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
		RuleVersion: c.RuleVersion, Proof: proof,
	}
}

func countAuthNonces(e testEnv) int {
	n := 0
	e.k.IterateAuthNonces(e.ctx, func(string) bool {
		n++
		return true
	})
	return n
}

// P2-3: expires_at 9 days out (beyond the 7d cap) must be rejected even
// though the signature is valid; 1 hour out is accepted (positive control).
func TestWireHardening_FarFutureExpiresAtRejected(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	c := sampleCandidateFields()

	far := strconv.FormatInt(e.ctx.BlockTime().Unix()+9*86400, 10)
	_, err := e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c, "8601", far, false))
	require.ErrorIs(t, err, types.ErrAuthMalformed,
		"expires_at beyond the validity cap must fail as malformed")

	_, err = e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c, "8602", wireExp(e.ctx), false))
	require.NoError(t, err, "expires_at within the cap must pass")
}

// P2-1: once a candidate exists, a same-content publish with a CORRUPTED
// signature must fail instead of riding the idempotent short-circuit; a
// correctly signed same-content publish stays idempotent (both a fresh nonce
// and a full record replay of the original msg).
func TestWireHardening_IdempotentPublishRequiresValidSignature(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	c := sampleCandidateFields()

	orig := publishMsgFor(t, e, pubPriv, c, "8701", wireExp(e.ctx), false)
	_, err := e.server.PublishCandidate(e.ctx, orig)
	require.NoError(t, err)

	// same content, corrupted signature -> rejected before idempotency
	tampered := publishMsgFor(t, e, pubPriv, c, "8702", wireExp(e.ctx), true)
	_, err = e.server.PublishCandidate(e.ctx, tampered)
	require.Error(t, err, "idempotent short-circuit must not bypass signature verification")

	// same content, valid fresh nonce -> idempotent success
	repeat := publishMsgFor(t, e, pubPriv, c, "8703", wireExp(e.ctx), false)
	_, err = e.server.PublishCandidate(e.ctx, repeat)
	require.NoError(t, err)

	// full record replay of the original msg (nonce already consumed) ->
	// still idempotent success, retry semantics preserved
	_, err = e.server.PublishCandidate(e.ctx, orig)
	require.NoError(t, err)
}

// P1-2: a consumed nonce is pruned once its expiry passes the retention
// window (validity + grace), triggered by the next write for the same key.
func TestWireHardening_AuthNoncesPrunedAfterRetention(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	c := sampleCandidateFields()

	_, err := e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c, "9701", wireExp(e.ctx), false))
	require.NoError(t, err)
	require.Equal(t, 1, countAuthNonces(e))

	// advance past expiry (3600s) + retention (8d) + slack
	e.ctx = e.ctx.WithBlockTime(e.ctx.BlockTime().Add(time.Duration(3600+691200+60) * time.Second))

	// different content -> new entitlement, forces a real nonce consume
	c2 := sampleCandidateFields()
	c2.Amount = "40000"
	c2.PavID = "2222222222222222222222222222222222222222222222222222222222222299"
	c2.EntitlementID = wirev1.EntitlementIDOf(c2)
	_, err = e.server.PublishCandidate(e.ctx, publishMsgFor(t, e, pubPriv, c2, "9702", wireExp(e.ctx), false))
	require.NoError(t, err)

	require.Equal(t, 1, countAuthNonces(e),
		"expired nonce must be pruned by the next consume; only the fresh nonce remains")
}
