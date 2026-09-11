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

// recordingICS721 stands in for the ICS-721 keeper. getRefundClassId delegates
// the "not a voucher of this packet's channel" shape to GetVoucherClassID, so
// the fake records every call and answers with a caller-chosen value. The tests
// below assert *that* the resolution is delegated (and, for the shapes that are
// decided locally, that it is not); how the resolver picks between a local
// class and a trace path is the ICS-721 keeper's own contract.
type recordingICS721 struct {
	calls  []string
	answer string
}

func (m *recordingICS721) OnAcknowledgementPacket(
	sdk.Context, channeltypes.Packet, nfttransfertypes.NonFungibleTokenPacketData, channeltypes.Acknowledgement,
) error {
	return nil
}

func (m *recordingICS721) OnTimeoutPacket(
	sdk.Context, channeltypes.Packet, nfttransfertypes.NonFungibleTokenPacketData,
) error {
	return nil
}

func (m *recordingICS721) GetVoucherClassID(_ sdk.Context, classID string) (string, error) {
	m.calls = append(m.calls, classID)
	if m.answer == "" {
		return classID, nil
	}
	return m.answer, nil
}

// TestGetRefundClassId covers the four observable shapes of `data.ClassId`:
// bare native id → as-is; matching (port, channel) voucher → ibc/<hash>;
// anything else → delegated to the ICS-721 resolver with a
// `cross_channel_refund` event (which keeps the full path) for multi-hop
// observability.
//
// Shapes 1 and 2 are resolved locally, so consulting the resolver there would
// both waste a store read and let a future resolver bug override a decision the
// string already settles. Shapes 3/4 must consult it: a class id can legally
// contain "/" (idString allows it), and only the resolver knows whether such an
// id names a real local class or is an ICS-721 trace path.
func TestGetRefundClassId(t *testing.T) {
	const resolverAnswer = "resolver-answer"
	packet := channeltypes.Packet{
		SourcePort:    nfttransfertypes.PortID,
		SourceChannel: "channel-0",
		Sequence:      42,
	}

	t.Run("bare native class is returned unchanged", func(t *testing.T) {
		ctx := newRefundCtx(t)
		resolver := &recordingICS721{answer: resolverAnswer}
		k := NewKeeper(resolver)

		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{ClassId: "kitty"})
		require.NoError(t, err)
		require.Equal(t, "kitty", got)
		require.Empty(t, resolver.calls, "a bare class id needs no resolver lookup")
		// Bare class path must NOT emit a cross-channel event.
		_, found := findEvent(ctx, "cross_channel_refund")
		require.False(t, found, "bare class should not trigger cross_channel_refund event")
	})

	t.Run("voucher with matching prefix returns canonical ibc hash", func(t *testing.T) {
		ctx := newRefundCtx(t)
		resolver := &recordingICS721{answer: resolverAnswer}
		k := NewKeeper(resolver)

		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: nfttransfertypes.PortID + "/channel-0/kitty",
		})
		require.NoError(t, err)
		require.Contains(t, got, "ibc/")
		require.NotEqual(t, resolverAnswer, got, "a matching prefix is settled locally, not by the resolver")
		require.Empty(t, resolver.calls, "a matching prefix needs no resolver lookup")
		_, found := findEvent(ctx, "cross_channel_refund")
		require.False(t, found, "matching prefix should not trigger cross_channel_refund event")
	})

	// A voucher whose channel prefix does NOT match this packet's (port,
	// channel) is a multi-hop ICS-721 case: the local voucher id comes from the
	// ICS-721 keeper, while the event keeps the full path for observability.
	t.Run("voucher with different channel delegates with observability event", func(t *testing.T) {
		ctx := newRefundCtx(t)
		resolver := &recordingICS721{answer: resolverAnswer}
		k := NewKeeper(resolver)

		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: nfttransfertypes.PortID + "/channel-1/kitty",
		})
		require.NoError(t, err)
		require.Equal(t, resolverAnswer, got)
		require.Equal(t, []string{nfttransfertypes.PortID + "/channel-1/kitty"}, resolver.calls)
		// Must emit an event so ops can spot multi-hop usage.
		ev, found := findEvent(ctx, "cross_channel_refund")
		require.True(t, found, "cross-channel voucher must emit cross_channel_refund event")
		requireAttribute(t, ev, "class_id", nfttransfertypes.PortID+"/channel-1/kitty")
		requireAttribute(t, ev, "source_port", nfttransfertypes.PortID)
		requireAttribute(t, ev, "source_channel", "channel-0")
		requireAttribute(t, ev, "sequence", "42")
	})

	t.Run("voucher with different port delegates with observability event", func(t *testing.T) {
		ctx := newRefundCtx(t)
		resolver := &recordingICS721{answer: resolverAnswer}
		k := NewKeeper(resolver)
		otherPacket := channeltypes.Packet{
			SourcePort:    "transfer",
			SourceChannel: "channel-0",
			Sequence:      7,
		}
		got, err := k.getRefundClassId(ctx, otherPacket, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: nfttransfertypes.PortID + "/channel-0/kitty",
		})
		require.NoError(t, err)
		require.Equal(t, resolverAnswer, got)
		require.Equal(t, []string{nfttransfertypes.PortID + "/channel-0/kitty"}, resolver.calls)
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
		resolver := &recordingICS721{answer: resolverAnswer}
		k := NewKeeper(resolver)

		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: nfttransfertypes.PortID + "/channel-0abc/foo",
		})
		require.NoError(t, err)
		require.Equal(t, resolverAnswer, got)
		require.Equal(t, []string{nfttransfertypes.PortID + "/channel-0abc/foo"}, resolver.calls)
		_, found := findEvent(ctx, "cross_channel_refund")
		require.True(t, found, "near-miss prefix must emit cross_channel_refund event")
	})

	// A natively issued class id may itself contain "/" (idString allows it),
	// and such an id is NOT an ICS-721 trace path. Only the ICS-721 keeper can
	// tell the two apart -- it answers with the id itself when a class of that
	// name exists locally. Deriving ibc/<hash> from the string shape instead
	// would hand the module-side refund a class that does not exist.
	t.Run("local class containing a slash is returned unchanged", func(t *testing.T) {
		ctx := newRefundCtx(t)
		resolver := &recordingICS721{answer: "sub/collection"}
		k := NewKeeper(resolver)

		got, err := k.getRefundClassId(ctx, packet, nfttransfertypes.NonFungibleTokenPacketData{
			ClassId: "sub/collection",
		})
		require.NoError(t, err)
		require.Equal(t, "sub/collection", got)
		require.Equal(t, []string{"sub/collection"}, resolver.calls,
			"an ambiguous class id must be resolved by the ICS-721 keeper, not by string shape")
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
