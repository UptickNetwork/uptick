package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
)

// TestTokenPairSetGet tests the full token pair lifecycle:
// Set → Get → IsRegistered → Delete → (gone)
func TestTokenPairSetGet_Lifecycle(t *testing.T) {
	k, ctx := setupKeeper(t)

	tp := cw721types.NewTokenPair("0x1234567890123456789012345678901234567890", "class001")
	id := tp.GetID()

	// Should not be registered yet
	require.False(t, k.IsTokenPairRegistered(ctx, id))

	// Set
	k.SetTokenPair(ctx, tp)

	// Now registered
	require.True(t, k.IsTokenPairRegistered(ctx, id))

	// Get
	got, found := k.GetTokenPair(ctx, id)
	require.True(t, found)
	require.Equal(t, tp.Cw721Address, got.Cw721Address)
	require.Equal(t, tp.ClassId, got.ClassId)

	// Delete
	k.DeleteTokenPair(ctx, tp)

	// Should be gone
	_, found = k.GetTokenPair(ctx, id)
	require.False(t, found)
	require.False(t, k.IsTokenPairRegistered(ctx, id))
}

func TestTokenPairSetMultiple(t *testing.T) {
	k, ctx := setupKeeper(t)

	pairs := []cw721types.TokenPair{
		cw721types.NewTokenPair("0xAAAA00000000000000000000000000000000AAAA", "class-a"),
		cw721types.NewTokenPair("0xBBBB00000000000000000000000000000000BBBB", "class-b"),
		cw721types.NewTokenPair("0xCCCC00000000000000000000000000000000CCCC", "class-c"),
	}

	for _, tp := range pairs {
		k.SetTokenPair(ctx, tp)
	}

	allPairs := k.GetTokenPairs(ctx)
	require.Len(t, allPairs, 3)
}

func TestTokenPairGet_NilID(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, found := k.GetTokenPair(ctx, nil)
	require.False(t, found)
}

func TestCW721Map_SetGet(t *testing.T) {
	k, ctx := setupKeeper(t)

	tp := cw721types.NewTokenPair("0xDEAD00000000000000000000000000000000DEAD", "class-dead")
	id := tp.GetID()

	k.SetTokenPair(ctx, tp)
	k.SetCW721Map(ctx, tp.Cw721Address, id)

	// Get from map
	gotID := k.GetCW721Map(ctx, tp.Cw721Address)
	require.Equal(t, id, gotID)

	// IsCW721Registered
	require.True(t, k.IsCW721Registered(ctx, tp.Cw721Address))

	// Not registered for unknown address
	require.False(t, k.IsCW721Registered(ctx, "0x0000000000000000000000000000000000000000"))
}

func TestClassMap_SetGet(t *testing.T) {
	k, ctx := setupKeeper(t)

	tp := cw721types.NewTokenPair("0xBEEF00000000000000000000000000000000BEEF", "class-beef")
	id := tp.GetID()

	k.SetTokenPair(ctx, tp)
	k.SetClassMap(ctx, tp.ClassId, id)

	// Get from map
	gotID := k.GetClassMap(ctx, tp.ClassId)
	require.Equal(t, id, gotID)

	// IsClassRegistered
	require.True(t, k.IsClassRegistered(ctx, tp.ClassId))

	// Not registered for unknown class
	require.False(t, k.IsClassRegistered(ctx, "unknown-class"))
}

func TestGetTokenPairID(t *testing.T) {
	k, ctx := setupKeeper(t)

	tp := cw721types.NewTokenPair("0xCAFE00000000000000000000000000000000CAFE", "class-cafe")
	id := tp.GetID()

	k.SetTokenPair(ctx, tp)
	k.SetCW721Map(ctx, tp.Cw721Address, id)
	k.SetClassMap(ctx, tp.ClassId, id)

	// Lookup by cw721 address
	gotID := k.GetTokenPairID(ctx, tp.Cw721Address)
	require.Equal(t, id, gotID)

	// Lookup by class ID
	gotID = k.GetTokenPairID(ctx, tp.ClassId)
	require.Equal(t, id, gotID)
}

