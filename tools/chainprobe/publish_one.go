package main

// Publish-one mode: broadcast exactly one MsgPublishCandidate whose body is the
// candidate the protocol service already built and stored.
//
// Why this exists: the relay executor accepts only pre-signed publishes
// (uge-dev-v1-publish-presigned-request-0.1), i.e. a developer wallet must sign
// the WireProof. To prove the *service's* chain write path (and to have a
// candidate the service can later approve) the publish has to carry the exact
// UGE candidate body: the keeper recomputes candidate_hash from the message,
// and the service compares that hash with its own local candidate. A candidate
// minted by chainprobe's own flow can never match, so the body is loaded from
// the service instead.
//
//	UGE_PUBLISH_CANDIDATE_FILE=/tmp/uge-publish.json ./chainprobe

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

type ugeCredential struct {
	CredentialVersion string `json:"credential_version"`
	AppID             string `json:"app_id"`
	PoolID            string `json:"pool_id"`
	PavID             string `json:"pav_id"`
	EvidenceRoot      string `json:"evidence_root"`
	GasEvidenceRoot   string `json:"gas_evidence_root"`
	Amount            string `json:"amount"`
	Denom             string `json:"denom"`
	GasUsed           string `json:"gas_used"`
	QualifiedTxCount  string `json:"qualified_tx_count"`
	RuleVersion       string `json:"rule_version"`
	IssuedAt          string `json:"issued_at"`
	ExpiresAt         string `json:"expires_at"`
	CredentialID      string `json:"credential_id"`
	SignerKeyID       string `json:"signer_key_id"`
	SignatureBase64   string `json:"signature_base64"`
}

type ugeCandidateDocument struct {
	PoolID           string `json:"pool_id"`
	PublisherKeyID   string `json:"publisher_key_id"`
	PublisherKeyFile string `json:"publisher_key_file"`
	RegisterKey      bool   `json:"register_key"`
	Candidate        struct {
		SchemaVersion    string `json:"schema_version"`
		AppID            string `json:"app_id"`
		Epoch            string `json:"epoch"`
		Amount           string `json:"amount"`
		Denom            string `json:"denom"`
		GasUsed          string `json:"gas_used"`
		QualifiedTxCount string `json:"qualified_tx_count"`
		PavID            string `json:"pav_id"`
		EvidenceRoot     string `json:"evidence_root"`
		GasEvidenceRoot  string `json:"gas_evidence_root"`
		RuleVersion      string `json:"rule_version"`
		EntitlementID    string `json:"entitlement_id"`
	} `json:"candidate"`
	Credential ugeCredential `json:"credential"`
}

// loadEd25519 reads a PKCS#8 Ed25519 private key. The role key files the
// deployment already ships (approver.key.pem, publisher.key.pem) are in this
// format, so the same file can be used here instead of inventing a new key.
func loadEd25519(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("%s holds no PEM block", path)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%s is %T, not Ed25519", path, key)
	}
	return priv, nil
}

func (p *probe) publishFromCandidateFile(file string) {
	raw, err := os.ReadFile(file)
	if err != nil {
		p.die("read %s: %v", file, err)
	}
	var doc ugeCandidateDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		p.die("decode %s: %v", file, err)
	}
	if doc.PoolID == "" || doc.PublisherKeyID == "" || doc.PublisherKeyFile == "" {
		p.die("pool_id, publisher_key_id and publisher_key_file are required")
	}
	pubPriv, err := loadEd25519(doc.PublisherKeyFile)
	if err != nil {
		p.die("publisher key: %v", err)
	}
	pubRaw := pubPriv.Public().(ed25519.PublicKey)

	if doc.RegisterKey {
		p.ok("RegisterRoleKey (uge publisher)", &types.MsgRegisterRoleKey{
			Authority: p.from.String(),
			Role:      types.EntitlementRolePublisher,
			KeyId:     doc.PublisherKeyID,
			Pubkey:    []byte(pubRaw),
			Version:   1,
			KeyType:   types.RoleKeyTypeEd25519,
		})
	}

	c := wirev1.Candidate{
		SchemaVersion:    doc.Candidate.SchemaVersion,
		EntitlementID:    doc.Candidate.EntitlementID,
		AppID:            doc.Candidate.AppID,
		Epoch:            doc.Candidate.Epoch,
		Amount:           doc.Candidate.Amount,
		Denom:            doc.Candidate.Denom,
		GasUsed:          doc.Candidate.GasUsed,
		QualifiedTxCount: doc.Candidate.QualifiedTxCount,
		PavID:            doc.Candidate.PavID,
		EvidenceRoot:     doc.Candidate.EvidenceRoot,
		GasEvidenceRoot:  doc.Candidate.GasEvidenceRoot,
		RuleVersion:      doc.Candidate.RuleVersion,
	}
	hash, err := wirev1.CandidateHash(c)
	if err != nil {
		p.die("candidate hash: %v", err)
	}
	fmt.Printf("uge candidate: app=%s pav=%s entitlement=%s amount=%s hash=%s\n",
		c.AppID, c.PavID, c.EntitlementID, c.Amount, hash)

	now := time.Now()
	expires := strconv.FormatInt(now.Add(time.Hour).Unix(), 10)
	nonce := strconv.FormatInt(now.UnixNano(), 10)
	record := map[string]string{
		"protocol_version": "1", "operation": "PUBLISH", "role": "publisher",
		"chain_id": p.cfg.chainID, "pool_id": doc.PoolID, "key_id": doc.PublisherKeyID,
		"nonce": nonce, "expires_at": expires,
		"entitlement_id": c.EntitlementID, "app_id": c.AppID,
		"candidate_hash": hash, "rule_version": c.RuleVersion,
	}

	cred := types.AssessmentCredential{
		CredentialVersion: doc.Credential.CredentialVersion,
		AppId:             doc.Credential.AppID,
		PoolId:            doc.Credential.PoolID,
		PavId:             doc.Credential.PavID,
		EvidenceRoot:      doc.Credential.EvidenceRoot,
		GasEvidenceRoot:   doc.Credential.GasEvidenceRoot,
		Amount:            doc.Credential.Amount,
		Denom:             doc.Credential.Denom,
		GasUsed:           doc.Credential.GasUsed,
		QualifiedTxCount:  doc.Credential.QualifiedTxCount,
		RuleVersion:       doc.Credential.RuleVersion,
		IssuedAt:          doc.Credential.IssuedAt,
		ExpiresAt:         doc.Credential.ExpiresAt,
		CredentialId:      doc.Credential.CredentialID,
		SignerKeyId:       doc.Credential.SignerKeyID,
		SignatureBase64:   doc.Credential.SignatureBase64,
	}

	res := p.ok("PublishCandidate (uge candidate)", &types.MsgPublishCandidate{
		Submitter: p.from.String(), ChainId: p.cfg.chainID, PoolId: doc.PoolID,
		EntitlementId: c.EntitlementID, AppId: c.AppID, Epoch: c.Epoch,
		Amount: c.Amount, Denom: c.Denom, GasUsed: c.GasUsed,
		QualifiedTxCount: c.QualifiedTxCount, PavId: c.PavID,
		EvidenceRoot: c.EvidenceRoot, GasEvidenceRoot: c.GasEvidenceRoot,
		RuleVersion: c.RuleVersion,
		Proof:       p.signWire(pubPriv, record),
		Credential:  cred,
	})
	fmt.Printf("published: tx=%s height=%s\n", res.hash, res.height)
}
