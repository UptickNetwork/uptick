// The executable form of scripts/check-appmining-contract.sh.
//
// # WHY THIS FILE EXISTS BEFORE THE MODULE DOES
//
// x/appmining is not in this tree yet. These rules are still worth pinning now,
// for the ordinary reason: a constraint nobody checks is a constraint that gives
// way the first time someone is shipping under pressure. Two of the three rules
// below describe failures that cannot be detected after the fact -- an EVM import
// makes a contract a possible caller (invalidating every permission check written
// against sdk.AccAddress), and reaching into a commitment duplicates the rule it
// was meant to make underivable.
//
// The tests therefore pass vacuously-but-honestly today: they report that the
// module is absent and the rules are armed. They turn live the moment the module
// lands, which is the point -- see TestAppminingModuleAbsentUntilItLands.
package appmining

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// moduleDir is this file's own directory: x/appmining. The guard is anchored to
// where it will actually be, so a moved module cannot silently escape its own
// contract.
const moduleDir = "."

// Production Go files in this package, excluding tests.
func productionFiles(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		t.Fatalf("read %s: %v", moduleDir, err)
	}

	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		out = append(out, name)
	}
	return out
}

// TestAppminingModuleAbsentUntilItLands documents the current state and keeps the
// guard honest about it.
//
// The temptation with a guard written ahead of its subject is to let it pass
// quietly, which would make "the module isn't here" indistinguishable from "the
// module is here and clean". So the absence is asserted explicitly. When someone
// creates the first real file, this test fails and gets rewritten -- at which
// point TestNoEVMImport and friends below are already in place to run against it.
func TestAppminingModuleAbsentUntilItLands(t *testing.T) {
	if files := productionFiles(t); len(files) != 0 {
		t.Skipf("x/appmining now has %d production file(s) (%v); the placeholder rules now apply, "+
			"and this absence guard should be deleted in the same commit that landed them",
			len(files), files)
	}

	t.Log("x/appmining has no production Go files yet; contract guard is armed and inert")
}

// TestNoEVMImport pins rule [1].
//
// UGE is a Cosmos-side protocol: its emitter is CosmWasm, its economics are bank
// transfers out of a module account. The moment this module imports cosmos/evm,
// the EVM account model applies to it -- a contract becomes a possible caller --
// and every permission check written against sdk.AccAddress has to be
// re-derived against a caller that can be code. That is a design change, so it
// has to be a deliberate one rather than an import someone added while fixing
// something unrelated.
//
// Matched on the import path so an aliased import (`evm "github.com/cosmos/evm/..."`)
// cannot slip past.
func TestNoEVMImport(t *testing.T) {
	forbidden := []string{
		`"github.com/cosmos/evm`,
		`"github.com/ethereum/go-ethereum`,
	}

	for _, file := range productionFiles(t) {
		src := readFile(t, file)
		for _, needle := range forbidden {
			if strings.Contains(src, needle) {
				t.Errorf("%s imports %s.\n"+
					"UGE is a Cosmos-side protocol; an EVM import makes a contract a possible caller and "+
					"invalidates every sdk.AccAddress permission check in this module. If this import is "+
					"genuinely required, it is a design decision to raise before merge, not a local fix.",
					file, needle)
			}
		}
	}
}

// TestNoPavCommitmentFieldAccess pins rule [2].
//
// The PAV commitment is opaque by construction: a submitter commits, a third
// party challenges, settlement reads the verdict. Code that reaches INTO the
// commitment -- unmarshalling it, reading one field, comparing one field -- is
// re-deriving what the commitment was made to make underivable, and it does so
// with consensus consequences: the challenge window or the epoch boundary then
// lives in two places and drifts.
//
// Matching getters rather than the bare type name is deliberate. The type is the
// Msg's field type and may be referenced freely; what must not appear is a
// per-field read.
func TestNoPavCommitmentFieldAccess(t *testing.T) {
	// Getters that expose a single field of a struct. Their presence in this
	// module is the signal; the commitment type may still be passed around.
	fieldGetters := []string{
		".GetAppId(",
		".GetEpoch(",
		".GetCommitment(",
		".GetChallenger(",
		".GetChallengeReason(",
		".GetSettled(",
	}

	for _, file := range productionFiles(t) {
		src := readFile(t, file)
		// Only flag a getter when it is plausibly aimed at a commitment or a
		// PAV, so an unrelated Epoch getter elsewhere is not caught.
		for _, getter := range fieldGetters {
			if !strings.Contains(src, getter) {
				continue
			}
			if mentionsCommitmentOrPav(src) {
				t.Errorf("%s reads a single field off a PAV commitment (%s).\n"+
					"The commitment is opaque by design. Per-field access duplicates the challenge window "+
					"or epoch boundary into a second place that then drifts. The verdict and the height "+
					"are the whole interface.",
					file, getter)
			}
		}
	}
}

// TestNoChainAppIDPrefix pins rule [3].
//
// app_id is the base64url-no-padding form of a 32-byte SHA-256 -- 43 characters,
// an opaque identifier the chain deliberately never resolves (UGE confirmed the
// chain end does not parse it). Putting it under a prefixed key contradicts that
// at the storage layer and turns "app_id" and "chain/<app_id>" into two
// spellings of one thing, with the separator living inside the key space.
//
// The ban is on this specific literal only. Storing app_id under a module-owned
// binary prefix is the expected shape and is untouched by this rule.
func TestNoChainAppIDPrefix(t *testing.T) {
	for _, file := range productionFiles(t) {
		src := readFile(t, file)
		if strings.Contains(src, `"chain/"`) {
			t.Errorf("%s contains the literal \"chain/\".\n"+
				"app_id is an opaque base64url-no-padding SHA-256 the chain does not parse; a \"chain/\" "+
				"prefix makes a protocol claim about ownership that the storage layer was chosen not to "+
				"make. Use a module-owned binary prefix instead.",
				file)
		}
	}
}

// mentionsCommitmentOrPav reports whether the source refers to the commitment or
// to a PAV at all, which is what makes a field getter suspicious here.
func mentionsCommitmentOrPav(src string) bool {
	for _, needle := range []string{"Commitment", "commitment", "PAV", "Pav", "pav"} {
		if strings.Contains(src, needle) {
			return true
		}
	}
	return false
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := fs.ReadFile(fs.FS(os.DirFS(moduleDir)), name)
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Join(moduleDir, name), err)
	}
	return string(b)
}
