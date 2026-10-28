package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// Vector IDs mirror x/appmining/spec/acceptance-vectors-0.5.json (T1 phase).

func setupBoundPublishEnv(t *testing.T) (testEnv, ed25519.PrivateKey, ed25519.PrivateKey, []byte) {
	t.Helper()
	e, _, _ := setupP1(t)
	appBound, _ := hex.DecodeString("1111111111111111111111111111111111111111111111111111111111111111")
	params := e.k.GetParams(e.ctx)
	params.EntitlementAllowedAssessmentVersions = []string{types.DefaultAssessmentVersion}
	require.NoError(t, e.k.SetParams(e.ctx, params))

	signerPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x51}, ed25519.SeedSize))
	_, err := e.server.RegisterAssessmentSigner(e.ctx, &types.MsgRegisterAssessmentSigner{
		Authority: e.auth.String(), KeyId: "assess-vec-1",
		Pubkey: signerPriv.Public().(ed25519.PublicKey), Version: 1,
	})
	require.NoError(t, err)

	boundPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x52}, ed25519.SeedSize))
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "bound-vec", Pubkey: boundPriv.Public().(ed25519.PublicKey), Version: 1, AppId: appBound,
	})
	require.NoError(t, err)
	return e, signerPriv, boundPriv, appBound
}

func makeCred(t *testing.T, signerPriv ed25519.PrivateKey, cand wirev1.Candidate, mutate func(*types.AssessmentCredential)) types.AssessmentCredential {
	t.Helper()
	cred := types.AssessmentCredential{
		CredentialVersion: types.DefaultAssessmentVersion,
		AppId:             cand.AppID,
		PoolId:            testPoolID,
		PavId:             cand.PavID,
		EvidenceRoot:      cand.EvidenceRoot,
		GasEvidenceRoot:   cand.GasEvidenceRoot,
		Amount:            cand.Amount,
		Denom:             cand.Denom,
		GasUsed:           cand.GasUsed,
		QualifiedTxCount:  cand.QualifiedTxCount,
		RuleVersion:       cand.RuleVersion,
		IssuedAt:          "1700000000",
		ExpiresAt:         "1893456000",
		SignerKeyId:       "assess-vec-1",
	}
	if mutate != nil {
		mutate(&cred)
	}
	signAssessment(t, signerPriv, &cred)
	return cred
}

func publishBound(t *testing.T, e testEnv, boundPriv ed25519.PrivateKey, cand wirev1.Candidate, nonce string, cred types.AssessmentCredential) error {
	t.Helper()
	hash, err := wirev1.CandidateHash(cand)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "bound-vec",
		"nonce": nonce, "expires_at": "1893456000",
		"entitlement_id": cand.EntitlementID, "app_id": cand.AppID,
		"candidate_hash": hash, "rule_version": cand.RuleVersion,
	}
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: cand.EntitlementID, AppId: cand.AppID, Epoch: cand.Epoch,
		Amount: cand.Amount, Denom: cand.Denom, GasUsed: cand.GasUsed,
		QualifiedTxCount: cand.QualifiedTxCount, PavId: cand.PavID,
		EvidenceRoot: cand.EvidenceRoot, GasEvidenceRoot: cand.GasEvidenceRoot,
		RuleVersion: cand.RuleVersion, Proof: signProof(t, boundPriv, rec),
		Credential: cred,
	})
	return err
}

func TestVector_BOUND_PUBLISHER_REQUIRES_CREDENTIAL(t *testing.T) {
	e, _, boundPriv, _ := setupBoundPublishEnv(t)
	cand := sampleCandidateFields()
	err := publishBound(t, e, boundPriv, cand, "9401", types.AssessmentCredential{})
	require.ErrorIs(t, err, types.ErrAssessmentCredentialInvalid)
}

func TestVector_GLOBAL_PUBLISHER_WITHOUT_CREDENTIAL_OK(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	_ = publishForDeposit(t, e, pubPriv, "9402")
}

func TestVector_ROLE_KEY_APP_MISMATCH(t *testing.T) {
	TestT1a_PublishCandidateAppBoundMismatch(t)
}

func TestVector_REGISTER_APPLICATION_EMBEDDED_PUBLISHER(t *testing.T) {
	TestT1a_RegisterApplicationEmbeddedPublisher(t)
}

