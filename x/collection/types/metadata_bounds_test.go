package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cosmossdk.io/x/nft"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
)

// readNonTestGoFiles returns filename -> source for every non-test .go file in
// dir, with line comments stripped so prose about a bound cannot trip a
// source-level assertion. CWD is the package directory, matching the idiom
// app/export_guard_test.go uses.
func readNonTestGoFiles(t *testing.T, dir string) map[string]string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	out := make(map[string]string)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)

		var b strings.Builder
		for _, line := range strings.Split(string(raw), "\n") {
			if idx := strings.Index(line, "//"); idx != -1 {
				line = line[:idx]
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		out[name] = b.String()
	}

	require.NotEmpty(t, out, "no Go sources read from %s", dir)
	return out
}

// TestDenomMetadataBoundHasOneImplementation is the source-level pin that makes
// the F-001 asymmetry unrepeatable by construction rather than by review.
//
// The bug was not a missing check; it was the bound being spelled in one place
// (keeper.SaveDenom) and not in another (ValidateGenesis) after v0.4.1 added it
// to the first only. A test that asserts the *values* agree would still pass if
// a fourth copy appeared with the right numbers today and drifted tomorrow. So
// this test asserts the bound has exactly one home -- validation.go -- and that
// the three genesis-path call sites reach it by CALL, never by re-deriving it.
//
// If this test fails, do not add your file to the allow-list below. Call
// ValidateDenomMetadataBounds (or ClampDenomMetadataBounds) instead.
func TestDenomMetadataBoundHasOneImplementation(t *testing.T) {
	const (
		canonicalFile = "validation.go"
		// msgs.go is grandfathered: MsgIssueDenom.ValidateBasic carries its own
		// comparison with its own error type and wording. It is an independent,
		// stricter gate on user issuance that runs before SaveDenom is ever
		// reached, so it cannot create a validate/import asymmetry (genesis
		// files never go through ValidateBasic). Listed explicitly, and only
		// here, so a *new* file cannot join it by accident: folding it into the
		// shared predicate would change a user-facing error message, which is a
		// decision for its own change -- not this fix.
		grandfatheredFile = "msgs.go"
	)
	bounds := []string{"MaxDenomSchemaLen", "MaxDenomDataLen"}

	typesSources := readNonTestGoFiles(t, ".")
	for name, src := range typesSources {
		if name == canonicalFile || name == grandfatheredFile {
			continue
		}
		for _, bound := range bounds {
			require.NotContains(t, src, bound,
				"%s names %s; the denom metadata bound must be read from %s "+
					"(via ValidateDenomMetadataBounds / ClampDenomMetadataBounds), not re-derived -- "+
					"two sides disagreeing about it is exactly the F-001 asymmetry",
				name, bound, canonicalFile)
		}
	}

	// The genesis-path call sites must actually call the predicate.
	for name, want := range map[string]string{
		"genesis.go": "ValidateDenomMetadataBounds(",
		"builder.go": "ValidateDenomMetadataBounds(",
	} {
		require.Contains(t, typesSources[name], want,
			"%s must obtain the bound from %s (%s missing)", name, canonicalFile, want)
	}

	// The keeper side must call it too -- and must not name the constants at all.
	keeperSources := readNonTestGoFiles(t, "../keeper")
	for name, src := range keeperSources {
		for _, bound := range bounds {
			require.NotContains(t, src, bound,
				"x/collection/keeper/%s names %s; keeper.SaveDenom must call "+
					"types.ValidateDenomMetadataBounds rather than compare against the constant",
				name, bound)
		}
	}
	require.Contains(t, keeperSources["denom.go"], "types.ValidateDenomMetadataBounds(",
		"keeper.SaveDenom must share the predicate with ValidateGenesis")
	require.Contains(t, keeperSources["collection.go"], "types.ClampDenomMetadataBounds(",
		"the export path must clamp through the shared helper rather than re-deriving the bound")
}

