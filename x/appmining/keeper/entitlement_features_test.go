package keeper_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/keeper"
	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func eventAttr(ev sdk.Event, key string) string {
	for _, a := range ev.Attributes {
		if string(a.Key) == key {
			return string(a.Value)
		}
	}
	return ""
}

func findEvent(events []sdk.Event, typ string) (sdk.Event, bool) {
	for _, ev := range events {
		if ev.Type == typ {
			return ev, true
		}
	}
	return sdk.Event{}, false
}

func TestFeature_AuthNonceReplayRejected(t *testing.T) {
	e, pubPriv, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))

	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1301"))
	require.NoError(t, err)

	// Second candidate; reuse approver nonce → ErrAuthReplay (before pool exhaustion).
	c2 := sampleCandidateFields()
	c2.Amount = "100"
	c2.PavID = "22222222222222222222222222222222222222222222222222222222222222aa"
	c2.EntitlementID = wirev1.EntitlementIDOf(c2)
	hash, err := wirev1.CandidateHash(c2)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "1302", "expires_at": "1893456000",
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
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c2, "1301"))
	require.ErrorIs(t, err, types.ErrAuthReplay)
}

func TestFeature_AuthExpiredRejected(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))

	msg := approveMsg(t, e, aprPriv, c, "1310")
	msg.Proof.ExpiresAt = "1" // far in the past
	// must re-sign with expired expires_at in the record
	rec := map[string]string{
		"protocol_version": "1", "operation": "APPROVE", "role": "approver",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "apr-1",
		"nonce": "1310", "expires_at": "1",
		"entitlement_id": c.EntitlementID, "app_id": c.AppID,
		"candidate_hash":            cand.CandidateHash,
		"challenge_resolution_hash": "",
		"amount":                    c.Amount, "denom": c.Denom, "rule_version": c.RuleVersion,
	}
	msg.Proof = signProof(t, aprPriv, rec)
	msg.ChallengeResolutionHash = ""
	_, err := e.server.ApproveEntitlement(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrAuthExpired)
}

func TestFeature_PoolPausedRejectsPublish(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	_, err := e.server.SetEntitlementPoolPaused(e.ctx, &types.MsgSetEntitlementPoolPaused{
		Authority: e.auth.String(), PoolId: testPoolID, Paused: true,
	})
	require.NoError(t, err)

	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "1320", "expires_at": "1893456000",
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
	require.ErrorIs(t, err, types.ErrPoolPaused)
}

func TestFeature_PausedRoleKeyRejectsApprove(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	_, err := e.server.SetRoleKeyStatus(e.ctx, &types.MsgSetRoleKeyStatus{
		Authority: e.auth.String(), Role: types.EntitlementRoleApprover,
		KeyId: "apr-1", Status: types.RoleKeyStatusPaused,
	})
	require.NoError(t, err)

	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err = e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1330"))
	require.ErrorIs(t, err, types.ErrRoleKeyInactive)
}

func TestFeature_FreezeOnce(t *testing.T) {
	e, riskPriv, _, _, c := setupP3(t)
	now := e.ctx.BlockTime().Unix()
	_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+3600, "1340"))
	require.NoError(t, err)
	// Already FROZEN → not claimable path (cannot freeze again operationally).
	_, err = e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+3700, "1341"))
	require.ErrorIs(t, err, types.ErrNotClaimable)

	// FreezeCount guard: CLAIMABLE with freeze_count already spent.
	e2, risk2, _, _, c2 := setupP3(t)
	ent, _ := e2.k.GetEntitlement(e2.ctx, c2.EntitlementID)
	ent.FreezeCount = 1
	e2.k.SetEntitlement(e2.ctx, ent)
	now2 := e2.ctx.BlockTime().Unix()
	_, err = e2.server.FreezeEntitlement(e2.ctx, freezeMsg(t, e2, risk2, c2, now2+3600, "1342"))
	require.ErrorIs(t, err, types.ErrFreezeOnce)
}

func TestFeature_OpenChallengeAfterDeadlineRejected(t *testing.T) {
	e, _, _, _, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.ChallengeDeadline, 0))
	_, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeConfirmedRefundOrReversal,
		EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	require.ErrorIs(t, err, types.ErrChallengeWindowClosed)
}

