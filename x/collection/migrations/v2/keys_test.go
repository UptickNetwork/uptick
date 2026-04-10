package v2

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestLegacyStoreKeys(t *testing.T) {
	require.Equal(t, append(append([]byte{}, PrefixDenom...), []byte("/denom-1")...), KeyDenom("denom-1"))
	require.Equal(t, append(append([]byte{}, PrefixDenomName...), []byte("/name-1")...), KeyDenomName("name-1"))
	require.Equal(t, append(append([]byte{}, PrefixCollection...), []byte("/denom-1")...), KeyCollection("denom-1"))
	require.Equal(t, append(append([]byte{}, PrefixNFT...), []byte("/denom-1/token-1")...), KeyNFT("denom-1", "token-1"))
}

func TestKeyOwner(t *testing.T) {
	addr := sdk.AccAddress("owner-addr-1")
	got := KeyOwner(addr, "denom-1", "token-1")
	expected := append(append([]byte{}, PrefixOwners...), []byte("/"+addr.String()+"/denom-1/token-1")...)
	require.Equal(t, expected, got)

	// nil address keeps only prefix and delimiter
	require.Equal(t, append(append([]byte{}, PrefixOwners...), delimiter...), KeyOwner(nil, "denom", "token"))
}
