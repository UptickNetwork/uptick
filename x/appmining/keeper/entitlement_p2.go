package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func (k msgServer) ApproveEntitlement(goCtx context.Context, msg *types.MsgApproveEntitlement) (*types.MsgApproveEntitlementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.ChainId != ctx.ChainID() {
		return nil, types.ErrChainIDMismatch
	}
	if _, found := k.GetEntitlement(ctx, msg.EntitlementId); found {
		return nil, types.ErrEntitlementExists
	}
	cand, found := k.GetCandidate(ctx, msg.EntitlementId)
	if !found {
		return nil, types.ErrCandidateNotFound
	}
	if cand.Status != types.CandidateStatusPublished {
		return nil, types.ErrCandidateNotPublished
	}
	if msg.PoolId != cand.PoolId {
		return nil, types.ErrPoolMismatch
	}
	pool, params, err := k.requireEntitlementFlow(ctx, cand.PoolId)
	if err != nil {
		return nil, err
	}
	// F-1 (§3): the candidate rule_version must be explicitly allowed by
	// governance params; an empty allowlist denies every approval.
	if !params.RuleVersionAllowed(cand.RuleVersion) {
		return nil, types.ErrRuleVersionNotAllowed
	}

	now := observedTime(ctx.BlockTime().Unix(), cand.MaxObservedTime)
	// Challenge window must have elapsed: now >= published_at + 604800
	if now < cand.PublishedAt+types.ChallengeWindowSeconds {
		return nil, types.ErrChallengeWindow
	}
	if k.HasOpenOrUpheldEntitlementChallenge(ctx, msg.EntitlementId) {
		return nil, types.ErrOpenOrUpheldChallenge
	}

	appID, err := appIDBytesFromHex(cand.AppId)
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

	cmp, err := types.CompareAmountStrings(pool.Available, cand.Amount)
	if err != nil {
		return nil, err
	}
	if cmp < 0 {
		return nil, types.ErrInsufficientAvailable
	}

	record := map[string]string{
		"protocol_version":          "1",
		"operation":                 "APPROVE",
		"role":                      "approver",
		"chain_id":                  msg.ChainId,
		"pool_id":                   msg.PoolId,
		"key_id":                    msg.Proof.KeyId,
		"nonce":                     msg.Proof.Nonce,
		"expires_at":                msg.Proof.ExpiresAt,
		"entitlement_id":            msg.EntitlementId,
		"app_id":                    cand.AppId,
		"candidate_hash":            cand.CandidateHash,
		"challenge_resolution_hash": msg.ChallengeResolutionHash,
		"amount":                    cand.Amount,
		"denom":                     cand.Denom,
		"rule_version":              cand.RuleVersion,
	}
	if err := k.verifyAndConsumeWireAuth(ctx, types.EntitlementRoleApprover, msg.Proof, record, now); err != nil {
		return nil, err
	}

	// UGE DEC-01 scheme C — mutually exclusive approve paths:
	//  - non-empty hash → must bind a DISMISSED ResolutionDigest (else 78)
	//  - empty hash → window elapsed (already checked) and zero challenges (else 71)
	resolutionBound := false
	if msg.ChallengeResolutionHash != "" {
		if !k.HasDismissedEntitlementChallengeResolution(ctx, msg.EntitlementId, msg.ChallengeResolutionHash) {
			return nil, types.ErrResolutionUnbound
		}
		resolutionBound = true
	} else if k.HasAnyEntitlementChallenge(ctx, msg.EntitlementId) {
		return nil, types.ErrOpenOrUpheldChallenge
	}

	newAvailable, err := types.SubAmountStrings(pool.Available, cand.Amount)
	if err != nil {
		return nil, err
	}
	newReserved, err := types.AddAmountStrings(pool.Reserved, cand.Amount)
	if err != nil {
		return nil, err
	}
	pool.Available = newAvailable
	pool.Reserved = newReserved
	if err := types.ValidatePoolInvariant(pool); err != nil {
		return nil, err
	}
	k.SetEntitlementPool(ctx, pool)

	ent := types.Entitlement{
		EntitlementId:    msg.EntitlementId,
		AppId:            cand.AppId,
		OwnerAtLastQuery: app.Owner,
		Amount:           cand.Amount,
		Denom:            cand.Denom,
		State:            types.EntitlementStateClaimable,
		FreezeUntil:      0,
		FreezeCount:      0,
		ApprovalTx:       currentTxHashHex(ctx),
		EvidenceRoot:     cand.EvidenceRoot,
		RuleVersion:      cand.RuleVersion,
		PoolId:           cand.PoolId,
		ApprovedAt:       now,
		MaxObservedTime:  now,
	}
	k.SetEntitlement(ctx, ent)

	cand.MaxObservedTime = now
	k.SetCandidate(ctx, cand)

	boundAttr := "false"
	if resolutionBound {
		boundAttr = "true"
	}
	emitV1StateEvent(ctx, types.EventTypeEntitlementApproved, "approve",
		cand.PoolId, msg.EntitlementId, cand.AppId,
		cand.Status.String(), ent.State.String(), cand.Amount, cand.Denom, msg.ChallengeResolutionHash,
		sdk.NewAttribute(types.AttributeKeyAvailable, pool.Available),
		sdk.NewAttribute(types.AttributeKeyResolutionBound, boundAttr),
	)
	return &types.MsgApproveEntitlementResponse{}, nil
}

