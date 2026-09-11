package app

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	"github.com/cosmos/evm/ibc"
	cosmoserc20types "github.com/cosmos/evm/x/erc20/types"

	storetypes "cosmossdk.io/store/types"
)

// This file closes the "in-flight packet" half of the migration review: a
// packet that was sent by the v0.3.3 binary but whose acknowledgement or timeout
// is delivered after the v0.4.0 upgrade, when the state it was written against
// has partly been deleted.
//
// The concern is specific. v0.3.3's custom MsgTransferERC20 wrote an
// IBCTransferProvenance record per outbound packet and its refund hook read that
// record back:
//
//	v0.3.3 x/erc20/keeper/msg_server.go:538
//	    if !k.ConsumeIBCTransferProvenance(ctx, packet, data) { return nil }
//
// v0.4.0 deletes those records (app/upgrades/v040/upgrades.go:227) together
// with the legacy OWNER_MODULE token pairs (Step 3). If the replacement hook
// treated "no provenance" or "no token pair" as a failure it would return an
// error, and an error from OnAcknowledgementPacket aborts the ack transaction -
// including the refund the transfer module had just made in the same
// transaction. The packet would then be un-acknowledgeable forever and the funds
// would be stuck.
//
// Two facts make that impossible, and this test pins the second one:
//
//  1. Nothing in the v0.4.x tree reads provenance any more. The identifier only
//     survives inside the v040 cleanup code that deletes it, so the records are
//     unreachable rather than merely stale.
//  2. The replacement hook treats an unresolvable denom as a no-op, not an
//     error: cosmos/evm v0.6.2 x/erc20/keeper/ibc_callbacks.go
//     ConvertCoinToERC20FromPacket returns nil when GetTokenPair finds nothing,
//     and even a failed conversion only emits an event before returning nil.
//
// The user-visible consequence of (2) is mild and worth stating precisely:
// v0.3.3 converted the ERC20 into the Cosmos representation *before* sending, so
// ibc-go's own refund hands back exactly what the sender held at send time. What
// is lost is the automatic re-conversion to ERC20, which the sender can redo
// with MsgConvertCoin.
func TestLegacyERC20PacketRefundCannotStrandThePacket(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	sender := sdk.AccAddress(bytes.Repeat([]byte{0x5a}, 20)).String()
	data := transfertypes.FungibleTokenPacketData{
		Denom:    "erc20legacy", // a denom whose token pair the upgrade deleted
		Amount:   "1000",
		Sender:   sender,
		Receiver: sender,
	}
	packet := channeltypes.Packet{
		Sequence:           1,
		SourcePort:         transfertypes.PortID,
		SourceChannel:      "channel-0",
		DestinationPort:    transfertypes.PortID,
		DestinationChannel: "channel-1",
		Data:               transfertypes.ModuleCdc.MustMarshalJSON(&data),
	}
	ack := channeltypes.NewErrorAcknowledgement(errors.New("counterparty rejected"))

	// The error acknowledgement path. Returning nil here is what keeps the
	// refund that the transfer module made earlier in the same transaction alive.
	require.NoError(t, app.Erc20Keeper.OnAcknowledgementPacket(ctx, packet, data, ack),
		"a packet whose token pair no longer exists must not fail the acknowledgement")

	// The timeout path takes the same branch without inspecting the ack.
	require.NoError(t, app.Erc20Keeper.OnTimeoutPacket(ctx, packet, data),
		"a packet whose token pair no longer exists must not fail the timeout")

	// It must have been a genuine no-op, not an attempted-and-failed conversion.
	for _, ev := range ctx.EventManager().Events() {
		require.NotEqual(t, cosmoserc20types.EventTypeFailedConvertERC20, ev.Type,
			"the legacy denom must be skipped before any conversion is attempted")
	}

	// Negative control: the hook is not simply incapable of failing. A sender
	// the address codec cannot decode is rejected, which proves the nil above
	// came from the "no token pair" branch rather than from a function that
	// swallows everything.
	bad := data
	bad.Sender = "not-an-address"
	require.Error(t, app.Erc20Keeper.OnAcknowledgementPacket(ctx, packet, bad, ack),
		"the hook must still reject input it genuinely cannot process")
}

