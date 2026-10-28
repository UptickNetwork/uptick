package keeper

import (
	"context"
	"strconv"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return msgServer{Keeper: k}
}

func (k msgServer) RegisterApplication(goCtx context.Context, msg *types.MsgRegisterApplication) (*types.MsgRegisterApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.PauseState != nil && params.PauseState.RegistrationPaused {
		return nil, types.ErrRegistrationPaused
	}
	if msg.Authority != params.Authority {
		return nil, sdkerrors.Wrap(types.ErrUnauthorized, "not params authority")
	}
	if _, found := k.GetApplication(ctx, msg.AppId); found {
		return nil, types.ErrApplicationExists
	}
	if err := types.ValidateOptionalCommitment(msg.MetadataCommitment); err != nil {
		return nil, err
	}
	app := types.Application{
		AppId:                msg.AppId,
		Owner:                msg.Owner,
		AttestationSubmitter: msg.AttestationSubmitter,
		GasPayers:            nil,
		Status:               types.ApplicationStatusActive,
		MetadataCommitment:   msg.MetadataCommitment,
	}
	k.SetApplication(ctx, app)
	ctx.EventManager().EmitEvent(eventWithAppID(
		types.EventTypeApplicationRegistered, msg.AppId,
		sdk.NewAttribute(types.AttributeKeyOwner, msg.Owner),
		sdk.NewAttribute(types.AttributeKeySubmitter, msg.AttestationSubmitter),
	))
	// R2: optional embedded publisher key bound to this app in the same tx.
	if len(msg.PublisherPubkey) > 0 {
		if err := k.persistActiveRoleKey(
			ctx, msg.Authority, types.EntitlementRolePublisher,
			msg.PublisherKeyId, msg.PublisherPubkey, 1, msg.AppId,
			types.RoleKeyTypeEd25519,
		); err != nil {
			return nil, err
		}
	}
	return &types.MsgRegisterApplicationResponse{}, nil
}

func (k msgServer) SetApplicationStatus(goCtx context.Context, msg *types.MsgSetApplicationStatus) (*types.MsgSetApplicationStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, sdkerrors.Wrap(types.ErrUnauthorized, "not params authority")
	}
	app, found := k.GetApplication(ctx, msg.AppId)
	if !found {
		return nil, types.ErrApplicationNotFound
	}
	if err := types.CanSetApplicationStatus(app.Status, msg.Status); err != nil {
		return nil, err
	}
	old := app.Status
	app.Status = msg.Status
	k.SetApplication(ctx, app)
	ctx.EventManager().EmitEvent(eventWithAppID(
		types.EventTypeApplicationStatusChanged, msg.AppId,
		sdk.NewAttribute(types.AttributeKeyOldStatus, old.String()),
		sdk.NewAttribute(types.AttributeKeyNewStatus, msg.Status.String()),
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
	))
	return &types.MsgSetApplicationStatusResponse{}, nil
}

