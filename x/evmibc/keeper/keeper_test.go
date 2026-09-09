package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cw721keep "github.com/UptickNetwork/uptick/x/cw721/keeper"
	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

// newRefundCtx builds a minimal sdk.Context with an event manager wired up
// so we can inspect events emitted by getRefundClassId. It deliberately
// uses an empty Keeper (no store / codec needed) because getRefundClassId
// only emits through ctx.EventManager().
func newRefundCtx(t *testing.T) sdk.Context {
	t.Helper()
	memDB := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	memDB.MountStoreWithDB(storetypes.NewKVStoreKey("dummy"), storetypes.StoreTypeIAVL, nil)
	require.NoError(t, memDB.LoadLatestVersion())
	return sdk.NewContext(memDB, tmproto.Header{}, false, log.NewNopLogger()).
		WithKVGasConfig(storetypes.KVGasConfig())
}

// findEvent returns the first event of the given type in the context's
// emitted events, plus whether it was found.
func findEvent(ctx sdk.Context, eventType string) (sdk.Event, bool) {
	for _, e := range ctx.EventManager().Events() {
		if e.Type == eventType {
			return e, true
		}
	}
	return sdk.Event{}, false
}

func TestNewKeeper(t *testing.T) {
	// NewKeeper requires a non-nil ibcnfttransferkeeper.Keeper (value type)
	ik := ibcnfttransferkeeper.Keeper{}
	k := NewKeeper(ik)
	require.NotNil(t, &k)
}

func TestSetCw721Keeper(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})
	require.NotNil(t, &k)

	// Setting a nil cw721Keeper (not a pointer type, can't be nil)
	// Just verify set/get works - setting it to zero value
	k.SetCw721Keeper(cw721keep.Keeper{})
}

func TestSetErc721Keeper(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})
	require.NotNil(t, &k)

	k.SetErc721Keeper(erc721keeper.Keeper{})
}

func TestGetVoucherClassID(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})

	voucherClassID := k.GetVoucherClassID(nfttransfertypes.PortID, "channel-0", "class-1")
	require.NotEmpty(t, voucherClassID)
	require.Contains(t, voucherClassID, "ibc/")

	// Same params produce same result
	voucherClassID2 := k.GetVoucherClassID(nfttransfertypes.PortID, "channel-0", "class-1")
	require.Equal(t, voucherClassID, voucherClassID2)

	// Different channel produces different result
	differentChannel := k.GetVoucherClassID(nfttransfertypes.PortID, "channel-1", "class-1")
	require.NotEqual(t, voucherClassID, differentChannel)
}

func TestGetVoucherClassID_EmptyPort(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})
	voucherClassID := k.GetVoucherClassID("", "channel-0", "class-empty")
	require.NotEmpty(t, voucherClassID)
}

func TestGetVoucherClassID_Deterministic(t *testing.T) {
	ik := ibcnfttransferkeeper.Keeper{}
	k1 := NewKeeper(ik)
	k2 := NewKeeper(ik)

	result1 := k1.GetVoucherClassID(nfttransfertypes.PortID, "channel-0", "class-1")
	result2 := k2.GetVoucherClassID(nfttransfertypes.PortID, "channel-0", "class-1")
	require.Equal(t, result1, result2)
}

