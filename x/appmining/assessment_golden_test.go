package appmining_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// assessmentGolden mirrors UGE spec/assessment-credential-golden-0.1.json.
type assessmentGolden struct {
	SchemaVersion string            `json:"schema_version"`
	Domain        string            `json:"domain"`
	FieldOrder    []string          `json:"field_order"`
	TestKey       assessmentTestKey `json:"test_key"`
	Fields        map[string]string `json:"fields"`
	CredentialID  string            `json:"credential_id"`
	SignatureB64  string            `json:"signature_base64"`
	FrameLengths  struct {
		IDPayload int `json:"id_payload"`
		Signed    int `json:"signed"`
	} `json:"frame_lengths"`
}

type assessmentTestKey struct {
	SeedHex       string `json:"seed_hex"`
	PublicKeyHex  string `json:"public_key_hex"`
}

func loadAssessmentGolden(t *testing.T) assessmentGolden {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "spec", "assessment-credential-golden-0.1.json")
	bz, err := os.ReadFile(path)
	require.NoError(t, err)
	var g assessmentGolden
	require.NoError(t, json.Unmarshal(bz, &g))
	return g
}

func TestAssessmentCredentialGolden_FrameIDAndVerify(t *testing.T) {
	g := loadAssessmentGolden(t)
	require.Equal(t, wirev1.DomainAssessment, g.Domain)
	require.Equal(t, wirev1.AssessmentFieldOrder, g.FieldOrder)

	record := make(map[string]string, len(g.Fields))
	for k, v := range g.Fields {
		record[k] = v
	}

	// ID frame: credential_id forced empty.
	idRecord := make(map[string]string, len(record))
	for k, v := range record {
		idRecord[k] = v
	}
	idRecord["credential_id"] = ""
	idFrame, err := wirev1.AssessmentBytes(idRecord)
	require.NoError(t, err)
	require.Equal(t, g.FrameLengths.IDPayload, len(idFrame), "id_payload frame length")

	gotID, err := wirev1.AssessmentCredentialID(record)
	require.NoError(t, err)
	require.Equal(t, g.CredentialID, gotID)
	require.Equal(t, g.Fields["credential_id"], gotID)

	// Signed frame: credential_id filled.
	signedFrame, err := wirev1.AssessmentBytes(record)
	require.NoError(t, err)
	require.Equal(t, g.FrameLengths.Signed, len(signedFrame), "signed frame length")

	seed, err := hex.DecodeString(g.TestKey.SeedHex)
	require.NoError(t, err)
	require.Len(t, seed, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	wantPub, err := hex.DecodeString(g.TestKey.PublicKeyHex)
	require.NoError(t, err)
	require.Equal(t, wantPub, []byte(pub))

	require.True(t, wirev1.VerifyAssessment(record, g.SignatureB64, pub), "golden signature must verify")

	// Deterministic re-sign: same key + same frame → byte-identical signature.
	resign := ed25519.Sign(priv, signedFrame)
	require.Equal(t, g.SignatureB64, base64.StdEncoding.EncodeToString(resign))
}

func TestAssessmentCredentialGolden_TamperRejects(t *testing.T) {
	g := loadAssessmentGolden(t)
	record := make(map[string]string, len(g.Fields))
	for k, v := range g.Fields {
		record[k] = v
	}
	seed, err := hex.DecodeString(g.TestKey.SeedHex)
	require.NoError(t, err)
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)

	tampered := make(map[string]string, len(record))
	for k, v := range record {
		tampered[k] = v
	}
	tampered["amount"] = "40001"
	require.False(t, wirev1.VerifyAssessment(tampered, g.SignatureB64, pub))
}