func (k msgServer) SubmitAttestation(goCtx context.Context, msg *types.MsgSubmitAttestation) (*types.MsgSubmitAttestationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.PauseState != nil && params.PauseState.AttestationPaused {
		return nil, types.ErrAttestationPaused
	}
	att := msg.Attestation
	app, found := k.GetApplication(ctx, att.AppId)
	if !found {
		return nil, types.ErrApplicationNotFound
	}
	if msg.Submitter != app.AttestationSubmitter {
		return nil, types.ErrUnauthorized
	}
	if !types.ProfileAllowed(params, att.Profile, att.ProfileVersion) {
		return nil, types.ErrUnsupportedProfile
	}
	budget, ok := k.GetEpochBudget(ctx, att.Epoch)
	if !ok {
		return nil, types.ErrInvalidEpoch
	}
	if budget.Closed {
		return nil, types.ErrInvalidEpoch
	}
	existing, exists := k.GetAttestation(ctx, att.AppId, att.Epoch)
	var existingPtr *types.AttestationCommitment
	if exists {
		existingPtr = &existing
	}
	if err := types.CanSubmitAttestation(app, existingPtr, att.Epoch); err != nil {
		return nil, err
	}
	// Soft reservation: pending non-terminal charges for this epoch + this
	// request must fit in remaining budget (first-writer wins; no proto field).
	charge, err := types.AttestationBudgetCharge(att)
	if err != nil {
		return nil, err
	}
	if err := k.ensureEpochBudgetRoom(ctx, budget, charge); err != nil {
		return nil, err
	}

	height := ctx.BlockHeight()
	att.Submitter = msg.Submitter
	att.SubmittedHeight = height
	att.ChallengeDeadlineHeight = height + int64(params.ChallengeWindowBlocks)
	att.Status = types.AttestationStatusChallengeWindow
	k.SetAttestation(ctx, att)

	ctx.EventManager().EmitEvent(eventWithAppID(
		types.EventTypeAttestationSubmitted, att.AppId,
		sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatUint(att.Epoch, 10)),
		sdk.NewAttribute(types.AttributeKeyProfile, att.Profile),
		sdk.NewAttribute(types.AttributeKeyProfileVersion, att.ProfileVersion),
		sdk.NewAttribute(types.AttributeKeyAttestationHash, hexBytes(att.AttestationHash)),
		sdk.NewAttribute(types.AttributeKeyDeadline, strconv.FormatInt(att.ChallengeDeadlineHeight, 10)),
		sdk.NewAttribute(types.AttributeKeySubmitter, att.Submitter),
	))
	return &types.MsgSubmitAttestationResponse{ChallengeDeadlineHeight: att.ChallengeDeadlineHeight}, nil
}

func (k msgServer) ChallengeAttestation(goCtx context.Context, msg *types.MsgChallengeAttestation) (*types.MsgChallengeAttestationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.PauseState != nil && params.PauseState.ChallengePaused {
		return nil, types.ErrChallengePaused
	}
	att, found := k.GetAttestation(ctx, msg.AppId, msg.Epoch)
	if !found {
		return nil, types.ErrAttestationNotFound
	}
	if err := types.CanChallenge(att, ctx.BlockHeight()); err != nil {
		return nil, err
	}

	challenger, err := sdk.AccAddressFromBech32(msg.Challenger)
	if err != nil {
		return nil, err
	}
	bondAmt, err := types.ParseAmount(params.ChallengeBondAmount)
	if err != nil {
		return nil, err
	}
	coin := sdk.NewCoin(params.ChallengeBondDenom, bondAmt)
	if err := k.bank.SendCoinsFromAccountToModule(ctx, challenger, types.ChallengeEscrowName, sdk.NewCoins(coin)); err != nil {
		return nil, sdkerrors.Wrap(err, "escrow challenge bond")
	}

	id := k.GetNextChallengeID(ctx)
	chal := types.Challenge{
		ChallengeId:  id,
		AppId:        msg.AppId,
		Epoch:        msg.Epoch,
		Challenger:   msg.Challenger,
		EvidenceHash: msg.EvidenceHash,
		BondDenom:    params.ChallengeBondDenom,
		BondAmount:   params.ChallengeBondAmount,
		Status:       types.ChallengeStatusOpen,
	}
	k.SetChallenge(ctx, chal)
	k.SetNextChallengeID(ctx, id+1)
	k.SetOpenChallengeIndex(ctx, msg.AppId, msg.Epoch, id)
	att = types.ApplyChallenge(att)
	k.SetAttestation(ctx, att)

	ctx.EventManager().EmitEvent(eventWithAppID(
		types.EventTypeAttestationChallenged, msg.AppId,
		sdk.NewAttribute(types.AttributeKeyChallengeID, strconv.FormatUint(id, 10)),
		sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatUint(msg.Epoch, 10)),
		sdk.NewAttribute(types.AttributeKeyChallenger, msg.Challenger),
		sdk.NewAttribute(types.AttributeKeyEvidenceHash, hexBytes(msg.EvidenceHash)),
	))
	return &types.MsgChallengeAttestationResponse{ChallengeId: id}, nil
}

