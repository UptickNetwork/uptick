package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDenomID(t *testing.T) {
	require.NoError(t, ValidateDenomID("abc"))
	require.NoError(t, ValidateDenomID("uptick-custom-denom"))
	require.Error(t, ValidateDenomID("A-!"))
	require.Error(t, ValidateDenomID("abc\x00def"))
	require.Error(t, ValidateDenomID("uptick-abc/def"))
	require.Error(t, ValidateDenomID("ibc-token"))
}

// A comma inside a denom ID would end up in the classId component of NFT UIDs
// (CreateNFTUID -> "<nftId>,<classId>") and break the last-comma round-trip in
// GetNFTFromUID. The "uptick-" prefix branch of ValidateDenomID bypasses the
// character-set regex, so comma rejection must cover both branches.
func TestValidateDenomID_RejectsComma(t *testing.T) {
	// regex branch (already rejected by the charset, pinned here explicitly)
	require.Error(t, ValidateDenomID("a,b"))
	// uptick- prefixed branch: previously accepted, must now be rejected
	require.Error(t, ValidateDenomID("uptick-a,b"))
	require.Error(t, ValidateDenomID("uptick-denom,with,commas"))
	// the UID parser round-trips only comma-free classIds
	require.NoError(t, ValidateDenomID("uptick-abcdef"))
}

func TestValidateTokenID(t *testing.T) {
	require.NoError(t, ValidateTokenID("abc"))
	require.Error(t, ValidateTokenID("ab"))
	require.Error(t, ValidateTokenID(strings.Repeat("a", MaxDenomLen+1)))
	require.Error(t, ValidateTokenID("ab\x00c"))
	require.Error(t, ValidateTokenID("ab/c"))
}

func TestValidateTokenURI(t *testing.T) {
	require.NoError(t, ValidateTokenURI("https://example.com/nft/1"))
	require.Error(t, ValidateTokenURI(strings.Repeat("u", MaxTokenURILen+1)))
}

func TestModifyAndModified(t *testing.T) {
	require.False(t, Modified(DoNotModify))
	require.True(t, Modified("new-value"))
	require.Equal(t, "origin", Modify("origin", DoNotModify))
	require.Equal(t, "new-value", Modify("origin", "new-value"))
}

func TestValidateKeywordsAndIsIBCDenom(t *testing.T) {
	require.Error(t, ValidateKeywords("ibc-token"))
	require.NoError(t, ValidateKeywords("custom-token"))
	require.True(t, IsIBCDenom("ibc/ABCDEF"))
	require.False(t, IsIBCDenom("denom"))
}
