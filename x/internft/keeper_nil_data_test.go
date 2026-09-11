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

// TestGetClassToleratesMissingClassData is the integration half of the P2-8
// fix: it drives the ICS-721 port the way nft-transfer does and pins that a
// class stored without the DenomMetadata wrapper is still visible.
//
// Before the fix BuildMetadata returned "unsupported class metadata" for such a
// class, so this GetClass answered not-found and nft-transfer aborted the
// transfer of a class that plainly exists on chain.
func TestGetClassToleratesMissingClassData(t *testing.T) {
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

	require.NoError(t, ik.CreateOrUpdateClass(ctx, "legacy", "ipfs://legacy", ""))

	// Strip the metadata wrapper so the stored class looks like a
	// pre-migration / third-party record. Reaching through NFTkeeper() is
	// deliberate: no public path writes a class without the wrapper, which is
	// exactly why the tolerant read path needs its own test.
	nk := collectionKeeper.NFTkeeper()
	class, has := nk.GetClass(ctx, "legacy")
	require.True(t, has)
	class.Data = nil
	require.NoError(t, nk.UpdateClass(ctx, class))

	got, ok := ik.GetClass(ctx, "legacy")
	require.True(t, ok, "a class whose Data is nil must still be exported over ICS-721")
	require.Equal(t, "legacy", got.GetID())
	require.Equal(t, "ipfs://legacy", got.GetURI())
	require.NotEmpty(t, got.GetData(), "the exported class data must not be empty")

	// Reverse sentinel: tolerance for a degraded record must not become
	// tolerance for an absent one.
	_, ok = ik.GetClass(ctx, "no-such-class")
	require.False(t, ok, "an unknown class must still be reported as not found")
}
