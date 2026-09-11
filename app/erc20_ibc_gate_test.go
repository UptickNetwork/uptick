package app

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	evmibc "github.com/cosmos/evm/ibc"
	cosmoserc20types "github.com/cosmos/evm/x/erc20/types"

	uptickkeepers "github.com/UptickNetwork/uptick/app/keepers"
)

// This file guards the inbound-registration gate (app/keepers/erc20_ibc_gate.go).
//
// The defect it closes: cosmos/evm v0.6.2 x/erc20/keeper/ibc_callbacks.go:100
// creates a token pair - and a dynamic precompile - for any previously unseen
// `ibc/` denom, without consulting Params.PermissionlessRegistration, while
// MsgRegisterERC20 does consult it (keeper/msg_server.go:181-185). The number of
// such pairs is bounded only by the denominations counterparty chains choose to
// send, and the callback zeroes its KV gas config (ibc_callbacks.go:53-56) so the
// relayer does not pay for the writes.
//
// The three behavioural tests below are deliberately split so that a *widened*
// gate cannot pass them either: one pins the default behaviour, one pins the
// suppression, and one pins that nothing except registration is affected.

const (
	// inboundChannel is the channel the inbound packets in this file arrive on.
	// inboundPacketCarrying reuses it as the denom prefix that makes a packet
	// deliver an exact denom.
	inboundChannel = "channel-4"
	inboundPort    = transfertypes.PortID
)

// TestInboundAutoRegistrationRunsUnderDefaultParams is the baseline. The gate
// must be invisible while PermissionlessRegistration is on, which is the shipped
// configuration (x/erc20/types/params.go DefaultParams).
func TestInboundAutoRegistrationRunsUnderDefaultParams(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	require.True(t, app.Erc20Keeper.GetParams(ctx).PermissionlessRegistration,
		"DefaultParams enables permissionless registration; this test pins the baseline the gate must preserve")

	gate := uptickkeepers.NewERC20IBCGate(&app.Erc20Keeper)
	packet, recvDenom := inboundPacket(t, "transfer/channel-0/uatom")
	require.True(t, strings.HasPrefix(recvDenom, "ibc/"),
		"an unseen foreign denom credits as an ibc/<hash> voucher, which is the input the registration branch keys off")

	ack := channeltypes.NewResultAcknowledgement([]byte{1})
	out := gate.OnRecvPacket(ctx, packet, ack)

	require.True(t, out.Success(), "the ICS-20 credit must survive the callback")
	require.True(t, app.Erc20Keeper.IsDenomRegistered(ctx, recvDenom),
		"with the switch on the pair is still created, exactly as the ungated keeper did")
}

// TestInboundAutoRegistrationHonoursPermissionlessRegistration is the fix. With
// the switch off, a denom nobody registered stays a bank voucher instead of
// silently becoming an ERC20 with a precompile behind it.
func TestInboundAutoRegistrationHonoursPermissionlessRegistration(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	ctx, _ := baseCtx.CacheContext()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	require.NoError(t, app.Erc20Keeper.SetParams(ctx, cosmoserc20types.NewParams(true, false)))
	require.False(t, app.Erc20Keeper.GetParams(ctx).PermissionlessRegistration)

	gate := uptickkeepers.NewERC20IBCGate(&app.Erc20Keeper)
	packet, recvDenom := inboundPacket(t, "transfer/channel-0/uosmo")

	ack := channeltypes.NewResultAcknowledgement([]byte{1})
	out := gate.OnRecvPacket(ctx, packet, ack)

	require.True(t, out.Success(), "gating registration must not fail the transfer")
	require.Equal(t, ack.Acknowledgement(), out.Acknowledgement(),
		"the acknowledgement is returned untouched, so the voucher credit is still committed")

	require.False(t, app.Erc20Keeper.IsDenomRegistered(ctx, recvDenom),
		"the governance switch must cover the IBC path: no pair, no dynamic precompile, no unbounded state growth")

	var suppressed bool
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == uptickkeepers.EventTypeAutoRegistrationSuppressed {
			suppressed = true
		}
	}
	require.True(t, suppressed,
		"a suppressed registration has to be observable, otherwise it is indistinguishable from a packet that never matched")
}

