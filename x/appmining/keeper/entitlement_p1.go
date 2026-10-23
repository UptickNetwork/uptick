package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func (k msgServer) requireEntitlementFlow(ctx sdk.Context, poolID string) (types.Pool, types.Params, error) {
	params := k.GetParams(ctx)
	if params.PauseState != nil && params.PauseState.EntitlementPaused {
		return types.Pool{}, params, types.ErrEntitlementPaused
	}
	pool, found := k.GetEntitlementPool(ctx, poolID)
	if !found {
		return types.Pool{}, params, types.ErrPoolNotFound
	}
	if pool.Paused {
		return types.Pool{}, params, types.ErrPoolPaused
	}
	modAddr := k.ak.GetModuleAddress(types.EntitlementPoolName)
	bal := k.bank.GetBalance(ctx, modAddr, pool.Denom)
	if !types.EntitlementFlowEnabled(pool, params, types.AmountString(bal.Amount)) {
		return types.Pool{}, params, types.ErrPoolNotEnabled
	}
	return pool, params, nil
}

func appIDBytesFromHex(appIDHex string) ([]byte, error) {
	b, err := hex.DecodeString(appIDHex)
	if err != nil || len(b) != types.AppIDLength {
		return nil, types.ErrInvalidAppID
	}
	return b, nil
}

func (k msgServer) PublishCandidate(goCtx context.Context, msg *types.MsgPublishCandidate) (*types.MsgPublishCandidateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.ChainId != ctx.ChainID() {
		return nil, types.ErrChainIDMismatch
	}
	pool, _, err := k.requireEntitlementFlow(ctx, msg.PoolId)
	if err != nil {
		return nil, err
	}
	if msg.Denom != pool.Denom {
		return nil, types.ErrInvalidDenom
	}
	appID, err := appIDBytesFromHex(msg.AppId)
	if err != nil {
		return nil, err
	}
	app, found := k.GetApplication(ctx, appID)
	if !found {
		return nil, types.ErrApplicationNotFound
	}
	if app.Status != types.ApplicationStatusActive {
		return nil, types.ErrApplicationNotActive
	}

	cand := wirev1.Candidate{
		SchemaVersion:    wirev1.CandidateSchemaVersion,
		EntitlementID:    msg.EntitlementId,
		AppID:            msg.AppId,
		Epoch:            msg.Epoch,
		Amount:           msg.Amount,
		Denom:            msg.Denom,
		GasUsed:          msg.GasUsed,
		QualifiedTxCount: msg.QualifiedTxCount,
		PavID:            msg.PavId,
		EvidenceRoot:     msg.EvidenceRoot,
		GasEvidenceRoot:  msg.GasEvidenceRoot,
		RuleVersion:      msg.RuleVersion,
	}
	hash, err := wirev1.CandidateHash(cand)
	if err != nil {
		return nil, types.ErrInvalidCommitment
	}

	now := ctx.BlockTime().Unix()
	if existing, ok := k.GetCandidate(ctx, msg.EntitlementId); ok {
		if existing.CandidateHash == hash {
			// idempotent same content
			return &types.MsgPublishCandidateResponse{
				CandidateHash:     existing.CandidateHash,
				ChallengeDeadline: existing.ChallengeDeadline,
			}, nil
		}
		return nil, types.ErrCandidateConflict
	}

	record := map[string]string{
		"protocol_version": "1",
		"operation":        "PUBLISH",
		"role":             "publisher",
		"chain_id":         msg.ChainId,
		"pool_id":          msg.PoolId,
		"key_id":           msg.Proof.KeyId,
		"nonce":            msg.Proof.Nonce,
		"expires_at":       msg.Proof.ExpiresAt,
		"entitlement_id":   msg.EntitlementId,
		"app_id":           msg.AppId,
		"candidate_hash":   hash,
		"rule_version":     msg.RuleVersion,
	}
	if err := k.verifyAndConsumeWireAuth(ctx, types.EntitlementRolePublisher, msg.Proof, record, now); err != nil {
		return nil, err
	}

	stored := types.Candidate{
		EntitlementId:     msg.EntitlementId,
		AppId:             msg.AppId,
		Epoch:             msg.Epoch,
		Amount:            msg.Amount,
		Denom:             msg.Denom,
		GasUsed:           msg.GasUsed,
		QualifiedTxCount:  msg.QualifiedTxCount,
		PavId:             msg.PavId,
		EvidenceRoot:      msg.EvidenceRoot,
		GasEvidenceRoot:   msg.GasEvidenceRoot,
		RuleVersion:       msg.RuleVersion,
		CandidateHash:     hash,
		PoolId:            msg.PoolId,
		PublishTxHash:     currentTxHashHex(ctx),
		PublishHeight:     ctx.BlockHeight(),
		PublishedAt:       now,
		ChallengeDeadline: now + types.ChallengeWindowSeconds,
		Publisher:         msg.Submitter,
		Status:            types.CandidateStatusPublished,
		MaxObservedTime:   now,
	}
	k.SetCandidate(ctx, stored)

	emitV1StateEvent(ctx, types.EventTypeCandidatePublished, "publish",
		msg.PoolId, msg.EntitlementId, msg.AppId,
		"", stored.Status.String(), msg.Amount, msg.Denom, msg.EvidenceRoot)
	return &types.MsgPublishCandidateResponse{
		CandidateHash:     hash,
		ChallengeDeadline: stored.ChallengeDeadline,
	}, nil
}

