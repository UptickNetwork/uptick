package types

import (
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/x/nft"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
)

// newTokenBuilderTestHarness returns a TokenBuilder plus the codec needed to
// read NFTMetadata.Data back out of the Any it stores.
func newTokenBuilderTestHarness() (TokenBuilder, codec.Codec) {
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	return NewTokenBuilder(cdc), cdc
}

// storedTokenMetadata decodes the Any TokenBuilder.Build attached, i.e. exactly
// what a later genesis export / ICS-721 send would read.
func storedTokenMetadata(t *testing.T, cdc codec.Codec, token nft.NFT) NFTMetadata {
	t.Helper()
	require.NotNil(t, token.Data, "the built token must carry a metadata blob")
	var meta NFTMetadata
	require.NoError(t, cdc.Unmarshal(token.Data.GetValue(), &meta))
	return meta
}

// TestValidateTokenMetadataBoundsIsInclusive pins the boundary itself, so the
// Build tests below can rely on it independently of JSON re-encoding overhead.
func TestValidateTokenMetadataBoundsIsInclusive(t *testing.T) {
	require.NoError(t, ValidateTokenMetadataBounds(""), "empty data is not the bound's business")
	require.NoError(t, ValidateTokenMetadataBounds(strings.Repeat("a", MaxTokenDataLen)),
		"the bound is inclusive: exactly MaxTokenDataLen must be accepted")
	require.Error(t, ValidateTokenMetadataBounds(strings.Repeat("a", MaxTokenDataLen+1)))
	require.ErrorIs(t, ValidateTokenMetadataBounds(strings.Repeat("a", MaxTokenDataLen+1)), ErrInvalidNFT)
}

// TestBuildRejectsOversizedTokenMetadata pins the ICS-721 receive-side gate.
//
// TokenBuilder.Build is the only place an inbound packet's tokenData becomes a
// stored token -- x/internft calls it from Mint (keeper.go:94, the receive
// path) and from Transfer (:119) -- and both of its exits must enforce the same
// bound. The non-JSON exit is the sharper one: it persists the packet's raw
// decoded bytes verbatim, so a counterparty chain picked those bytes exactly.
func TestBuildRejectsOversizedTokenMetadata(t *testing.T) {
	tb, cdc := newTokenBuilderTestHarness()

	oversized := strings.Repeat("z", MaxTokenDataLen+1)

	t.Run("non-json exit rejects an oversized blob", func(t *testing.T) {
		_, err := tb.Build("ibc/abc", "1", "ipfs://token", Base64.EncodeToString([]byte(oversized)))
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidNFT)
	})

	// The JSON exit re-encodes only the keys Build does not recognise, so an
	// oversized blob there needs an unknown key to survive into
	// NFTMetadata.Data.
	t.Run("json exit rejects an oversized leftover map", func(t *testing.T) {
		tokenDataJSON := fmt.Sprintf(`{"%s":{"%s":"%s"}}`,
			"unknown_extra_field", KeyMediaFieldValue, oversized)

		_, err := tb.Build("ibc/abc", "1", "ipfs://token", Base64.EncodeToString([]byte(tokenDataJSON)))
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidNFT)
	})

	// Positive control: the bound is inclusive, and the bytes it allows really
	// do reach the store. Without this the "rejects" cases above would pass
	// just as well against a blanket reject.
	t.Run("non-json exit accepts exactly the limit", func(t *testing.T) {
		atLimit := strings.Repeat("z", MaxTokenDataLen)

		token, err := tb.Build("ibc/abc", "1", "ipfs://token", Base64.EncodeToString([]byte(atLimit)))
		require.NoError(t, err)
		require.Equal(t, "1", token.Id)

		meta := storedTokenMetadata(t, cdc, token)
		require.Len(t, meta.Data, MaxTokenDataLen, "the stored blob is what the bound measures")
		require.Equal(t, atLimit, meta.Data)
	})

	t.Run("json exit accepts a large but legal leftover map", func(t *testing.T) {
		tokenDataJSON := fmt.Sprintf(`{"%s":{"%s":"%s"}}`,
			"unknown_extra_field", KeyMediaFieldValue, strings.Repeat("z", MaxTokenDataLen-128))

		token, err := tb.Build("ibc/abc", "1", "ipfs://token", Base64.EncodeToString([]byte(tokenDataJSON)))
		require.NoError(t, err)

		meta := storedTokenMetadata(t, cdc, token)
		require.NotEmpty(t, meta.Data)
		require.LessOrEqual(t, len(meta.Data), MaxTokenDataLen)
	})

	// Ordinary traffic must be untouched: a token that carries no extra keys
	// still builds, and its name survives.
	t.Run("ordinary token is unaffected", func(t *testing.T) {
		tokenDataJSON := fmt.Sprintf(`{"%s":{"%s":"Token One"}}`,
			TokenKeyName, KeyMediaFieldValue)

		token, err := tb.Build("ibc/abc", "1", "ipfs://token", Base64.EncodeToString([]byte(tokenDataJSON)))
		require.NoError(t, err)

		meta := storedTokenMetadata(t, cdc, token)
		require.Equal(t, "Token One", meta.Name)
		require.Empty(t, meta.Data, "the recognised key is lifted out of the leftover map")
	})

	// Empty tokenData means "no metadata": Base64.DecodeString("") yields no
	// bytes, the JSON parse fails, and the raw blob is "" -- which is legal.
	// The bound must not invent a rejection for it.
	t.Run("empty tokenData is not the bound's business", func(t *testing.T) {
		token, err := tb.Build("ibc/abc", "1", "ipfs://token", "")
		require.NoError(t, err)

		meta := storedTokenMetadata(t, cdc, token)
		require.Empty(t, meta.Data)
	})

	// Unchanged behaviour: malformed base64 still fails on decode, before any
	// bound is consulted.
	t.Run("malformed base64 still fails on decode", func(t *testing.T) {
		_, err := tb.Build("ibc/abc", "1", "ipfs://token", "not!base64!")
		require.Error(t, err)
	})
}

