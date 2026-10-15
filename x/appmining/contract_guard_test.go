// Contract guards for x/appmining.
//
// Rules:
//  1. No EVM / go-ethereum imports in production code (Cosmos-side settlement).
//  2. Do not treat opaque hash payloads as structured protocol data
//     (no json.Unmarshal / proto.Unmarshal of attestation_hash / evidence_root /
//     result_commitment beyond storing or comparing them).
//  3. No literal "chain/" app_id key prefix.
package appmining

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const moduleDir = "."

// productionFiles recursively lists production .go files under the module,
// excluding tests and generated *.pb.go / *.pb.gw.go.
func productionFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(moduleDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == "testdata" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if strings.HasSuffix(name, ".pb.go") || strings.HasSuffix(name, ".pb.gw.go") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", moduleDir, err)
	}
	return out
}

func TestNoEVMImport(t *testing.T) {
	forbidden := []string{
		`"github.com/cosmos/evm`,
		`"github.com/ethereum/go-ethereum`,
	}
	files := productionFiles(t)
	if len(files) == 0 {
		t.Fatal("expected production files under x/appmining; module has landed")
	}
	for _, file := range files {
		src := readFile(t, file)
		for _, needle := range forbidden {
			if strings.Contains(src, needle) {
				t.Errorf("%s imports %s — EVM imports are forbidden in x/appmining", file, needle)
			}
		}
	}
}

func TestNoOpaqueHashUnmarshal(t *testing.T) {
	// Opaque hash fields must not be decoded as structured payloads.
	needles := []string{
		"json.Unmarshal(attestation",
		"json.Unmarshal(AttestationHash",
		"proto.Unmarshal(att.AttestationHash",
		"proto.Unmarshal(attestation.AttestationHash",
		"proto.Unmarshal(a.AttestationHash",
		"proto.Unmarshal(c.EvidenceHash",
		"json.Unmarshal(evidence",
		"json.Unmarshal(result_commitment",
	}
	for _, file := range productionFiles(t) {
		src := readFile(t, file)
		lower := strings.ToLower(src)
		for _, needle := range needles {
			if strings.Contains(lower, strings.ToLower(needle)) {
				t.Errorf("%s appears to decode an opaque hash field (%s)", file, needle)
			}
		}
	}
}

func TestNoChainAppIDPrefix(t *testing.T) {
	for _, file := range productionFiles(t) {
		src := readFile(t, file)
		if strings.Contains(src, `"chain/"`) {
			t.Errorf("%s contains literal \"chain/\" app_id key prefix", file)
		}
	}
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}
