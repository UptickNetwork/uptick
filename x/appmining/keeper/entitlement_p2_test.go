package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func setupP2(t *testing.T) (testEnv, ed25519.PrivateKey, ed25519.PrivateKey, ed25519.PrivateKey, wirev1.Candidate) {
	t.Helper()
	e, pubPriv, revPriv := setupP1(t)

	// top up pool so Approve of 31500 succeeds
	e.bank.fundAccount(e.owner, types.DefaultChallengeBondDenom, math.NewInt(100_000))
	_, err := e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Funder: e.owner.String(), PoolId: testPoolID, Amount: "50000",
	})
	require.NoError(t, err)
	params := e.k.GetParams(e.ctx)
	params.EntitlementEnableMinFunded = "1000"
	// F-1: rule_version allowlist must cover the sample candidate.
	params.EntitlementAllowedRuleVersions = []string{sampleCandidateFields().RuleVersion}
	require.NoError(t, e.k.SetParams(e.ctx, params))

	aprSeed := bytes.Repeat([]byte{0x03}, ed25519.SeedSize)
	aprPriv := ed25519.NewKeyFromSeed(aprSeed)
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleApprover,
		KeyId: "apr-1", Pubkey: aprPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	require.NoError(t, err)

	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "100", "expires_at": "1893456000",
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

	return e, pubPriv, revPriv, aprPriv, c
}

// approveMsg builds the V0 no-challenge approve path (UGE DEC-01 scheme C):
// empty challenge_resolution_hash after the window, with zero challenges on-chain.
func approveMsg(t *testing.T, e testEnv, aprPriv ed25519.PrivateKey, c wirev1.Candidate, nonce string) *types.MsgApproveEntitlement {
	t.Helper()
	return approveMsgWithResolution(t, e, aprPriv, c, nonce, "")
}

// approveMsgWithResolution builds a bound approve (non-empty digest must match a
// DISMISSED ResolutionDigest already on-chain).
func approveMsgWithResolution(t *testing.T, e testEnv, aprPriv ed25519.PrivateKey, c wirev1.Candidate, nonce, resolutionDigest string) *types.MsgApproveEntitlement {
	t.Helper()
	cand, ok := e.k.GetCandidate(e.ctx, c.EntitlementID)
	require.True(t, ok)
	rec := map[string]string{
		"protocol_version": "1", "operation": "APPROVE", "role": "approver",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "apr-1",
		"nonce": nonce, "expires_at": "1893456000",
		"entitlement_id": c.EntitlementID, "app_id": c.AppID,
		"candidate_hash":            cand.CandidateHash,
		"challenge_resolution_hash": resolutionDigest,
		"amount":                    c.Amount, "denom": c.Denom, "rule_version": c.RuleVersion,
	}
	return &types.MsgApproveEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId:           c.EntitlementID,
		ChallengeResolutionHash: resolutionDigest,
		Proof:                   signProof(t, aprPriv, rec),
	}
}

func TestVector_APPROVAL_BEFORE_SEVEN_DAYS(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)

	// now = published_at + 604799 → reject
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds-1, 0))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "201"))
	require.ErrorIs(t, err, types.ErrChallengeWindow)

	// now = published_at + 604800 → ok
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "202"))
	require.NoError(t, err)

	ent, ok := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.True(t, ok)
	require.Equal(t, types.EntitlementStateClaimable, ent.State)
	pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.Equal(t, "19500", pool.Available) // 51000 - 31500 (1000+50000)
	require.Equal(t, "31500", pool.Reserved)
	require.Equal(t, "0", pool.Paid)
}

func TestVector_FIXED_POOL_EXHAUSTION_AND_CROSS_EPOCH_RESERVATION(t *testing.T) {
	e, pubPriv, _, aprPriv, c1 := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c1.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c1, "301"))
	require.NoError(t, err)

	// second candidate with amount 40000 > remaining available 19500
	c2 := sampleCandidateFields()
	c2.Amount = "40000"
	c2.PavID = "2222222222222222222222222222222222222222222222222222222222222299"
	c2.EntitlementID = wirev1.EntitlementIDOf(c2)
	hash, err := wirev1.CandidateHash(c2)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "302", "expires_at": "1893456000",
		"entitlement_id": c2.EntitlementID, "app_id": c2.AppID,
		"candidate_hash": hash, "rule_version": c2.RuleVersion,
	}
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: c2.EntitlementID, AppId: c2.AppID, Epoch: c2.Epoch, Amount: c2.Amount,
		Denom: c2.Denom, GasUsed: c2.GasUsed, QualifiedTxCount: c2.QualifiedTxCount,
		PavId: c2.PavID, EvidenceRoot: c2.EvidenceRoot, GasEvidenceRoot: c2.GasEvidenceRoot,
		RuleVersion: c2.RuleVersion, Proof: signProof(t, pubPriv, rec),
	})
	require.NoError(t, err)

	cand2, _ := e.k.GetCandidate(e.ctx, c2.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand2.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c2, "303"))
	require.ErrorIs(t, err, types.ErrInsufficientAvailable)
}

func TestVector_OWNER_CLAIM_ONCE_AND_CROSS_APP_REJECTED(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "401"))
	require.NoError(t, err)

	// Cross-app: register app B owned by payer, then payer claims A's entitlement.
	appB, _ := hex.DecodeString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	_, err = e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority: e.auth.String(), AppId: appB, Owner: e.payer.String(), AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)

	_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.payer.String(), EntitlementId: c.EntitlementID,
	})
	require.ErrorIs(t, err, types.ErrBeneficiaryMustBeOwner)

	_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.owner.String(), EntitlementId: c.EntitlementID,
	})
	require.NoError(t, err)

	ent, _ := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.Equal(t, types.EntitlementStateClaimed, ent.State)
	pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.Equal(t, "0", pool.Reserved)
	require.Equal(t, "31500", pool.Paid)

	_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.owner.String(), EntitlementId: c.EntitlementID,
	})
	require.ErrorIs(t, err, types.ErrAlreadyClaimed)
}
