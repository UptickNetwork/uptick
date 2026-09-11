package app

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

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