func (k msgServer) ResolveChallenge(goCtx context.Context, msg *types.MsgResolveChallenge) (*types.MsgResolveChallengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, sdkerrors.Wrap(types.ErrUnauthorized, "not params authority")
	}
	chal, found := k.GetChallenge(ctx, msg.ChallengeId)
	if !found {
		return nil, types.ErrChallengeNotFound
	}
	att, found := k.GetAttestation(ctx, chal.AppId, chal.Epoch)
	if !found {
		return nil, types.ErrAttestationNotFound
	}
	att, chal, err := types.ApplyResolveChallenge(att, chal, msg.Uphold, msg.ResolutionHash, ctx.BlockHeight())
	if err != nil {
		return nil, err
	}
	k.DeleteOpenChallengeIndex(ctx, chal.AppId, chal.Epoch, chal.ChallengeId)

	bondAmt, err := types.ParseAmount(chal.BondAmount)
	if err != nil {
		return nil, err
	}
	coin := sdk.NewCoin(chal.BondDenom, bondAmt)
	bondFlow := "refund_in"
	if msg.Uphold {
		// Slash bond: move from challenge escrow into reward pool (Q-05).
		att.Status = types.AttestationStatusRejected
		bondFlow = "slash_in"
		if err := k.bank.SendCoinsFromModuleToModule(ctx, types.ChallengeEscrowName, types.RewardPoolName, sdk.NewCoins(coin)); err != nil {
			return nil, sdkerrors.Wrap(err, "slash challenge bond into reward pool")
		}
		// Supersede every other OPEN challenge and fully refund their bonds (Q-04).
		for _, oid := range k.ListOpenChallengeIDs(ctx, chal.AppId, chal.Epoch) {
			if oid == chal.ChallengeId {
				continue
			}
			other, ok := k.GetChallenge(ctx, oid)
			if !ok || other.Status != types.ChallengeStatusOpen {
				continue
			}
			other.Status = types.ChallengeStatusSuperseded
			other.ResolvedHeight = ctx.BlockHeight()
			other.ResolutionHash = msg.ResolutionHash
			k.DeleteOpenChallengeIndex(ctx, other.AppId, other.Epoch, other.ChallengeId)
			otherAmt, err := types.ParseAmount(other.BondAmount)
			if err != nil {
				return nil, err
			}
			otherAddr, err := sdk.AccAddressFromBech32(other.Challenger)
			if err != nil {
				return nil, err
			}
			otherCoin := sdk.NewCoin(other.BondDenom, otherAmt)
			if err := k.bank.SendCoinsFromModuleToAccount(ctx, types.ChallengeEscrowName, otherAddr, sdk.NewCoins(otherCoin)); err != nil {
				return nil, sdkerrors.Wrapf(err, "refund superseded challenge %d", oid)
			}
			k.SetChallenge(ctx, other)
			ctx.EventManager().EmitEvent(eventWithAppID(
				types.EventTypeChallengeResolved, other.AppId,
				sdk.NewAttribute(types.AttributeKeyChallengeID, strconv.FormatUint(oid, 10)),
				sdk.NewAttribute(types.AttributeKeyStatus, other.Status.String()),
				sdk.NewAttribute(types.AttributeKeyBondFlow, "refund_in"),
			))
		}
	} else {
		challenger, err := sdk.AccAddressFromBech32(chal.Challenger)
		if err != nil {
			return nil, err
		}
		if err := k.bank.SendCoinsFromModuleToAccount(ctx, types.ChallengeEscrowName, challenger, sdk.NewCoins(coin)); err != nil {
			return nil, sdkerrors.Wrap(err, "refund challenge bond")
		}
		if !k.HasOpenChallenge(ctx, chal.AppId, chal.Epoch) && att.Status != types.AttestationStatusRejected {
			att.Status = types.AttestationStatusChallengeWindow
		}
	}
	k.SetChallenge(ctx, chal)
	k.SetAttestation(ctx, att)

	ctx.EventManager().EmitEvent(eventWithAppID(
		types.EventTypeChallengeResolved, chal.AppId,
		sdk.NewAttribute(types.AttributeKeyChallengeID, strconv.FormatUint(chal.ChallengeId, 10)),
		sdk.NewAttribute(types.AttributeKeyStatus, chal.Status.String()),
		sdk.NewAttribute(types.AttributeKeyResolutionHash, hexBytes(msg.ResolutionHash)),
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyBondFlow, bondFlow),
	))
	return &types.MsgResolveChallengeResponse{}, nil
}

