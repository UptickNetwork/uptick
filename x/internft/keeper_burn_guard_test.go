package internft

import (
	"strings"
	"testing"

	"cosmossdk.io/log"
	rootstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	nfttransfertypes "github.com/bianjieai/nft-transfer/types"

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
)

// boundNFTs stands in for the checker the application wires in app/keepers
// (convertedNFTChecker, which ORs x/erc721 and x/cw721): it answers "this
// native NFT has a contract-side counterpart" for exactly one token.
type boundNFTs struct{ classID, nftID string }

func (b boundNFTs) IsConvertedNFT(_ sdk.Context, classID, nftID string) bool {
	return classID == b.classID && nftID == b.nftID
}

// newBurnGuardFixture builds a real collection keeper over a real (in-memory)
// store, wrapped in the ICS-721 adapter nft-transfer is constructed with. The
// store is real so the burn is observable in state.
func newBurnGuardFixture(t *testing.T) (InterNftKeeper, collectionkeeper.Keeper, sdk.Context) {
	t.Helper()

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	collectiontypes.RegisterInterfaces(cdc.InterfaceRegistry())

	db := dbm.NewMemDB()
	cms := rootstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(collectiontypes.StoreKey)
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	storeSvc := &internftKVStoreService{store: internftKVStoreAdapter{inner: cms.GetKVStore(key)}}
	ck := collectionkeeper.NewKeeper(cdc, storeSvc, &nftAccountKeeper{}, &nftBankKeeper{})
	ik := NewInterNftKeeper(cdc, ck, &internftAccountKeeper{})

	return ik, ck, sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
}

// destroyBranchClass is shaped like a class nft-transfer destroys rather than
// escrows: its id carries this chain's own ICS-721 (port, channel) prefix.
const destroyBranchClass = "nonfungibletokentransfer/channel-0/pinned"

// TestICS721BurnRefusesABoundNFT pins the burn guard on the ICS-721 path.
//
// Two functions can destroy a native NFT. x/collection.RemoveNFT
// (keeper/nft.go:223) consults the conversion guard and refuses when a contract
// token is escrowed. InterNftKeeper.Burn instead reaches the underlying x/nft
// keeper through Keeper.NFTkeeper() because nft-transfer calls it via the
// ICS721Keeper interface, so it never passed that guard and used to destroy
// bound NFTs.
//
// The guard costs the ICS-721 path nothing: MsgTransferERC721/MsgTransferCW721
// always take the escrow branch, because a conversion class is always
// "uptick-<contract>" (TestOrdinaryConversionClassCanNeverReachTheDestroyBranch
// pins that), so nothing legitimate reaches Burn with a pair recorded.
func TestICS721BurnRefusesABoundNFT(t *testing.T) {
	ik, ck, ctx := newBurnGuardFixture(t)
	owner := sdk.AccAddress([]byte("owner"))

	require.NoError(t, ik.CreateOrUpdateClass(ctx, destroyBranchClass, "ipfs://class", ""))
	require.NoError(t, ik.Mint(ctx, destroyBranchClass, "token1", "ipfs://token", "", owner))
	require.NoError(t, ik.Mint(ctx, destroyBranchClass, "token2", "", "", owner))

	// Wire the check exactly as app/keepers does: once, on the collection keeper.
	// The adapter above was built before this call and must still see it.
	ck.SetConvertedNFTChecker(boundNFTs{classID: destroyBranchClass, nftID: "token1"})

	// The collection module's own burn refuses the bound token.
	err := ck.RemoveNFT(ctx, destroyBranchClass, "token1", owner)
	require.ErrorIs(t, err, collectiontypes.ErrNFTBoundToContract,
		"burning a bound native NFT through the collection module must be refused")

	// The ICS-721 path must refuse it too; before the guard it returned nil and the
	// token was gone from the store.
	err = ik.Burn(ctx, destroyBranchClass, "token1")
	require.ErrorIs(t, err, collectiontypes.ErrNFTBoundToContract,
		"the ICS-721 burn must not destroy a native NFT whose contract half is escrowed")
	_, stillThere := ik.GetNFT(ctx, destroyBranchClass, "token1")
	require.True(t, stillThere, "a refused burn must leave the native token in the store")

	// Negative control: the same call succeeds for a token the checker does not
	// report as bound, so the refusal above came from the guard and not from an
	// unrelated precondition.
	require.NoError(t, ik.Burn(ctx, destroyBranchClass, "token2"))
	_, ok := ik.GetNFT(ctx, destroyBranchClass, "token2")
	require.False(t, ok, "an unbound token must still be destroyed -- the guard is not a blanket refusal")
}

