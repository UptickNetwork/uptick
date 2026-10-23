package wirev1_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

type goldenFile struct {
	SchemaVersion             string            `json:"schema_version"`
	PublicKeyDERHex           string            `json:"public_key_der_hex"`
	Candidate                 json.RawMessage   `json:"candidate"`
	CandidateHash             string            `json:"candidate_hash"`
	Authorization             map[string]string `json:"authorization"`
	AuthorizationDigestSHA256 string            `json:"authorization_digest_sha256"`
	SignatureBase64           string            `json:"signature_base64"`
}

type goldenCandidate struct {
	SchemaVersion    string          `json:"schema_version"`
	EntitlementID    string          `json:"entitlement_id"`
	AppID            string          `json:"app_id"`
	Epoch            json.RawMessage `json:"epoch"`
	Amount           string          `json:"amount"`
	Denom            string          `json:"denom"`
	GasUsed          string          `json:"gas_used"`
	QualifiedTxCount json.RawMessage `json:"qualified_tx_count"`
	PavID            string          `json:"pav_id"`
	EvidenceRoot     string          `json:"evidence_root"`
	GasEvidenceRoot  string          `json:"gas_evidence_root"`
	RuleVersion      string          `json:"rule_version"`
}

func loadGolden(t *testing.T) goldenFile {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "..", "..", "spec", "app-mining-v1-wire-golden-1.0.json")
	bz, err := os.ReadFile(path)
	require.NoError(t, err)
	var g goldenFile
	require.NoError(t, json.Unmarshal(bz, &g))
	return g
}

func parseUintField(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var asStr string
	if err := json.Unmarshal(raw, &asStr); err == nil {
		return asStr
	}
	var asNum float64
	require.NoError(t, json.Unmarshal(raw, &asNum))
	return strconv.FormatInt(int64(asNum), 10)
}

// TestE7_WireGolden_CandidateHashAndAuthDigest is workorder §7.2 E-7.
func TestE7_WireGolden_CandidateHashAndAuthDigest(t *testing.T) {
	g := loadGolden(t)
	require.Equal(t, "uge-app-mining-wire-golden-1.0", g.SchemaVersion)

	var rawCand goldenCandidate
	require.NoError(t, json.Unmarshal(g.Candidate, &rawCand))
	c := wirev1.Candidate{
		SchemaVersion:    rawCand.SchemaVersion,
		EntitlementID:    rawCand.EntitlementID,
		AppID:            rawCand.AppID,
		Epoch:            parseUintField(t, rawCand.Epoch),
		Amount:           rawCand.Amount,
		Denom:            rawCand.Denom,
		GasUsed:          rawCand.GasUsed,
		QualifiedTxCount: parseUintField(t, rawCand.QualifiedTxCount),
		PavID:            rawCand.PavID,
		EvidenceRoot:     rawCand.EvidenceRoot,
		GasEvidenceRoot:  rawCand.GasEvidenceRoot,
		RuleVersion:      rawCand.RuleVersion,
	}

	require.Equal(t, c.EntitlementID, wirev1.EntitlementIDOf(c), "entitlement_id must match identity hash")

	gotHash, err := wirev1.CandidateHash(c)
	require.NoError(t, err)
	require.Equal(t, g.CandidateHash, gotHash)

	digest, err := wirev1.AuthorizationDigest(g.Authorization)
	require.NoError(t, err)
	require.Equal(t, g.AuthorizationDigestSHA256, digest)
}

// TestE7_WireGolden_VerifyEd25519Signature is workorder §7.2 E-7.
func TestE7_WireGolden_VerifyEd25519Signature(t *testing.T) {
	g := loadGolden(t)

	// Fixture: Ed25519 seed = 32 bytes of 0x01 (see golden test_only_key_notice).
	seed := bytes.Repeat([]byte{0x01}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	// Cross-check DER-embedded raw key from golden.
	der, err := hex.DecodeString(g.PublicKeyDERHex)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(der), 32)
	require.Equal(t, []byte(pub), der[len(der)-32:])

	require.True(t, wirev1.VerifyAuthorization(g.Authorization, g.SignatureBase64, pub))

	// Tamper reject.
	tampered := make(map[string]string, len(g.Authorization))
	for k, v := range g.Authorization {
		tampered[k] = v
	}
	tampered["amount"] = "31501"
	require.False(t, wirev1.VerifyAuthorization(tampered, g.SignatureBase64, pub))
}

func TestAuthorization_RejectsWrongRole(t *testing.T) {
	rec := map[string]string{
		"protocol_version":          "1",
		"operation":                 "APPROVE",
		"role":                      "publisher", // wrong
		"chain_id":                  "uptick_7777-1",
		"pool_id":                   "ecosystem-v0",
		"key_id":                    "test-key-1",
		"nonce":                     "1",
		"expires_at":                "1893456000",
		"entitlement_id":            "970c4147badb6d3d95505024ae52d123f436f09957666f9a5b940bbf90a9ceb9",
		"app_id":                    "1111111111111111111111111111111111111111111111111111111111111111",
		"candidate_hash":            "a2824b8d6cd1e9d361fb91838acffc8648a7124031a8bd68c4cc44c0aa5e3dc1",
		"challenge_resolution_hash": "5555555555555555555555555555555555555555555555555555555555555555",
		"amount":                    "31500",
		"denom":                     "auptick",
		"rule_version":              "gas-reward-candidate-0.1",
	}
	_, err := wirev1.AuthorizationBytes(rec)
	require.Error(t, err)
}

func TestCandidateHash_RejectsIdentityMismatch(t *testing.T) {
	c := wirev1.Candidate{
		SchemaVersion:    wirev1.CandidateSchemaVersion,
		EntitlementID:    "0000000000000000000000000000000000000000000000000000000000000000",
		AppID:            "1111111111111111111111111111111111111111111111111111111111111111",
		Epoch:            "1",
		Amount:           "31500",
		Denom:            "auptick",
		GasUsed:          "21000",
		QualifiedTxCount: "1",
		PavID:            "2222222222222222222222222222222222222222222222222222222222222222",
		EvidenceRoot:     "3333333333333333333333333333333333333333333333333333333333333333",
		GasEvidenceRoot:  "4444444444444444444444444444444444444444444444444444444444444444",
		RuleVersion:      "gas-reward-candidate-0.1",
	}
	_, err := wirev1.CandidateHash(c)
	require.Error(t, err)
}
