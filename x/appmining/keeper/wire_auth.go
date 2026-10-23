package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// verifyAndConsumeWireAuth validates an ACTIVE role key Ed25519 proof and
// consumes the nonce. record must already include all wire fields including
// protocol_version, operation, role, chain_id, pool_id, key_id, nonce, expires_at.
func (k Keeper) verifyAndConsumeWireAuth(
	ctx sdk.Context,
	role types.EntitlementRole,
	proof types.WireProof,
	record map[string]string,
	nowUnix int64,
) error {
	wantRole, ok := types.RoleWire(role)
	if !ok {
		return types.ErrInvalidRole
	}
	if record["role"] != wantRole {
		return types.ErrUnauthorized
	}
	if record["key_id"] != proof.KeyId || record["nonce"] != proof.Nonce || record["expires_at"] != proof.ExpiresAt {
		return types.ErrAuthMalformed
	}
	exp, err := strconv.ParseInt(proof.ExpiresAt, 10, 64)
	if err != nil {
		return types.ErrAuthMalformed
	}
	if nowUnix > exp {
		return types.ErrAuthExpired
	}
	rk, found := k.GetRoleKey(ctx, role, proof.KeyId)
	if !found {
		return types.ErrRoleKeyNotFound
	}
	if rk.Status != types.RoleKeyStatusActive {
		return types.ErrRoleKeyInactive
	}
	if k.HasAuthNonce(ctx, role, proof.KeyId, proof.Nonce) {
		return types.ErrAuthReplay
	}
	if !wirev1.VerifyAuthorization(record, proof.SignatureBase64, rk.Pubkey) {
		return types.ErrUnauthorized
	}
	k.SetAuthNonce(ctx, role, proof.KeyId, proof.Nonce)
	return nil
}

// observedTime returns max(blockTime, watermark) for anti-rollback comparisons.
func observedTime(blockUnix, watermark int64) int64 {
	if blockUnix > watermark {
		return blockUnix
	}
	return watermark
}
