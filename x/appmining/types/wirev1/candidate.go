package wirev1

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// CandidateFields is the fixed order for UGE-APP-MINING-CANDIDATE-WIRE-V1.
var CandidateFields = []string{
	"schema_version", "entitlement_id", "app_id", "epoch", "amount", "denom",
	"gas_used", "qualified_tx_count", "pav_id", "evidence_root", "gas_evidence_root", "rule_version",
}

const CandidateSchemaVersion = "uge-app-mining-candidate-0.1"

// Candidate is the full candidate commitment record (all values as strings).
type Candidate struct {
	SchemaVersion    string
	EntitlementID    string
	AppID            string
	Epoch            string
	Amount           string
	Denom            string
	GasUsed          string
	QualifiedTxCount string
	PavID            string
	EvidenceRoot     string
	GasEvidenceRoot  string
	RuleVersion      string
}

func (c Candidate) asMap() map[string]string {
	return map[string]string{
		"schema_version":     c.SchemaVersion,
		"entitlement_id":     c.EntitlementID,
		"app_id":             c.AppID,
		"epoch":              c.Epoch,
		"amount":             c.Amount,
		"denom":              c.Denom,
		"gas_used":           c.GasUsed,
		"qualified_tx_count": c.QualifiedTxCount,
		"pav_id":             c.PavID,
		"evidence_root":      c.EvidenceRoot,
		"gas_evidence_root":  c.GasEvidenceRoot,
		"rule_version":       c.RuleVersion,
	}
}

// EntitlementIDOf recomputes candidate identity (UGE-APP-MINING-CANDIDATE-V1).
// Fields: app_id, epoch, pav_id, evidence_root, gas_evidence_root, gas_used, rule_version, denom, amount.
func EntitlementIDOf(c Candidate) string {
	fields := []string{
		c.AppID, c.Epoch, c.PavID, c.EvidenceRoot, c.GasEvidenceRoot,
		c.GasUsed, c.RuleVersion, c.Denom, c.Amount,
	}
	h := sha256.Sum256([]byte(DomainEntitlementID + "\x00" + strings.Join(fields, "\x00")))
	return hex.EncodeToString(h[:])
}

// CandidateHash returns SHA-256 hex of the candidate wire frame.
func CandidateHash(c Candidate) (string, error) {
	m := c.asMap()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := append([]string(nil), CandidateFields...)
	sort.Strings(want)
	if strings.Join(keys, "|") != strings.Join(want, "|") || c.SchemaVersion != CandidateSchemaVersion {
		return "", fmt.Errorf("invalid candidate schema")
	}
	for _, name := range []string{"entitlement_id", "app_id", "pav_id", "evidence_root", "gas_evidence_root"} {
		if err := requireHash(name, m[name]); err != nil {
			return "", err
		}
	}
	for _, name := range []string{"epoch", "amount", "gas_used", "qualified_tx_count"} {
		if err := requireUint(name, m[name]); err != nil {
			return "", err
		}
	}
	if err := requireID("denom", c.Denom); err != nil {
		return "", fmt.Errorf("invalid candidate policy")
	}
	if err := requireID("rule_version", c.RuleVersion); err != nil {
		return "", fmt.Errorf("invalid candidate policy")
	}
	if EntitlementIDOf(c) != c.EntitlementID {
		return "", fmt.Errorf("candidate identity mismatch")
	}
	raw, err := frame(DomainCandidate, m, CandidateFields)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