// TestGetRefundClassId covers the four observable shapes of `data.ClassId`:
// bare native id → as-is; matching (port, channel) voucher → ibc/<hash>;
// different-channel or different-port voucher → ibc/<hash> + `cross_channel_refund`
// event (which keeps the full path) for multi-hop observability.
func TestGetRefundClassId(t *testing.T) {
	k := NewKeeper(ibcnfttransferkeeper.Keeper{})
	packet := channeltypes.Packet{
		SourcePort:    nfttransfertypes.PortID,
		SourceChannel: "channel-0",
		Sequence:      42,
	}

	t.Run("bare native class is returned unchanged", func(t *testing.T) {
		ctx := newRefundCtx(t)
		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{ClassId: "kitty"})
		require.NoError(t, err)
		require.Equal(t, "kitty", got)
		// Bare class path must NOT emit a cross-channel event.
		_, found := findEvent(ctx, "cross_channel_refund")
		require.False(t, found, "bare class should not trigger cross_channel_refund event")
	})

	t.Run("voucher with matching prefix returns canonical ibc hash", func(t *testing.T) {
		ctx := newRefundCtx(t)
		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: nfttransfertypes.PortID + "/channel-0/kitty",
		})
		require.NoError(t, err)
		require.Contains(t, got, "ibc/")
		_, found := findEvent(ctx, "cross_channel_refund")
		require.False(t, found, "matching prefix should not trigger cross_channel_refund event")
	})

	// A voucher whose channel prefix does NOT match this packet's (port,
	// channel) is a multi-hop ICS-721 case: the local voucher id is derived
	// from the full path, while the event keeps the full path for observability.
	t.Run("voucher with different channel derives local id with observability event", func(t *testing.T) {
		ctx := newRefundCtx(t)
		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: nfttransfertypes.PortID + "/channel-1/kitty",
		})
		require.NoError(t, err)
		require.Equal(t, nfttransfertypes.ParseClassTrace(nfttransfertypes.PortID+"/channel-1/kitty").IBCClassID(), got)
		// Must emit an event so ops can spot multi-hop usage.
		ev, found := findEvent(ctx, "cross_channel_refund")
		require.True(t, found, "cross-channel voucher must emit cross_channel_refund event")
		requireAttribute(t, ev, "class_id", nfttransfertypes.PortID+"/channel-1/kitty")
		requireAttribute(t, ev, "source_port", nfttransfertypes.PortID)
		requireAttribute(t, ev, "source_channel", "channel-0")
		requireAttribute(t, ev, "sequence", "42")
	})

	t.Run("voucher with different port derives local id with observability event", func(t *testing.T) {
		ctx := newRefundCtx(t)
		otherPacket := channeltypes.Packet{
			SourcePort:    "transfer",
			SourceChannel: "channel-0",
			Sequence:      7,
		}
		got, err := k.getRefundClassId(ctx, otherPacket, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: nfttransfertypes.PortID + "/channel-0/kitty",
		})
		require.NoError(t, err)
		require.Equal(t, nfttransfertypes.ParseClassTrace(nfttransfertypes.PortID+"/channel-0/kitty").IBCClassID(), got)
		ev, found := findEvent(ctx, "cross_channel_refund")
		require.True(t, found, "cross-port voucher must emit cross_channel_refund event")
		requireAttribute(t, ev, "class_id", nfttransfertypes.PortID+"/channel-0/kitty")
		requireAttribute(t, ev, "source_port", "transfer")
		requireAttribute(t, ev, "source_channel", "channel-0")
		requireAttribute(t, ev, "sequence", "7")
	})

	// Shape 2 with a class id that starts with the packet's (port, channel)
	// as a substring but not as the full canonical prefix -- e.g.
	// "transfer/channel-0abc/foo". HasPrefix sees the trailing "/" mismatch
	// and correctly routes this to the cross-channel branch.
	t.Run("near-miss prefix (channel-0abc) is treated as cross-channel", func(t *testing.T) {
		ctx := newRefundCtx(t)
		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: nfttransfertypes.PortID + "/channel-0abc/foo",
		})
		require.NoError(t, err)
		require.Equal(t, nfttransfertypes.ParseClassTrace(nfttransfertypes.PortID+"/channel-0abc/foo").IBCClassID(), got)
		_, found := findEvent(ctx, "cross_channel_refund")
		require.True(t, found, "near-miss prefix must emit cross_channel_refund event")
	})
}

// requireAttribute asserts that the given event has an attribute with the
// given key/value pair. Useful for verifying event payloads without
// relying on attribute order.
func requireAttribute(t *testing.T, ev sdk.Event, key, want string) {
	t.Helper()
	for _, a := range ev.Attributes {
		if string(a.Key) == key {
			require.Equal(t, want, string(a.Value),
				"attribute %q has value %q, want %q", key, string(a.Value), want)
			return
		}
	}
	t.Fatalf("event %q has no attribute %q (have %v)", ev.Type, key, ev.Attributes)
}