func (k msgServer) FinalizeAttestation(goCtx context.Context, msg *types.MsgFinalizeAttestation) (*types.MsgFinalizeAttestationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.PauseState != nil && params.PauseState.SettlementPaused {
		return nil, types.ErrSettlementPaused
	}
	app, found := k.GetApplication(ctx, msg.AppId)
	if !found {
		return nil, types.ErrApplicationNotFound
	}
	if app.Status != types.ApplicationStatusActive {
		return nil, types.ErrApplicationNotActive
	}
	if _, found := k.GetSettlement(ctx, msg.AppId, msg.Epoch); found {
		return nil, types.ErrSettlementExists
	}
	att, found := k.GetAttestation(ctx, msg.AppId, msg.Epoch)
	if !found {
		return nil, types.ErrAttestationNotFound
	}
	if att.Status == types.AttestationStatusRejected {
		return nil, types.ErrAttestationFinalized
	}
	if err := types.CanFinalize(att, ctx.BlockHeight(), k.HasOpenChallenge(ctx, msg.AppId, msg.Epoch)); err != nil {
		return nil, err
	}
	budget, ok := k.GetEpochBudget(ctx, msg.Epoch)
	if !ok {
		return nil, types.ErrBudgetUnavailable
	}
	reward, credit, newSpent, err := types.ComputeSettlementAmounts(att, budget, params.GasCreditShareBps)
	if err != nil {
		return nil, err
	}

	rewardAmt, err := types.ParseAmount(reward)
	if err != nil {
		return nil, err
	}
	if !rewardAmt.IsZero() {
		poolAddr := k.ak.GetModuleAddress(types.RewardPoolName)
		bal := k.bank.GetBalance(ctx, poolAddr, budget.Denom)
		if bal.Amount.LT(rewardAmt) {
			return nil, types.ErrPoolInsufficient
		}
		owner, err := sdk.AccAddressFromBech32(app.Owner)
		if err != nil {
			return nil, err
		}
		if err := k.bank.SendCoinsFromModuleToAccount(ctx, types.RewardPoolName, owner, sdk.NewCoins(sdk.NewCoin(budget.Denom, rewardAmt))); err != nil {
			return nil, sdkerrors.Wrap(err, "release reward")
		}
	}

	budget.Spent = newSpent
	k.StoreEpochBudget(ctx, budget)

	if !types.IsZeroAmount(credit) {
		acct := k.GetOrCreateGasCredit(ctx, msg.AppId)
		newBal, err := types.AddAmountStrings(acct.Balance, credit)
		if err != nil {
			return nil, err
		}
		newIssued, err := types.AddAmountStrings(acct.TotalIssued, credit)
		if err != nil {
			return nil, err
		}
		acct.Balance = newBal
		acct.TotalIssued = newIssued
		k.SetGasCredit(ctx, acct)
		ctx.EventManager().EmitEvent(eventWithAppID(
			types.EventTypeGasCreditIssued, msg.AppId,
			sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatUint(msg.Epoch, 10)),
			sdk.NewAttribute(types.AttributeKeyAmount, credit),
		))
	}

	settlement := types.BuildSettlement(att, reward, credit, ctx.BlockHeight())
	k.SetSettlement(ctx, settlement)
	att = types.ApplyFinalize(att)
	k.SetAttestation(ctx, att)

	ctx.EventManager().EmitEvent(eventWithAppID(
		types.EventTypeAttestationFinalized, msg.AppId,
		sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatUint(msg.Epoch, 10)),
		sdk.NewAttribute(types.AttributeKeyProfile, att.Profile),
		sdk.NewAttribute(types.AttributeKeyProfileVersion, att.ProfileVersion),
		sdk.NewAttribute(types.AttributeKeyAttestationHash, hexBytes(att.AttestationHash)),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	if !rewardAmt.IsZero() {
		ctx.EventManager().EmitEvent(eventWithAppID(
			types.EventTypeRewardReleased, msg.AppId,
			sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatUint(msg.Epoch, 10)),
			sdk.NewAttribute(types.AttributeKeyDenom, budget.Denom),
			sdk.NewAttribute(types.AttributeKeyAmount, reward),
		))
	}
	return &types.MsgFinalizeAttestationResponse{Settlement: settlement}, nil
}