func TestFeature_ChallengeOpenedEventIndexKey(t *testing.T) {
	// Doc 14 §4: Comet tx_search query entitlement_challenge_opened.entitlement_id
	e, pubPriv, _ := setupP1(t)
	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "1340", "expires_at": "1893456000",
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

	e.ctx = e.ctx.WithEventManager(sdk.NewEventManager())
	openRes, err := e.server.OpenChallenge(e.ctx, &types.MsgOpenChallenge{
		Challenger: e.payer.String(), EntitlementId: c.EntitlementID,
		EvidenceType:   types.EvidenceTypeReceiptQbtMismatch,
		EvidenceDigest: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	})
	require.NoError(t, err)

	ev, ok := findEvent(e.ctx.EventManager().Events(), types.EventTypeEntitlementChallengeOpened)
	require.True(t, ok)
	require.Equal(t, c.EntitlementID, eventAttr(ev, types.AttributeKeyEntitlementID))
	require.Equal(t, openRes.ChallengeId, eventAttr(ev, types.AttributeKeyChallengeID))
	require.Equal(t, "open_challenge", eventAttr(ev, types.AttributeKeyAction))

	// Empty index-events → index all (app.toml default). Explicit allow-list also works.
	abciEv := sdk.Events{ev}.ToABCIEvents()
	indexedAll := sdk.MarkEventsToIndex(abciEv, nil)
	require.True(t, indexedAll[0].Attributes[0].Index)
	allow := map[string]struct{}{
		types.EventTypeEntitlementChallengeOpened + "." + types.AttributeKeyEntitlementID: {},
	}
	indexedSel := sdk.MarkEventsToIndex(abciEv, allow)
	var entitlementIndexed bool
	for _, a := range indexedSel[0].Attributes {
		if a.Key == types.AttributeKeyEntitlementID {
			entitlementIndexed = a.Index
		}
	}
	require.True(t, entitlementIndexed)
}

func TestFeature_ApproveEventSchemaAndTxHash(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	txBytes := []byte("feature-approve-tx-bytes")
	wantHash := hex.EncodeToString(tmhash.Sum(txBytes))
	e.ctx = e.ctx.
		WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0)).
		WithTxBytes(txBytes).
		WithEventManager(sdk.NewEventManager())

	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1350"))
	require.NoError(t, err)

	ent, ok := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.True(t, ok)
	require.Equal(t, wantHash, ent.ApprovalTx)

	ev, ok := findEvent(e.ctx.EventManager().Events(), types.EventTypeEntitlementApproved)
	require.True(t, ok)
	require.Equal(t, testChainID, eventAttr(ev, types.AttributeKeyChainID))
	require.Equal(t, "approve", eventAttr(ev, types.AttributeKeyAction))
	require.Equal(t, types.EntitlementStateClaimable.String(), eventAttr(ev, types.AttributeKeyNewState))
	require.Equal(t, wantHash, eventAttr(ev, types.AttributeKeyTxHash))
	require.Equal(t, c.Amount, eventAttr(ev, types.AttributeKeyAmount))
	require.Equal(t, testPoolID, eventAttr(ev, types.AttributeKeyPoolID))
	require.Equal(t, "false", eventAttr(ev, types.AttributeKeyResolutionBound))
}

func TestFeature_PublishStoresTxHash(t *testing.T) {
	e, pubPriv, _ := setupP1(t)
	c := sampleCandidateFields()
	hash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	txBytes := []byte("feature-publish-tx")
	wantHash := hex.EncodeToString(tmhash.Sum(txBytes))
	e.ctx = e.ctx.WithTxBytes(txBytes)

	rec := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": testChainID, "pool_id": testPoolID, "key_id": "pub-1",
		"nonce": "1360", "expires_at": "1893456000",
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
	got, ok := e.k.GetCandidate(e.ctx, c.EntitlementID)
	require.True(t, ok)
	require.Equal(t, wantHash, got.PublishTxHash)
}

