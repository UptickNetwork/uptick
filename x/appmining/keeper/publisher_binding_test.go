package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func TestT1a_RegisterRoleKeyAppBound(t *testing.T) {
	e := setupTest(t)
	appID, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority: e.auth.String(), AppId: appID, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)

	pk := bytes.Repeat([]byte{0xab}, 32)
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "bound-1", Pubkey: pk, Version: 1, AppId: appID,
	})
	require.NoError(t, err)

	rk, ok := e.k.GetRoleKey(e.ctx, types.EntitlementRolePublisher, "bound-1")
	require.True(t, ok)
	require.Equal(t, appID, rk.AppId)

	// unknown app rejected
	otherApp := bytes.Repeat([]byte{0xbb}, 32)
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "bound-2", Pubkey: bytes.Repeat([]byte{0xac}, 32), Version: 1, AppId: otherApp,
	})
	require.ErrorIs(t, err, types.ErrApplicationNotFound)

	// Roles filter returns bound keys only
	all, err := e.k.Roles(e.ctx, &types.QueryRolesRequest{Role: types.EntitlementRolePublisher})
	require.NoError(t, err)
	require.Len(t, all.Keys, 1)

	filtered, err := e.k.Roles(e.ctx, &types.QueryRolesRequest{
		Role: types.EntitlementRolePublisher, AppId: appID,
	})
	require.NoError(t, err)
	require.Len(t, filtered.Keys, 1)

	miss, err := e.k.Roles(e.ctx, &types.QueryRolesRequest{
		Role: types.EntitlementRolePublisher, AppId: otherApp,
	})
	require.NoError(t, err)
	require.Empty(t, miss.Keys)
}

func TestT1a_RegisterApplicationEmbeddedPublisher(t *testing.T) {
	e := setupTest(t)
	appID, _ := hex.DecodeString("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	pk := bytes.Repeat([]byte{0xcd}, 32)

	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority:            e.auth.String(),
		AppId:                appID,
		Owner:                e.owner.String(),
		AttestationSubmitter: e.sub.String(),
		PublisherPubkey:      pk,
		PublisherKeyId:       "emb-pub-1",
	})
	require.NoError(t, err)

	rk, ok := e.k.GetRoleKey(e.ctx, types.EntitlementRolePublisher, "emb-pub-1")
	require.True(t, ok)
	require.Equal(t, types.RoleKeyStatusActive, rk.Status)
	require.Equal(t, appID, rk.AppId)
	require.Equal(t, pk, rk.Pubkey)

	// key_id conflict on second app
	app2, _ := hex.DecodeString("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	_, err = e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority:            e.auth.String(),
		AppId:                app2,
		Owner:                e.owner.String(),
		AttestationSubmitter: e.sub.String(),
		PublisherPubkey:      bytes.Repeat([]byte{0xce}, 32),
		PublisherKeyId:       "emb-pub-1",
	})
	require.ErrorIs(t, err, types.ErrRoleKeyConflict)
}

func TestT1a_PublishCandidateAppBoundMismatch(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	appBound, _ := hex.DecodeString("1111111111111111111111111111111111111111111111111111111111111111")
	otherApp, _ := hex.DecodeString("2222222222222222222222222222222222222222222222222222222222222222")

	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority: e.auth.String(), AppId: otherApp, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)

	boundPK := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x05}, ed25519.SeedSize))
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "bound-pub", Pubkey: boundPK.Public().(ed25519.PublicKey), Version: 1, AppId: appBound,
	})
	require.NoError(t, err)

	// mismatch: key bound to appBound, publish for otherApp → 79
	cand := sampleCandidateFields()
	cand.AppID = hex.EncodeToString(otherApp)
	cand.EntitlementID = wirev1.EntitlementIDOf(cand)
	hash, err := wirev1.CandidateHash(cand)
	require.NoError(t, err)

	record := map[string]string{
		"protocol_version": "1",
		"operation":        "PUBLISH",
		"role":             "publisher",
		"chain_id":         testChainID,
		"pool_id":          testPoolID,
		"key_id":           "bound-pub",
		"nonce":            "9101",
		"expires_at":       "1893456000",
		"entitlement_id":   cand.EntitlementID,
		"app_id":           cand.AppID,
		"candidate_hash":   hash,
		"rule_version":     cand.RuleVersion,
	}
	proof := signProof(t, boundPK, record)
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: cand.EntitlementID, AppId: cand.AppID, Epoch: cand.Epoch,
		Amount: cand.Amount, Denom: cand.Denom, GasUsed: cand.GasUsed,
		QualifiedTxCount: cand.QualifiedTxCount, PavId: cand.PavID,
		EvidenceRoot: cand.EvidenceRoot, GasEvidenceRoot: cand.GasEvidenceRoot,
		RuleVersion: cand.RuleVersion, Proof: proof,
	})
	require.ErrorIs(t, err, types.ErrRoleKeyAppMismatch)

	// matching bound key still publishes when AssessmentCredential is provided
	params := e.k.GetParams(e.ctx)
	params.EntitlementAllowedAssessmentVersions = []string{types.DefaultAssessmentVersion}
	require.NoError(t, e.k.SetParams(e.ctx, params))
	signerPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x0a}, ed25519.SeedSize))
	e.k.SetAssessmentSigner(e.ctx, types.AssessmentSigner{
		KeyId: "assess-t1a", Pubkey: signerPriv.Public().(ed25519.PublicKey),
		Status: types.RoleKeyStatusActive, Version: 1, RegisteredHeight: 1,
	})

	candOK := sampleCandidateFields()
	cred := types.AssessmentCredential{
		CredentialVersion: types.DefaultAssessmentVersion,
		AppId:             candOK.AppID, PoolId: testPoolID, PavId: candOK.PavID,
		EvidenceRoot: candOK.EvidenceRoot, GasEvidenceRoot: candOK.GasEvidenceRoot,
		Amount: candOK.Amount, Denom: candOK.Denom, GasUsed: candOK.GasUsed,
		QualifiedTxCount: candOK.QualifiedTxCount, RuleVersion: candOK.RuleVersion,
		IssuedAt: "1700000000", ExpiresAt: "1893456000", SignerKeyId: "assess-t1a",
	}
	signAssessment(t, signerPriv, &cred)

	hashOK, err := wirev1.CandidateHash(candOK)
	require.NoError(t, err)
	recOK := map[string]string{
		"protocol_version": "1",
		"operation":        "PUBLISH",
		"role":             "publisher",
		"chain_id":         testChainID,
		"pool_id":          testPoolID,
		"key_id":           "bound-pub",
		"nonce":            "9102",
		"expires_at":       "1893456000",
		"entitlement_id":   candOK.EntitlementID,
		"app_id":           candOK.AppID,
		"candidate_hash":   hashOK,
		"rule_version":     candOK.RuleVersion,
	}
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: candOK.EntitlementID, AppId: candOK.AppID, Epoch: candOK.Epoch,
		Amount: candOK.Amount, Denom: candOK.Denom, GasUsed: candOK.GasUsed,
		QualifiedTxCount: candOK.QualifiedTxCount, PavId: candOK.PavID,
		EvidenceRoot: candOK.EvidenceRoot, GasEvidenceRoot: candOK.GasEvidenceRoot,
		RuleVersion: candOK.RuleVersion, Proof: signProof(t, boundPK, recOK),
		Credential: cred,
	})
	require.NoError(t, err)

	// global key (pub-1 from setupP1) still works without credential in transition period
	_ = pubPriv
}