func (k msgServer) GrantGasPayer(goCtx context.Context, msg *types.MsgGrantGasPayer) (*types.MsgGrantGasPayerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	app, found := k.GetApplication(ctx, msg.AppId)
	if !found {
		return nil, types.ErrApplicationNotFound
	}
	if app.Status == types.ApplicationStatusRevoked {
		return nil, types.ErrApplicationRevoked
	}
	if msg.Owner != app.Owner {
		return nil, types.ErrUnauthorized
	}
	for _, p := range app.GasPayers {
		if p == msg.GasPayer {
			return &types.MsgGrantGasPayerResponse{}, nil
		}
	}
	params := k.GetParams(ctx)
	if uint32(len(app.GasPayers)) >= params.EffectiveMaxGasPayers() {
		return nil, types.ErrGasPayerLimitExceeded
	}
	if err := k.SetGasPayerIndex(ctx, msg.GasPayer, msg.AppId); err != nil {
		return nil, err
	}
	app.GasPayers = append(app.GasPayers, msg.GasPayer)
	k.SetApplication(ctx, app)
	return &types.MsgGrantGasPayerResponse{}, nil
}

func (k msgServer) SetAttestationSubmitter(goCtx context.Context, msg *types.MsgSetAttestationSubmitter) (*types.MsgSetAttestationSubmitterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	app, found := k.GetApplication(ctx, msg.AppId)
	if !found {
		return nil, types.ErrApplicationNotFound
	}
	if app.Status == types.ApplicationStatusRevoked {
		return nil, types.ErrApplicationRevoked
	}
	if msg.Owner != app.Owner {
		return nil, types.ErrUnauthorized
	}
	old := app.AttestationSubmitter
	app.AttestationSubmitter = msg.AttestationSubmitter
	k.SetApplication(ctx, app)
	ctx.EventManager().EmitEvent(eventWithAppID(
		types.EventTypeSubmitterRotated, msg.AppId,
		sdk.NewAttribute(types.AttributeKeyOwner, msg.Owner),
		sdk.NewAttribute(types.AttributeKeyPrevSubmitter, old),
		sdk.NewAttribute(types.AttributeKeySubmitter, msg.AttestationSubmitter),
	))
	return &types.MsgSetAttestationSubmitterResponse{}, nil
}

func (k msgServer) RevokeGasPayer(goCtx context.Context, msg *types.MsgRevokeGasPayer) (*types.MsgRevokeGasPayerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	app, found := k.GetApplication(ctx, msg.AppId)
	if !found {
		return nil, types.ErrApplicationNotFound
	}
	if msg.Owner != app.Owner {
		return nil, types.ErrUnauthorized
	}
	out := make([]string, 0, len(app.GasPayers))
	for _, p := range app.GasPayers {
		if p != msg.GasPayer {
			out = append(out, p)
		}
	}
	app.GasPayers = out
	k.SetApplication(ctx, app)
	k.DeleteGasPayerIndex(ctx, msg.GasPayer)
	return &types.MsgRevokeGasPayerResponse{}, nil
}

func (k msgServer) ConsumeGasCredit(goCtx context.Context, msg *types.MsgConsumeGasCredit) (*types.MsgConsumeGasCreditResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if !k.GetParams(ctx).AllowManualGasCreditConsume {
		return nil, types.ErrManualConsumeDisabled
	}
	remaining, err := k.Keeper.ConsumeGasCredit(ctx, msg.AppId, msg.GasPayer, msg.Amount, msg.SourceTxHash)
	if err != nil {
		return nil, err
	}
	return &types.MsgConsumeGasCreditResponse{RemainingBalance: remaining}, nil
}

