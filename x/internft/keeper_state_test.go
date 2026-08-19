package internft

import (
	"context"
	"testing"

	coreaddress "cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	rootstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
)

func TestInterNftKeeperStateRoundTrip(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	collectiontypes.RegisterInterfaces(cdc.InterfaceRegistry())

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(collectiontypes.StoreKey)
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	storeSvc := &internftKVStoreService{store: internftKVStoreAdapter{inner: cms.GetKVStore(key)}}
	collectionKeeper := collectionkeeper.NewKeeper(cdc, storeSvc, &nftAccountKeeper{}, &nftBankKeeper{})
	ik := NewInterNftKeeper(cdc, collectionKeeper, &internftAccountKeeper{})

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	owner := sdk.AccAddress([]byte("owner"))
	recipient := sdk.AccAddress([]byte("recipient"))

	require.NoError(t, ik.CreateOrUpdateClass(ctx, "class1", "ipfs://class", ""))
	require.NoError(t, ik.Mint(ctx, "class1", "token1", "ipfs://token", "", owner))
	require.Equal(t, owner, ik.GetOwner(ctx, "class1", "token1"))

	token, ok := ik.GetNFT(ctx, "class1", "token1")
	require.True(t, ok)
	require.Equal(t, "token1", token.GetID())
	require.Equal(t, "ipfs://token", token.GetURI())

	require.NoError(t, ik.Transfer(ctx, "class1", "token1", "", recipient))
	require.Equal(t, recipient, ik.GetOwner(ctx, "class1", "token1"))

	require.NoError(t, ik.Burn(ctx, "class1", "token1"))
	_, ok = ik.GetNFT(ctx, "class1", "token1")
	require.False(t, ok)
}

type nftAccountKeeper struct{}

func (a *nftAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return authtypes.NewModuleAddress(moduleName)
}
func (a *nftAccountKeeper) GetAccount(context.Context, sdk.AccAddress) sdk.AccountI {
	return nil
}
func (a *nftAccountKeeper) AddressCodec() coreaddress.Codec {
	return codecAddress.NewBech32Codec("uptick")
}

type nftBankKeeper struct{}

func (b *nftBankKeeper) SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins {
	return nil
}

type internftAccountKeeper struct{}

func (a *internftAccountKeeper) NewAccountWithAddress(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	return authtypes.NewBaseAccountWithAddress(addr)
}
func (a *internftAccountKeeper) SetAccount(context.Context, sdk.AccountI) {}
func (a *internftAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return authtypes.NewModuleAddress(moduleName)
}

type internftKVStoreAdapter struct {
	inner storetypes.KVStore
}

func (a internftKVStoreAdapter) Get(key []byte) ([]byte, error) { return a.inner.Get(key), nil }
func (a internftKVStoreAdapter) Has(key []byte) (bool, error)   { return a.inner.Has(key), nil }
func (a internftKVStoreAdapter) Iterator(start, end []byte) (store.Iterator, error) {
	return a.inner.Iterator(start, end), nil
}
func (a internftKVStoreAdapter) ReverseIterator(start, end []byte) (store.Iterator, error) {
	return a.inner.ReverseIterator(start, end), nil
}
func (a internftKVStoreAdapter) Set(key, value []byte) error { a.inner.Set(key, value); return nil }
func (a internftKVStoreAdapter) Delete(key []byte) error     { a.inner.Delete(key); return nil }

type internftKVStoreService struct {
	store store.KVStore
}

func (s *internftKVStoreService) OpenKVStore(context.Context) store.KVStore { return s.store }

var _ store.KVStoreService = (*internftKVStoreService)(nil)
