package keeper

import (
	"bytes"
	"context"
	"strconv"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func (k msgServer) RegisterAssessmentSigner(goCtx context.Context, msg *types.MsgRegisterAssessmentSigner) (*types.MsgRegisterAssessmentSignerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, types.ErrUnauthorized
	}
	if existing, found := k.GetAssessmentSigner(ctx, msg.KeyId); found {
		switch {
		case bytes.Equal(existing.Pubkey, msg.Pubkey) && existing.Status == types.RoleKeyStatusActive:
			return &types.MsgRegisterAssessmentSignerResponse{}, nil // idempotent
		case existing.Status == types.RoleKeyStatusRevoked:
			// re-register after revoke allowed (new material or same)
		case !bytes.Equal(existing.Pubkey, msg.Pubkey):
			return nil, types.ErrRoleKeyConflict
		default:
			// same pubkey, non-active non-revoked → reactivate via overwrite
		}
	}
	s := types.AssessmentSigner{
		KeyId:            msg.KeyId,
		Pubkey:           append([]byte(nil), msg.Pubkey...),
		Status:           types.RoleKeyStatusActive,
		Version:          msg.Version,
		RegisteredHeight: ctx.BlockHeight(),
	}
	k.SetAssessmentSigner(ctx, s)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAssessmentSignerRegistered,
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyKeyID, msg.KeyId),
		sdk.NewAttribute(types.AttributeKeyVersion, strconv.FormatUint(msg.Version, 10)),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgRegisterAssessmentSignerResponse{}, nil
}

func (k msgServer) SetAssessmentSignerStatus(goCtx context.Context, msg *types.MsgSetAssessmentSignerStatus) (*types.MsgSetAssessmentSignerStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, types.ErrUnauthorized
	}
	s, found := k.GetAssessmentSigner(ctx, msg.KeyId)
	if !found {
		return nil, types.ErrRoleKeyNotFound
	}
	if s.Status == types.RoleKeyStatusRevoked && msg.Status != types.RoleKeyStatusRevoked {
		return nil, sdkerrors.Wrap(types.ErrInvalidRoleKeyStatus, "revoked assessment signers cannot be reactivated; register a new key_id")
	}
	prev := s.Status
	s.Status = msg.Status
	k.SetAssessmentSigner(ctx, s)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAssessmentSignerStatusChanged,
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyKeyID, msg.KeyId),
		sdk.NewAttribute(types.AttributeKeyOldStatus, prev.String()),
		sdk.NewAttribute(types.AttributeKeyNewStatus, msg.Status.String()),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgSetAssessmentSignerStatusResponse{}, nil
}
