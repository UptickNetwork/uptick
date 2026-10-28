package keeper

import (
	"bytes"
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
	if !k.IsEntitlementFlowEnabled(ctx, pool) {
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
	pool, params, err := k.requireEntitlementFlow(ctx, msg.PoolId)
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

	// R1: app-bound ACTIVE publisher keys must match msg.app_id (64-hex ↔ 32B);
	// global keys (empty AppId) remain allowed during the transition period.
	publisherAppBound := false
	if rk, found := k.GetRoleKey(ctx, types.EntitlementRolePublisher, msg.Proof.KeyId); found {
		if len(rk.AppId) > 0 {
			publisherAppBound = true
			if !bytes.Equal(rk.AppId, appID) {
				return nil, types.ErrRoleKeyAppMismatch
			}
		}
	}

	credentialID, err := k.requirePublishCredential(ctx, params, msg, publisherAppBound, now)
	if err != nil {
		return nil, err
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
		CredentialId:      credentialID,
	}
	k.SetCandidate(ctx, stored)
	if credentialID != "" {
		k.SetCredentialIndex(ctx, credentialID, msg.EntitlementId)
	}

	emitV1StateEvent(ctx, types.EventTypeCandidatePublished, "publish",
		msg.PoolId, msg.EntitlementId, msg.AppId,
		"", stored.Status.String(), msg.Amount, msg.Denom, msg.EvidenceRoot,
		sdk.NewAttribute(types.AttributeKeyCredentialID, credentialID))
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
	pool, params, err := k.requireEntitlementFlow(ctx, cand.PoolId)
	if err != nil {
		return nil, err
	}

	access := params.EffectiveChallengeOpenAccess()
	switch access {
	case types.ChallengeOpenAccessNone:
		// Emergency switch: public open disabled (UGE R4).
		return nil, types.ErrUnauthorized
	case types.ChallengeOpenAccessOpen, types.ChallengeOpenAccessDeposit:
	default:
		return nil, types.ErrUnauthorized
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

	depositAmt, depositDenom := "", ""
	if access == types.ChallengeOpenAccessDeposit {
		depositAmt = params.ChallengeDepositAmount
		depositDenom = params.ChallengeDepositDenom
		if depositAmt == "" || types.IsZeroAmount(depositAmt) || depositDenom == "" {
			return nil, types.ErrInvalidAmount
		}
		if depositDenom != pool.Denom {
			// Deposit must match the candidate pool denom so confiscation can
			// credit that pool without cross-denom bookkeeping.
			return nil, types.ErrInvalidDenom
		}
		challenger, err := sdk.AccAddressFromBech32(msg.Challenger)
		if err != nil {
			return nil, err
		}
		amt, err := types.ParseAmount(depositAmt)
		if err != nil {
			return nil, err
		}
		coin := sdk.NewCoin(depositDenom, amt)
		if err := k.bank.SendCoinsFromAccountToModule(ctx, challenger, types.ChallengeEscrowName, sdk.NewCoins(coin)); err != nil {
			return nil, err
		}
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
		DepositAmount:   depositAmt,
		DepositDenom:    depositDenom,
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
		sdk.NewAttribute(types.AttributeKeyBondFlow, access),
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
	bondFlow := ""
	switch msg.Outcome {
	case "UPHELD":
		chal.Status = types.EntitlementChallengeStatusUpheld
		cand.Status = types.CandidateStatusRejected
		if err := k.releaseEntitlementChallengeDeposit(ctx, chal, true, cand.PoolId); err != nil {
			return nil, err
		}
		if chal.DepositAmount != "" {
			bondFlow = "refund_in"
		}
	case "DISMISSED":
		chal.Status = types.EntitlementChallengeStatusDismissed
		if err := k.releaseEntitlementChallengeDeposit(ctx, chal, false, cand.PoolId); err != nil {
			return nil, err
		}
		if chal.DepositAmount != "" {
			bondFlow = "slash_in"
		}
	default:
		return nil, types.ErrInvalidOutcome
	}
	k.SetEntitlementChallenge(ctx, chal)
	k.DeleteOpenEntitlementChallenge(ctx, chal.CandidateId, chal.Challenger)
	cand.MaxObservedTime = now
	k.SetCandidate(ctx, cand)

	extra := []sdk.Attribute{
		sdk.NewAttribute(types.AttributeKeyChallengeID, chal.ChallengeId),
		sdk.NewAttribute(types.AttributeKeyStatus, chal.Status.String()),
	}
	if bondFlow != "" {
		extra = append(extra, sdk.NewAttribute(types.AttributeKeyBondFlow, bondFlow))
	}
	emitV1StateEvent(ctx, types.EventTypeEntitlementChallengeResolved, "resolve",
		cand.PoolId, chal.CandidateId, cand.AppId,
		prevCand, cand.Status.String(), cand.Amount, cand.Denom, msg.RationaleDigest,
		extra...,
	)
	return &types.MsgResolveEntitlementChallengeResponse{}, nil
}

// releaseEntitlementChallengeDeposit settles a V1 deposit after resolve.
// refund=true (UPHELD) returns coins to the challenger; false (DISMISSED)
// moves them into the entitlement pool and credits funded+available.
func (k Keeper) releaseEntitlementChallengeDeposit(ctx sdk.Context, chal types.EntitlementChallenge, refund bool, poolID string) error {
	if chal.DepositAmount == "" || chal.DepositDenom == "" {
		return nil
	}
	amt, err := types.ParseAmount(chal.DepositAmount)
	if err != nil {
		return err
	}
	coin := sdk.NewCoin(chal.DepositDenom, amt)
	if refund {
		challenger, err := sdk.AccAddressFromBech32(chal.Challenger)
		if err != nil {
			return err
		}
		return k.bank.SendCoinsFromModuleToAccount(ctx, types.ChallengeEscrowName, challenger, sdk.NewCoins(coin))
	}
	if err := k.bank.SendCoinsFromModuleToModule(ctx, types.ChallengeEscrowName, types.EntitlementPoolName, sdk.NewCoins(coin)); err != nil {
		return err
	}
	pool, found := k.GetEntitlementPool(ctx, poolID)
	if !found {
		return types.ErrPoolNotFound
	}
	newFunded, err := types.AddAmountStrings(pool.Funded, chal.DepositAmount)
	if err != nil {
		return err
	}
	newAvailable, err := types.AddAmountStrings(pool.Available, chal.DepositAmount)
	if err != nil {
		return err
	}
	pool.Funded = newFunded
	pool.Available = newAvailable
	if err := types.ValidatePoolInvariant(pool); err != nil {
		return err
	}
	k.SetEntitlementPool(ctx, pool)
	return nil
}
