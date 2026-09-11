package internft

import (
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

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
)

// boundNFTs stands in for the checker the application wires in app/keepers
// (convertedNFTChecker, which ORs x/erc721 and x/cw721): it answers "this
// native NFT has a contract-side counterpart" for exactly one token.
type boundNFTs struct{ classID, nftID string }

func (b boundNFTs) IsConvertedNFT(_ sdk.Context, classID, nftID string) bool {
	return classID == b.classID && nftID == b.nftID
}

// newBurnGuardFixture builds a real collection keeper over a real (in-memory)
// store, wrapped in the ICS-721 adapter that nft-transfer is constructed with.
// The store is real so the burn is observable in state; the adapters are the
// same minimal stubs the other x/internft tests use.
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

// TestICS721BurnMustNotConsultTheConvertedNFTGuard pins the one burn path that
// has to bypass x/collection's cross-module conversion guard, because the
// "obvious fix" (routing it through collection.RemoveNFT) is a regression.
//
// Two different functions can destroy a native NFT:
//
//   - x/collection.RemoveNFT (keeper/nft.go:223) consults the wired
//     ConvertedNFTChecker and refuses when a contract token is escrowed. That is
//     the right call for a plain "burn my NFT" request: the escrowed contract
//     half would lose its last on-chain record.
//   - InterNftKeeper.Burn goes straight to the underlying x/nft keeper
//     (NFTkeeper(), collection/keeper/keeper.go:43) and consults nothing,
//     because nft-transfer reaches it through the ICS721Keeper interface.
//
// The bypass is load-bearing. nft-transfer picks escrow or burn from the class
// prefix alone (nft-transfer keeper/relay.go:238-248): a class whose id starts
// with this chain's own (port, channel) is burned on the way out, and a class
// that does not is escrowed. MsgTransferERC721/MsgTransferCW721 convert first
// and then send, so the packet always carries a class that was just paired with
// a contract token (convertEvm2Cosmos writes the pair, then the send burns the
// native half). A guard here would therefore fail every such send with
// ErrNFTBoundToContract instead of the intended "escrow the contract side,
// burn the native side" pairing -- which is why the burn is deliberately bare.
//
// The residual, and the reason this is worth a test rather than a comment: the
// same bare burn also fires for a locally issued denom whose id merely spells a
// (port, channel) prefix. If such a denom has a conversion pair and its holder
// sends the native NFT over that channel, the burn destroys the last record of
// the escrowed contract token, and ConvertERC721/ConvertCW721 can no longer
// recover it ("is not escrowed by the module account"). That is self-inflicted
// (the holder burns their own token), so it is documented rather than blocked;
// blocking it would need a signal the class prefix cannot carry.
func TestICS721BurnMustNotConsultTheConvertedNFTGuard(t *testing.T) {
	ik, ck, ctx := newBurnGuardFixture(t)
	owner := sdk.AccAddress([]byte("owner"))

	// Shaped like a class nft-transfer will burn rather than escrow: its id
	// carries this chain's own ICS-721 (port, channel) prefix.
	const classID = "nonfungibletokentransfer/channel-0/pinned"

	require.NoError(t, ik.CreateOrUpdateClass(ctx, classID, "ipfs://class", ""))
	require.NoError(t, ik.Mint(ctx, classID, "token1", "ipfs://token", "", owner))
	require.NoError(t, ik.Mint(ctx, classID, "token2", "", "", owner))

	// With the application's checker wired in, the collection module's own burn
	// refuses a bound token. This is the guard doing its job.
	ck.SetConvertedNFTChecker(boundNFTs{classID: classID, nftID: "token1"})
	err := ck.RemoveNFT(ctx, classID, "token1", owner)
	require.ErrorIs(t, err, collectiontypes.ErrNFTBoundToContract,
		"burning a bound native NFT through the collection module must be refused")

	// Negative control: the same call succeeds for a token the checker does not
	// report as bound, so the refusal above came from the guard and not from an
	// unrelated precondition (existence or ownership).
	require.NoError(t, ck.RemoveNFT(ctx, classID, "token2", owner))

	// The ICS-721 path burns the bound token anyway. If this ever starts
	// failing, someone has routed Burn through the guard and broken
	// MsgTransferERC721 / MsgTransferCW721 -- see the doc comment.
	require.NoError(t, ik.Burn(ctx, classID, "token1"))
	_, ok := ik.GetNFT(ctx, classID, "token1")
	require.False(t, ok,
		"the ICS-721 burn removed the native token while the contract-side binding was still recorded")
}