// TestERC20IBCGateDelegatesEverythingButRegistration pins the scope of the gate.
// With the switch off, every packet whose callback would not create a pair must
// produce exactly what the bare keeper produces. Without this, turning the gate
// into "drop all callbacks while the switch is off" would still pass the two
// tests above - while silently disabling conversions and the error
// acknowledgement the transfer module relies on.
func TestERC20IBCGateDelegatesEverythingButRegistration(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	const contract = "0xAbC0000000000000000000000000000000000aBc"
	externalDenom := cosmoserc20types.CreateDenom(contract)
	moduleDenom := "ibc/" + strings.Repeat("AB", 32)

	setup, _ := baseCtx.CacheContext()
	setup = setup.WithGasMeter(storetypes.NewInfiniteGasMeter())
	require.NoError(t, app.Erc20Keeper.SetParams(setup, cosmoserc20types.NewParams(true, false)))

	// Two pairs the callback can meet on the receive side: one whose owner sends
	// it down the mint branch, one whose owner makes it a no-op.
	require.NoError(t, app.Erc20Keeper.SetToken(setup, cosmoserc20types.NewTokenPair(
		common.HexToAddress(contract), externalDenom, cosmoserc20types.OWNER_EXTERNAL)),
		"the pair MsgRegisterERC20 produces")
	require.NoError(t, app.Erc20Keeper.SetToken(setup, cosmoserc20types.NewTokenPair(
		common.HexToAddress("0xDeF0000000000000000000000000000000000dEf"), moduleDenom, cosmoserc20types.OWNER_MODULE)),
		"the pair RegisterERC20Extension produces")

	cases := []struct {
		name  string
		build func(t *testing.T) (channeltypes.Packet, string)
		// keeperSucceeds records what the ungated callback does with this
		// packet. The three cases disagree on it, so the equality assertions
		// below cannot be satisfied by three no-ops agreeing with each other.
		keeperSucceeds bool
	}{
		{
			name: "undecodable payload",
			build: func(t *testing.T) (channeltypes.Packet, string) {
				t.Helper()
				return channeltypes.Packet{
					Sequence:           3,
					SourcePort:         inboundPort,
					SourceChannel:      inboundChannel,
					DestinationPort:    inboundPort,
					DestinationChannel: "channel-9",
					Data:               []byte("{ not a FungibleTokenPacketData"),
				}, ""
			},
			// The callback answers with an error acknowledgement.
			keeperSucceeds: false,
		},
		{
			name: "existing external pair hits the mint branch",
			build: func(t *testing.T) (channeltypes.Packet, string) {
				t.Helper()
				return inboundPacketCarrying(t, externalDenom)
			},
			// The mint branch runs and fails: no ERC20 code is deployed at the
			// pair's derived address, so ConvertCoinNativeERC20 cannot read a
			// balance and the callback answers with an error acknowledgement.
			keeperSucceeds: false,
		},
		{
			name: "existing module pair falls through",
			build: func(t *testing.T) (channeltypes.Packet, string) {
				t.Helper()
				return inboundPacketCarrying(t, moduleDenom)
			},
			// Nothing to convert, the acknowledgement is returned untouched.
			keeperSucceeds: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packet, recvDenom := tc.build(t)

			gateCtx, _ := setup.CacheContext()
			gateCtx = gateCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
			keeperCtx, _ := setup.CacheContext()
			keeperCtx = keeperCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())

			ack := channeltypes.NewResultAcknowledgement([]byte{1})
			viaGate := uptickkeepers.NewERC20IBCGate(&app.Erc20Keeper).OnRecvPacket(gateCtx, packet, ack)
			viaKeeper := app.Erc20Keeper.OnRecvPacket(keeperCtx, packet, ack)

			require.Equal(t, tc.keeperSucceeds, viaKeeper.Success(),
				"the ungated callback changed behaviour; this case no longer exercises the branch it claims to")

			require.Equal(t, viaKeeper.Success(), viaGate.Success(),
				"the gate changed whether the callback succeeded")
			require.Equal(t, viaKeeper.Acknowledgement(), viaGate.Acknowledgement(),
				"the gate changed the acknowledgement the relayer would write")
			require.Equal(t, eventTypes(keeperCtx), eventTypes(gateCtx),
				"the gate changed the events the callback emits")

			if recvDenom != "" {
				require.Equal(t,
					app.Erc20Keeper.IsDenomRegistered(keeperCtx, recvDenom),
					app.Erc20Keeper.IsDenomRegistered(gateCtx, recvDenom),
					"the gate changed whether %s is registered", recvDenom)
			}
		})
	}
}