func (k msgServer) SetEpochBudget(goCtx context.Context, msg *types.MsgSetEpochBudget) (*types.MsgSetEpochBudgetResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, sdkerrors.Wrap(types.ErrUnauthorized, "not params authority")
	}
	b := msg.Budget
	if !types.DenomAllowed(params, b.Denom) {
		return nil, types.ErrInvalidDenom
	}
	if b.Spent == "" {
		b.Spent = "0"
	}
	if existing, ok := k.GetEpochBudget(ctx, b.Epoch); ok {
		// Denom is immutable once the epoch budget is opened (P0 audit).
		if b.Denom != existing.Denom {
			return nil, sdkerrors.Wrapf(types.ErrInvalidDenom,
				"cannot change epoch %d denom from %s to %s", b.Epoch, existing.Denom, b.Denom)
		}
		// Do not decrease spent; preserve spent if re-setting limit upward only.
		b.Spent = existing.Spent
		b.OpenedHeight = existing.OpenedHeight
		if b.Closed && !existing.Closed {
			b.ClosedHeight = ctx.BlockHeight()
		} else {
			b.ClosedHeight = existing.ClosedHeight
		}
		cmp, err := types.CompareAmountStrings(b.Spent, b.Limit)
		if err != nil {
			return nil, err
		}
		if cmp > 0 {
			return nil, types.ErrBudgetExceeded
		}
	} else if b.OpenedHeight == 0 {
		b.OpenedHeight = ctx.BlockHeight()
	}
	poolAddr := k.ak.GetModuleAddress(types.RewardPoolName)
	bal := k.bank.GetBalance(ctx, poolAddr, b.Denom)
	// Bound open commitments: sum(remaining limit) across non-closed budgets of
	// this denom, substituting this message's new limit for the same epoch.
	committed, err := k.openBudgetCommitment(ctx, b.Denom, b)
	if err != nil {
		return nil, err
	}
	if bal.Amount.LT(committed) {
		return nil, sdkerrors.Wrapf(types.ErrPoolInsufficient,
			"reward pool %s < open budget commitment %s (incl. epoch %d)", bal.Amount, committed, b.Epoch)
	}
	k.StoreEpochBudget(ctx, b)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeEpochBudgetSet,
		sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatUint(b.Epoch, 10)),
		sdk.NewAttribute(types.AttributeKeyDenom, b.Denom),
		sdk.NewAttribute(types.AttributeKeyLimit, b.Limit),
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
	))
	return &types.MsgSetEpochBudgetResponse{}, nil
}

func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		// pause_authority cannot resume via UpdateParams (Q-06)
		if msg.Authority == params.PauseAuthority {
			return nil, types.ErrResumeRequiresAuthority
		}
		return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "expected %s got %s", params.Authority, msg.Authority)
	}
	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) Pause(goCtx context.Context, msg *types.MsgPause) (*types.MsgPauseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.PauseAuthority != params.PauseAuthority {
		return nil, sdkerrors.Wrap(types.ErrUnauthorized, "not pause authority")
	}
	if !types.PauseBitsHasAnyTrue(msg.PauseBits) {
		return nil, types.ErrPauseOnlyTrue
	}
	cur := types.PauseState{}
	if params.PauseState != nil {
		cur = *params.PauseState
	}
	// Pause message may only OR true bits; clearing is resume and authority-only.
	next := types.ApplyPauseOR(cur, msg.PauseBits)
	params.PauseState = &next
	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeModulePaused,
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.PauseAuthority),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgPauseResponse{}, nil
}

