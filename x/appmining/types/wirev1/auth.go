package wirev1

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// Common authorization envelope fields (fixed order before op-specific fields).
var CommonFields = []string{
	"protocol_version", "operation", "role", "chain_id", "pool_id", "key_id", "nonce", "expires_at",
}

// OpFields maps operation -> ordered business fields (after CommonFields).
var OpFields = map[string][]string{
	"PUBLISH": {"entitlement_id", "app_id", "candidate_hash", "rule_version"},
	"RESOLVE": {"challenge_id", "candidate_id", "outcome", "rationale_digest"},
	"APPROVE": {"entitlement_id", "app_id", "candidate_hash", "challenge_resolution_hash", "amount", "denom", "rule_version"},
	"FREEZE":  {"entitlement_id", "evidence_type", "evidence_digest", "freeze_until"},
	"REVOKE":  {"entitlement_id", "evidence_type", "evidence_digest", "rationale_digest"},
}

var expectedRole = map[string]string{
	"PUBLISH": "publisher",
	"RESOLVE": "reviewer",
	"APPROVE": "approver",
	"FREEZE":  "risk",
}

// AuthorizationBytes returns canonical wire bytes for an authorization record.
func AuthorizationBytes(record map[string]string) ([]byte, error) {
	op := record["operation"]
	fields, ok := OpFields[op]
	if !ok {
		return nil, fmt.Errorf("unsupported operation")
	}
	expected := append(append([]string{}, CommonFields...), fields...)
	if err := exactKeys(record, expected); err != nil {
		return nil, err
	}
	if record["protocol_version"] != "1" {
		return nil, fmt.Errorf("invalid protocol version or role")
	}
	if want, ok := expectedRole[op]; ok {
		if record["role"] != want {
			return nil, fmt.Errorf("invalid protocol version or role")
		}
	} else if op == "REVOKE" {
		if record["role"] != "risk" && record["role"] != "audit" {
			return nil, fmt.Errorf("invalid protocol version or role")
		}
	}
	for _, name := range []string{"chain_id", "pool_id", "key_id", "rule_version", "denom"} {
		if v, ok := record[name]; ok {
			if err := requireID(name, v); err != nil {
				return nil, err
			}
		}
	}
	for _, name := range []string{
		"entitlement_id", "app_id", "candidate_hash", "challenge_id", "candidate_id",
		"rationale_digest", "challenge_resolution_hash", "evidence_digest",
	} {
		if v, ok := record[name]; ok {
			if err := requireHash(name, v); err != nil {
				return nil, err
			}
		}
	}
	for _, name := range []string{"nonce", "expires_at", "freeze_until", "amount"} {
		if v, ok := record[name]; ok {
			if err := requireUint(name, v); err != nil {
				return nil, err
			}
		}
	}
	if v, ok := record["outcome"]; ok {
		if v != "UPHELD" && v != "DISMISSED" {
			return nil, fmt.Errorf("invalid outcome")
		}
	}
	if v, ok := record["evidence_type"]; ok {
		if _, ok := evidenceTypes[v]; !ok {
			return nil, fmt.Errorf("invalid evidence type")
		}
	}
	return frame(DomainAuth, record, expected)
}

// AuthorizationDigest returns SHA-256 hex of AuthorizationBytes.
func AuthorizationDigest(record map[string]string) (string, error) {
	raw, err := AuthorizationBytes(record)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// VerifyAuthorization checks an Ed25519 signature (raw 64-byte, standard base64)
// over AuthorizationBytes. pubkey must be raw 32-byte Ed25519 public key.
func VerifyAuthorization(record map[string]string, signatureB64 string, pubkey []byte) bool {
	if len(pubkey) != ed25519.PublicKeySize {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	// Reject non-canonical base64 (JS checks round-trip equality).
	if base64.StdEncoding.EncodeToString(sig) != signatureB64 {
		return false
	}
	raw, err := AuthorizationBytes(record)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pubkey), raw, sig)
}