func TestNFTPairMapping_SetGet(t *testing.T) {
	k, ctx := setupKeeper(t)

	contractAddr := "0xCAFE00000000000000000000000000000000CAFE"
	tokenID := "42"
	classID := "class-nft"
	nftID := "nft-42"

	k.SetNFTPairByContractTokenID(ctx, contractAddr, tokenID, classID, nftID)

	// Get by contract+token
	got := k.GetNFTPairByContractTokenID(ctx, contractAddr, tokenID)
	require.NotEmpty(t, got)

	// Get reverse mapping
	k.SetNFTPairByClassNFTID(ctx, classID, nftID, contractAddr, tokenID)
	gotReverse := k.GetNFTPairByClassNFTID(ctx, classID, nftID)
	require.NotEmpty(t, gotReverse)
}

func TestNFTPairMapping_Delete(t *testing.T) {
	k, ctx := setupKeeper(t)

	contractAddr := "0xCAFE00000000000000000000000000000000CAFE"
	tokenID := "42"
	classID := "class-nft"
	nftID := "nft-42"

	// Set
	k.SetNFTPairByContractTokenID(ctx, contractAddr, tokenID, classID, nftID)

	// Verify exists
	got := k.GetNFTPairByContractTokenID(ctx, contractAddr, tokenID)
	require.NotEmpty(t, got)

	// Delete
	k.DeleteNFTPairByTokenID(ctx, contractAddr, tokenID)
	got = k.GetNFTPairByContractTokenID(ctx, contractAddr, tokenID)
	require.Empty(t, got)
}

func TestSetNFTPairs_Idempotent(t *testing.T) {
	k, ctx := setupKeeper(t)

	contractAddr := "0xCAFE00000000000000000000000000000000CAFE"
	tokenID := "42"
	classID := "class-idem"
	nftID := "nft-42"

	// First call sets the mapping
	k.SetNFTPairs(ctx, contractAddr, tokenID, classID, nftID)
	first := k.GetNFTPairByContractTokenID(ctx, contractAddr, tokenID)
	require.NotEmpty(t, first)

	// Second call with same params is idempotent
	k.SetNFTPairs(ctx, contractAddr, tokenID, classID, nftID)
	second := k.GetNFTPairByContractTokenID(ctx, contractAddr, tokenID)
	require.Equal(t, first, second)
}

func TestGetPair_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, err := k.GetPair(ctx, "unknown-token")
	require.Error(t, err)
	require.ErrorIs(t, err, cw721types.ErrTokenPairNotFound)
}

func TestGetPair_Found(t *testing.T) {
	k, ctx := setupKeeper(t)

	tp := cw721types.NewTokenPair("0xABCD00000000000000000000000000000000ABCD", "class-abcd")
	id := tp.GetID()

	k.SetTokenPair(ctx, tp)
	k.SetCW721Map(ctx, tp.Cw721Address, id)

	got, err := k.GetPair(ctx, tp.Cw721Address)
	require.NoError(t, err)
	require.Equal(t, tp.Cw721Address, got.Cw721Address)
	require.Equal(t, tp.ClassId, got.ClassId)
}

func TestCW721Map_Delete(t *testing.T) {
	k, ctx := setupKeeper(t)

	addr := "uptick1cw721contract"
	id := []byte("pair-id")
	k.SetCW721Map(ctx, addr, id)
	require.Equal(t, id, k.GetCW721Map(ctx, addr))

	k.DeleteCW721Map(ctx, addr)
	require.Empty(t, k.GetCW721Map(ctx, addr))
	require.False(t, k.IsCW721Registered(ctx, addr))
}

func TestGetWasmCodeID(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, err := k.GetWasmCodeID(ctx)
	require.ErrorIs(t, err, cw721types.ErrCW721CodeNotFound)

	k.SetWasmCode(ctx, cw721types.ModuleName, 7)
	codeID, err := k.GetWasmCodeID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(7), codeID)

	k.SetWasmCode(ctx, cw721types.AccModuleAddress.String(), 9)
	codeID, err = k.GetWasmCodeID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(9), codeID)
}