// TestERC20IBCMiddlewareIsWiredToTheGate asserts the wiring. The gate is only
// worth anything if cosmoserc20.NewIBCMiddleware is handed it rather than the
// bare keeper, and that is not observable from a running app: the middleware
// stores the interface it is given.
//
// The assertion is made on the source for the same reason
// cmd/uptickd/genesis_denom_order_test.go makes its ordering assertion there -
// and with the same guard against prose: comments are stripped first, so
// mentioning NewERC20IBCGate in a comment cannot satisfy it.
func TestERC20IBCMiddlewareIsWiredToTheGate(t *testing.T) {
	raw, err := os.ReadFile("keepers/keepers.go")
	require.NoError(t, err)

	code := stripLineComments(string(raw))
	flat := strings.Join(strings.Fields(code), " ")

	// strings.Contains rather than require.Contains: the latter prints the whole
	// flattened file on failure, which buries the message.
	require.True(t,
		strings.Contains(flat, "transferStack := cosmoserc20.NewIBCMiddleware( NewERC20IBCGate("),
		"the ICS-20 transfer middleware must be handed NewERC20IBCGate, not the bare ERC20 keeper: "+
			"upstream's inbound RegisterERC20Extension branch reads no governance parameter, so "+
			"reverting this re-opens the bypass of Params.PermissionlessRegistration")
}

// inboundPacket builds the ICS-20 packet a counterparty chain sends when it moves
// one of its own denoms to Uptick, and returns the denom Uptick holds after the
// credit. The trace does not match the receiving channel, so the shared transfer
// module keeps it and the voucher is a fresh ibc/<hash>.
func inboundPacket(t *testing.T, rawDenom string) (channeltypes.Packet, string) {
	t.Helper()

	receiver := sdk.AccAddress(bytes.Repeat([]byte{0x7a}, 20)).String()
	data := transfertypes.FungibleTokenPacketData{
		Denom:    rawDenom,
		Amount:   "1000",
		Sender:   receiver,
		Receiver: receiver,
	}
	packet := channeltypes.Packet{
		Sequence:           11,
		SourcePort:         inboundPort,
		SourceChannel:      inboundChannel,
		DestinationPort:    inboundPort,
		DestinationChannel: "channel-9",
		Data:               transfertypes.ModuleCdc.MustMarshalJSON(&data),
	}

	// Derived with the same helper the gate and the callback use, so the test
	// cannot silently disagree with them about what was delivered.
	coin := evmibc.GetReceivedCoin(packet, transfertypes.Token{
		Denom:  transfertypes.ExtractDenomFromPath(data.Denom),
		Amount: data.Amount,
	})

	return packet, coin.Denom
}

// inboundPacketCarrying builds a packet that delivers exactly denom: the
// transfer module strips a trace prefix when it matches the receiving channel
// (ibc-go v10 Denom.HasPrefix / getReceivedCoin), so
// "<transfer>/<inboundChannel>/<denom>" credits as <denom>. A packet sent back
// over the channel a token left by really does look like this.
func inboundPacketCarrying(t *testing.T, denom string) (channeltypes.Packet, string) {
	t.Helper()

	packet, received := inboundPacket(t, inboundPort+"/"+inboundChannel+"/"+denom)
	require.Equal(t, denom, received, "the helper failed to construct a packet carrying %s", denom)

	return packet, received
}

func eventTypes(ctx sdk.Context) []string {
	events := ctx.EventManager().Events()
	types := make([]string, 0, len(events))
	for _, ev := range events {
		types = append(types, ev.Type)
	}
	return types
}

// stripLineComments removes line comments so a marker appearing only in prose
// cannot satisfy (or defeat) an assertion made on the source.
func stripLineComments(src string) string {
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx != -1 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}
