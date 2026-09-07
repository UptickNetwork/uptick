package cw721

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"

	"github.com/UptickNetwork/uptick/x/cw721/keeper"
	"github.com/UptickNetwork/uptick/x/cw721/types"
)

const (
	rtContract = "0xCAFE00000000000000000000000000000000CAFE"
	rtOwner    = "uptick1wjjvcy9t6ycnwssh69t6cf5zxzg6pk7pmhdznm"
)

func newRoundTripKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(types.StoreKey + "-t")
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(key, cdc, nil, collectionkeeper.Keeper{}, nil, nil)
	ctx := testutil.DefaultContext(key, tkey)
	return k, ctx
}

// seedPairAndRuntimeState registers a collection pair and writes the
// per-token runtime state: a bidirectional conversion binding (with a custom,
// non-derivable token ID) and the IBC refund receiver.
func seedPairAndRuntimeState(t *testing.T, k keeper.Keeper, ctx sdk.Context) (tokenUID, nftUID string) {
	t.Helper()

	pair := types.NewTokenPair(rtContract, "kitty")
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetCW721Map(ctx, pair.Cw721Address, pair.GetID())

	tokenUID = types.CreateTokenUID(rtContract, "42")
	nftUID = types.CreateNFTUID("kitty", "custom-nft")
	k.SetNFTUIDPairByTokenUID(ctx, tokenUID, nftUID)
	k.SetNFTUIDPairByNFTUID(ctx, nftUID, tokenUID)

	k.SetCwAddressByContractTokenId(ctx, rtContract, "42", rtOwner)

	return tokenUID, nftUID
}

// TestGenesisRoundTripPreservesPerTokenState is the H-02 regression: export
// must carry the per-token bindings and refund receivers, and importing that
// export into a fresh store must restore them exactly.
func TestGenesisRoundTripPreservesPerTokenState(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)
	tokenUID, nftUID := seedPairAndRuntimeState(t, k, ctx)

	exported := ExportGenesis(ctx, k)

	require.Len(t, exported.NftUidPairs, 1, "per-token binding missing from export (H-02)")
	require.Equal(t, tokenUID, exported.NftUidPairs[0].TokenUid)
	require.Equal(t, nftUID, exported.NftUidPairs[0].NftUid)
	require.Len(t, exported.RefundReceivers, 1, "refund receiver missing from export (H-02)")
	require.Equal(t, rtContract, exported.RefundReceivers[0].ContractAddress)
	require.Equal(t, "42", exported.RefundReceivers[0].TokenId)
	require.Equal(t, rtOwner, exported.RefundReceivers[0].Owner)

	// Import into a fresh store (replicating InitGenesis's two phases) and
	// read everything back.
	k2, ctx2 := newRoundTripKeeper(t)
	importedPair := types.NewTokenPair(rtContract, "kitty")
	k2.SetTokenPair(ctx2, importedPair)
	k2.SetClassMap(ctx2, importedPair.ClassId, importedPair.GetID())
	k2.SetCW721Map(ctx2, importedPair.Cw721Address, importedPair.GetID())
	require.NoError(t, importPerTokenState(ctx2, k2, *exported))

	require.Equal(t, []byte(nftUID), k2.GetNFTUIDPairByTokenUID(ctx2, tokenUID))
	require.Equal(t, []byte(tokenUID), k2.GetTokenUIDPairByNFTUID(ctx2, nftUID))
	require.Equal(t, []byte(rtOwner), k2.GetCwAddressByContractTokenId(ctx2, rtContract, "42"))

	// A re-export of the imported state must be byte-identical to the first
	// export: the genesis file is now a true snapshot, not a lossy summary.
	reExported := ExportGenesis(ctx2, k2)
	first, err := exported.Marshal()
	require.NoError(t, err)
	second, err := reExported.Marshal()
	require.NoError(t, err)
	require.Equal(t, first, second, "export -> import -> export must round-trip losslessly")
}

// TestExportRejectsInconsistentIndex: a forward entry without its reverse
// half is corrupt state; export must abort instead of writing a genesis that
// loses the binding.
func TestExportRejectsInconsistentIndex(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)
	seedPairAndRuntimeState(t, k, ctx)

	// Add a forward binding whose reverse half is missing.
	k.SetNFTUIDPairByTokenUID(
		ctx,
		types.CreateTokenUID(rtContract, "99"),
		types.CreateNFTUID("kitty", "ghost"),
	)

	require.Panics(t, func() {
		ExportGenesis(ctx, k)
	})
}

// TestExportRejectsOrphanedRefundRecord: a refund record whose contract is
// not a registered pair cannot be attributed; export must fail loudly rather
// than silently drop refund state.
func TestExportRejectsOrphanedRefundRecord(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)
	seedPairAndRuntimeState(t, k, ctx)

	k.SetCwAddressByContractTokenId(ctx, "0xdeadbeef0000000000000000000000000000beef", "7", rtOwner)

	require.Panics(t, func() {
		ExportGenesis(ctx, k)
	})
}

// TestImportRejectsDuplicateTokenUID: one-to-one must hold after import.
func TestImportRejectsDuplicateTokenUID(t *testing.T) {
	dupToken := types.CreateTokenUID(rtContract, "1")
	gs := types.GenesisState{
		Params: types.DefaultParams(),
		NftUidPairs: []types.NFTUIDPair{
			{TokenUid: dupToken, NftUid: types.CreateNFTUID("kitty", "nft1")},
			{TokenUid: dupToken, NftUid: types.CreateNFTUID("kitty", "nft2")},
		},
	}

	k, ctx := newRoundTripKeeper(t)
	require.Error(t, importPerTokenState(ctx, k, gs))
}

// TestImportRejectsUnregisteredRefundContract.
func TestImportRejectsUnregisteredRefundContract(t *testing.T) {
	gs := types.GenesisState{
		Params: types.DefaultParams(),
		RefundReceivers: []types.RefundReceiver{
			{ContractAddress: "0xdeadbeef0000000000000000000000000000beef", TokenId: "7", Owner: rtOwner},
		},
	}

	k, ctx := newRoundTripKeeper(t)
	require.Error(t, importPerTokenState(ctx, k, gs))
}

// TestRefundKeySplitMatchesRegisteredContract covers the splitter used by
// ExportRefundReceivers.
func TestRefundKeySplitMatchesRegisteredContract(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)
	seedPairAndRuntimeState(t, k, ctx)

	receivers, err := k.ExportRefundReceivers(ctx)
	require.NoError(t, err)
	require.Len(t, receivers, 1)
	require.True(t, strings.EqualFold(rtContract, receivers[0].ContractAddress))
	require.Equal(t, "42", receivers[0].TokenId)
}
