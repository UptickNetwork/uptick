package internft

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
)

// newBurnCtx builds a minimal sdk.Context with an event manager wired
// up. The internft.Burn guard only reads/writes through ctx.EventManager
// and ctx.Logger, so an in-memory store is enough.
func newBurnCtx(t *testing.T) sdk.Context {
	t.Helper()
	memDB := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	memDB.MountStoreWithDB(storetypes.NewKVStoreKey("dummy"), storetypes.StoreTypeIAVL, nil)
	require.NoError(t, memDB.LoadLatestVersion())
	return sdk.NewContext(memDB, tmproto.Header{}, false, log.NewNopLogger()).
		WithKVGasConfig(storetypes.KVGasConfig())
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

// TestInterNftKeeper_BurnSkipOnMissingNFT verifies the no-panic + skip-event
// contract of Burn when the NFT is absent (production path is exercised by
// the nft-transfer module's own suite).
func TestInterNftKeeper_BurnSkipOnMissingNFT(t *testing.T) {
	ik := InterNftKeeper{} // zero-value: nk is nil; the guard must not touch it
	ctx := newBurnCtx(t)

	// Must complete without panicking and without returning an error,
	// even though the underlying nft keeper is nil (which would
	// definitely panic if reached).
	require.NotPanics(t, func() {
		err := ik.Burn(ctx, "class-1", "token-1")
		require.NoError(t, err)
	})

	ev, ok := findBurnEvent(ctx, "inter_nft_burn_skip")
	require.True(t, ok, "Burn must emit inter_nft_burn_skip when NFT is missing")
	require.Equal(t, "class-1", string(ev.Attributes[0].Value),
		"first attribute should be class_id")
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
