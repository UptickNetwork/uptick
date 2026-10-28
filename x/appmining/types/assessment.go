package types

// IsEmpty reports whether the credential was omitted (protobuf default / all zero).
func (m *AssessmentCredential) IsEmpty() bool {
	if m == nil {
		return true
	}
	return m.CredentialVersion == "" &&
		m.AppId == "" &&
		m.PoolId == "" &&
		m.PavId == "" &&
		m.EvidenceRoot == "" &&
		m.GasEvidenceRoot == "" &&
		m.Amount == "" &&
		m.Denom == "" &&
		m.GasUsed == "" &&
		m.QualifiedTxCount == "" &&
		m.RuleVersion == "" &&
		m.IssuedAt == "" &&
		m.ExpiresAt == "" &&
		m.CredentialId == "" &&
		m.SignerKeyId == "" &&
		m.SignatureBase64 == ""
}

// Record returns the wirev1 assessment field map.
func (m AssessmentCredential) Record() map[string]string {
	return map[string]string{
		"credential_version": m.CredentialVersion,
		"app_id":             m.AppId,
		"pool_id":            m.PoolId,
		"pav_id":             m.PavId,
		"evidence_root":      m.EvidenceRoot,
		"gas_evidence_root":  m.GasEvidenceRoot,
		"amount":             m.Amount,
		"denom":              m.Denom,
		"gas_used":           m.GasUsed,
		"qualified_tx_count": m.QualifiedTxCount,
		"rule_version":       m.RuleVersion,
		"issued_at":          m.IssuedAt,
		"expires_at":         m.ExpiresAt,
		"credential_id":      m.CredentialId,
		"signer_key_id":      m.SignerKeyId,
	}
}

// DefaultAssessmentVersion is the first allowed assessment credential_version.
const DefaultAssessmentVersion = "uge-assessment-0.1"

// AssessmentVersionAllowed reports whether credential_version is on the allowlist.
// Empty allowlist denies all versions (default-deny, same pattern as rule_version).
func (p Params) AssessmentVersionAllowed(v string) bool {
	for _, allowed := range p.EntitlementAllowedAssessmentVersions {
		if allowed == v {
			return true
		}
	}
	return false
}
