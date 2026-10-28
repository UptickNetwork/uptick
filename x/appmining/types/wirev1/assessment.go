package wirev1

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const DomainAssessment = "UGE-APP-MINING-ASSESSMENT-V1"

// AssessmentFieldOrder is the frozen UGE AssessmentCredential frame order (§3).
var AssessmentFieldOrder = []string{
	"credential_version",
	"app_id",
	"pool_id",
	"pav_id",
	"evidence_root",
	"gas_evidence_root",
	"amount",
	"denom",
	"gas_used",
	"qualified_tx_count",
	"rule_version",
	"issued_at",
	"expires_at",
	"credential_id",
	"signer_key_id",
}

// AssessmentBytes returns the canonical assessment frame bytes (all fields as given).
func AssessmentBytes(record map[string]string) ([]byte, error) {
	return frame(DomainAssessment, record, AssessmentFieldOrder)
}

// AssessmentCredentialID returns SHA-256 hex of the assessment frame with
// credential_id forced empty. This avoids the self-referential fixed-point
// while still covering every other committed field (UGE §3 intent).
func AssessmentCredentialID(record map[string]string) (string, error) {
	clone := make(map[string]string, len(record))
	for k, v := range record {
		clone[k] = v
	}
	clone["credential_id"] = ""
	raw, err := AssessmentBytes(clone)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// RequireCredentialIDBound ensures record["credential_id"] equals AssessmentCredentialID.
func RequireCredentialIDBound(record map[string]string) error {
	id, ok := record["credential_id"]
	if !ok || id == "" {
		return fmt.Errorf("credential_id missing")
	}
	got, err := AssessmentCredentialID(record)
	if err != nil {
		return err
	}
	if got != id {
		return fmt.Errorf("credential_id mismatch")
	}
	return nil
}

// VerifyAssessment checks Ed25519 signature over AssessmentBytes (with filled credential_id).
func VerifyAssessment(record map[string]string, signatureB64 string, pubkey []byte) bool {
	if len(pubkey) != ed25519.PublicKeySize {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	if base64.StdEncoding.EncodeToString(sig) != signatureB64 {
		return false
	}
	raw, err := AssessmentBytes(record)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pubkey), raw, sig)
}
