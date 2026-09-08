package keeper

import (
	"testing"

	errorsmod "cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

const (
	guardContract = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	guardClassID  = "class-1"
)

func guardMsg(tokenIDs, nftIDs []string) *types.MsgConvertERC721 {
	return &types.MsgConvertERC721{
		EvmContractAddress: guardContract,
		EvmTokenIds:        tokenIDs,
		CosmosTokenIds:     nftIDs,
		ClassId:            guardClassID,
	}
}

func TestValidateNoMappingConflict_RejectsBoundToken(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	require.NoError(t, k.SetNFTPairs(ctx, guardContract, "1", guardClassID, "nft-bound"))

	// Token "1" is bound to nft-bound; asking to convert it as nft-other would
	// release a module-escrowed token that belongs to a different NFT.
	err := k.validateNoMappingConflict(ctx, guardMsg([]string{"1"}, []string{"nft-other"}))
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, types.ErrNFTMappingConflict), "got %v", err)
}

func TestValidateNoMappingConflict_AllowsMatchingBinding(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	require.NoError(t, k.SetNFTPairs(ctx, guardContract, "1", guardClassID, "nft-bound"))

	// The caller converts exactly the NFT the token is bound to.
	require.NoError(t, k.validateNoMappingConflict(ctx, guardMsg([]string{"1"}, []string{"nft-bound"})))
}

func TestValidateNoMappingConflict_AllowsUnboundToken(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	require.NoError(t, k.validateNoMappingConflict(ctx, guardMsg([]string{"99"}, []string{"nft-new"})))
}

func TestValidateNoMappingConflict_RejectsAnyConflictingBatchEntry(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	require.NoError(t, k.SetNFTPairs(ctx, guardContract, "2", guardClassID, "nft-bound"))

	// Only the second entry conflicts; the whole batch must still be rejected.
	err := k.validateNoMappingConflict(ctx, guardMsg([]string{"1", "2"}, []string{"nft-a", "nft-b"}))
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, types.ErrNFTMappingConflict), "got %v", err)
}

// convertEvm2Cosmos must reject a conflicting binding before it mints or
// transfers the NFT — the conflict must not surface only after side effects
// exist.
//
// The minimal harness leaves the EVM keeper nil, so reaching any EVM call
// panics; a clean pass also proves the guard runs before those calls.
func TestConvertEvm2Cosmos_RejectsConflictBeforeAnySideEffect(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	require.NoError(t, k.SetNFTPairs(ctx, guardContract, "1", guardClassID, "nft-bound"))

	pair := types.NewTokenPair(common.HexToAddress(guardContract), guardClassID)

	_, err := k.convertEvm2Cosmos(ctx, pair, guardMsg([]string{"1"}, []string{"nft-other"}), common.Address{})
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, types.ErrNFTMappingConflict), "got %v", err)

	// No side effect: the existing binding is untouched.
	require.Equal(t,
		types.CreateNFTUID(guardClassID, "nft-bound"),
		string(k.GetNFTPairByContractTokenID(ctx, guardContract, "1")),
	)
	// The rejected NFT id must not have been written anywhere.
	require.Empty(t, k.GetTokenUIDPairByNFTUID(ctx, types.CreateNFTUID(guardClassID, "nft-other")))
}

func TestConvertEvm2Cosmos_RejectsBatchSizeOverLimit(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	pair := types.NewTokenPair(common.HexToAddress(guardContract), guardClassID)

	tokenIDs := make([]string, maxERC721BatchSize+1)
	nftIDs := make([]string, maxERC721BatchSize+1)
	for i := range tokenIDs {
		tokenIDs[i] = "1"
		nftIDs[i] = "nft"
	}

	_, err := k.convertEvm2Cosmos(ctx, pair, guardMsg(tokenIDs, nftIDs), common.Address{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds maximum")
}

func TestConvertEvm2Cosmos_RejectsLengthMismatch(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	pair := types.NewTokenPair(common.HexToAddress(guardContract), guardClassID)

	_, err := k.convertEvm2Cosmos(ctx, pair, guardMsg([]string{"1", "2"}, []string{"nft-a"}), common.Address{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "length mismatch")
}
