package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

const (
	expireWindowSeconds = 60
	expireGraceSeconds  = 60
	expireDepositAmount = "500"
)

// expireEnv builds a pool that can approve the sample candidate (funded +
// allowlist + approver key) and a short sealed challenge window (60s) with a
// 60s reviewer grace, so the timeout path is reachable inside a test.
func expireEnv(t *testing.T) (testEnv, ed25519.PrivateKey, ed25519.PrivateKey, ed25519.PrivateKey) {
	t.Helper()
	e, pubPriv, revPriv := setupP1(t)

	e.bank.fundAccount(e.auth, types.DefaultChallengeBondDenom, math.NewInt(200_000))
	_, err := e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Authority: e.auth.String(), PoolId: testPoolID, Amount: "50000",
	})
	require.NoError(t, err)

	params := e.k.GetParams(e.ctx)
	params.EntitlementEnableMinFunded = "1000"
	params.EntitlementAllowedRuleVersions = []string{sampleCandidateFields().RuleVersion}
	params.ChallengeWindowSeconds = expireWindowSeconds
	params.ChallengeResolutionGraceSeconds = expireGraceSeconds
	params.ChallengeOpenAccess = types.ChallengeOpenAccessDeposit
	params.ChallengeDepositAmount = expireDepositAmount
	params.ChallengeDepositDenom = types.DefaultChallengeBondDenom
	require.NoError(t, e.k.SetParams(e.ctx, params))

	aprPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x03}, ed25519.SeedSize))
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRoleApprover,
		KeyId: "apr-1", Pubkey: aprPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	require.NoError(t, err)

	e.bank.fundAccount(e.payer, types.DefaultChallengeBondDenom, math.NewInt(5_000))
	return e, pubPriv, revPriv, aprPriv
}

// openDepositChallenge publishes the sample candidate and opens a challenge
// against it, returning the challenge id.
func openDepositChallenge(t *testing.T, e testEnv, pubPriv ed25519.PrivateKey, nonce, evidenceDigest string) string {
	t.Helper()
	c := publishForDeposit(t, e, pubPriv, nonce)
	res, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger:     e.payer.String(),
		EntitlementId:  c.EntitlementID,
		EvidenceType:   types.EvidenceTypeTxMissingOrUnfinalized,
		EvidenceDigest: evidenceDigest,
	})
	require.NoError(t, err)
	return res.ChallengeId
}

func expireChallenge(e testEnv, cid string) (*types.MsgExpireEntitlementChallengeResponse, error) {
	return e.server.ExpireEntitlementChallenge(e.ctx, &types.MsgExpireEntitlementChallenge{
		Submitter:   e.sub.String(),
		PoolId:      testPoolID,
		ChallengeId: cid,
	})
}

// resolutionDeadline returns opened_at + sealed window + grace for the test env.
func resolutionDeadline(t *testing.T, e testEnv, cid string) int64 {
	t.Helper()
	chal, found := e.k.GetEntitlementChallenge(e.ctx, cid)
	require.True(t, found)
	cand, found := e.k.GetCandidate(e.ctx, chal.CandidateId)
	require.True(t, found)
	params := e.k.GetParams(e.ctx)
	return types.ChallengeResolutionDeadline(cand, chal, params.EffectiveChallengeResolutionGraceSeconds())
}

func TestExpire_RejectedBeforeGraceElapses(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9001", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	deadline := resolutionDeadline(t, e, cid)
	// One second early: the reviewer still owns the decision.
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline-1, 0))
	_, err := expireChallenge(e, cid)
	require.ErrorIs(t, err, types.ErrChallengeNotExpired)

	chal, _ := e.k.GetEntitlementChallenge(e.ctx, cid)
	require.Equal(t, types.EntitlementChallengeStatusOpen, chal.Status)
}

func TestExpire_AtDeadlineRefundsAndKeepsCandidatePublished(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9002", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	chal, _ := e.k.GetEntitlementChallenge(e.ctx, cid)
	before := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)

	deadline := resolutionDeadline(t, e, cid)
	require.EqualValues(t, chal.OpenedAt+expireWindowSeconds+expireGraceSeconds, deadline,
		"resolution deadline must be opened_at + sealed window + grace")
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline, 0))

	res, err := expireChallenge(e, cid)
	require.NoError(t, err)
	require.Equal(t, types.ExpiredResolutionDigest(cid), res.ResolutionDigest)

	chal, _ = e.k.GetEntitlementChallenge(e.ctx, cid)
	require.Equal(t, types.EntitlementChallengeStatusExpired, chal.Status)
	require.Equal(t, res.ResolutionDigest, chal.ResolutionDigest)
	require.Empty(t, chal.Reviewer, "timeout must not attribute a reviewer")
	require.EqualValues(t, deadline, chal.ResolvedAt)

	cand, _ := e.k.GetCandidate(e.ctx, chal.CandidateId)
	require.Equal(t, types.CandidateStatusPublished, cand.Status,
		"neutral timeout must not reject the candidate")

	after := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)
	require.Equal(t, before.Amount.Add(math.NewInt(500)), after.Amount,
		"deposit must be fully refunded to the challenger")

	// The open-challenge index entry is cleared so a future challenge is possible.
	_, found := e.k.GetOpenEntitlementChallenge(e.ctx, chal.CandidateId, chal.Challenger)
	require.False(t, found)
}

