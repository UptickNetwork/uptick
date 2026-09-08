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
	"cosmossdk.io/x/nft"
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

// TestExportGenesisNilData ensures a legacy / migrated NFT with nil Data does not
// make genesis export panic (it is exported with empty metadata instead).
func (s *KeeperTestSuite) TestExportGenesisNilData() {
	creator := sdk.AccAddress([]byte("creator-nil"))
	s.Require().NoError(s.keeper.SaveDenom(s.ctx, "denom-nil", "Nil", "", "NIL", creator, false, false, "", "", "", ""))

	// Mint directly via the underlying nft keeper to produce a nil-Data NFT
	// (SaveNFT always attaches NFTMetadata).
	s.Require().NoError(s.nftKpr.Mint(s.ctx, nft.NFT{
		ClassId: "denom-nil",
		Id:      "nft-nil",
		Uri:     "ipfs://nil",
	}, creator))

	gs := s.keeper.ExportGenesis(s.ctx)
	s.Require().NotNil(gs)
	s.Require().Len(gs.Collections, 1)
	s.Require().Len(gs.Collections[0].NFTs, 1)
}

// legacy / migrated classes with nil Data must not crash
// GetDenomInfo. The expected behavior is to return zero-value metadata rather
// than a "has no metadata" error.
func (s *KeeperTestSuite) TestGetDenomInfoNilData() {
	// Save a class directly via the underlying nft keeper with nil Data.
	// GetDenomInfo will then encounter the nil Data on read.
	s.Require().NoError(s.nftKpr.SaveClass(s.ctx, nft.Class{
		Id:     "denom-c",
		Name:   "Class C",
		Symbol: "C",
	}))

	d, err := s.keeper.GetDenomInfo(s.ctx, "denom-c")
	s.Require().NoError(err)
	s.Require().NotNil(d)
	s.Require().Equal("denom-c", d.Id)
	s.Require().Equal("Class C", d.Name)
	s.Require().Empty(d.Creator)
	s.Require().Empty(d.Schema)
}

// A legacy / migrated class with nil Data must not abort GetCollections: the
// call succeeds and the nil-data class yields a valid entry with zero-value
// metadata.
func (s *KeeperTestSuite) TestGetCollectionsIncludesNilDataClass() {
	creator := sdk.AccAddress([]byte("creator"))
	s.Require().NoError(s.keeper.SaveDenom(s.ctx, "good", "Good", "", "G", creator, false, false, "", "", "", ""))
	s.Require().NoError(s.keeper.SaveDenom(s.ctx, "good2", "Good2", "", "G2", creator, false, false, "", "", "", ""))

	// Insert a class with nil Data directly via the underlying keeper -- the
	// only path that produces a nil-Data class on chain (SaveDenom always
	// wraps metadata).
	s.Require().NoError(s.nftKpr.SaveClass(s.ctx, nft.Class{
		Id:     "nildataclass",
		Name:   "NilData",
		Symbol: "ND",
	}))

	cs, err := s.keeper.GetCollections(s.ctx)
	s.Require().NoError(err)
	// All three classes are returned. The nil-data one survives the loop
	// with zero-value metadata instead of crashing genesis export.
	s.Require().Len(cs, 3)
	ids := make(map[string]types.Denom, 3)
	for _, c := range cs {
		ids[c.Denom.Id] = c.Denom
	}
	s.Require().Contains(ids, "good")
	s.Require().Contains(ids, "good2")
	s.Require().Contains(ids, "nildataclass")
	// The nil-data class is exposed with empty Creator / Schema -- this is
	// the post-M-C contract that downstream queries (Collection, Denom)
	// rely on.
	s.Require().Empty(ids["nildataclass"].Creator)
	s.Require().Empty(ids["nildataclass"].Schema)
}

// Defense-in-depth: SaveNFT must reject empty inputs rather than
// silently writing a corrupt state entry (empty key / zero address).
func (s *KeeperTestSuite) TestSaveNFTRejectsEmptyInputs() {
	creator := sdk.AccAddress([]byte("creator-empty"))
	s.Require().NoError(s.keeper.SaveDenom(s.ctx, "denom", "D", "", "D", creator, false, false, "", "", "", ""))

	// Empty denom ID.
	s.Require().ErrorContains(s.keeper.SaveNFT(s.ctx, "", "1", "n", "", "", "", creator),
		"denom ID cannot be empty")

	// Empty token ID.
	s.Require().ErrorContains(s.keeper.SaveNFT(s.ctx, "denom", "", "n", "", "", "", creator),
		"token ID cannot be empty")

	// Empty / nil receiver.
	s.Require().ErrorContains(s.keeper.SaveNFT(s.ctx, "denom", "1", "n", "", "", "", nil),
		"receiver cannot be empty")
	s.Require().ErrorContains(s.keeper.SaveNFT(s.ctx, "denom", "1", "n", "", "", "", sdk.AccAddress{}),
		"receiver cannot be empty")

	// No state was written on any of these failures.
	s.Require().False(s.keeper.HasNFT(s.ctx, "denom", "1"))
}

// GetNFTs must downgrade a single NFT with undecodable metadata to empty
// metadata rather than aborting the whole query (matches GetCollections
// behavior). The bad record stays in the result so callers iterating the
// chain state see the same shape they would have seen before the bug.
func (s *KeeperTestSuite) TestGetNFTsSkipsUndecodableNFT() {
	creator := sdk.AccAddress([]byte("creator-undec"))
	s.Require().NoError(s.keeper.SaveDenom(s.ctx, "denom-un", "U", "", "U", creator, false, false, "", "", "", ""))
	// Mint a clean NFT first.
	s.Require().NoError(s.keeper.SaveNFT(s.ctx, "denom-un", "good", "Good", "", "", "", creator))

	// Mint a second NFT, then poison its Data so Unmarshal fails.
	s.Require().NoError(s.keeper.SaveNFT(s.ctx, "denom-un", "bad", "Bad", "", "", "", creator))
	bad, ok := s.nftKpr.GetNFT(s.ctx, "denom-un", "bad")
	s.Require().True(ok)
	bad.Data = &codectypes.Any{TypeUrl: "/cosmos.bad.Type", Value: []byte{0xff, 0xfe}}
	s.Require().NoError(s.nftKpr.Update(s.ctx, bad))

	nfts, err := s.keeper.GetNFTs(s.ctx, "denom-un")
	s.Require().NoError(err, "undecodable NFT must not abort the whole query")
	s.Require().Len(nfts, 2)

	got := make(map[string]types.BaseNFT, 2)
	for _, n := range nfts {
		got[n.GetID()] = n.(types.BaseNFT)
	}
	// The clean NFT keeps its real metadata.
	s.Require().Equal("Good", got["good"].Name)
	// The poisoned NFT is downgraded to empty metadata.
	s.Require().Empty(got["bad"].Name)
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
