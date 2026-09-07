package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestGetNftData_AllEmpty(t *testing.T) {
	result, err := getNftData("", "", "", 0)
	require.NoError(t, err)
	require.NotEmpty(t, result)
}

func TestGetNftData_SavedOnly(t *testing.T) {
	result, err := getNftData("", "", "saved-value", 0)
	require.NoError(t, err)
	require.Equal(t, "saved-value", result)
}

func TestGetNftData_OrgOnly(t *testing.T) {
	result, err := getNftData("org-value", "", "", 0)
	require.NoError(t, err)
	require.Equal(t, "org-value", result)
}

func TestGetNftData_SavedEqualsOrg(t *testing.T) {
	result, err := getNftData("org-value", "", "org-value", 0)
	require.NoError(t, err)
	require.Equal(t, "org-value", result)
}

func TestGetNftData_SavedDiffersOrg(t *testing.T) {
	_, err := getNftData("org-value", "", "different-value", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not correct")
}

func TestGetNftData_NftType0_NftId(t *testing.T) {
	// nftType 0: nftId → NFT ID from token ID
	result, err := getNftData("", "0x1234", "", 0)
	require.NoError(t, err)
	require.Contains(t, result, cw721types.DefaultPrefix)
}

func TestGetNftData_NftType1_ClassId(t *testing.T) {
	// nftType 1: classId → class ID from contract address
	result, err := getNftData("", "0xabcdef", "", 1)
	require.NoError(t, err)
	require.Contains(t, result, cw721types.DefaultPrefix)
	require.Contains(t, result, "abcdef")
}

func TestGetNftData_NftType2_TokenId(t *testing.T) {
	// nftType 2: tokenId → token ID from NFT ID (strip uptick prefix)
	result, err := getNftData("", "uptick-1234", "", 2)
	require.NoError(t, err)
	require.Equal(t, "1234", result)
}

func TestGetNftData_NftType3_ContractAddress(t *testing.T) {
	// nftType 3: contract address → contract address from class ID
	_, err := getNftData("", "uptick-abcdef", "", 3)
	require.NoError(t, err)
}

func TestCreateNftDataByType(t *testing.T) {
	tests := []struct {
		name    string
		nftType int
		input   string
	}{
		{"nftId from tokenId", 0, "0x1234"},
		{"classId from address", 1, "0xabcdef"},
		{"tokenId from nftId", 2, "uptick-1234"},
		{"contract from classId", 3, "uptick-abcdef"},
		{"unknown returns empty", 99, "test"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := createNftDataByType(tc.input, tc.nftType)
			if tc.nftType == 99 {
				require.Empty(t, result)
			} else {
				require.NotEmpty(t, result)
			}
		})
	}
}

func TestGetNftDataErrorByType(t *testing.T) {
	tests := []struct {
		name    string
		nftType int
		errType error
	}{
		{"nft id error", 0, cw721types.ErrNftIdNotCorrect},
		{"class id error", 1, cw721types.ErrClassIdNotCorrect},
		{"token id error", 2, cw721types.ErrTokenIdNotCorrect},
		{"contract address error", 3, cw721types.ErrContractAddressNotCorrect},
		{"unknown returns nil", 99, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := getNftDataErrorByType("expected", "got", tc.nftType)
			if tc.nftType == 99 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.errType)
			}
		})
	}
}

func TestGetContractAddressAndTokenIds_ProvidedContract(t *testing.T) {
	k, ctx := setupKeeper(t)
	contract := sdk.AccAddress([]byte("cw721contractaddrxx")).String()

	_, _, err := k.GetContractAddressAndTokenIds(ctx, &cw721types.MsgConvertNFT{
		ClassId:         "class-1",
		NftIds:          []string{"nft-1"},
		ContractAddress: contract,
	})
	require.ErrorIs(t, err, cw721types.ErrContractAddressNotCorrect)
}

func TestGetContractAddressAndTokenIds_InvalidContract(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, _, err := k.GetContractAddressAndTokenIds(ctx, &cw721types.MsgConvertNFT{
		ClassId:         "class-1",
		NftIds:          []string{"nft-1"},
		ContractAddress: "not-bech32",
	})
	require.ErrorIs(t, err, cw721types.ErrContractAddressNotCorrect)
}

func TestGetContractAddressAndTokenIds_MissingCode(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, _, err := k.GetContractAddressAndTokenIds(ctx, &cw721types.MsgConvertNFT{
		ClassId: "class-1",
		NftIds:  []string{"nft-1"},
	})
	require.ErrorIs(t, err, cw721types.ErrCW721CodeNotFound)
}

// TestGetContractAddressAndTokenIds_RejectsTokenIDCollision reproduces the
// registry-poisoning attack for CW721: a caller's unbound NFT must not resolve
// to a token already bound to a different NFT (which would release the escrowed
// victim token).
func TestGetContractAddressAndTokenIds_RejectsTokenIDCollision(t *testing.T) {
	k, ctx := setupKeeper(t)
	contract := sdk.AccAddress([]byte("cw721contractaddrxx")).String()
	pair := cw721types.NewTokenPair(contract, "class-1")
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetCW721Map(ctx, pair.Cw721Address, pair.GetID())

	// A previously-converted victim token is bound to nft-victim.
	require.NoError(t, k.SetNFTPairs(ctx, contract, "7", "class-1", "nft-victim"))

	_, _, err := k.GetContractAddressAndTokenIds(ctx, &cw721types.MsgConvertNFT{
		ClassId:         "class-1",
		NftIds:          []string{"nft-attacker"},
		ContractAddress: contract,
		TokenIds:        []string{"7"},
	})
	require.ErrorIs(t, err, cw721types.ErrNFTMappingConflict)
}
