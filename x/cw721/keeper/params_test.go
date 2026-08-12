package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	"github.com/stretchr/testify/require"
)

func setupKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()

	// Create a minimal codec that can marshal/unmarshal proto types
	interfaceRegistry := types.NewInterfaceRegistry()
	cw721types.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey(cw721types.StoreKey)

	// Create in-memory DB
	memDB := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	memDB.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
	err := memDB.LoadLatestVersion()
	require.NoError(t, err)

	ctx := sdk.NewContext(memDB, tmproto.Header{}, false, log.NewNopLogger()).WithKVGasConfig(storetypes.KVGasConfig())

	k := Keeper{
		storeKey: storeKey,
		cdc:      cdc,
		// accountKeeper, nftKeeper, wasmKeeper, ibcTransferKeeper are nil
		// but params/token_pairs operations don't call them
	}

	return k, ctx
}

func TestGetParams_Default(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := k.GetParams(ctx)
	require.True(t, params.EnableCw721)
	require.True(t, params.EnableEVMHook)
}

func TestSetParams_And_GetParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	newParams := cw721types.NewParams(false, false)
	err := k.SetParams(ctx, newParams)
	require.NoError(t, err)

	got := k.GetParams(ctx)
	require.False(t, got.EnableCw721)
	require.False(t, got.EnableEVMHook)
}

func TestGetEnableCw721(t *testing.T) {
	k, ctx := setupKeeper(t)
	// Default is true
	require.True(t, k.GetEnableCw721(ctx))

	// Set to false
	err := k.SetParams(ctx, cw721types.NewParams(false, false))
	require.NoError(t, err)
	require.False(t, k.GetEnableCw721(ctx))
}

func TestGetEnableEVMHook(t *testing.T) {
	k, ctx := setupKeeper(t)
	// Default is true
	require.True(t, k.GetEnableEVMHook(ctx))

	// Set to false
	err := k.SetParams(ctx, cw721types.NewParams(true, false))
	require.NoError(t, err)
	require.False(t, k.GetEnableEVMHook(ctx))
}

func TestSetParams_Override(t *testing.T) {
	k, ctx := setupKeeper(t)

	// First set
	err := k.SetParams(ctx, cw721types.NewParams(false, false))
	require.NoError(t, err)
	got := k.GetParams(ctx)
	require.False(t, got.EnableCw721)

	// Override
	err = k.SetParams(ctx, cw721types.NewParams(true, true))
	require.NoError(t, err)
	got = k.GetParams(ctx)
	require.True(t, got.EnableCw721)
	require.True(t, got.EnableEVMHook)
}