func (k msgServer) FundRewardPool(goCtx context.Context, msg *types.MsgFundRewardPool) (*types.MsgFundRewardPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	// Permissionless deposit into reward pool (allowlisted denoms only).
	// Governance authority remains separate for budgets/params (gov accounts
	// cannot practically hold/spend liquid balances as Msg funders).
	if !types.DenomAllowed(params, msg.Denom) {
		return nil, types.ErrInvalidDenom
	}
	amt, err := types.ParseAmount(msg.Amount)
	if err != nil {
		return nil, err
	}
	if !amt.IsPositive() {
		return nil, sdkerrors.Wrap(types.ErrInvalidAmount, "fund amount must be > 0")
	}
	from, err := sdk.AccAddressFromBech32(msg.Funder)
	if err != nil {
		return nil, err
	}
	coins := sdk.NewCoins(sdk.NewCoin(msg.Denom, amt))
	if err := k.bank.SendCoinsFromAccountToModule(ctx, from, types.RewardPoolName, coins); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRewardPoolFunded,
		sdk.NewAttribute(types.AttributeKeyFunder, msg.Funder),
		sdk.NewAttribute(types.AttributeKeyDenom, msg.Denom),
		sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgFundRewardPoolResponse{}, nil
}

// ensureEpochBudgetRoom rejects submit when spent + pending non-terminal
// charges + thisCharge would exceed the epoch limit.
func (k Keeper) ensureEpochBudgetRoom(ctx sdk.Context, budget types.EpochBudget, thisCharge string) error {
	remaining, err := remainingBudget(budget)
	if err != nil {
		return err
	}
	pending, err := k.pendingEpochCharge(ctx, budget.Epoch)
	if err != nil {
		return err
	}
	add, err := types.ParseAmount(thisCharge)
	if err != nil {
		return err
	}
	if pending.Add(add).GT(remaining) {
		return types.ErrBudgetExceeded
	}
	return nil
}

// pendingEpochCharge sums AttestationBudgetCharge for non-terminal attestations
// in the given epoch.
func (k Keeper) pendingEpochCharge(ctx sdk.Context, epoch uint64) (sdkmath.Int, error) {
	total := sdkmath.ZeroInt()
	store := k.store(ctx)
	it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixAttestation)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var a types.AttestationCommitment
		k.cdc.MustUnmarshal(it.Value(), &a)
		if a.Epoch != epoch || !types.AttestationCountsTowardBudget(a.Status) {
			continue
		}
		charge, err := types.AttestationBudgetCharge(a)
		if err != nil {
			return sdkmath.ZeroInt(), err
		}
		amt, err := types.ParseAmount(charge)
		if err != nil {
			return sdkmath.ZeroInt(), err
		}
		total = total.Add(amt)
	}
	return total, nil
}

// openBudgetCommitment returns sum(limit-spent) for non-closed budgets of denom,
// using proposed as the value for its epoch (replace-or-insert).
func (k Keeper) openBudgetCommitment(ctx sdk.Context, denom string, proposed types.EpochBudget) (sdkmath.Int, error) {
	total := sdkmath.ZeroInt()
	seen := false
	store := k.store(ctx)
	it := storetypes.KVStorePrefixIterator(store, types.KeyPrefixEpochBudget)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var b types.EpochBudget
		k.cdc.MustUnmarshal(it.Value(), &b)
		if b.Denom != denom || b.Closed {
			continue
		}
		cur := b
		if b.Epoch == proposed.Epoch {
			cur = proposed
			seen = true
		}
		remaining, err := remainingBudget(cur)
		if err != nil {
			return sdkmath.ZeroInt(), err
		}
		total = total.Add(remaining)
	}
	if !seen && !proposed.Closed {
		remaining, err := remainingBudget(proposed)
		if err != nil {
			return sdkmath.ZeroInt(), err
		}
		total = total.Add(remaining)
	}
	return total, nil
}

func remainingBudget(b types.EpochBudget) (sdkmath.Int, error) {
	limit, err := types.ParseAmount(b.Limit)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	spent, err := types.ParseAmount(b.Spent)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	if spent.GT(limit) {
		return sdkmath.ZeroInt(), types.ErrBudgetExceeded
	}
	return limit.Sub(spent), nil
}