func entitlementChallengeID(poolID, entitlementID, challenger string, openedAt int64, evidenceType, evidenceDigest string) string {
	h := sha256.New()
	h.Write([]byte(poolID))
	h.Write([]byte{0})
	h.Write([]byte(entitlementID))
	h.Write([]byte{0})
	h.Write([]byte(challenger))
	h.Write([]byte{0})
	h.Write([]byte(strconv.FormatInt(openedAt, 10)))
	h.Write([]byte{0})
	h.Write([]byte(evidenceType))
	h.Write([]byte{0})
	h.Write([]byte(evidenceDigest))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}

func (k msgServer) OpenChallenge(goCtx context.Context, msg *types.MsgOpenChallenge) (*types.MsgOpenChallengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cand, found := k.GetCandidate(ctx, msg.EntitlementId)
	if !found {
		return nil, types.ErrCandidateNotFound
	}
	if cand.Status != types.CandidateStatusPublished {
		return nil, types.ErrCandidateNotPublished
	}
	if _, _, err := k.requireEntitlementFlow(ctx, cand.PoolId); err != nil {
		return nil, err
	}

	blockUnix := ctx.BlockTime().Unix()
	now := observedTime(blockUnix, cand.MaxObservedTime)
	if now >= cand.ChallengeDeadline {
		return nil, types.ErrChallengeWindowClosed
	}
	if _, ok := k.GetOpenEntitlementChallenge(ctx, msg.EntitlementId, msg.Challenger); ok {
		return nil, types.ErrDuplicateOpenChallenge
	}
	evWire, ok := types.EvidenceTypeWire(msg.EvidenceType)
	if !ok {
		return nil, types.ErrInvalidEvidenceType
	}
	ref := msg.ReferenceDigest
	if ref == "" {
		ref = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	id := entitlementChallengeID(cand.PoolId, msg.EntitlementId, msg.Challenger, now, evWire, msg.EvidenceDigest)
	chal := types.EntitlementChallenge{
		ChallengeId:     id,
		CandidateId:     msg.EntitlementId,
		Challenger:      msg.Challenger,
		EvidenceType:    msg.EvidenceType,
		EvidenceDigest:  msg.EvidenceDigest,
		ReferenceDigest: ref,
		Status:          types.EntitlementChallengeStatusOpen,
		OpenedAt:        now,
	}
	k.SetEntitlementChallenge(ctx, chal)
	k.SetOpenEntitlementChallenge(ctx, msg.EntitlementId, msg.Challenger, id)

	cand.MaxObservedTime = now
	k.SetCandidate(ctx, cand)

	emitV1StateEvent(ctx, types.EventTypeEntitlementChallengeOpened, "open_challenge",
		cand.PoolId, msg.EntitlementId, cand.AppId,
		"", chal.Status.String(), cand.Amount, cand.Denom, msg.EvidenceDigest,
		sdk.NewAttribute(types.AttributeKeyChallengeID, id),
		sdk.NewAttribute(types.AttributeKeyChallenger, msg.Challenger),
	)
	return &types.MsgOpenChallengeResponse{ChallengeId: id}, nil
}

func (k msgServer) ResolveEntitlementChallenge(goCtx context.Context, msg *types.MsgResolveEntitlementChallenge) (*types.MsgResolveEntitlementChallengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.ChainId != ctx.ChainID() {
		return nil, types.ErrChainIDMismatch
	}
	chal, found := k.GetEntitlementChallenge(ctx, msg.ChallengeId)
	if !found {
		return nil, types.ErrEntitlementChallengeNotFound
	}
	if chal.Status != types.EntitlementChallengeStatusOpen {
		return nil, types.ErrEntitlementChallengeNotOpen
	}
	cand, found := k.GetCandidate(ctx, chal.CandidateId)
	if !found {
		return nil, types.ErrCandidateNotFound
	}
	if msg.PoolId != cand.PoolId {
		return nil, types.ErrPoolMismatch
	}
	if _, _, err := k.requireEntitlementFlow(ctx, cand.PoolId); err != nil {
		return nil, err
	}

	now := observedTime(ctx.BlockTime().Unix(), cand.MaxObservedTime)
	record := map[string]string{
		"protocol_version": "1",
		"operation":        "RESOLVE",
		"role":             "reviewer",
		"chain_id":         msg.ChainId,
		"pool_id":          msg.PoolId,
		"key_id":           msg.Proof.KeyId,
		"nonce":            msg.Proof.Nonce,
		"expires_at":       msg.Proof.ExpiresAt,
		"challenge_id":     msg.ChallengeId,
		"candidate_id":     chal.CandidateId,
		"outcome":          msg.Outcome,
		"rationale_digest": msg.RationaleDigest,
	}
	if err := k.verifyAndConsumeWireAuth(ctx, types.EntitlementRoleReviewer, msg.Proof, record, now); err != nil {
		return nil, err
	}

	chal.Reviewer = msg.Proof.KeyId
	chal.ResolutionDigest = msg.RationaleDigest
	chal.ResolvedAt = now
	prevCand := cand.Status.String()
	switch msg.Outcome {
	case "UPHELD":
		chal.Status = types.EntitlementChallengeStatusUpheld
		cand.Status = types.CandidateStatusRejected
	case "DISMISSED":
		chal.Status = types.EntitlementChallengeStatusDismissed
	default:
		return nil, types.ErrInvalidOutcome
	}
	k.SetEntitlementChallenge(ctx, chal)
	k.DeleteOpenEntitlementChallenge(ctx, chal.CandidateId, chal.Challenger)
	cand.MaxObservedTime = now
	k.SetCandidate(ctx, cand)

	emitV1StateEvent(ctx, types.EventTypeEntitlementChallengeResolved, "resolve",
		cand.PoolId, chal.CandidateId, cand.AppId,
		prevCand, cand.Status.String(), cand.Amount, cand.Denom, msg.RationaleDigest,
		sdk.NewAttribute(types.AttributeKeyChallengeID, chal.ChallengeId),
		sdk.NewAttribute(types.AttributeKeyStatus, chal.Status.String()),
	)
	return &types.MsgResolveEntitlementChallengeResponse{}, nil
}
