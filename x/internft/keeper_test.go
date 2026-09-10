package internft

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
)

// newBurnKeeper builds a REAL collection-backed InterNftKeeper over an
// in-memory commit multi-store, with the gas-metered KV store wired into the
// store service. The previous version of these tests used a zero-value
// InterNftKeeper, which only worked because Burn() recovered every panic —
// i.e. the test pinned the very anti-pattern under audit. A real keeper lets
// us assert both halves of the contract: a missing NFT skips cleanly, while a
// store-level panic (OutOfGas) must propagate.
//
// gasLimit is consumed by every KV read: pass a small positive value for
// normal paths, or 0 to force OutOfGas on the first read.
func newBurnKeeper(t *testing.T, gasLimit uint64) (InterNftKeeper, sdk.Context) {
	t.Helper()

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	collectiontypes.RegisterInterfaces(cdc.InterfaceRegistry())

	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(collectiontypes.StoreKey)
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewGasMeter(gasLimit)).
		WithKVGasConfig(storetypes.KVGasConfig())

	// ctx.KVStore wraps the raw store in gaskv bound to ctx's gas meter, so
	// reads actually consume gas and an exhausted meter panics with OutOfGas.
	storeSvc := &internftKVStoreService{
		store: internftKVStoreAdapter{inner: ctx.KVStore(key)},
	}
	collectionKeeper := collectionkeeper.NewKeeper(cdc, storeSvc, &nftAccountKeeper{}, &nftBankKeeper{})
	return NewInterNftKeeper(cdc, collectionKeeper, &internftAccountKeeper{}), ctx
}

// findEvent returns the first event of the given type.
func findBurnEvent(ctx sdk.Context, eventType string) (sdk.Event, bool) {
	for _, e := range ctx.EventManager().Events() {
		if e.Type == eventType {
			return e, true
		}
	}
	return sdk.Event{}, false
}

// TestInterNftKeeper_BurnSkipOnMissingNFT verifies the skip half of the
// contract: when the NFT genuinely does not exist, Burn emits the skip event
// and returns nil so the surrounding IBC refund callback can keep going.
func TestInterNftKeeper_BurnSkipOnMissingNFT(t *testing.T) {
	ik, ctx := newBurnKeeper(t, 1_000_000)

	require.NotPanics(t, func() {
		err := ik.Burn(ctx, "class-1", "token-1")
		require.NoError(t, err)
	})

	ev, ok := findBurnEvent(ctx, "inter_nft_burn_skip")
	require.True(t, ok, "Burn must emit inter_nft_burn_skip when NFT is missing")
	require.Equal(t, "class-1", string(ev.Attributes[0].Value),
		"first attribute should be class_id")
}

// TestInterNftKeeper_BurnPropagatesOutOfGas is the counter-contract to the
// test above: an exhausted gas meter must abort the call instead of being
// downgraded to a "token already gone" skip. Before the blanket recover() was
// removed this call returned nil and emitted inter_nft_burn_skip, which made a
// failed transaction look like a successful ICS-721 burn.
func TestInterNftKeeper_BurnPropagatesOutOfGas(t *testing.T) {
	ik, ctx := newBurnKeeper(t, 0)

	require.Panics(t, func() {
		_ = ik.Burn(ctx, "class-1", "token-1")
	}, "OutOfGas must propagate out of Burn instead of being recovered")

	_, ok := findBurnEvent(ctx, "inter_nft_burn_skip")
	require.False(t, ok, "a panicking Burn must not emit a success-shaped skip event")
}

func TestInterClass_Getters(t *testing.T) {
	c := InterClass{
		ID:   "class-alpha",
		URI:  "https://metadata.uptick.network/class/alpha",
		Data: `{"name":"Alpha Collection","description":"Test"}`,
	}
	require.Equal(t, "class-alpha", c.GetID())
	require.Equal(t, "https://metadata.uptick.network/class/alpha", c.GetURI())
	require.Equal(t, `{"name":"Alpha Collection","description":"Test"}`, c.GetData())
}

func TestInterClass_Empty(t *testing.T) {
	c := InterClass{}
	require.Empty(t, c.GetID())
	require.Empty(t, c.GetURI())
	require.Empty(t, c.GetData())
}

func TestInterToken_Getters(t *testing.T) {
	token := InterToken{
		ClassID: "class-1",
		ID:      "token-42",
		URI:     "https://metadata.uptick.network/token/42",
		Data:    `{"attributes":[{"trait_type":"Color","value":"Red"}]}`,
	}
	require.Equal(t, "class-1", token.GetClassID())
	require.Equal(t, "token-42", token.GetID())
	require.Equal(t, "https://metadata.uptick.network/token/42", token.GetURI())
	require.Equal(t, `{"attributes":[{"trait_type":"Color","value":"Red"}]}`, token.GetData())
}

func TestInterToken_Empty(t *testing.T) {
	token := InterToken{}
	require.Empty(t, token.GetClassID())
	require.Empty(t, token.GetID())
	require.Empty(t, token.GetURI())
	require.Empty(t, token.GetData())
}

func TestInterNftKeeper_Fields(t *testing.T) {
	// Verify keeper struct can be created with zero values
	k := InterNftKeeper{}
	require.Nil(t, k.cdc)
	require.Nil(t, k.ak)
}

func TestInterClass_IBCClassType(t *testing.T) {
	// Verify InterClass satisfies the IBC class interface
	// (GetID, GetURI, GetData methods)
	var class interface {
		GetID() string
		GetURI() string
		GetData() string
	}

	c := InterClass{ID: "c1", URI: "uri", Data: "data"}
	class = c
	require.Equal(t, "c1", class.GetID())
	require.Equal(t, "uri", class.GetURI())
	require.Equal(t, "data", class.GetData())
}

func TestInterToken_IBCNFTType(t *testing.T) {
	// Verify InterToken satisfies the IBC NFT interface
	// (GetClassID, GetID, GetURI, GetData methods)
	var nft interface {
		GetClassID() string
		GetID() string
		GetURI() string
		GetData() string
	}

	token := InterToken{ClassID: "c1", ID: "t1", URI: "uri", Data: "data"}
	nft = token
	require.Equal(t, "c1", nft.GetClassID())
	require.Equal(t, "t1", nft.GetID())
	require.Equal(t, "uri", nft.GetURI())
	require.Equal(t, "data", nft.GetData())
}
