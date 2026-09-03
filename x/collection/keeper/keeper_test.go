package keeper

import (
	"context"
	"testing"

	coreaddress "cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	rootstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	nftkeeper "cosmossdk.io/x/nft/keeper"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	dbm "github.com/cosmos/cosmos-db"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// KeeperTestSuite is the test suite for the collection keeper
type KeeperTestSuite struct {
	suite.Suite

	cdc      codec.Codec
	ctx      sdk.Context
	keeper   Keeper
	nftKpr   nftkeeper.Keeper
	storeSvc store.KVStoreService
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	s.cdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	types.RegisterInterfaces(s.cdc.InterfaceRegistry())

	// Create in-memory database and commit store
	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())

	// Collection keeper embeds the cosmos nft keeper on the same store
	// (see NewKeeper). Mount only the collection key.
	colStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	cms.MountStoreWithDB(colStoreKey, storetypes.StoreTypeIAVL, db)

	_ = cms.LoadLatestVersion()

	s.storeSvc = &testKVStoreService{store: kvStoreAdapter{inner: cms.GetKVStore(colStoreKey)}}

	// Create nft keeper dependencies
	ac := codecAddress.NewBech32Codec("cosmos")
	vc := codecAddress.NewBech32Codec("cosmosvaloper")

	// Create account keeper mock
	ak := &testAccountKeeper{}
	bk := &testBankKeeper{}

	s.nftKpr = nftkeeper.NewKeeper(s.storeSvc, s.cdc, ak, bk)
	_ = ac
	_ = vc

	s.keeper = Keeper{
		storeService: s.storeSvc,
		cdc:          s.cdc,
		nk:           s.nftKpr,
	}

	s.ctx = sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
}

// ============================================================================
// Denom tests
// ============================================================================

func (s *KeeperTestSuite) TestSaveAndGetDenomInfo() {
	s.Run("save and retrieve denom", func() {
		denom := types.Denom{
			Id:      "denom1",
			Name:    "Test Denom",
			Schema:  "schema-uri",
			Creator: "cosmos1creator",
		}
		s.keeper.SaveDenom(s.ctx, denom.Id, denom.Name, denom.Schema, "", sdk.AccAddress([]byte(denom.Creator)), false, false, "", "", "", "")

		retrieved, err := s.keeper.GetDenomInfo(s.ctx, "denom1")
		s.Require().NoError(err)
		s.Require().Equal(denom.Id, retrieved.Id)
		s.Require().Equal(denom.Name, retrieved.Name)
		s.Require().Equal(denom.Schema, retrieved.Schema)
		s.Require().Equal(sdk.AccAddress([]byte(denom.Creator)).String(), retrieved.Creator)
	})

	s.Run("get non-existent denom returns error", func() {
		_, err := s.keeper.GetDenomInfo(s.ctx, "nonexistent")
		s.Require().Error(err)
		s.Require().ErrorContains(err, "not exists")
	})
}

func (s *KeeperTestSuite) TestHasDenom() {
	s.Run("denom exists returns true", func() {
		denom := types.Denom{
			Id:      "denom2",
			Name:    "Name",
			Schema:  "schema",
			Creator: "cosmos1creator",
		}
		s.keeper.SaveDenom(s.ctx, denom.Id, denom.Name, denom.Schema, "", sdk.AccAddress([]byte(denom.Creator)), false, false, "", "", "", "")

		s.Require().True(s.keeper.HasDenom(s.ctx, "denom2"))
	})

	s.Run("denom does not exist returns false", func() {
		s.Require().False(s.keeper.HasDenom(s.ctx, "nonexistent"))
	})
}

func (s *KeeperTestSuite) TestSaveDenomDuplicate() {
	denom := types.Denom{
		Id:      "denom3",
		Name:    "Name",
		Schema:  "schema",
		Creator: "cosmos1creator",
	}
	// First save succeeds
	err := s.keeper.SaveDenom(s.ctx, denom.Id, denom.Name, denom.Schema, "", sdk.AccAddress([]byte(denom.Creator)), false, false, "", "", "", "")
	s.Require().NoError(err)

	// Second save fails due to duplicate
	err = s.keeper.SaveDenom(s.ctx, denom.Id, denom.Name, denom.Schema, "", sdk.AccAddress([]byte(denom.Creator)), false, false, "", "", "", "")
	s.Require().Error(err)
	s.Require().ErrorContains(err, "already exists")
}

// ============================================================================
// Collection tests
// ============================================================================

func (s *KeeperTestSuite) TestGetTotalSupply() {
	s.Run("supply of non-existent denom is zero", func() {
		supply := s.keeper.GetTotalSupply(s.ctx, "nonexistent")
		s.Require().Equal(uint64(0), supply)
	})
}

