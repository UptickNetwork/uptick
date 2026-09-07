package erc721

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"

	"github.com/UptickNetwork/uptick/x/erc721/keeper"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

const (
	rtContract = "0x1111111111111111111111111111111111111111"
	rtOwner    = "0x2222222222222222222222222222222222222222"
)

func newRoundTripKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(types.StoreKey + "-t")
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(key, cdc, nil, nftkeeper.Keeper{}, nil, ibcnfttransferkeeper.Keeper{})
	ctx := testutil.DefaultContext(key, tkey)
	return k, ctx
}

// seedPairAndRuntimeState registers a collection pair and writes the
// per-token runtime state: a bidirectional conversion binding (with a custom,
// non-derivable token ID) and the IBC refund receiver.
func seedPairAndRuntimeState(t *testing.T, k keeper.Keeper, ctx sdk.Context) (tokenUID, nftUID string) {
	t.Helper()

	contract := common.HexToAddress(rtContract)
	pair := types.NewTokenPair(contract, "kitty")
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, contract, pair.GetID())

	tokenUID = types.CreateTokenUID(rtContract, "42")
	nftUID = types.CreateNFTUID("kitty", "custom-nft")
	k.SetNFTUIDPairByTokenUID(ctx, tokenUID, nftUID)
	k.SetNFTUIDPairByNFTUID(ctx, nftUID, tokenUID)

	k.SetEvmAddressByContractTokenId(ctx, rtContract, "42", rtOwner)

	return tokenUID, nftUID
}

// TestGenesisRoundTripPreservesPerTokenState is a regression test: export
// must carry the per-token bindings and refund receivers, and importing that
// export into a fresh store must restore them exactly. The old export (params
// + collection pairs only) failed both halves.
func TestGenesisRoundTripPreservesPerTokenState(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)
	tokenUID, nftUID := seedPairAndRuntimeState(t, k, ctx)

	exported := ExportGenesis(ctx, k)

	require.Len(t, exported.NftUidPairs, 1, "per-token binding missing from export")
	require.Equal(t, tokenUID, exported.NftUidPairs[0].TokenUid)
	require.Equal(t, nftUID, exported.NftUidPairs[0].NftUid)
	require.Len(t, exported.RefundReceivers, 1, "refund receiver missing from export")
	require.Equal(t, rtContract, exported.RefundReceivers[0].EvmContractAddress)
	require.Equal(t, "42", exported.RefundReceivers[0].TokenId)
	require.Equal(t, rtOwner, exported.RefundReceivers[0].EvmAddress)

	// Import into a fresh store and read everything back. InitGenesis runs in
	// two phases: the collection-level TokenPairs loop, then
	// importPerTokenState. The per-token half is the exact code path (the
	// account-keeper plumbing in InitGenesis itself cannot be unit-tested
	// because SDK v0.53's authkeeper.AccountKeeper is a concrete struct).
	k2, ctx2 := newRoundTripKeeper(t)
	importedPair := types.NewTokenPair(common.HexToAddress(rtContract), "kitty")
	k2.SetTokenPair(ctx2, importedPair)
	k2.SetClassMap(ctx2, importedPair.ClassId, importedPair.GetID())
	k2.SetERC721Map(ctx2, common.HexToAddress(rtContract), importedPair.GetID())
	require.NoError(t, importPerTokenState(ctx2, k2, *exported))

	require.Equal(t, []byte(nftUID), k2.GetNFTUIDPairByTokenUID(ctx2, tokenUID))
	require.Equal(t, []byte(tokenUID), k2.GetTokenUIDPairByNFTUID(ctx2, nftUID))
	require.Equal(t, []byte(rtOwner), k2.GetEvmAddressByContractTokenId(ctx2, rtContract, "42"))
	require.Equal(t, []byte(rtOwner), k2.GetEvmRefundReceiver(ctx2, rtContract, "42", "42"))

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

	k.SetEvmAddressByContractTokenId(ctx, "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "7", rtOwner)

	require.Panics(t, func() {
		ExportGenesis(ctx, k)
	})
}

// TestImportRejectsDuplicateNFTUID: one-to-one must hold after import.
func TestImportRejectsDuplicateNFTUID(t *testing.T) {
	dupNft := types.CreateNFTUID("kitty", "nft1")
	gs := types.GenesisState{
		Params: types.DefaultParams(),
		NftUidPairs: []types.NFTUIDPair{
			{TokenUid: types.CreateTokenUID(rtContract, "1"), NftUid: dupNft},
			{TokenUid: types.CreateTokenUID(rtContract, "2"), NftUid: dupNft},
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
			{EvmContractAddress: "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", TokenId: "7", EvmAddress: rtOwner},
		},
	}

	k, ctx := newRoundTripKeeper(t)
	require.Error(t, importPerTokenState(ctx, k, gs))
}

