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

// ValidateRoleKeyMaterial checks pubkey/address bytes for the given key type + role.
func ValidateRoleKeyMaterial(role EntitlementRole, keyType RoleKeyType, material []byte) error {
	switch keyType {
	case RoleKeyTypeEd25519:
		return ValidateEd25519PubKey(material)
	case RoleKeyTypeSecp256k1EvmAddress:
		if role != EntitlementRolePublisher {
			return ErrRoleKeyTypeUnsupported
		}
		return ValidateEvmAddressBytes(material)
	default:
		return ErrRoleKeyTypeUnsupported
	}
}
