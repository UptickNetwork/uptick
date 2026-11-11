package keeper_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// Why a reviewer may hold an EVM-address key: adjudicating a challenge needs a
// signature over the RESOLVE preimage, and the party who adjudicates should be
// able to produce it from the wallet they already use (Keplr / MetaMask, which
// sign that preimage as EIP-191 personal_sign) instead of exporting an Ed25519
// secret onto an operations host. Registration is gated in
// types.RoleKeyTypeAllowed and verification in
// types.RoleKeyTypeOperationAllowed; this test walks the real keeper path to
// prove there is no third gate hiding in between.
func TestSecp256k1ReviewerResolve(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	pubRec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "9101", "expires_at": wireExp(e.ctx),
		"entitlement_id": c.EntitlementID, "app_id": c.AppID,
		"candidate_hash": hash, "rule_version": c.RuleVersion,
	}
	_, err = e.server.PublishCandidate(e.ctx, &types.MsgPublishCandidate{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch, Amount: c.Amount,
		Denom: c.Denom, GasUsed: c.GasUsed, QualifiedTxCount: c.QualifiedTxCount,
		PavId: c.PavID, EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
		RuleVersion: c.RuleVersion, Proof: signProof(t, pubPriv, pubRec),
	})
	require.NoError(t, err)

	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeReceiptQbtMismatch,
		EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	require.NoError(t, err)

	priv, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(priv.PublicKey)

	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(),
		Role:      types.EntitlementRoleReviewer,
		KeyId:     "mm-reviewer-1",
		Pubkey:    addr.Bytes(),
		Version:   1,
		KeyType:   types.RoleKeyTypeSecp256k1EvmAddress,
	})
	require.NoError(t, err)

	resolveRec := map[string]string{
		"protocol_version": "1", "operation": "RESOLVE", "role": "reviewer",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "mm-reviewer-1",
		"nonce": "9102", "expires_at": wireExp(e.ctx),
		"challenge_id": openRes.ChallengeId, "candidate_id": c.EntitlementID,
		"outcome":          "DISMISSED",
		"rationale_digest": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}

	// A signature from a different EVM key must not pass: the verifier recovers
	// the signer and compares it with the registered address, so this is the
	// case that distinguishes "type allowed" from "any EVM key allowed".
	other, err := crypto.GenerateKey()
	require.NoError(t, err)
	_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: openRes.ChallengeId, Outcome: "DISMISSED",
		RationaleDigest: resolveRec["rationale_digest"],
		Proof:           ethSignWire(t, other, resolveRec),
	})
	require.ErrorIs(t, err, types.ErrWireProofRecoverMismatch)

	_, err = e.server.ResolveEntitlementChallenge(e.ctx, &types.MsgResolveEntitlementChallenge{
		Submitter: e.sub.String(), ChainId: testChainID, PoolId: testPoolID,
		ChallengeId: openRes.ChallengeId, Outcome: "DISMISSED",
		RationaleDigest: resolveRec["rationale_digest"],
		Proof:           ethSignWire(t, priv, resolveRec),
	})
	require.NoError(t, err)

	chal, found := e.k.GetEntitlementChallenge(e.ctx, openRes.ChallengeId)
	require.True(t, found)
	require.Equal(t, types.EntitlementChallengeStatusDismissed, chal.Status)
	require.Equal(t, "mm-reviewer-1", chal.Reviewer)
}