// TestRefundKeySplitMatchesRegisteredContract covers the splitter used by
// ExportRefundReceivers, including case-insensitive fallback for hex
// addresses stored with differing case.
func TestRefundKeySplitMatchesRegisteredContract(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)
	seedPairAndRuntimeState(t, k, ctx)

	receivers, err := k.ExportRefundReceivers(ctx)
	require.NoError(t, err)
	require.Len(t, receivers, 1)
	require.True(t, strings.EqualFold(rtContract, receivers[0].EvmContractAddress))
	require.Equal(t, "42", receivers[0].TokenId)
}

func TestRefundKeySplitMatchesChecksumPair(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)

	checksum := "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	pair := types.TokenPair{Erc721Address: checksum, ClassId: "kitty"}
	k.SetTokenPair(ctx, pair)
	k.SetEvmAddressByContractTokenId(ctx, strings.ToLower(checksum), "42", rtOwner)

	receivers, err := k.ExportRefundReceivers(ctx)
	require.NoError(t, err)
	require.Len(t, receivers, 1)
	require.Equal(t, "42", receivers[0].TokenId)
}

func TestDeletePairPerTokenStateClearsOrphans(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)
	tokenUID, nftUID := seedPairAndRuntimeState(t, k, ctx)
	pair := types.NewTokenPair(common.HexToAddress(rtContract), "kitty")

	k.DeletePairPerTokenState(ctx, pair)

	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, tokenUID))
	require.Empty(t, k.GetTokenUIDPairByNFTUID(ctx, nftUID))
	require.Empty(t, k.GetEvmAddressByContractTokenId(ctx, rtContract, "42"))

	exported := ExportGenesis(ctx, k)
	require.Empty(t, exported.NftUidPairs)
	require.Empty(t, exported.RefundReceivers)
}

func TestGetTokenPairsPanicsOnCorruptValue(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(types.StoreKey + "-t")
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(key, cdc, nil, nftkeeper.Keeper{}, nil, ibcnfttransferkeeper.Keeper{})
	ctx := testutil.DefaultContext(key, tkey)

	store := ctx.KVStore(key)
	store.Set(append(append([]byte{}, types.KeyPrefixTokenPair...), []byte("x")...), []byte("not-a-proto"))

	require.Panics(t, func() { _ = k.GetTokenPairs(ctx) })
}

// TestGenesisRoundTripPreservesDualKeyRefundReceivers covers the runtime
// refund store writing TWO keys per receiver (cosmos NFT id and EVM token id)
// and requires export -> import to preserve both, not just the first one.
func TestGenesisRoundTripPreservesDualKeyRefundReceivers(t *testing.T) {
	k, ctx := newRoundTripKeeper(t)
	pair := types.NewTokenPair(common.HexToAddress(rtContract), "kitty")
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, common.HexToAddress(rtContract), pair.GetID())

	// cosmos NFT id "42" and EVM token id "99" for the same owner.
	k.SetEvmAddressByContractTokenId(ctx, rtContract, "42", rtOwner)
	k.SetEvmAddressByContractTokenId(ctx, rtContract, "99", rtOwner)

	exported := ExportGenesis(ctx, k)
	require.Len(t, exported.RefundReceivers, 2, "both refund keys must be exported")

	k2, ctx2 := newRoundTripKeeper(t)
	importedPair := types.NewTokenPair(common.HexToAddress(rtContract), "kitty")
	k2.SetTokenPair(ctx2, importedPair)
	k2.SetClassMap(ctx2, importedPair.ClassId, importedPair.GetID())
	k2.SetERC721Map(ctx2, common.HexToAddress(rtContract), importedPair.GetID())
	require.NoError(t, importPerTokenState(ctx2, k2, *exported))

	require.Equal(t, []byte(rtOwner), k2.GetEvmAddressByContractTokenId(ctx2, rtContract, "42"))
	require.Equal(t, []byte(rtOwner), k2.GetEvmAddressByContractTokenId(ctx2, rtContract, "99"))
	// The runtime lookup helper checks the cosmos-id key first, then the EVM
	// token-id key — both must resolve after the round trip.
	require.Equal(t, []byte(rtOwner), k2.GetEvmRefundReceiver(ctx2, rtContract, "42", "99"))

	// Re-export must reproduce the first export byte-for-byte.
	reExported := ExportGenesis(ctx2, k2)
	first, err := exported.Marshal()
	require.NoError(t, err)
	second, err := reExported.Marshal()
	require.NoError(t, err)
	require.Equal(t, first, second, "dual-key refund state must round-trip losslessly")
}
