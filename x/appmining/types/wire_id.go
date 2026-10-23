package types

import "regexp"

// Ed25519PubKeyLength is the raw Ed25519 public key size in bytes.
const Ed25519PubKeyLength = 32

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
