package keeper

import (
	"testing"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/nft"
	nftkeeper "cosmossdk.io/x/nft/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

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
	s.cdc = codec.NewProtoCodec(nil)
	types.RegisterInterfaces(s.cdc.InterfaceRegistry())

	// Create in-memory database and commit store
	db := dbm.NewMemDB()
	cms := storetypes.NewCommitMultiStore(db, log.NewNopLogger(), nil)

	// Create store service for the nft module store key
	nftStoreKey := storetypes.NewKVStoreKey(nft.StoreKey)
	cms.MountStoreWithDB(nftStoreKey, storetypes.StoreTypeIAVL, db)

	// Create collection store key
	colStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	cms.MountStoreWithDB(colStoreKey, storetypes.StoreTypeIAVL, db)

	_ = cms.LoadLatestVersion()

	s.storeSvc = &testKVStoreService{store: cms.GetKVStore(colStoreKey)}

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

	s.ctx = sdk.NewContext(cms, sdk.Header{}, false, log.NewNopLogger())
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
		s.keeper.SaveDenom(s.ctx, denom)

		retrieved, err := s.keeper.GetDenomInfo(s.ctx, "denom1")
		s.Require().NoError(err)
		s.Require().Equal(denom.Id, retrieved.Id)
		s.Require().Equal(denom.Name, retrieved.Name)
		s.Require().Equal(denom.Schema, retrieved.Schema)
		s.Require().Equal(denom.Creator, retrieved.Creator)
	})

	s.Run("get non-existent denom returns error", func() {
		_, err := s.keeper.GetDenomInfo(s.ctx, "nonexistent")
		s.Require().Error(err)
		s.Require().ErrorContains(err, "not found")
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
		s.keeper.SaveDenom(s.ctx, denom)

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
	err := s.keeper.SaveDenom(s.ctx, denom)
	s.Require().NoError(err)

	// Second save fails due to duplicate
	err = s.keeper.SaveDenom(s.ctx, denom)
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
		d := types.NewDenom("myid", "MyName", "schema-abc", "cosmos1creator")
		require.Equal(t, "myid", d.Id)
		require.Equal(t, "MyName", d.Name)
		require.Equal(t, "schema-abc", d.Schema)
		require.Equal(t, "cosmos1creator", d.Creator)
	})

	t.Run("Denom String returns ID", func(t *testing.T) {
		d := types.NewDenom("abc", "Name", "schema", "cosmos1creator")
		require.Equal(t, "abc", d.String())
	})
}

// ============================================================================
// Test mocks
// ============================================================================

// testKVStoreService is a minimal KVStoreService implementation for testing
type testKVStoreService struct {
	store storetypes.KVStore
}

func (s *testKVStoreService) OpenKVStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(ctx.MultiStore().GetStoreKey("collection"))
}

var _ store.KVStoreService = (*testKVStoreService)(nil)

// testAccountKeeper implements nft.AccountKeeper for testing
type testAccountKeeper struct{}

func (a *testAccountKeeper) GetAccount(_ sdk.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}
func (a *testAccountKeeper) SetAccount(_ sdk.Context, _ sdk.AccAddress) {}
func (a *testAccountKeeper) GetModuleAccount(_ sdk.Context, _ string) sdk.ModuleAccountI {
	return nil
}
func (a *testAccountKeeper) GetModuleAddress(_ string) sdk.AccAddress { return nil }
func (a *testAccountKeeper) GetSequence(_ sdk.Context, _ sdk.AccAddress) (uint64, error) {
	return 0, nil
}
func (a *testAccountKeeper) GetParams(_ sdk.Context) sdk.Params                       { return sdk.Params{} }
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
func (b *testBankKeeper) SpendableCoins(_ sdk.Context, _ sdk.AccAddress) sdk.Coins { return nil }
func (b *testBankKeeper) BlockedAddr(_ sdk.AccAddress) bool                        { return false }
func (b *testBankKeeper) GetBalance(_ sdk.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	return sdk.Coin{}
}