// TestTokenMetadataBoundStaysOnTheReceiveSide is the source-level pin that keeps
// this fix from becoming the F-001 asymmetry in reverse.
//
// F-001 was a bound added to the write side while the validate/import side
// stayed lax, so an oversized record could be exported, accepted by
// `validate-genesis`, and then panic InitGenesis. The token bound is
// deliberately NOT like that: no genesis path runs through TokenBuilder.Build,
// so gating Build cannot desynchronise export from validation. That is a
// property of the call graph, not of any single comparison, so this test
// asserts it structurally:
//
//   - the bound has exactly one home (builder.go) and never appears elsewhere
//     in the package -- in particular not in genesis.go, whose per-NFT loop is
//     the validate/import side;
//   - both exits of TokenBuilder.Build call it (one definition + two call
//     sites), because Build returning early from only one branch is the
//     "synced only halfway" shape the class side shipped with for schema+data.
//
// Reverse control: adding `ValidateTokenMetadataBounds(nft.GetData())` to
// types/genesis.go, or deleting either call from builder.go, turns this test
// red. Both were exercised when the guard was written.
func TestTokenMetadataBoundStaysOnTheReceiveSide(t *testing.T) {
	const canonicalFile = "builder.go"

	sources := readNonTestGoFiles(t, ".")

	for name, src := range sources {
		if name == canonicalFile {
			continue
		}
		require.NotContains(t, src, "MaxTokenDataLen",
			"%s names the token metadata bound; it must stay in %s, re-derived nowhere", name, canonicalFile)
		require.NotContains(t, src, "ValidateTokenMetadataBounds(",
			"%s calls the token metadata predicate; it must stay in %s", name, canonicalFile)
	}

	require.Contains(t, sources[canonicalFile], "ValidateTokenMetadataBounds(")

	// One definition plus exactly two call sites: the two exits of
	// TokenBuilder.Build. Comments are stripped by readNonTestGoFiles, so this
	// counts code only.
	require.Equal(t, 3, strings.Count(sources[canonicalFile], "ValidateTokenMetadataBounds("),
		"builder.go must hold one definition and exactly two call sites -- "+
			"both exits of TokenBuilder.Build must enforce the bound")

	// The validate/import side must not gate on it. This is the assertion that
	// would fail if the F-001 asymmetry were reintroduced here.
	require.NotContains(t, sources["genesis.go"], "ValidateTokenMetadataBounds(",
		"ValidateGenesis must not enforce the token bound: the export carries "+
			"pre-existing oversized token data verbatim, so a validate-side gate "+
			"would reject the chain's own backup")
	require.NotContains(t, sources["genesis.go"], "MaxTokenDataLen")
}

// TestTokenDataBoundDoesNotReuseTheDenomConstant guards the naming decision
// recorded on MaxTokenDataLen. The token blob and the denom blob are different
// fields on different records; expressing the token bound through the denom
// constant would imply a coupling that does not exist, and would make the two
// impossible to change independently. It also re-states, for the token side,
// what metadata_bounds_test.go already requires of builder.go for the denom
// side.
func TestTokenDataBoundDoesNotReuseTheDenomConstant(t *testing.T) {
	require.Greater(t, MaxTokenDataLen, 0, "sanity: the bound must be set")

	src := readNonTestGoFiles(t, ".")["builder.go"]
	require.NotContains(t, src, "MaxDenomDataLen",
		"the token bound must not be expressed through the denom constant; "+
			"builder.go may reach the denom bound only via ValidateDenomMetadataBounds")
	require.Contains(t, src, "ValidateDenomMetadataBounds(",
		"the class-side receive gate must still share the denom predicate")
}
