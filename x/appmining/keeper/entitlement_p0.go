package keeper

import (
	"bytes"
	"context"
	"strconv"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func (k msgServer) OpenEntitlementPool(goCtx context.Context, msg *types.MsgOpenEntitlementPool) (*types.MsgOpenEntitlementPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, types.ErrUnauthorized
	}
	if !types.DenomAllowed(params, msg.Denom) {
		return nil, types.ErrInvalidDenom
	}
	if _, found := k.GetEntitlementPool(ctx, msg.PoolId); found {
		return nil, types.ErrPoolExists
	}
	pool := types.EmptyPool(msg.PoolId, msg.Denom, msg.Version)
	if err := types.ValidatePoolInvariant(pool); err != nil {
		return nil, err
	}
	k.SetEntitlementPool(ctx, pool)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeEntitlementPoolOpened,
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyPoolID, msg.PoolId),
		sdk.NewAttribute(types.AttributeKeyDenom, msg.Denom),
		sdk.NewAttribute(types.AttributeKeyVersion, strconv.FormatUint(msg.Version, 10)),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgOpenEntitlementPoolResponse{}, nil
}

func (k msgServer) FundEntitlementPool(goCtx context.Context, msg *types.MsgFundEntitlementPool) (*types.MsgFundEntitlementPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.PauseState != nil && params.PauseState.EntitlementPaused {
		return nil, types.ErrEntitlementPaused
	}
	pool, found := k.GetEntitlementPool(ctx, msg.PoolId)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	if pool.Paused {
		return nil, types.ErrPoolPaused
	}
	amt, err := types.ParseAmount(msg.Amount)
	if err != nil {
		return nil, err
	}
	from, err := sdk.AccAddressFromBech32(msg.Funder)
	if err != nil {
		return nil, err
	}
	coins := sdk.NewCoins(sdk.NewCoin(pool.Denom, amt))
	if err := k.bank.SendCoinsFromAccountToModule(ctx, from, types.EntitlementPoolName, coins); err != nil {
		return nil, err
	}
	newFunded, err := types.AddAmountStrings(pool.Funded, msg.Amount)
	if err != nil {
		return nil, err
	}
	newAvailable, err := types.AddAmountStrings(pool.Available, msg.Amount)
	if err != nil {
		return nil, err
	}
	pool.Funded = newFunded
	pool.Available = newAvailable
	if err := types.ValidatePoolInvariant(pool); err != nil {
		return nil, err
	}
	k.SetEntitlementPool(ctx, pool)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeEntitlementPoolFunded,
		sdk.NewAttribute(types.AttributeKeyFunder, msg.Funder),
		sdk.NewAttribute(types.AttributeKeyPoolID, msg.PoolId),
		sdk.NewAttribute(types.AttributeKeyDenom, pool.Denom),
		sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount),
		sdk.NewAttribute(types.AttributeKeyFunded, pool.Funded),
		sdk.NewAttribute(types.AttributeKeyAvailable, pool.Available),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgFundEntitlementPoolResponse{}, nil
}

func (k msgServer) SetEntitlementPoolPaused(goCtx context.Context, msg *types.MsgSetEntitlementPoolPaused) (*types.MsgSetEntitlementPoolPausedResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, types.ErrUnauthorized
	}
	pool, found := k.GetEntitlementPool(ctx, msg.PoolId)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	prev := pool.Paused
	pool.Paused = msg.Paused
	k.SetEntitlementPool(ctx, pool)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeEntitlementPoolPaused,
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyPoolID, msg.PoolId),
		sdk.NewAttribute(types.AttributeKeyOldStatus, strconv.FormatBool(prev)),
		sdk.NewAttribute(types.AttributeKeyNewStatus, strconv.FormatBool(msg.Paused)),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgSetEntitlementPoolPausedResponse{}, nil
}

