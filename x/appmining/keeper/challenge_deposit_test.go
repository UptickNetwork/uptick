package keeper_test

import (
	"crypto/ed25519"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func enableDepositMode(t *testing.T, e testEnv, amount string) {
	t.Helper()
	params := e.k.GetParams(e.ctx)
	params.ChallengeOpenAccess = types.ChallengeOpenAccessDeposit
	params.ChallengeDepositAmount = amount
	params.ChallengeDepositDenom = types.DefaultChallengeBondDenom
	require.NoError(t, e.k.SetParams(e.ctx, params))
}

func publishForDeposit(t *testing.T, e testEnv, pubPriv ed25519.PrivateKey, nonce string) wirev1.Candidate {
	t.Helper()
	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": nonce, "expires_at": "1893456000",
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

func resolveForDeposit(t *testing.T, e testEnv, revPriv ed25519.PrivateKey, challengeID, entitlementID, outcome, nonce string) {
	t.Helper()
	rec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "rev-1",
		"nonce": nonce, "expires_at": "1893456000",
		"challenge_id": challengeID, "candidate_id": entitlementID,
		"outcome": outcome, "rationale_digest": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	}
	_, err := e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: challengeID, Outcome: outcome,
		RationaleDigest: rec["rationale_digest"],
		Proof:           signProof(t, revPriv, rec),
	})
	require.NoError(t, err)
}

func TestT1c_ChallengeOpenAccessNone(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	c := publishForDeposit(t, e, pubPriv, "8001")

	params := e.k.GetParams(e.ctx)
	params.ChallengeOpenAccess = types.ChallengeOpenAccessNone
	require.NoError(t, e.k.SetParams(e.ctx, params))

	_, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeTxMissingOrUnfinalized,
		EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)
}

func TestT1c_DepositUpheldRefunds(t *testing.T) {
	e, pubPriv, revPriv := setupP1(t)
	enableDepositMode(t, e, "500")
	e.bank.fundAccount(e.payer, types.DefaultChallengeBondDenom, math.NewInt(5000))
	c := publishForDeposit(t, e, pubPriv, "8010")

	before := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)
	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeTxMissingOrUnfinalized,
		EvidenceDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	require.NoError(t, err)
	afterOpen := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)
	require.True(t, before.Amount.Sub(afterOpen.Amount).Equal(math.NewInt(500)))

	chal, ok := e.k.GetEntitlementChallenge(e.ctx, openRes.ChallengeId)
	require.True(t, ok)
	require.Equal(t, "500", chal.DepositAmount)

	resolveForDeposit(t, e, revPriv, openRes.ChallengeId, c.EntitlementID, "UPHELD", "8011")
	afterResolve := e.bank.GetBalance(e.ctx, e.payer, types.DefaultChallengeBondDenom)
	require.True(t, afterResolve.Amount.Equal(before.Amount))
}

func TestT1c_DepositDismissedToPool(t *testing.T) {
	e, pubPriv, revPriv := setupP1(t)
	enableDepositMode(t, e, "700")
	e.bank.fundAccount(e.payer, types.DefaultChallengeBondDenom, math.NewInt(5000))
	c := publishForDeposit(t, e, pubPriv, "8020")

	poolBefore, ok := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.True(t, ok)

	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeReceiptQbtMismatch,
		EvidenceDigest: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	})
	require.NoError(t, err)

	resolveForDeposit(t, e, revPriv, openRes.ChallengeId, c.EntitlementID, "DISMISSED", "8021")

	poolAfter, ok := e.k.GetEntitlementPool(e.ctx, testPoolID)
	require.True(t, ok)
	fundedDelta, err := types.SubAmountStrings(poolAfter.Funded, poolBefore.Funded)
	require.NoError(t, err)
	require.Equal(t, "700", fundedDelta)
	availDelta, err := types.SubAmountStrings(poolAfter.Available, poolBefore.Available)
	require.NoError(t, err)
	require.Equal(t, "700", availDelta)
}

func TestT1c_OpenModeStillZeroDeposit(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	c := publishForDeposit(t, e, pubPriv, "8030")
	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeAppOrEmitterIneligible,
		EvidenceDigest: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	})
	require.NoError(t, err)
	chal, ok := e.k.GetEntitlementChallenge(e.ctx, openRes.ChallengeId)
	require.True(t, ok)
	require.Empty(t, chal.DepositAmount)
}
