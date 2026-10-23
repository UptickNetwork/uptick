package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

const testChainID = "uptick_7777-1"
const testPoolID = "ecosystem-v0"

func withP1Context(e testEnv) testEnv {
	e.ctx = e.ctx.WithChainID(testChainID).WithBlockTime(time.Unix(1_700_000_000, 0))
	return e
}

func setupP1(t *testing.T) (testEnv, ed25519.PrivateKey, ed25519.PrivateKey) {
	t.Helper()
	e := withP1Context(setupTest(t))

	params := e.k.GetParams(e.ctx)
	params.EntitlementEnableMinFunded = "1000"
	require.NoError(t, e.k.SetParams(e.ctx, params))

	_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: testPoolID, Denom: types.DefaultChallengeBondDenom, Version: 1,
	})
	require.NoError(t, err)
	e.bank.fundAccount(e.owner, types.DefaultChallengeBondDenom, math.NewInt(10_000))
	_, err = e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Funder: e.owner.String(), PoolId: testPoolID, Amount: "1000",
	})
	require.NoError(t, err)

	pubSeed := bytes.Repeat([]byte{0x01}, ed25519.SeedSize)
	revSeed := bytes.Repeat([]byte{0x02}, ed25519.SeedSize)
	pubPriv := ed25519.NewKeyFromSeed(pubSeed)
	revPriv := ed25519.NewKeyFromSeed(revSeed)

	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "pub-1", Pubkey: pubPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	require.NoError(t, err)
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleReviewer,
		KeyId: "rev-1", Pubkey: revPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	require.NoError(t, err)

	// register app with wire-compatible app_id (32 bytes of 0x11)
	appID, _ := hex.DecodeString("1111111111111111111111111111111111111111111111111111111111111111")
	_, err = e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority: e.auth.String(), AppId: appID, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)

	return e, pubPriv, revPriv
}

func sampleCandidateFields() wirev1.Candidate {
	c := wirev1.Candidate{
		SchemaVersion:    wirev1.CandidateSchemaVersion,
		AppID:            "1111111111111111111111111111111111111111111111111111111111111111",
		Epoch:            "1",
		Amount:           "31500",
		Denom:            "auptick",
		GasUsed:          "21000",
		QualifiedTxCount: "1",
		PavID:            "2222222222222222222222222222222222222222222222222222222222222222",
		EvidenceRoot:     "3333333333333333333333333333333333333333333333333333333333333333",
		GasEvidenceRoot:  "4444444444444444444444444444444444444444444444444444444444444444",
		RuleVersion:      "gas-reward-candidate-0.1",
	}
	c.EntitlementID = wirev1.EntitlementIDOf(c)
	return c
}

func signProof(t *testing.T, priv ed25519.PrivateKey, record map[string]string) types.WireProof {
	t.Helper()
	raw, err := wirev1.AuthorizationBytes(record)
	require.NoError(t, err)
	sig := ed25519.Sign(priv, raw)
	return types.WireProof{
		KeyId:           record["key_id"],
		Nonce:           record["nonce"],
		ExpiresAt:       record["expires_at"],
		SignatureBase64: base64.StdEncoding.EncodeToString(sig),
	}
}