func (k msgServer) ClaimEntitlement(goCtx context.Context, msg *types.MsgClaimEntitlement) (*types.MsgClaimEntitlementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ent, found := k.GetEntitlement(ctx, msg.EntitlementId)
	if !found {
		return nil, types.ErrEntitlementNotFound
	}
	if ent.State == types.EntitlementStateClaimed {
		return nil, types.ErrAlreadyClaimed
	}
	if ent.State == types.EntitlementStateRevoked {
		return nil, types.ErrNotClaimable
	}

	pool, _, err := k.requireEntitlementFlow(ctx, ent.PoolId)
	if err != nil {
		return nil, err
	}

	now := observedTime(ctx.BlockTime().Unix(), ent.MaxObservedTime)
	switch ent.State {
	case types.EntitlementStateClaimable:
		// ok
	case types.EntitlementStateFrozen:
		if now < ent.FreezeUntil {
			return nil, types.ErrNotClaimable
		}
		// freeze expired → treat as claimable
	default:
		return nil, types.ErrNotClaimable
	}

	appID, err := appIDBytesFromHex(ent.AppId)
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
	if msg.Owner != app.Owner {
		return nil, types.ErrBeneficiaryMustBeOwner
	}

	amt, err := types.ParseAmount(ent.Amount)
	if err != nil {
		return nil, err
	}
	ownerAddr, err := sdk.AccAddressFromBech32(app.Owner)
	if err != nil {
		return nil, err
	}
	coins := sdk.NewCoins(sdk.NewCoin(ent.Denom, amt))
	if err := k.bank.SendCoinsFromModuleToAccount(ctx, types.EntitlementPoolName, ownerAddr, coins); err != nil {
		return nil, sdkerrors.Wrap(err, "claim transfer")
	}

	newReserved, err := types.SubAmountStrings(pool.Reserved, ent.Amount)
	if err != nil {
		return nil, err
	}
	newPaid, err := types.AddAmountStrings(pool.Paid, ent.Amount)
	if err != nil {
		return nil, err
	}
	pool.Reserved = newReserved
	pool.Paid = newPaid
	if err := types.ValidatePoolInvariant(pool); err != nil {
		return nil, err
	}
	k.SetEntitlementPool(ctx, pool)

	prev := ent.State.String()
	ent.State = types.EntitlementStateClaimed
	ent.OwnerAtLastQuery = app.Owner
	ent.ClaimTx = currentTxHashHex(ctx)
	ent.MaxObservedTime = now
	k.SetEntitlement(ctx, ent)

	emitV1StateEvent(ctx, types.EventTypeEntitlementClaimed, "claim",
		ent.PoolId, ent.EntitlementId, ent.AppId,
		prev, ent.State.String(), ent.Amount, ent.Denom, "",
		sdk.NewAttribute(types.AttributeKeyOwner, app.Owner),
	)
	return &types.MsgClaimEntitlementResponse{Amount: ent.Amount, Denom: ent.Denom}, nil
}
