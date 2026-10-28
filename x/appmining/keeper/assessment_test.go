package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func signAssessment(t *testing.T, priv ed25519.PrivateKey, cred *types.AssessmentCredential) {
	t.Helper()
	id, err := wirev1.AssessmentCredentialID(cred.Record())
	require.NoError(t, err)
	cred.CredentialId = id
	raw, err := wirev1.AssessmentBytes(cred.Record())
	require.NoError(t, err)
	sig := ed25519.Sign(priv, raw)
	cred.SignatureBase64 = base64.StdEncoding.EncodeToString(sig)
}

func TestT1b_BoundKeyRequiresCredential(t *testing.T) {
	e, _, _ := setupP1(t)
	appBound, _ := hex.DecodeString("1111111111111111111111111111111111111111111111111111111111111111")

	params := e.k.GetParams(e.ctx)
	params.EntitlementAllowedAssessmentVersions = []string{types.DefaultAssessmentVersion}
	require.NoError(t, e.k.SetParams(e.ctx, params))

	boundPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x07}, ed25519.SeedSize))
	_, err := e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "bound-need-cred", Pubkey: boundPriv.Public().(ed25519.PublicKey), Version: 1, AppId: appBound,
	})
	require.NoError(t, err)

	cand := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(cand)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "bound-need-cred",
		"nonce": "9201", "expires_at": "1893456000",
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
	})
	require.ErrorIs(t, err, types.ErrAssessmentCredentialInvalid)
}

func TestT1b_CredentialHappyPathAndReuse(t *testing.T) {
	e, _, _ := setupP1(t)
	appBound, _ := hex.DecodeString("1111111111111111111111111111111111111111111111111111111111111111")

	params := e.k.GetParams(e.ctx)
	params.EntitlementAllowedAssessmentVersions = []string{types.DefaultAssessmentVersion}
	require.NoError(t, e.k.SetParams(e.ctx, params))

	signerPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x08}, ed25519.SeedSize))
	e.k.SetAssessmentSigner(e.ctx, types.AssessmentSigner{
		KeyId: "assess-1", Pubkey: signerPriv.Public().(ed25519.PublicKey),
		Status: types.RoleKeyStatusActive, Version: 1, RegisteredHeight: 1,
	})

	boundPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x09}, ed25519.SeedSize))
	_, err := e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "bound-ok", Pubkey: boundPriv.Public().(ed25519.PublicKey), Version: 1, AppId: appBound,
	})
	require.NoError(t, err)

	cand := sampleCandidateFields()
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
		SignerKeyId:       "assess-1",
	}
	signAssessment(t, signerPriv, &cred)

	hash, err := wirev1.CandidateHash(cand)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "bound-ok",
		"nonce": "9210", "expires_at": "1893456000",
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
	require.NoError(t, err)

	stored, ok := e.k.GetCandidate(e.ctx, cand.EntitlementID)
	require.True(t, ok)
	require.Equal(t, cred.CredentialId, stored.CredentialId)
	ent, ok := e.k.GetCredentialEntitlement(e.ctx, cred.CredentialId)
	require.True(t, ok)
	require.Equal(t, cand.EntitlementID, ent)

	// Pre-seed index then publish a new candidate with a freshly signed credential
	// that happens to collide on credential_id only if we force the index hit:
	// build cand2 + matching cred2, then overwrite index with cred2's id first.
	cand2 := sampleCandidateFields()
	cand2.Epoch = "2"
	cand2.EntitlementID = wirev1.EntitlementIDOf(cand2)
	cred2 := types.AssessmentCredential{
		CredentialVersion: types.DefaultAssessmentVersion,
		AppId:             cand2.AppID,
		PoolId:            testPoolID,
		PavId:             cand2.PavID,
		EvidenceRoot:      cand2.EvidenceRoot,
		GasEvidenceRoot:   cand2.GasEvidenceRoot,
		Amount:            cand2.Amount,
		Denom:             cand2.Denom,
		GasUsed:           cand2.GasUsed,
		QualifiedTxCount:  cand2.QualifiedTxCount,
		RuleVersion:       cand2.RuleVersion,
		IssuedAt:          "1700000000",
		ExpiresAt:         "1893456000",
		SignerKeyId:       "assess-1",
	}
	signAssessment(t, signerPriv, &cred2)
	e.k.SetCredentialIndex(e.ctx, cred2.CredentialId, "prior-entitlement")

	hash2, err := wirev1.CandidateHash(cand2)
	require.NoError(t, err)
	rec2 := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "bound-ok",
		"nonce": "9211", "expires_at": "1893456000",
		"entitlement_id": cand2.EntitlementID, "app_id": cand2.AppID,
		"candidate_hash": hash2, "rule_version": cand2.RuleVersion,
	}
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: cand2.EntitlementID, AppId: cand2.AppID, Epoch: cand2.Epoch,
		Amount: cand2.Amount, Denom: cand2.Denom, GasUsed: cand2.GasUsed,
		QualifiedTxCount: cand2.QualifiedTxCount, PavId: cand2.PavID,
		EvidenceRoot: cand2.EvidenceRoot, GasEvidenceRoot: cand2.GasEvidenceRoot,
		RuleVersion: cand2.RuleVersion, Proof: signProof(t, boundPriv, rec2),
		Credential: cred2,
	})
	require.ErrorIs(t, err, types.ErrAssessmentCredentialReuse)
}
