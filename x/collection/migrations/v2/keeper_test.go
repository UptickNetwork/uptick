package v2

import (
	"testing"

	nftkeeper "cosmossdk.io/x/nft/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/stretchr/testify/require"
)

func TestStoreKeyBuilders(t *testing.T) {
	classID := "class-1"
	tokenID := "token-1"
	owner := sdk.AccAddress("owner-addr-1")

	require.Equal(t, append(append([]byte{}, nftkeeper.ClassTotalSupply...), []byte(classID)...), classTotalSupply(classID))
	require.Equal(t, append(append(append([]byte{}, nftkeeper.NFTKey...), []byte(classID)...), nftkeeper.Delimiter...), nftStoreKey(classID))
	require.Equal(
		t,
		append(append(append(append([]byte{}, nftkeeper.OwnerKey...), []byte(classID)...), nftkeeper.Delimiter...), []byte(tokenID)...),
		ownerStoreKey(classID, tokenID),
	)

	prefixOwner := address.MustLengthPrefix(owner)
	expectedOwnerClassKey := append(
		append(
			append(
				append([]byte{}, nftkeeper.NFTOfClassByOwnerKey...),
				prefixOwner...,
			),
			nftkeeper.Delimiter...,
		),
		[]byte(classID)...,
	)
	expectedOwnerClassKey = append(expectedOwnerClassKey, nftkeeper.Delimiter...)
	require.Equal(t, expectedOwnerClassKey, nftOfClassByOwnerStoreKey(owner, classID))
}

func TestUnsafeConvertHelpers(t *testing.T) {
	s := "hello-world"
	b := UnsafeStrToBytes(s)
	require.Equal(t, []byte("hello-world"), b)
	require.Equal(t, s, UnsafeBytesToStr(b))
}
