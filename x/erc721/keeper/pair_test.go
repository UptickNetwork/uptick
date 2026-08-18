package keeper

import (
	"testing"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	store "cosmossdk.io/store"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/UptickNetwork/uptick/x/erc721/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func setupKeeperContext(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), nil)
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeDB, db)
	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, tmproto.Header{}, false, log.NewNopLogger())
	k := Keeper{
		storeKey: storeKey,
		cdc:      types.ModuleCdc,
	}

	return k, ctx
}

func TestKeeperGetTokenPairID(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	pair := types.NewTokenPair(common.HexToAddress("0x1111111111111111111111111111111111111111"), "class/one")
	id := pair.GetID()

	k.SetClassMap(ctx, pair.ClassId, id)
	k.SetERC721Map(ctx, pair.GetERC721Contract(), id)

	require.Equal(t, id, k.GetTokenPairID(ctx, pair.ClassId))
	require.Equal(t, id, k.GetTokenPairID(ctx, pair.Erc721Address))
}

func TestKeeperGetPair(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	t.Run("token id map not found", func(t *testing.T) {
		_, err := k.GetPair(ctx, "class/missing")
		require.Error(t, err)
		require.True(t, errorsmod.IsOf(err, types.ErrTokenPairNotFound))
	})

	t.Run("pair data not found", func(t *testing.T) {
		k.SetClassMap(ctx, "class/only-mapping", []byte("dangling-id"))
		_, err := k.GetPair(ctx, "class/only-mapping")
		require.Error(t, err)
		require.True(t, errorsmod.IsOf(err, types.ErrTokenPairNotFound))
	})

	t.Run("success", func(t *testing.T) {
		pair := types.NewTokenPair(common.HexToAddress("0x2222222222222222222222222222222222222222"), "class/exist")
		id := pair.GetID()
		k.SetTokenPair(ctx, pair)
		k.SetClassMap(ctx, pair.ClassId, id)

		got, err := k.GetPair(ctx, pair.ClassId)
		require.NoError(t, err)
		require.Equal(t, pair, got)
	})
}

func TestIsERC721Registered_LegacyBytesKey(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	addr := common.HexToAddress("0x3333333333333333333333333333333333333333")
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC721)
	store.Set(addr.Bytes(), []byte("legacy-id"))

	require.True(t, k.IsERC721Registered(ctx, addr))
}

func TestEvmRefundReceiver(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	owner := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	mixedContract := "0xBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBb"
	cosmosID := "nft-1"
	evmID := "42"

	k.SetEvmRefundReceiver(ctx, mixedContract, []string{cosmosID}, []string{evmID}, owner)

	got := k.GetEvmRefundReceiver(ctx, mixedContract, cosmosID, evmID)
	require.Equal(t, []byte(owner), got)

	got = k.GetEvmRefundReceiver(ctx, mixedContract, cosmosID, "unused")
	require.Equal(t, []byte(owner), got)

	got = k.GetEvmRefundReceiver(ctx, mixedContract, "missing", evmID)
	require.Equal(t, []byte(owner), got)

	require.Empty(t, k.GetEvmRefundReceiver(ctx, mixedContract, "missing", "missing"))
}