// The rest of this file closes the denom-form half of the same review.
//
// ConvertCoinToERC20FromPacket does mix two denom forms: it resolves the pair
// with the raw packet denom (ibc_callbacks.go:200, a literal denom-store key)
// and then gates the conversion on ibc.GetSentCoin(data.Denom, ...).Denom,
// which normalises the denom through transfertypes.ExtractDenomFromPath
// (:207, :224). The two forms only diverge when the denom carries a trace, i.e.
// when it is a path shaped "<port>/<channel>/<base>" whose second segment is a
// channel or client identifier.
//
// Nothing the chain can store as a token pair denom has that shape:
//
//	external ERC20, cosmos/evm scheme    MsgRegisterERC20 -> CreateDenom("erc20:"+addr)
//	external ERC20, legacy uptick scheme v0.4.0 keeps the "erc20/"+addr pairs
//	IBC coin auto-extension              RegisterERC20Extension -> "ibc/<hash>", OWNER_MODULE
//
// and the third shape never reaches the gate at all: the switch at :212 answers
// "no-op, received coin is a native coin" for every OWNER_MODULE pair first. So
// on this chain the gate always compares GetSentCoin(denom).Denom against the
// very denom that was just found in the store, and the callback cannot skip a
// refund it should have converted.
//
// Mainnet confirms it: the erc20 store holds four token pairs and all four are
// OWNER_MODULE pairs over "ibc/..." denoms, with no OWNER_EXTERNAL pair at all
// (testnet holds none). The assertions below are bound to the upstream helpers
// that would have to change for that to stop being true, so a future cosmos/evm
// that starts registering trace-shaped denoms turns this red instead of
// silently dropping the automatic re-conversion to ERC20.
func TestERC20RefundDenomFormsCannotDiverge(t *testing.T) {
	const contract = "0xAbC0000000000000000000000000000000000aBc"

	shapes := []struct {
		name  string
		denom string
		owner cosmoserc20types.Owner
	}{
		{"external ERC20, cosmos/evm scheme", cosmoserc20types.CreateDenom(contract), cosmoserc20types.OWNER_EXTERNAL},
		{"external ERC20, legacy uptick scheme", "erc20/" + contract, cosmoserc20types.OWNER_EXTERNAL},
		{"IBC coin auto-extension", "ibc/" + strings.Repeat("AB", 32), cosmoserc20types.OWNER_MODULE},
	}

	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			require.NoError(t, sdk.ValidateDenom(shape.denom))

			// The denom the lookup at :200 uses is already the normalised form,
			// so the form computed at :207 cannot differ from it.
			require.True(t, transfertypes.ExtractDenomFromPath(shape.denom).IsNative())
			require.Equal(t, shape.denom, transfertypes.ExtractDenomFromPath(shape.denom).IBCDenom())
			require.Equal(t, shape.denom, ibc.GetSentCoin(shape.denom, "1").Denom)

			pair := cosmoserc20types.NewTokenPair(common.HexToAddress(contract), shape.denom, shape.owner)
			require.Equal(t, shape.owner == cosmoserc20types.OWNER_EXTERNAL, pair.IsNativeERC20(),
				"only OWNER_EXTERNAL pairs reach the IsDenomRegistered gate at :224")
			require.Equal(t, shape.owner == cosmoserc20types.OWNER_MODULE, pair.IsNativeCoin(),
				"OWNER_MODULE pairs return from case 1 at :212 before the gate is read")
		})
	}

	// Negative control: the helper really does produce two different forms, so
	// the assertions above are not vacuous.
	trace := transfertypes.ExtractDenomFromPath("transfer/channel-0/uatom")
	require.False(t, trace.IsNative())
	require.NotEqual(t, "transfer/channel-0/uatom", trace.IBCDenom())

	// And the reason a legacy "erc20/0x..." denom stays on the safe side of it:
	// ExtractDenomFromPath only reads a hop out of the second segment when that
	// segment looks like a channel or client identifier.
	require.False(t, channeltypes.IsValidChannelID(contract))
	require.True(t, channeltypes.IsValidChannelID("channel-0"))
}

// TestERC20RefundOfAnExternalPairReachesTheConversion is the end-to-end half.
// With a pair whose denom and owner are exactly what MsgRegisterERC20 writes,
// the refund hook must get past the denom gate.
//
// The pair is registered with no contract behind it, so the EVM call that
// follows fails and the hook records EventTypeFailedConvertERC20 - and that
// event is the evidence: it can only be emitted after the gate, and a gate that
// fired would leave no event at all (that is what the test above this one pins
// for the unroutable denom).
func TestERC20RefundOfAnExternalPairReachesTheConversion(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	require.True(t, app.Erc20Keeper.GetParams(ctx).EnableErc20,
		"the gate under test also requires erc20 to be enabled")

	contract := common.HexToAddress("0xAbC0000000000000000000000000000000000aBc")
	denom := cosmoserc20types.CreateDenom(contract.Hex())
	require.NoError(t, app.Erc20Keeper.SetToken(ctx, cosmoserc20types.NewTokenPair(
		contract, denom, cosmoserc20types.OWNER_EXTERNAL,
	)), "the token pair state MsgRegisterERC20 produces")
	require.True(t, app.Erc20Keeper.IsDenomRegistered(ctx, denom))

	sender := sdk.AccAddress(bytes.Repeat([]byte{0x6b}, 20)).String()
	data := transfertypes.FungibleTokenPacketData{
		Denom:    denom,
		Amount:   "1000",
		Sender:   sender,
		Receiver: sender,
	}
	packet := channeltypes.Packet{
		Sequence:           2,
		SourcePort:         transfertypes.PortID,
		SourceChannel:      "channel-0",
		DestinationPort:    transfertypes.PortID,
		DestinationChannel: "channel-1",
		Data:               transfertypes.ModuleCdc.MustMarshalJSON(&data),
	}

	require.NoError(t, app.Erc20Keeper.OnTimeoutPacket(ctx, packet, data))

	var attempted bool
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == cosmoserc20types.EventTypeFailedConvertERC20 {
			attempted = true
		}
	}
	require.True(t, attempted,
		"GetSentCoin(data.Denom).Denom must still resolve to the registered denom, "+
			"otherwise the refund is skipped before any conversion is attempted")
}
