package types

import (
	"strings"
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestSanitizeERC721Name(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"strips leading digits", "123MyToken", "MyToken"},
		{"removes invalid first char", "@abc", "abc"},
		{"keeps slash", "foo/bar", "foo/bar"},
		{"strips ibc prefix recursively", "ibc/ibc/erc721/name", "name"},
		{"strips erc721 prefix", "erc721/foo", "foo"},
		{"truncates to 128", strings.Repeat("a", 150), strings.Repeat("a", 128)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeERC721Name(tc.in)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestEqualStringSlice(t *testing.T) {
	t.Parallel()
	require.True(t, EqualStringSlice(nil, nil))
	require.True(t, EqualStringSlice([]string{"a"}, []string{"a"}))
	require.False(t, EqualStringSlice([]string{"a"}, []string{"b"}))
	require.False(t, EqualStringSlice([]string{"a"}, []string{"a", "b"}))
}

func TestEqualMetadata(t *testing.T) {
	t.Parallel()
	base := banktypes.Metadata{
		Description: "d",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "base", Exponent: 0, Aliases: []string{"x"}},
		},
		Base:    "base",
		Display: "disp",
		Name:    "n",
		Symbol:  "S",
	}
	other := base
	require.NoError(t, EqualMetadata(base, other))

	other.Symbol = "T"
	require.Error(t, EqualMetadata(base, other))

	dupUnits := base
	dupUnits.DenomUnits = append([]*banktypes.DenomUnit{}, base.DenomUnits...)
	dupUnits.DenomUnits = append(dupUnits.DenomUnits, &banktypes.DenomUnit{Denom: "u", Exponent: 6})
	require.Error(t, EqualMetadata(base, dupUnits))
}

func TestRemoveAddress0x(t *testing.T) {
	t.Parallel()
	require.Equal(t, "abcd", removeAddress0x("0xabcd"))
	require.Equal(t, "abcd", removeAddress0x("abcd"))
}

func TestCreateClassIDFromContractAddress(t *testing.T) {
	t.Parallel()
	addr := "0xAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAb"
	require.Equal(t, "uptick-AbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAb", CreateClassIDFromContractAddress(addr))
}

func TestCreateContractAddressFromClassID(t *testing.T) {
	t.Parallel()
	classID := "uptick-AbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAb"
	require.Equal(t, "AbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAbCdEfAb", CreateContractAddressFromClassID(classID))
}

func TestCreateNFTIDFromTokenID(t *testing.T) {
	t.Parallel()
	id := CreateNFTIDFromTokenID("0x00aa")
	require.Equal(t, "uptick00aa", id)
}

func TestCreateTokenUIDAndNFTUID(t *testing.T) {
	t.Parallel()
	require.Equal(t, "tid,caddr", CreateTokenUID("caddr", "tid"))
	require.Equal(t, "nid,cid", CreateNFTUID("cid", "nid"))
}

func TestGetNFTFromUID(t *testing.T) {
	t.Parallel()
	a, b := GetNFTFromUID("x,y")
	require.Equal(t, "x", a)
	require.Equal(t, "y", b)
	a, b = GetNFTFromUID("bad")
	require.Equal(t, "", a)
	require.Equal(t, "", b)
}
