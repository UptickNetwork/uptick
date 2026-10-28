package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func TestAssessmentSigner_RegisterRevokeList(t *testing.T) {
	e := setupTest(t)
	priv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x21}, ed25519.SeedSize))
	pub := priv.Public().(ed25519.PublicKey)

	_, err := e.server.RegisterAssessmentSigner(e.ctx, &types.MsgRegisterAssessmentSigner{
		Authority: e.auth.String(), KeyId: "uge-assess-01", Pubkey: pub, Version: 1,
	})
	require.NoError(t, err)

	s, ok := e.k.GetAssessmentSigner(e.ctx, "uge-assess-01")
	require.True(t, ok)
	require.Equal(t, types.RoleKeyStatusActive, s.Status)
	require.Equal(t, pub, ed25519.PublicKey(s.Pubkey))

	// idempotent same material
	_, err = e.server.RegisterAssessmentSigner(e.ctx, &types.MsgRegisterAssessmentSigner{
		Authority: e.auth.String(), KeyId: "uge-assess-01", Pubkey: pub, Version: 1,
	})
	require.NoError(t, err)

	// conflict on different pubkey without revoke
	other := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x22}, ed25519.SeedSize)).Public().(ed25519.PublicKey)
	_, err = e.server.RegisterAssessmentSigner(e.ctx, &types.MsgRegisterAssessmentSigner{
		Authority: e.auth.String(), KeyId: "uge-assess-01", Pubkey: other, Version: 2,
	})
	require.ErrorIs(t, err, types.ErrRoleKeyConflict)

	_, err = e.server.SetAssessmentSignerStatus(e.ctx, &types.MsgSetAssessmentSignerStatus{
		Authority: e.auth.String(), KeyId: "uge-assess-01", Status: types.RoleKeyStatusRevoked,
	})
	require.NoError(t, err)

	// revoke → re-register with new pubkey
	_, err = e.server.RegisterAssessmentSigner(e.ctx, &types.MsgRegisterAssessmentSigner{
		Authority: e.auth.String(), KeyId: "uge-assess-01", Pubkey: other, Version: 2,
	})
	require.NoError(t, err)

	var listed []types.AssessmentSigner
	e.k.IterateAssessmentSigners(e.ctx, func(s types.AssessmentSigner) bool {
		listed = append(listed, s)
		return false
	})
	require.Len(t, listed, 1)
	require.Equal(t, types.RoleKeyStatusActive, listed[0].Status)
	require.Equal(t, other, ed25519.PublicKey(listed[0].Pubkey))
}

func TestAssessmentSigner_UnauthorizedAndKeyID(t *testing.T) {
	e := setupTest(t)
	pub := bytes.Repeat([]byte{0x33}, 32)

	_, err := e.server.RegisterAssessmentSigner(e.ctx, &types.MsgRegisterAssessmentSigner{
		Authority: e.owner.String(), KeyId: "uge-assess-02", Pubkey: pub, Version: 1,
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	msg := &types.MsgRegisterAssessmentSigner{
		Authority: e.auth.String(), KeyId: "BAD_ID", Pubkey: pub, Version: 1,
	}
	require.Error(t, msg.ValidateBasic())

	_, err = e.server.SetAssessmentSignerStatus(e.ctx, &types.MsgSetAssessmentSignerStatus{
		Authority: e.auth.String(), KeyId: "missing-key", Status: types.RoleKeyStatusPaused,
	})
	require.ErrorIs(t, err, types.ErrRoleKeyNotFound)
}

func TestAssessmentSigner_MsgPathAcceptedByVerify(t *testing.T) {
	e, _, _ := setupP1(t)
	params := e.k.GetParams(e.ctx)
	params.EntitlementAllowedAssessmentVersions = []string{types.DefaultAssessmentVersion}
	require.NoError(t, e.k.SetParams(e.ctx, params))

	signerPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x44}, ed25519.SeedSize))
	_, err := e.server.RegisterAssessmentSigner(e.ctx, &types.MsgRegisterAssessmentSigner{
		Authority: e.auth.String(),
		KeyId:     "assess-msg-1",
		Pubkey:    signerPriv.Public().(ed25519.PublicKey),
		Version:   1,
	})
	require.NoError(t, err)

	appBound := bytes.Repeat([]byte{0x11}, 32)
	boundPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x45}, ed25519.SeedSize))
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "bound-msg", Pubkey: boundPriv.Public().(ed25519.PublicKey), Version: 1, AppId: appBound,
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
		SignerKeyId:       "assess-msg-1",
	}
	signAssessment(t, signerPriv, &cred)

	hash, err := wirev1.CandidateHash(cand)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "bound-msg",
		"nonce": "9301", "expires_at": "1893456000",
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
}