func TestVector_ASSESSMENT_CREDENTIAL_VALID(t *testing.T) {
	e, signerPriv, boundPriv, _ := setupBoundPublishEnv(t)
	cand := sampleCandidateFields()
	cred := makeCred(t, signerPriv, cand, nil)
	require.NoError(t, publishBound(t, e, boundPriv, cand, "9410", cred))
}

func TestVector_ASSESSMENT_CREDENTIAL_EXPIRED(t *testing.T) {
	e, signerPriv, boundPriv, _ := setupBoundPublishEnv(t)
	cand := sampleCandidateFields()
	cred := makeCred(t, signerPriv, cand, func(c *types.AssessmentCredential) {
		c.IssuedAt = "1000"
		c.ExpiresAt = "2000"
	})
	e.ctx = e.ctx.WithBlockTime(time.Unix(2001, 0))
	err := publishBound(t, e, boundPriv, cand, "9411", cred)
	require.ErrorIs(t, err, types.ErrAssessmentCredentialInvalid)
}

func TestVector_ASSESSMENT_CREDENTIAL_FIELD_MISMATCH(t *testing.T) {
	e, signerPriv, boundPriv, _ := setupBoundPublishEnv(t)
	cand := sampleCandidateFields()
	cred := makeCred(t, signerPriv, cand, func(c *types.AssessmentCredential) {
		c.Amount = "999999"
	})
	err := publishBound(t, e, boundPriv, cand, "9412", cred)
	require.ErrorIs(t, err, types.ErrAssessmentCredentialInvalid)
}

func TestVector_ASSESSMENT_CREDENTIAL_REUSE(t *testing.T) {
	TestT1b_CredentialHappyPathAndReuse(t)
}

func TestVector_ASSESSMENT_VERSION_NOT_ALLOWLISTED(t *testing.T) {
	e, signerPriv, boundPriv, _ := setupBoundPublishEnv(t)
	// blank allowlist after setup → default-deny
	params := e.k.GetParams(e.ctx)
	params.EntitlementAllowedAssessmentVersions = nil
	require.NoError(t, e.k.SetParams(e.ctx, params))

	cand := sampleCandidateFields()
	cred := makeCred(t, signerPriv, cand, nil)
	err := publishBound(t, e, boundPriv, cand, "9413", cred)
	require.ErrorIs(t, err, types.ErrAssessmentCredentialInvalid)
}

func TestVector_CREDENTIAL_ID_HELD_AFTER_REJECTED(t *testing.T) {
	e, signerPriv, boundPriv, _ := setupBoundPublishEnv(t)
	cand := sampleCandidateFields()
	cred := makeCred(t, signerPriv, cand, nil)
	require.NoError(t, publishBound(t, e, boundPriv, cand, "9420", cred))

	stored, ok := e.k.GetCandidate(e.ctx, cand.EntitlementID)
	require.True(t, ok)
	stored.Status = types.CandidateStatusRejected
	e.k.SetCandidate(e.ctx, stored)
	require.True(t, e.k.HasCredentialID(e.ctx, cred.CredentialId))

	cand2 := sampleCandidateFields()
	cand2.Epoch = "2"
	cand2.EntitlementID = wirev1.EntitlementIDOf(cand2)
	// Force reuse of the same credential_id via index collision (sign fresh then overwrite id+index).
	cred2 := makeCred(t, signerPriv, cand2, nil)
	e.k.SetCredentialIndex(e.ctx, cred2.CredentialId, cand.EntitlementID)
	err := publishBound(t, e, boundPriv, cand2, "9421", cred2)
	require.ErrorIs(t, err, types.ErrAssessmentCredentialReuse)
}

func TestVector_CHALLENGE_OPEN_ACCESS_OPEN_NO_DEPOSIT(t *testing.T) {
	TestT1c_OpenModeStillZeroDeposit(t)
}

func TestVector_CHALLENGE_OPEN_ACCESS_NONE_REJECTED(t *testing.T) {
	TestT1c_ChallengeOpenAccessNone(t)
}

func TestVector_CHALLENGE_DEPOSIT_UPHELD_REFUND(t *testing.T) {
	TestT1c_DepositUpheldRefunds(t)
}

func TestVector_CHALLENGE_DEPOSIT_DISMISSED_TO_POOL(t *testing.T) {
	TestT1c_DepositDismissedToPool(t)
}