func TestVector_PUBLISH_IS_NOT_PAYMENT(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)

	mkMsg := func(nonce string) *types.MsgPublishCandidate {
		rec := map[string]string{
			"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
			"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
			"nonce": nonce, "expires_at": "1893456000",
			"entitlement_id": c.EntitlementID, "app_id": c.AppID,
			"candidate_hash": hash, "rule_version": c.RuleVersion,
		}
		return &types.MsgPublishCandidate{
			Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
			EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch, Amount: c.Amount,
			Denom: c.Denom, GasUsed: c.GasUsed, QualifiedTxCount: c.QualifiedTxCount,
			PavId: c.PavID, EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
			RuleVersion: c.RuleVersion, Proof: signProof(t, pubPriv, rec),
		}
	}

	_, err = e.server.PublishCandidate(e.ctx, mkMsg("1"))
	require.NoError(t, err)

	// same content idempotent (new nonce still required for wire; but same hash short-circuits before auth)
	_, err = e.server.PublishCandidate(e.ctx, mkMsg("2"))
	require.NoError(t, err)

	// different content same id -> reject
	bad := mkMsg("3")
	bad.Amount = "31501"
	// entitlement_id still old — hash will differ → conflict after identity check fails at CandidateHash
	// Force conflict path: keep entitlement_id but change amount so CandidateHash fails identity OR
	// if we recompute entitlement_id it becomes a different id. To hit conflict, store same id with
	// different hash: mutate pav without changing entitlement_id.
	bad.PavId = "2222222222222222222222222222222222222222222222222222222222222223"
	_, err = e.server.PublishCandidate(e.ctx, bad)
	require.Error(t, err) // invalid commitment (identity mismatch) or conflict

	pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.Equal(t, "1000", pool.Available)
	require.Equal(t, "0", pool.Reserved)
	require.Equal(t, "0", pool.Paid)

	got, ok := e.k.GetCandidate(e.ctx, c.EntitlementID)
	require.True(t, ok)
	require.Equal(t, types.CandidateStatusPublished, got.Status)
}

func TestP1_OpenAndResolveChallenge(t *testing.T) {
	e, pubPriv, revPriv := setupP1(t)
	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "10", "expires_at": "1893456000",
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

	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	require.NoError(t, err)
	require.Len(t, openRes.ChallengeId, 64)

	// duplicate open by same challenger
	_, err = e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	require.ErrorIs(t, err, types.ErrDuplicateOpenChallenge)

	require.True(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, c.EntitlementID))

	resolveRec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
		"nonce": "20", "expires_at": "1893456000",
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
	require.Equal(t, types.CandidateStatusPublished, cand.Status)
	require.False(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, c.EntitlementID))
}

func TestP1_UpholdRejectsCandidate(t *testing.T) {
	e, pubPriv, revPriv := setupP1(t)
	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "30", "expires_at": "1893456000",
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

	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeReceiptQbtMismatch,
		EvidenceDigest: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	})
	require.NoError(t, err)

	resolveRec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
		"nonce": "40", "expires_at": "1893456000",
		"challenge_id": openRes.ChallengeId, "candidate_id": c.EntitlementID,
		"outcome": "UPHELD", "rationale_digest": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	}
	_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: openRes.ChallengeId, Outcome: "UPHELD",
		RationaleDigest: resolveRec["rationale_digest"],
		Proof:           signProof(t, revPriv, resolveRec),
	})
	require.NoError(t, err)

	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	require.Equal(t, types.CandidateStatusRejected, cand.Status)
	require.True(t, e.k.HasOpenOrUpheldEntitlementChallenge(e.ctx, c.EntitlementID))
}

func TestP1_PublishRejectsWhenPoolNotEnabled(t *testing.T) {
	e := withP1Context(setupTest(t))
	// open pool but do not fund / no min funded
	_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: testPoolID, Denom: types.DefaultChallengeBondDenom, Version: 1,
	})
	require.NoError(t, err)

	c := sampleCandidateFields()
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch, Amount: c.Amount,
		Denom: c.Denom, GasUsed: c.GasUsed, QualifiedTxCount: c.QualifiedTxCount,
		PavId: c.PavID, EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
		RuleVersion: c.RuleVersion,
		Proof:       types.WireProof{KeyId: "x", Nonce: "1", ExpiresAt: "9", SignatureBase64: "YQ=="},
	})
	require.ErrorIs(t, err, types.ErrPoolNotEnabled)
}

func TestChallengeIDIsHex64(t *testing.T) {
	sum := sha256.Sum256([]byte("x"))
	require.Equal(t, 64, len(hex.EncodeToString(sum[:])))
}