// TestConversionGuardReachesCopiesTakenBeforeWiring pins the mechanism the
// guard depends on, because the obvious implementation silently does not work.
//
// The collection keeper is copied by value into x/erc721, x/cw721 and
// x/internft, all of which are constructed *before* the checker can exist (it
// is built from x/erc721 and x/cw721 themselves). A plain field would be frozen
// as nil inside those copies, so the single wiring call in app/keepers would
// only reach the one instance the app holds -- and in particular x/erc721's
// refund path, which burns through its own nftKeeper copy, would run unguarded.
// That is a silent regression, not a compile error. Sharing one slot through a
// pointer is what makes the late wiring reach every copy.
func TestConversionGuardReachesCopiesTakenBeforeWiring(t *testing.T) {
	ik, ck, ctx := newBurnGuardFixture(t)

	// Both copies predate the wiring. erc721KeeperCopy stands in for the nftKeeper
	// field x/erc721 holds (it calls BurnNFT -> RemoveNFT on it); internftCopy
	// already lives inside the adapter returned above.
	erc721KeeperCopy := ck
	require.False(t, erc721KeeperCopy.IsConvertedNFT(ctx, "someclass", "token1"),
		"nothing is bound before the checker is wired")

	ck.SetConvertedNFTChecker(boundNFTs{classID: "someclass", nftID: "token1"})

	require.True(t, erc721KeeperCopy.IsConvertedNFT(ctx, "someclass", "token1"),
		"a copy taken before the wiring must observe it, or x/erc721's refund path runs unguarded")
	require.True(t, ik.IsConvertedNFT(ctx, "someclass", "token1"),
		"the ICS-721 adapter holds a copy of the collection keeper too")
	require.False(t, ik.IsConvertedNFT(ctx, "someclass", "token2"),
		"sibling tokens have no binding of their own")

	// An unwired keeper reports false rather than panicking: module-only setups,
	// such as this package's other tests, rely on that.
	var zero collectionkeeper.Keeper
	require.False(t, zero.IsConvertedNFT(ctx, "someclass", "token1"))
	require.NotPanics(t, func() { zero.SetConvertedNFTChecker(boundNFTs{classID: "x", nftID: "y"}) })
}

// TestOrdinaryConversionClassCanNeverReachTheDestroyBranch is the evidence that
// the guard above does not restrict normal traffic.
//
// nft-transfer picks escrow or destroy from the class name alone (nft-transfer
// keeper/relay.go:238-248 -> types.IsAwayFromOrigin): a class carrying this
// chain's own "<port>/<channel>/" prefix is destroyed on the way out, anything
// else is escrowed. A conversion class is always "uptick-<contract>" (x/erc721
// and x/cw721 both derive it; both are asserted here) and contains no "/", so it
// can never carry an ICS-721 prefix: IsAwayFromOrigin is true for it and
// MsgTransferERC721/MsgTransferCW721 always escrow. They never reach Burn.
func TestOrdinaryConversionClassCanNeverReachTheDestroyBranch(t *testing.T) {
	const (
		port    = "nonfungibletokentransfer"
		channel = "channel-0"
	)
	contract := "0x" + strings.Repeat("ab", 20)

	erc721Class := erc721types.CreateClassIDFromContractAddress(contract)
	cw721Class := cw721types.CreateClassIDFromContractAddress(contract)

	// If either module ever stops deriving the class from the contract address
	// this assertion is the tripwire: the guard's safety argument rests on it.
	require.Equal(t, "uptick-"+strings.TrimPrefix(contract, "0x"), erc721Class)
	require.Equal(t, erc721Class, cw721Class, "the twin modules must derive the same class")
	require.NotContains(t, erc721Class, "/",
		"a conversion class must not contain a path separator, or it could impersonate an ICS-721 prefix")

	require.False(t, strings.HasPrefix(erc721Class, port+"/"+channel+"/"))
	require.True(t, nfttransfertypes.IsAwayFromOrigin(port, channel, erc721Class),
		"a conversion class must take the escrow branch, never the destroy branch")

	// Counter-assertion, so the line above cannot pass vacuously: a class that
	// does carry this chain's prefix takes the destroy branch.
	require.False(t, nfttransfertypes.IsAwayFromOrigin(port, channel, port+"/"+channel+"/"+erc721Class),
		"IsAwayFromOrigin must actually be false for prefixed classes")
}