func (k msgServer) RegisterRoleKey(goCtx context.Context, msg *types.MsgRegisterRoleKey) (*types.MsgRegisterRoleKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, types.ErrUnauthorized
	}
	if existing, found := k.GetRoleKey(ctx, msg.Role, msg.KeyId); found {
		switch {
		case bytes.Equal(existing.Pubkey, msg.Pubkey) && existing.Status == types.RoleKeyStatusActive:
			return &types.MsgRegisterRoleKeyResponse{}, nil // idempotent
		case existing.Status == types.RoleKeyStatusRevoked:
			// re-register allowed after revoke
		case !bytes.Equal(existing.Pubkey, msg.Pubkey):
			return nil, types.ErrRoleKeyConflict
		}
	}
	// F-2 (§1.5): the same pubkey bytes must never be ACTIVE under two roles,
	// otherwise Revoke's risk/audit dual signature is a no-op. Check every
	// other role before persisting the new registration.
	if err := k.rejectPubkeyActiveInOtherRole(ctx, msg.Role, msg.Pubkey); err != nil {
		return nil, err
	}
	rk := types.RoleKey{
		Role:             msg.Role,
		KeyId:            msg.KeyId,
		Pubkey:           append([]byte(nil), msg.Pubkey...),
		Status:           types.RoleKeyStatusActive,
		Version:          msg.Version,
		RegisteredHeight: ctx.BlockHeight(),
	}
	k.SetRoleKey(ctx, rk)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRoleKeyRegistered,
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyRole, msg.Role.String()),
		sdk.NewAttribute(types.AttributeKeyKeyID, msg.KeyId),
		sdk.NewAttribute(types.AttributeKeyVersion, strconv.FormatUint(msg.Version, 10)),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgRegisterRoleKeyResponse{}, nil
}

func (k msgServer) SetRoleKeyStatus(goCtx context.Context, msg *types.MsgSetRoleKeyStatus) (*types.MsgSetRoleKeyStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if msg.Authority != params.Authority {
		return nil, types.ErrUnauthorized
	}
	rk, found := k.GetRoleKey(ctx, msg.Role, msg.KeyId)
	if !found {
		return nil, types.ErrRoleKeyNotFound
	}
	if rk.Status == types.RoleKeyStatusRevoked && msg.Status != types.RoleKeyStatusRevoked {
		return nil, sdkerrors.Wrap(types.ErrInvalidRoleKeyStatus, "revoked keys cannot be reactivated; register a new key_id")
	}
	prev := rk.Status
	rk.Status = msg.Status
	k.SetRoleKey(ctx, rk)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRoleKeyStatusChanged,
		sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyRole, msg.Role.String()),
		sdk.NewAttribute(types.AttributeKeyKeyID, msg.KeyId),
		sdk.NewAttribute(types.AttributeKeyOldStatus, prev.String()),
		sdk.NewAttribute(types.AttributeKeyNewStatus, msg.Status.String()),
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgSetRoleKeyStatusResponse{}, nil
}

// rejectPubkeyActiveInOtherRole rejects registering a pubkey that is already
// ACTIVE under a different role (F-2, §1.5 "两把不同公钥").
func (k Keeper) rejectPubkeyActiveInOtherRole(ctx sdk.Context, role types.EntitlementRole, pubkey []byte) error {
	roles := []types.EntitlementRole{
		types.EntitlementRolePublisher,
		types.EntitlementRoleApprover,
		types.EntitlementRoleReviewer,
		types.EntitlementRoleRisk,
		types.EntitlementRoleAudit,
	}
	for _, other := range roles {
		if other == role {
			continue
		}
		var conflict bool
		k.IterateRoleKeys(ctx, other, func(rk types.RoleKey) bool {
			if rk.Status == types.RoleKeyStatusActive && bytes.Equal(rk.Pubkey, pubkey) {
				conflict = true
				return false
			}
			return true
		})
		if conflict {
			return types.ErrRoleKeyConflict
		}
	}
	return nil
}