// TestBuildRejectsOversizedClassMetadata pins the ICS-721 receive-side gate.
//
// ClassBuilder.Build is the only place an inbound packet's classData becomes a
// stored class, and it writes through the base nft keeper, bypassing
// SaveDenom. Unbounded input here is how a counterparty chain could plant a
// class that this chain exported, validated, and then panicked on -- so both of
// Build's exits must enforce the same bound.
func TestBuildRejectsOversizedClassMetadata(t *testing.T) {
	cb, moduleAddr := newBuilderTestClassBuilder()

	oversizedSchema := strings.Repeat("y", MaxDenomSchemaLen+1)
	oversizedData := strings.Repeat("z", MaxDenomDataLen+1)

	meta := &DenomMetadata{
		Creator:          moduleAddr.String(),
		MintRestricted:   true,
		UpdateRestricted: true,
	}
	anyMeta, err := codectypes.NewAnyWithValue(meta)
	require.NoError(t, err)

	encodeTemplate := func(mutate func(*DenomMetadata)) string {
		t.Helper()
		fresh := &DenomMetadata{
			Creator:          moduleAddr.String(),
			MintRestricted:   true,
			UpdateRestricted: true,
		}
		mutate(fresh)
		anyFresh, err := codectypes.NewAnyWithValue(fresh)
		require.NoError(t, err)

		encoded, err := cb.BuildMetadata(nft.Class{Id: "someclass", Data: anyFresh})
		require.NoError(t, err)
		return encoded
	}

	t.Run("json branch: oversized schema", func(t *testing.T) {
		encoded := encodeTemplate(func(m *DenomMetadata) { m.Schema = oversizedSchema })
		_, err := cb.Build("someclass", "ipfs://class", encoded)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidDenom)
	})

	t.Run("json branch: oversized data", func(t *testing.T) {
		// BuildMetadata only re-emits the *decoded* keys, so an oversized blob
		// cannot be produced by round-tripping one: it needs a classData whose
		// JSON carries an unrecognised key, which Build keeps in its leftover
		// map and re-marshals into DenomMetadata.Data.
		classDataJSON := fmt.Sprintf(`{"%s":{"%s":"%s"}}`,
			"unknown_extra_field", KeyMediaFieldValue, oversizedData)

		_, err := cb.Build("someclass", "ipfs://class", Base64.EncodeToString([]byte(classDataJSON)))
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidDenom)
	})

	// The non-JSON branch stores the raw decoded bytes as the metadata blob.
	t.Run("non-json branch: oversized blob", func(t *testing.T) {
		blob := Base64.EncodeToString([]byte(oversizedData))
		_, err := cb.Build("someclass", "ipfs://class", blob)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidDenom)
	})

	// Positive controls: the bound is inclusive, and a legal class still builds.
	t.Run("at limit is accepted", func(t *testing.T) {
		encoded := encodeTemplate(func(m *DenomMetadata) {
			m.Schema = strings.Repeat("y", MaxDenomSchemaLen)
		})
		class, err := cb.Build("someclass", "ipfs://class", encoded)
		require.NoError(t, err)
		require.Equal(t, "someclass", class.Id)

		// Large but under the bound: the gate must not be a blanket reject.
		classDataJSON := fmt.Sprintf(`{"%s":{"%s":"%s"}}`,
			"unknown_extra_field", KeyMediaFieldValue, strings.Repeat("z", MaxDenomDataLen-128))
		class, err = cb.Build("someclass", "ipfs://class", Base64.EncodeToString([]byte(classDataJSON)))
		require.NoError(t, err)
		require.NotNil(t, class.Data)
		require.NotEmpty(t, class.Data.GetValue(), "sanity: the class carries a metadata blob")
	})

	t.Run("small class is accepted", func(t *testing.T) {
		encoded, err := cb.BuildMetadata(nft.Class{Id: "someclass", Data: anyMeta})
		require.NoError(t, err)

		class, err := cb.Build("someclass", "ipfs://class", encoded)
		require.NoError(t, err)
		require.NotNil(t, class.Data)

		// Same route the export/query paths use to read the blob back.
		registry := codectypes.NewInterfaceRegistry()
		RegisterInterfaces(registry)
		var got DenomMetadata
		require.NoError(t, codec.NewProtoCodec(registry).Unmarshal(class.Data.GetValue(), &got))
		require.Equal(t, moduleAddr.String(), got.Creator)
	})

	t.Run("non-json branch: empty classData is not the bound's business", func(t *testing.T) {
		// An empty classData means "no metadata": CreateOrUpdateClass does not
		// even call Build for it, and Build must not invent a rejection either.
		_, err := cb.Build("someclass", "ipfs://class", Base64.EncodeToString(nil))
		require.NoError(t, err)
	})
}
