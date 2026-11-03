package keeper

import (
	"strconv"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// verifyWireAuth validates an ACTIVE role key Ed25519/secp256k1 proof WITHOUT
// consuming the nonce: record/proof field agreement, expiry (past and
// far-future bound), key status, and the signature itself.
//
// Split from consumeWireAuthNonce so idempotent no-op paths (e.g. PUBLISH
// short-circuit) can require a valid signature before returning success
// without burning a fresh nonce.
func (k Keeper) verifyWireAuth(
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
	// P2-3: bound the validity horizon. A proof is one-shot, but without an
	// upper bound the signer could mint credentials that stay live for years;
	// cap exp at now + MaxWireProofValiditySeconds.
	if exp > nowUnix+types.MaxWireProofValiditySeconds {
		return sdkerrors.Wrap(types.ErrAuthMalformed, "expires_at too far in the future")
	}
	rk, found := k.GetRoleKey(ctx, role, proof.KeyId)
	if !found {
		return types.ErrRoleKeyNotFound
	}
	if rk.Status != types.RoleKeyStatusActive {
		return types.ErrRoleKeyInactive
	}
	switch rk.KeyType {
	case types.RoleKeyTypeEd25519:
		if !wirev1.VerifyAuthorization(record, proof.SignatureBase64, rk.Pubkey) {
			return types.ErrUnauthorized
		}
	case types.RoleKeyTypeSecp256k1EvmAddress:
		if role != types.EntitlementRolePublisher || record["operation"] != "PUBLISH" {
			return types.ErrRoleKeyTypeUnsupported
		}
		switch wirev1.VerifyAuthorizationSecp256k1(record, proof.SignatureBase64, rk.Pubkey) {
		case "":
			// ok
		case "sig_length":
			return types.ErrWireProofSigLength
		case "high_s":
			return types.ErrWireProofHighS
		case "mismatch", "recover":
			return types.ErrWireProofRecoverMismatch
		default:
			return types.ErrUnauthorized
		}
	default:
		return types.ErrRoleKeyTypeUnsupported
	}
	return nil
}

// consumeWireAuthNonce enforces single use and records the nonce with the
// proof's expiry so expired entries can be pruned. Must run after
// verifyWireAuth on every state-changing path.
func (k Keeper) consumeWireAuthNonce(
	ctx sdk.Context,
	role types.EntitlementRole,
	proof types.WireProof,
) error {
	if k.HasAuthNonce(ctx, role, proof.KeyId, proof.Nonce) {
		return types.ErrAuthReplay
	}
	exp, err := strconv.ParseInt(proof.ExpiresAt, 10, 64)
	if err != nil {
		return types.ErrAuthMalformed
	}
	k.SetAuthNonce(ctx, role, proof.KeyId, proof.Nonce, exp)
	return nil
}

// verifyAndConsumeWireAuth validates an ACTIVE role key proof and consumes
// the nonce. record must already include all wire fields including
// protocol_version, operation, role, chain_id, pool_id, key_id, nonce, expires_at.
func (k Keeper) verifyAndConsumeWireAuth(
	ctx sdk.Context,
	role types.EntitlementRole,
	proof types.WireProof,
	record map[string]string,
	nowUnix int64,
) error {
	if err := k.verifyWireAuth(ctx, role, proof, record, nowUnix); err != nil {
		return err
	}
	return k.consumeWireAuthNonce(ctx, role, proof)
}

// observedTime returns max(blockTime, watermark) for anti-rollback comparisons.
func observedTime(blockUnix, watermark int64) int64 {
	if blockUnix > watermark {
		return blockUnix
	}
	return watermark
}
