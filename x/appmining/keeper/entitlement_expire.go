package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// SettlementTriggerAutomatic is the submitter recorded when the EndBlocker —
// not a transaction — settled a challenge. Distinguishes "the window expired and
// the chain closed it" from "someone paid gas to close it" in events and
// queries, without implying any party decided the outcome.
const SettlementTriggerAutomatic = "endblocker"

// ExpireEntitlementChallenge settles an OPEN challenge whose resolution window
// elapsed without a reviewer decision (neutral timeout outcome, §"超时未裁决").
//
// Two deliberate design points:
//
//  1. Permissionless, no wire proof: the outcome is fixed (EXPIRED + deposit
//     refunded), so the submitter has nothing to choose and nothing to gain.
//  2. It does NOT call requireEntitlementFlow. The timeout path exists to
//     release a stuck deposit, so it must keep working while the pool or the
//     entitlement flow is paused — that is precisely when an operator needs it.
//     This is safe because no pool money moves: the deposit only returns to the
//     challenger that paid it.
//
// The reviewer keeps the exclusive right to decide until the sealed resolution
// window + governance grace elapses; resolving is not blocked afterwards, but a
// late resolve then finds the challenge already settled.
//
// The same settlement is also driven automatically by the module EndBlocker
// (SweepExpiredChallenges), so the timeout outcome does not depend on anyone
// submitting this message. This handler stays as the manual / auditable path.
func (k msgServer) ExpireEntitlementChallenge(goCtx context.Context, msg *types.MsgExpireEntitlementChallenge) (*types.MsgExpireEntitlementChallengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
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

	params := k.GetParams(ctx)
	now := observedTime(ctx.BlockTime().Unix(), cand.MaxObservedTime)
	deadline := types.EffectiveResolutionDeadline(cand, chal, params.EffectiveChallengeResolutionGraceSeconds())
	if now < deadline {
		return nil, sdkerrors.Wrapf(types.ErrChallengeNotExpired,
			"now %d < resolution deadline %d (sealed window %d + grace %d)",
			now, deadline, cand.ChallengeDeadline-cand.PublishedAt,
			params.EffectiveChallengeResolutionGraceSeconds())
	}

	digest, err := k.settleExpiredChallenge(ctx, chal, cand, now, msg.Submitter)
	if err != nil {
		return nil, err
	}
	return &types.MsgExpireEntitlementChallengeResponse{ResolutionDigest: digest}, nil
}

// settleExpiredChallenge applies the neutral timeout outcome to one OPEN
// challenge. Callers MUST have verified that now >= the resolution deadline.
//
// Shared by the Msg handler and the EndBlocker sweep so both paths are
// byte-for-byte identical: same digest, same refund, same event, same candidate
// update. Returns the deterministic timeout digest.
func (k Keeper) settleExpiredChallenge(ctx sdk.Context, chal types.EntitlementChallenge, cand types.Candidate, now int64, submitter string) (string, error) {
	// Neutral outcome: full refund, nobody is punished, candidate keeps
	// PUBLISHED so the publisher can still be approved (digest-bound).
	if err := k.releaseEntitlementChallengeDeposit(ctx, chal, true, cand.PoolId); err != nil {
		return "", err
	}

	prevCand := cand.Status.String()
	digest := types.ExpiredResolutionDigest(chal.ChallengeId)

	chal.Status = types.EntitlementChallengeStatusExpired
	chal.ResolvedAt = now
	chal.Reviewer = "" // nobody decided; the window did
	chal.ResolutionDigest = digest
	k.SetEntitlementChallenge(ctx, chal) // also drops the timeout bucket
	k.DeleteOpenEntitlementChallenge(ctx, chal.CandidateId, chal.Challenger)

	cand.MaxObservedTime = now
	k.SetCandidate(ctx, cand)

	extra := []sdk.Attribute{
		sdk.NewAttribute(types.AttributeKeyChallengeID, chal.ChallengeId),
		sdk.NewAttribute(types.AttributeKeyStatus, chal.Status.String()),
		sdk.NewAttribute(types.AttributeKeySubmitter, submitter),
	}
	if chal.DepositAmount != "" {
		extra = append(extra, sdk.NewAttribute(types.AttributeKeyBondFlow, "refund_in"))
	}
	emitV1StateEvent(ctx, types.EventTypeEntitlementChallengeExpired, "expire",
		cand.PoolId, chal.CandidateId, cand.AppId,
		prevCand, cand.Status.String(), cand.Amount, cand.Denom, digest,
		extra...,
	)
	return digest, nil
}
