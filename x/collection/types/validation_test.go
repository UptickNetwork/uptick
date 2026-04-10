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
}

func TestValidateTokenID(t *testing.T) {
	require.NoError(t, ValidateTokenID("abc"))
	require.Error(t, ValidateTokenID("ab"))
	require.Error(t, ValidateTokenID(strings.Repeat("a", MaxDenomLen+1)))
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