func TestExpire_WorksWhilePoolPaused(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9003", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")

	pool, found := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.True(t, found)
	pool.Paused = true
	e.k.SetEntitlementPool(e.ctx, pool)

	deadline := resolutionDeadline(t, e, cid)
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline, 0))

	// Guard: the paused pool still blocks the money-moving flows...
	_, err := e.server.ApproveEntitlement(e.ctx, &types.MsgApproveEntitlement{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: mustCandidateID(t, e, cid),
	})
	require.ErrorIs(t, err, types.ErrPoolPaused)

	// ...but the timeout settlement must stay reachable, otherwise the deposit
	// is stuck exactly when an operator reaches for the emergency brake.
	_, err = expireChallenge(e, cid)
	require.NoError(t, err)

	chal, _ := e.k.GetEntitlementChallenge(e.ctx, cid)
	require.Equal(t, types.EntitlementChallengeStatusExpired, chal.Status)
}

func TestExpire_ThenApproveBindsDigest(t *testing.T) {
	e, pubPriv, _, aprPriv := expireEnv(t)
	c := publishForDeposit(t, e, pubPriv, "9004")
	res, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger:     e.payer.String(),
		EntitlementId:  c.EntitlementID,
		EvidenceType:   types.EvidenceTypeTxMissingOrUnfinalized,
		EvidenceDigest: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	})
	require.NoError(t, err)

	deadline := resolutionDeadline(t, e, res.ChallengeId)
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline, 0))
	exp, err := expireChallenge(e, res.ChallengeId)
	require.NoError(t, err)

	// Empty-hash approve stays blocked: the candidate does have a challenge
	// record, so DEC-01 scheme C requires a bound digest.
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "9101"))
	require.ErrorIs(t, err, types.ErrOpenOrUpheldChallenge)

	// A digest that is not the on-chain EXPIRED digest is rejected.
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsgWithResolution(t, e, aprPriv, c, "9102",
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"))
	require.ErrorIs(t, err, types.ErrResolutionUnbound)

	// The real digest unblocks approval; the candidate is paid out normally.
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsgWithResolution(t, e, aprPriv, c, "9103", exp.ResolutionDigest))
	require.NoError(t, err)

	ent, found := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.True(t, found)
	require.Equal(t, types.EntitlementStateClaimable, ent.State)
}

func TestExpire_IsTerminal(t *testing.T) {
	e, pubPriv, revPriv, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9005", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	deadline := resolutionDeadline(t, e, cid)
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline, 0))
	_, err := expireChallenge(e, cid)
	require.NoError(t, err)

	// Re-expiring the same challenge must fail.
	_, err = expireChallenge(e, cid)
	require.ErrorIs(t, err, types.ErrEntitlementChallengeNotOpen)

	// A late reviewer decision must fail too: the window already spoke.
	chal, _ := e.k.GetEntitlementChallenge(e.ctx, cid)
	rationale := "1111111111111111111111111111111111111111111111111111111111111111"
	rec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
		"nonce": "9201", "expires_at": wireExp(e.ctx),
		"challenge_id": cid, "candidate_id": chal.CandidateId,
		"outcome": "UPHELD", "rationale_digest": rationale,
	}
	_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: cid, Outcome: "UPHELD", RationaleDigest: rationale,
		Proof: signProof(t, revPriv, rec),
	})
	require.ErrorIs(t, err, types.ErrEntitlementChallengeNotOpen)
}

func TestExpire_UnknownChallengeAndWrongPool(t *testing.T) {
	e, pubPriv, _, _ := expireEnv(t)
	cid := openDepositChallenge(t, e, pubPriv, "9006", "2222222222222222222222222222222222222222222222222222222222222222")

	_, err := expireChallenge(e, "3333333333333333333333333333333333333333333333333333333333333333")
	require.ErrorIs(t, err, types.ErrEntitlementChallengeNotFound)

	deadline := resolutionDeadline(t, e, cid)
	e.ctx = e.ctx.WithBlockTime(time.Unix(deadline, 0))
	_, err = e.server.ExpireEntitlementChallenge(e.ctx, &types.MsgExpireEntitlementChallenge{
		Submitter: e.sub.String(), PoolId: "other-pool", ChallengeId: cid,
	})
	require.ErrorIs(t, err, types.ErrPoolMismatch)
}

func mustCandidateID(t *testing.T, e testEnv, cid string) string {
	t.Helper()
	chal, found := e.k.GetEntitlementChallenge(e.ctx, cid)
	require.True(t, found)
	return chal.CandidateId
}
