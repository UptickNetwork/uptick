package types

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"
)

func init() {
	proto.RegisterType((*AssessmentCredential)(nil), "uptick.appmining.v1.AssessmentCredential")
	proto.RegisterType((*AssessmentSigner)(nil), "uptick.appmining.v1.AssessmentSigner")
}

// AssessmentCredential is the UGE assessment attestation attached to PublishCandidate (T1b / R3).
// Empty / all-zero fields means "not provided".
type AssessmentCredential struct {
	CredentialVersion string `protobuf:"bytes,1,opt,name=credential_version,json=credentialVersion,proto3" json:"credential_version,omitempty"`
	AppId             string `protobuf:"bytes,2,opt,name=app_id,json=appId,proto3" json:"app_id,omitempty"`
	PoolId            string `protobuf:"bytes,3,opt,name=pool_id,json=poolId,proto3" json:"pool_id,omitempty"`
	PavId             string `protobuf:"bytes,4,opt,name=pav_id,json=pavId,proto3" json:"pav_id,omitempty"`
	EvidenceRoot      string `protobuf:"bytes,5,opt,name=evidence_root,json=evidenceRoot,proto3" json:"evidence_root,omitempty"`
	GasEvidenceRoot   string `protobuf:"bytes,6,opt,name=gas_evidence_root,json=gasEvidenceRoot,proto3" json:"gas_evidence_root,omitempty"`
	Amount            string `protobuf:"bytes,7,opt,name=amount,proto3" json:"amount,omitempty"`
	Denom             string `protobuf:"bytes,8,opt,name=denom,proto3" json:"denom,omitempty"`
	GasUsed           string `protobuf:"bytes,9,opt,name=gas_used,json=gasUsed,proto3" json:"gas_used,omitempty"`
	QualifiedTxCount  string `protobuf:"bytes,10,opt,name=qualified_tx_count,json=qualifiedTxCount,proto3" json:"qualified_tx_count,omitempty"`
	RuleVersion       string `protobuf:"bytes,11,opt,name=rule_version,json=ruleVersion,proto3" json:"rule_version,omitempty"`
	IssuedAt          string `protobuf:"bytes,12,opt,name=issued_at,json=issuedAt,proto3" json:"issued_at,omitempty"`
	ExpiresAt         string `protobuf:"bytes,13,opt,name=expires_at,json=expiresAt,proto3" json:"expires_at,omitempty"`
	CredentialId      string `protobuf:"bytes,14,opt,name=credential_id,json=credentialId,proto3" json:"credential_id,omitempty"`
	SignerKeyId       string `protobuf:"bytes,15,opt,name=signer_key_id,json=signerKeyId,proto3" json:"signer_key_id,omitempty"`
	SignatureBase64   string `protobuf:"bytes,16,opt,name=signature_base64,json=signatureBase64,proto3" json:"signature_base64,omitempty"`
}

func (m *AssessmentCredential) Reset()         { *m = AssessmentCredential{} }
func (m *AssessmentCredential) String() string { return fmt.Sprintf("%v", *m) }
func (*AssessmentCredential) ProtoMessage()    {}

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

// AssessmentSigner is an authority-registered Ed25519 key for AssessmentCredential (T1b).
// Not an EntitlementRole — separate registry.
type AssessmentSigner struct {
	KeyId            string        `protobuf:"bytes,1,opt,name=key_id,json=keyId,proto3" json:"key_id,omitempty"`
	Pubkey           []byte        `protobuf:"bytes,2,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	Status           RoleKeyStatus `protobuf:"varint,3,opt,name=status,proto3,enum=uptick.appmining.v1.RoleKeyStatus" json:"status,omitempty"`
	Version          uint64        `protobuf:"varint,4,opt,name=version,proto3" json:"version,omitempty"`
	RegisteredHeight int64         `protobuf:"varint,5,opt,name=registered_height,json=registeredHeight,proto3" json:"registered_height,omitempty"`
}

func (m *AssessmentSigner) Reset()         { *m = AssessmentSigner{} }
func (m *AssessmentSigner) String() string { return fmt.Sprintf("%v", *m) }
func (*AssessmentSigner) ProtoMessage()    {}

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
