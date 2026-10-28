package keeper_test

import (
	"crypto/ecdsa"
	"encoding/base64"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func ethSignWire(t *testing.T, priv *ecdsa.PrivateKey, record map[string]string) types.WireProof {
	t.Helper()
	raw, err := wirev1.AuthorizationBytes(record)
	require.NoError(t, err)
	hash := accounts.TextHash(raw)
	sig, err := crypto.Sign(hash, priv)
	require.NoError(t, err)
	sig[64] += 27
	return types.WireProof{
		KeyId:           record["key_id"],
		Nonce:           record["nonce"],
		ExpiresAt:       record["expires_at"],
		SignatureBase64: base64.StdEncoding.EncodeToString(sig),
	}
}

func TestDEC10_Secp256k1PublisherPublish(t *testing.T) {
	e, _, _ := setupP1(t)

	priv, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(priv.PublicKey)

	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(),
		Role:      types.EntitlementRolePublisher,
		KeyId:     "mm-1",
		Pubkey:    addr.Bytes(),
		Version:   1,
		KeyType:   types.RoleKeyTypeSecp256k1EvmAddress,
	})
	require.NoError(t, err)

	// non-publisher cannot register type 1
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(),
		Role:      types.EntitlementRoleReviewer,
		KeyId:     "mm-rev",
		Pubkey:    addr.Bytes(),
		Version:   1,
		KeyType:   types.RoleKeyTypeSecp256k1EvmAddress,
	})
	require.ErrorIs(t, err, types.ErrRoleKeyTypeUnsupported)

	cand := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(cand)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "mm-1",
		"nonce": "9601", "expires_at": "1893456000",
		"entitlement_id": cand.EntitlementID, "app_id": cand.AppID,
		"candidate_hash": hash, "rule_version": cand.RuleVersion,
	}
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: cand.EntitlementID, AppId: cand.AppID, Epoch: cand.Epoch,
		Amount: cand.Amount, Denom: cand.Denom, GasUsed: cand.GasUsed,
		QualifiedTxCount: cand.QualifiedTxCount, PavId: cand.PavID,
		EvidenceRoot: cand.EvidenceRoot, GasEvidenceRoot: cand.GasEvidenceRoot,
		RuleVersion: cand.RuleVersion, Proof: ethSignWire(t, priv, rec),
	})
	require.NoError(t, err)

	// wrong address key rejects
	other, err := crypto.GenerateKey()
	require.NoError(t, err)
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(),
		Role:      types.EntitlementRolePublisher,
		KeyId:     "mm-2",
		Pubkey:    crypto.PubkeyToAddress(other.PublicKey).Bytes(),
		Version:   1,
		KeyType:   types.RoleKeyTypeSecp256k1EvmAddress,
	})
	require.NoError(t, err)
	cand2 := sampleCandidateFields()
	cand2.Epoch = "2"
	cand2.EntitlementID = wirev1.EntitlementIDOf(cand2)
	hash2, err := wirev1.CandidateHash(cand2)
	require.NoError(t, err)
	rec2 := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "mm-2",
		"nonce": "9602", "expires_at": "1893456000",
		"entitlement_id": cand2.EntitlementID, "app_id": cand2.AppID,
		"candidate_hash": hash2, "rule_version": cand2.RuleVersion,
	}
	// sign with first key against mm-2 registration
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: cand2.EntitlementID, AppId: cand2.AppID, Epoch: cand2.Epoch,
		Amount: cand2.Amount, Denom: cand2.Denom, GasUsed: cand2.GasUsed,
		QualifiedTxCount: cand2.QualifiedTxCount, PavId: cand2.PavID,
		EvidenceRoot: cand2.EvidenceRoot, GasEvidenceRoot: cand2.GasEvidenceRoot,
		RuleVersion: cand2.RuleVersion, Proof: ethSignWire(t, priv, rec2),
	})
	require.ErrorIs(t, err, types.ErrWireProofRecoverMismatch)
}
