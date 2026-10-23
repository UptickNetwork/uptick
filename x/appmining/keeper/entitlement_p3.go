package keeper

import (
	"bytes"
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func (k msgServer) FreezeEntitlement(goCtx context.Context, msg *types.MsgFreezeEntitlement) (*types.MsgFreezeEntitlementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.ChainId != ctx.ChainID() {
		return nil, types.ErrChainIDMismatch
	}
	ent, found := k.GetEntitlement(ctx, msg.EntitlementId)
	if !found {
		return nil, types.ErrEntitlementNotFound
	}
	if msg.PoolId != ent.PoolId {
		return nil, types.ErrPoolMismatch
	}
	if _, _, err := k.requireEntitlementFlow(ctx, ent.PoolId); err != nil {
		return nil, err
	}
	if ent.State != types.EntitlementStateClaimable {
		if ent.State == types.EntitlementStateClaimed {
			return nil, types.ErrAlreadyClaimed
		}
		return nil, types.ErrNotClaimable
	}
	if ent.FreezeCount != 0 {
		return nil, types.ErrFreezeOnce
	}
	evWire, ok := types.EvidenceTypeWire(msg.EvidenceType)
	if !ok {
		return nil, types.ErrInvalidEvidenceType
	}

	now := observedTime(ctx.BlockTime().Unix(), ent.MaxObservedTime)
	freezeUntil, err := strconv.ParseInt(msg.FreezeUntil, 10, 64)
	if err != nil {
		return nil, types.ErrFreezeDuration
	}
	if freezeUntil <= now || freezeUntil > now+types.MaxFreezeSeconds {
		return nil, types.ErrFreezeDuration
	}

	record := map[string]string{
		"protocol_version": "1",
		"operation":        "FREEZE",
		"role":             "risk",
		"chain_id":         msg.ChainId,
		"pool_id":          msg.PoolId,
		"key_id":           msg.Proof.KeyId,
		"nonce":            msg.Proof.Nonce,
		"expires_at":       msg.Proof.ExpiresAt,
		"entitlement_id":   msg.EntitlementId,
		"evidence_type":    evWire,
		"evidence_digest":  msg.EvidenceDigest,
		"freeze_until":     msg.FreezeUntil,
	}
	if err := k.verifyAndConsumeWireAuth(ctx, types.EntitlementRoleRisk, msg.Proof, record, now); err != nil {
		return nil, err
	}

	prev := ent.State.String()
	ent.State = types.EntitlementStateFrozen
	ent.FreezeUntil = freezeUntil
	ent.FreezeCount = 1
	ent.MaxObservedTime = now
	k.SetEntitlement(ctx, ent)

	emitV1StateEvent(ctx, types.EventTypeEntitlementFrozen, "freeze",
		ent.PoolId, ent.EntitlementId, ent.AppId,
		prev, ent.State.String(), ent.Amount, ent.Denom, msg.EvidenceDigest,
		sdk.NewAttribute(types.AttributeKeyDeadline, msg.FreezeUntil),
	)
	return &types.MsgFreezeEntitlementResponse{}, nil
}

func (k msgServer) RevokeEntitlement(goCtx context.Context, msg *types.MsgRevokeEntitlement) (*types.MsgRevokeEntitlementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.ChainId != ctx.ChainID() {
		return nil, types.ErrChainIDMismatch
	}
	ent, found := k.GetEntitlement(ctx, msg.EntitlementId)
	if !found {
		return nil, types.ErrEntitlementNotFound
	}
	if msg.PoolId != ent.PoolId {
		return nil, types.ErrPoolMismatch
	}
	pool, _, err := k.requireEntitlementFlow(ctx, ent.PoolId)
	if err != nil {
		return nil, err
	}
	if ent.State == types.EntitlementStateClaimed {
		return nil, types.ErrAlreadyClaimed
	}
	if ent.State == types.EntitlementStateRevoked {
		return nil, types.ErrNotClaimable
	}

	now := observedTime(ctx.BlockTime().Unix(), ent.MaxObservedTime)
	// Must be effectively frozen: FROZEN and now < freeze_until
	if ent.State != types.EntitlementStateFrozen || now >= ent.FreezeUntil {
		return nil, types.ErrNotFrozen
	}
	evWire, ok := types.EvidenceTypeWire(msg.EvidenceType)
	if !ok {
		return nil, types.ErrInvalidEvidenceType
	}
	if msg.RiskProof.KeyId == msg.AuditProof.KeyId {
		return nil, types.ErrDualAuthRequired
	}
	// F-2 defense in depth (§1.5): the two proofs must be carried by two
	// DIFFERENT pubkeys. key_id inequality alone is not enough — the same
	// pubkey could have been registered under both roles (genesis import or
	// pre-hardening state).
	riskKey, riskFound := k.GetRoleKey(ctx, types.EntitlementRoleRisk, msg.RiskProof.KeyId)
	auditKey, auditFound := k.GetRoleKey(ctx, types.EntitlementRoleAudit, msg.AuditProof.KeyId)
	if riskFound && auditFound && bytes.Equal(riskKey.Pubkey, auditKey.Pubkey) {
		return nil, types.ErrDualAuthRequired
	}

	base := map[string]string{
		"protocol_version": "1",
		"operation":        "REVOKE",
		"chain_id":         msg.ChainId,
		"pool_id":          msg.PoolId,
		"entitlement_id":   msg.EntitlementId,
		"evidence_type":    evWire,
		"evidence_digest":  msg.EvidenceDigest,
		"rationale_digest": msg.RationaleDigest,
	}
	riskRec := copyMap(base)
	riskRec["role"] = "risk"
	riskRec["key_id"] = msg.RiskProof.KeyId
	riskRec["nonce"] = msg.RiskProof.Nonce
	riskRec["expires_at"] = msg.RiskProof.ExpiresAt
	if err := k.verifyAndConsumeWireAuth(ctx, types.EntitlementRoleRisk, msg.RiskProof, riskRec, now); err != nil {
		return nil, err
	}
	auditRec := copyMap(base)
	auditRec["role"] = "audit"
	auditRec["key_id"] = msg.AuditProof.KeyId
	auditRec["nonce"] = msg.AuditProof.Nonce
	auditRec["expires_at"] = msg.AuditProof.ExpiresAt
	if err := k.verifyAndConsumeWireAuth(ctx, types.EntitlementRoleAudit, msg.AuditProof, auditRec, now); err != nil {
		return nil, err
	}

	newReserved, err := types.SubAmountStrings(pool.Reserved, ent.Amount)
	if err != nil {
		return nil, err
	}
	newAvailable, err := types.AddAmountStrings(pool.Available, ent.Amount)
	if err != nil {
		return nil, err
	}
	pool.Reserved = newReserved
	pool.Available = newAvailable
	if err := types.ValidatePoolInvariant(pool); err != nil {
		return nil, err
	}
	k.SetEntitlementPool(ctx, pool)

	prev := ent.State.String()
	ent.State = types.EntitlementStateRevoked
	ent.MaxObservedTime = now
	k.SetEntitlement(ctx, ent)

	emitV1StateEvent(ctx, types.EventTypeEntitlementRevoked, "revoke",
		ent.PoolId, ent.EntitlementId, ent.AppId,
		prev, ent.State.String(), ent.Amount, ent.Denom, msg.EvidenceDigest,
		sdk.NewAttribute(types.AttributeKeyAvailable, pool.Available),
	)
	return &types.MsgRevokeEntitlementResponse{}, nil
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