func TestFeature_EntitlementPoolInvariant(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "1370"))
	require.NoError(t, err)

	msg, broken := keeper.EntitlementPoolInvariant(e.k)(e.ctx)
	require.False(t, broken, msg)

	pool, _ := e.k.GetEntitlementPool(e.ctx, testPoolID)
	pool.Available = "1" // break funded = available+reserved+paid
	e.k.SetEntitlementPool(e.ctx, pool)
	_, broken = keeper.EntitlementPoolInvariant(e.k)(e.ctx)
	require.True(t, broken)
}

func TestFeature_V1Queries(t *testing.T) {
	e, riskPriv, _, aprPriv, c := setupP3(t)

	qPool, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: testPoolID})
	require.NoError(t, err)
	require.True(t, qPool.EntitlementFlowEnabled)
	require.Equal(t, "31500", qPool.Pool.Reserved)

	qCand, err := e.k.Candidate(e.ctx, &types.QueryCandidateRequest{EntitlementId: c.EntitlementID})
	require.NoError(t, err)
	require.Equal(t, types.CandidateStatusPublished, qCand.Candidate.Status)

	qEnt, err := e.k.Entitlement(e.ctx, &types.QueryEntitlementRequest{EntitlementId: c.EntitlementID})
	require.NoError(t, err)
	require.Equal(t, types.EntitlementStateClaimable, qEnt.Entitlement.State)

	qByApp, err := e.k.EntitlementsByApp(e.ctx, &types.QueryEntitlementsByAppRequest{AppId: c.AppID})
	require.NoError(t, err)
	require.Len(t, qByApp.Entitlements, 1)

	qRoles, err := e.k.Roles(e.ctx, &types.QueryRolesRequest{Role: types.EntitlementRoleRisk})
	require.NoError(t, err)
	require.NotEmpty(t, qRoles.Keys)

	now := e.ctx.BlockTime().Unix()
	_, err = e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+3600, "1380"))
	require.NoError(t, err)
	_ = aprPriv
}

func TestFeature_RevokedRoleKeyCannotReactivate(t *testing.T) {
	e := setupTest(t)
	pk := make([]byte, 32)
	for i := range pk {
		pk[i] = byte(i + 1)
	}
	_, err := e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "pub-x", Pubkey: pk, Version: 1,
	})
	require.NoError(t, err)
	_, err = e.server.SetRoleKeyStatus(e.ctx, &types.MsgSetRoleKeyStatus{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "pub-x", Status: types.RoleKeyStatusRevoked,
	})
	require.NoError(t, err)
	_, err = e.server.SetRoleKeyStatus(e.ctx, &types.MsgSetRoleKeyStatus{
		Authority: e.auth.String(), Role: types.EntitlementRolePublisher,
		KeyId: "pub-x", Status: types.RoleKeyStatusActive,
	})
	require.Error(t, err)
}

func TestFeature_E6_FreezeExpiryUsesWatermark(t *testing.T) {
	e, riskPriv, _, _, c := setupP3(t)
	now := e.ctx.BlockTime().Unix()
	until := now + 3600
	_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, until, "1390"))
	require.NoError(t, err)

	// Observe past freeze_until then roll clock back — Claim still allowed via watermark.
	ent, _ := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	ent.MaxObservedTime = until
	e.k.SetEntitlement(e.ctx, ent)
	e.ctx = e.ctx.WithBlockTime(time.Unix(now+1, 0))

	_, err = e.server.ClaimEntitlement(e.ctx, &types.MsgClaimEntitlement{
		Owner: e.owner.String(), EntitlementId: c.EntitlementID,
	})
	require.NoError(t, err)
	got, _ := e.k.GetEntitlement(e.ctx, c.EntitlementID)
	require.Equal(t, types.EntitlementStateClaimed, got.State)
}

func TestFeature_RevokeRequiresBothProofs(t *testing.T) {
	e, riskPriv, auditPriv, _, c := setupP3(t)
	now := e.ctx.BlockTime().Unix()
	_, err := e.server.FreezeEntitlement(e.ctx, freezeMsg(t, e, riskPriv, c, now+7200, "1400"))
	require.NoError(t, err)

	msg := revokeMsg(t, e, riskPriv, auditPriv, c, "1401", "1402")
	msg.AuditProof = types.WireProof{} // empty audit
	_, err = e.server.RevokeEntitlement(e.ctx, msg)
	require.Error(t, err)
}
