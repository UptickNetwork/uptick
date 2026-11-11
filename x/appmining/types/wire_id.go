package types

import (
	"encoding/hex"
	"regexp"
	"strings"
)

// Ed25519PubKeyLength is the raw Ed25519 public key size in bytes.
const Ed25519PubKeyLength = 32

// EvmAddressLength is the raw EVM address size in bytes (RoleKeyTypeSecp256k1EvmAddress).
const EvmAddressLength = 20

// Wire ID / amount patterns align with UGE protocol/app-mining-v1/wire-v1.mjs.
var (
	wireIDPattern   = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
	wireHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ValidateWireID checks pool_id / key_id / denom / rule_version style identifiers.
func ValidateWireID(s string) error {
	if !wireIDPattern.MatchString(s) {
		return ErrInvalidWireID
	}
	return nil
}

// ValidateWireHash checks a 64-char lowercase hex digest.
func ValidateWireHash(s string) error {
	if !wireHashPattern.MatchString(s) {
		return ErrInvalidHash
	}
	return nil
}

// ValidateEd25519PubKey requires raw 32-byte Ed25519 public key.
func ValidateEd25519PubKey(pk []byte) error {
	if len(pk) != Ed25519PubKeyLength {
		return ErrInvalidCommitment
	}
	return nil
}

// ValidateEvmAddressBytes requires raw 20-byte EVM address.
func ValidateEvmAddressBytes(addr []byte) error {
	if len(addr) != EvmAddressLength {
		return ErrInvalidCommitment
	}
	return nil
}

// NormalizeEvmAddressInput accepts 20 raw bytes, 40 hex, or 0x+40 hex → 20 bytes.
func NormalizeEvmAddressInput(in []byte) ([]byte, error) {
	if len(in) == EvmAddressLength {
		return append([]byte(nil), in...), nil
	}
	s := strings.TrimSpace(string(in))
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	if len(s) != EvmAddressLength*2 {
		return nil, ErrInvalidCommitment
	}
	out, err := hex.DecodeString(s)
	if err != nil || len(out) != EvmAddressLength {
		return nil, ErrInvalidCommitment
	}
	return out, nil
}

// RoleKeyTypeAllowed reports whether keyType may be registered at all for role.
//
// An EVM-address key is *not* a weaker Ed25519 key: it can only authorize
// operations whose preimage the holder's wallet will sign as EIP-191
// personal_sign bytes (see RoleKeyTypeOperationAllowed). Registration is
// therefore gated per role, and verification is gated per operation — the two
// checks are deliberately separate so that a role which gains a new operation
// cannot silently gain EVM-address authorization for it.
func RoleKeyTypeAllowed(role EntitlementRole, keyType RoleKeyType) bool {
	switch keyType {
	case RoleKeyTypeEd25519:
		return true
	case RoleKeyTypeSecp256k1EvmAddress:
		// publisher: PUBLISH (DEC-10). reviewer: RESOLVE, so the party who
		// adjudicates a challenge can sign from a wallet they already hold
		// instead of exporting an Ed25519 secret into an operations host.
		return role == EntitlementRolePublisher || role == EntitlementRoleReviewer
	default:
		return false
	}
}

// RoleKeyTypeOperationAllowed reports whether a key of keyType may authorize
// operation for role.
//
// Ed25519 keys may authorize any operation the role owns; EVM-address keys are
// restricted to the operations whose canonical bytes the wallet signs as
// EIP-191 (`accounts.TextHash`), which is what wirev1.VerifyAuthorizationSecp256k1
// verifies. Widening this without widening the wallet path would let a
// registration exist that no wallet can ever produce a valid proof for.
func RoleKeyTypeOperationAllowed(role EntitlementRole, keyType RoleKeyType, operation string) bool {
	if keyType == RoleKeyTypeEd25519 {
		return true
	}
	if keyType != RoleKeyTypeSecp256k1EvmAddress {
		return false
	}
	switch role {
	case EntitlementRolePublisher:
		return operation == "PUBLISH"
	case EntitlementRoleReviewer:
		return operation == "RESOLVE"
	default:
		return false
	}
}

// ValidateRoleKeyMaterial checks pubkey/address bytes for the given key type + role.
func ValidateRoleKeyMaterial(role EntitlementRole, keyType RoleKeyType, material []byte) error {
	switch keyType {
	case RoleKeyTypeEd25519:
		return ValidateEd25519PubKey(material)
	case RoleKeyTypeSecp256k1EvmAddress:
		if !RoleKeyTypeAllowed(role, keyType) {
			return ErrRoleKeyTypeUnsupported
		}
		return ValidateEvmAddressBytes(material)
	default:
		return ErrRoleKeyTypeUnsupported
	}
}