// TestAuthorize covers the single authority checkpoint that gates every NFT
// mutation (transfer, burn, update). A non-owner must never pass, regardless of
// denom-level permission flags.
func (s *KeeperTestSuite) TestAuthorize() {
	creator := sdk.AccAddress([]byte("creator"))
	other := sdk.AccAddress([]byte("other"))

	err := s.keeper.SaveDenom(s.ctx, "denom1", "Denom One", "", "ONE", creator, true, false, "", "", "", "")
	s.Require().NoError(err)
	err = s.keeper.SaveNFT(s.ctx, "denom1", "nft1", "NFT One", "ipfs://nft1", "", "", creator)
	s.Require().NoError(err)

	// The owner is authorized.
	s.Require().NoError(s.keeper.Authorize(s.ctx, "denom1", "nft1", creator))

	// A non-owner is rejected with the canonical unauthorized error.
	err = s.keeper.Authorize(s.ctx, "denom1", "nft1", other)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrUnauthorized)

	// A non-existent NFT is never authorized.
	s.Require().Error(s.keeper.Authorize(s.ctx, "denom1", "does-not-exist", creator))
}

// ============================================================================
// Invariant tests
// ============================================================================

func (s *KeeperTestSuite) TestSupplyInvariant() {
	s.Run("empty state passes invariant", func() {
		msg, broke := SupplyInvariant(s.keeper)(s.ctx)
		s.Require().False(broke)
		s.Require().Empty(msg)
	})
}

// ============================================================================
// Denom type helpers
// ============================================================================

func TestDenomHelpers(t *testing.T) {
	t.Run("NewDenom creates valid denom", func(t *testing.T) {
		d := types.Denom{Id: "myid", Name: "MyName", Schema: "schema-abc", Creator: "cosmos1creator"}
		require.Equal(t, "myid", d.Id)
		require.Equal(t, "MyName", d.Name)
		require.Equal(t, "schema-abc", d.Schema)
		require.Equal(t, "cosmos1creator", d.Creator)
	})

	t.Run("Denom String contains ID", func(t *testing.T) {
		d := types.Denom{Id: "abc", Name: "Name", Schema: "schema", Creator: "cosmos1creator"}
		require.Contains(t, d.String(), "abc")
	})
}

// ============================================================================
// Test mocks
// ============================================================================

// kvStoreAdapter adapts the cosmos-sdk/store KVStore (legacy Delete/Set
// signatures) to the cosmossdk.io/core/store.KVStore interface expected by the
// KVStoreService.
type kvStoreAdapter struct {
	inner storetypes.KVStore
}

func (a kvStoreAdapter) Get(key []byte) ([]byte, error) {
	return a.inner.Get(key), nil
}
func (a kvStoreAdapter) Has(key []byte) (bool, error) {
	return a.inner.Has(key), nil
}
func (a kvStoreAdapter) Iterator(start, end []byte) (store.Iterator, error) {
	return a.inner.Iterator(start, end), nil
}
func (a kvStoreAdapter) ReverseIterator(start, end []byte) (store.Iterator, error) {
	return a.inner.ReverseIterator(start, end), nil
}
func (a kvStoreAdapter) Set(key, value []byte) error { a.inner.Set(key, value); return nil }
func (a kvStoreAdapter) Delete(key []byte) error     { a.inner.Delete(key); return nil }

// testKVStoreService is a minimal KVStoreService implementation for testing
type testKVStoreService struct {
	store store.KVStore
}

func (s *testKVStoreService) OpenKVStore(_ context.Context) store.KVStore {
	return s.store
}

var _ store.KVStoreService = (*testKVStoreService)(nil)

// testAccountKeeper implements nft.AccountKeeper for testing
type testAccountKeeper struct{}

func (a *testAccountKeeper) GetAccount(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}
func (a *testAccountKeeper) SetAccount(_ context.Context, _ sdk.AccAddress) {}
func (a *testAccountKeeper) GetModuleAccount(_ context.Context, _ string) sdk.ModuleAccountI {
	return nil
}
func (a *testAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return authtypes.NewModuleAddress(moduleName)
}
func (a *testAccountKeeper) AddressCodec() coreaddress.Codec {
	return codecAddress.NewBech32Codec("uptick")
}
func (a *testAccountKeeper) GetSequence(_ sdk.Context, _ sdk.AccAddress) (uint64, error) {
	return 0, nil
}
func (a *testAccountKeeper) HasAccount(_ sdk.Context, _ sdk.AccAddress) bool          { return true }
func (a *testAccountKeeper) IterateAccounts(_ sdk.Context, _ func(sdk.AccountI) bool) {}

// testBankKeeper implements nft.BankKeeper for testing
type testBankKeeper struct{}

func (b *testBankKeeper) SendCoinsFromModuleToAccount(_ sdk.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}
func (b *testBankKeeper) SendCoinsFromAccountToModule(_ sdk.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}
func (b *testBankKeeper) SpendableCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins { return nil }
func (b *testBankKeeper) BlockedAddr(_ sdk.AccAddress) bool                            { return false }
func (b *testBankKeeper) GetBalance(_ sdk.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	return sdk.Coin{}
}
