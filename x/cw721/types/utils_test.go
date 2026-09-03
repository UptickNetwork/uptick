package types

import (
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestSanitizeCW721Name(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name unchanged",
			input:    "MyCoolNFT",
			expected: "MyCoolNFT",
		},
		{
			name:     "leading numbers removed",
			input:    "123MyNFT",
			expected: "MyNFT",
		},
		{
			name:     "special characters removed",
			input:    "My@#$NFT!",
			expected: "MyNFT",
		},
		{
			name:     "ibc prefix removed correctly",
			input:    "ibc/MyNFT",
			expected: "MyNFT",
		},
		{
			name:     "nested ibc prefixes removed",
			input:    "ibc/ibc/MyNFT",
			expected: "MyNFT",
		},
		{
			name:     "cw721 prefix removed correctly",
			input:    "cw721/MyNFT",
			expected: "MyNFT",
		},
		{
			name:     "nested cw721 prefixes removed",
			input:    "cw721/cw721/MyNFT",
			expected: "MyNFT",
		},
		{
			name:     "mixed ibc and cw721 prefixes removed",
			input:    "ibc/cw721/MyNFT",
			expected: "MyNFT",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only numbers",
			input:    "1234567890",
			expected: "",
		},
		{
			name:     "very long name truncated",
			input:    "N" + string(make([]byte, 200)),
			expected: "N",
		},
		{
			name:     "slash preserved",
			input:    "MyNFT/V2",
			expected: "MyNFT/V2",
		},
		{
			name:     "hyphen preserved",
			input:    "My-NFT-V2",
			expected: "My-NFT-V2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeCW721Name(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestEqualMetadata_Equal(t *testing.T) {
	a := banktypes.Metadata{
		Base:        "base",
		Description: "desc",
		Display:     "display",
		Name:        "name",
		Symbol:      "SYM",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "base", Exponent: 0, Aliases: []string{}},
			{Denom: "display", Exponent: 6, Aliases: []string{"alias"}},
		},
	}
	b := banktypes.Metadata{
		Base:        "base",
		Description: "desc",
		Display:     "display",
		Name:        "name",
		Symbol:      "SYM",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "base", Exponent: 0, Aliases: []string{}},
			{Denom: "display", Exponent: 6, Aliases: []string{"alias"}},
		},
	}
	err := EqualMetadata(a, b)
	require.NoError(t, err)
}

func TestEqualMetadata_DifferentBase(t *testing.T) {
	a := banktypes.Metadata{Base: "uatom"}
	b := banktypes.Metadata{Base: "uosmo"}
	err := EqualMetadata(a, b)
	require.Error(t, err)
}

func TestEqualMetadata_DifferentDenomUnits(t *testing.T) {
	a := banktypes.Metadata{
		Base:       "base",
		DenomUnits: []*banktypes.DenomUnit{{Denom: "base", Exponent: 0}},
	}
	b := banktypes.Metadata{
		Base:       "base",
		DenomUnits: []*banktypes.DenomUnit{{Denom: "base", Exponent: 6}},
	}
	err := EqualMetadata(a, b)
	require.Error(t, err)
}

func TestEqualStringSlice(t *testing.T) {
	require.True(t, EqualStringSlice([]string{"a", "b"}, []string{"a", "b"}))
	require.False(t, EqualStringSlice([]string{"a", "b"}, []string{"a", "c"}))
	require.False(t, EqualStringSlice([]string{"a"}, []string{"a", "b"}))
	require.False(t, EqualStringSlice([]string{"a", "b"}, []string{"a"}))
	require.True(t, EqualStringSlice([]string{}, []string{}))
	require.True(t, EqualStringSlice(nil, []string{}))
	require.False(t, EqualStringSlice(nil, []string{"a"}))
}

func TestCreateClassIDFromContractAddress(t *testing.T) {
	addr := "0x1234567890123456789012345678901234567890"
	classID := CreateClassIDFromContractAddress(addr)
	require.Contains(t, classID, DefaultPrefix)
	require.Contains(t, classID, "1234567890123456789012345678901234567890")
}

func TestCreateContractAddressFromClassID(t *testing.T) {
	classID := "uptick-abcdef123456"
	addr := CreateContractAddressFromClassID(classID)
	require.Equal(t, "abcdef123456", addr)
}

func TestCreateClassIDAndContractAddress_RoundTrip(t *testing.T) {
	addr := "0x1234567890abcdef"
	classID := CreateClassIDFromContractAddress(addr)
	recovered := CreateContractAddressFromClassID(classID)
	require.Equal(t, "1234567890abcdef", recovered)
}

func TestGetNFTFromUID_Valid(t *testing.T) {
	nftID, classID := GetNFTFromUID("nft123,class456")
	require.Equal(t, "nft123", nftID)
	require.Equal(t, "class456", classID)
}

func TestGetNFTFromUID_Invalid(t *testing.T) {
	// No comma
	nftID, classID := GetNFTFromUID("single")
	require.Equal(t, "", nftID)
	require.Equal(t, "", classID)

	// Empty
	nftID3, classID3 := GetNFTFromUID("")
	require.Equal(t, "", nftID3)
	require.Equal(t, "", classID3)

	// Leading comma (empty first field)
	nftID4, classID4 := GetNFTFromUID(",y")
	require.Equal(t, "", nftID4)
	require.Equal(t, "", classID4)

	// Trailing comma (empty second field)
	nftID5, classID5 := GetNFTFromUID("x,")
	require.Equal(t, "", nftID5)
	require.Equal(t, "", classID5)
}

func TestGetNFTFromUID_CommaInFirstField(t *testing.T) {
	// L-3: the UID is "<content>,<context>"; the context (class id / contract)
	// never contains a comma, so a comma in the first field is handled by
	// splitting on the LAST comma.
	nftID, classID := GetNFTFromUID("a,b,c")
	require.Equal(t, "a,b", nftID)
	require.Equal(t, "c", classID)

	tok, addr := GetNFTFromUID("tok,en,0x1234")
	require.Equal(t, "tok,en", tok)
	require.Equal(t, "0x1234", addr)
}

func TestCreateTokenUID(t *testing.T) {
	uid := CreateTokenUID("0x1234", "token-1")
	require.Equal(t, "token-1,0x1234", uid)
}

func TestCreateNFTUID(t *testing.T) {
	uid := CreateNFTUID("class-1", "nft-1")
	require.Equal(t, "nft-1,class-1", uid)
}

func TestCreateNFTIDFromTokenID(t *testing.T) {
	nftID := CreateNFTIDFromTokenID("abc123")
	require.Contains(t, nftID, DefaultPrefix)
}

func TestCreateTokenIDFromNFTID(t *testing.T) {
	require.Equal(t, "abc123", CreateTokenIDFromNFTID(CreateNFTIDFromTokenID("abc123")))
	require.Equal(t, "1234", CreateTokenIDFromNFTID("uptick-1234"))
	require.Equal(t, "cool-nft", CreateTokenIDFromNFTID("cool-nft"))
}

func TestRemoveAddress0x(t *testing.T) {
	// Helper function declared in package - test via known exported functions
	classID := CreateClassIDFromContractAddress("0xABCDEF")
	require.Contains(t, classID, "ABCDEF")
	require.NotContains(t, classID, "0x")
}
