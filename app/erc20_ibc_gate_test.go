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
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

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

	return inboundPacketTo(t, rawDenom, "")
}

// inboundPacketTo is inboundPacket with the local recipient overridden. An empty
// receiver means the ordinary account inboundPacket uses; the override exists
// because the callback treats a module-account recipient differently -- it
// returns before the registration branch -- and that is a distinct packet shape
// the gate has to be checked against.
func inboundPacketTo(t *testing.T, rawDenom, receiver string) (channeltypes.Packet, string) {
	t.Helper()

	sender := sdk.AccAddress(bytes.Repeat([]byte{0x7a}, 20)).String()
	if receiver == "" {
		receiver = sender
	}
	data := transfertypes.FungibleTokenPacketData{
		Denom:    rawDenom,
		Amount:   "1000",
		Sender:   sender,
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

// TestGateCreatesNoPairForAnyShapeWhileTheSwitchIsOff is the drift canary for the
// gate's predicate.
//
// wouldAutoRegisterTokenPair reimplements the condition under which upstream's
// OnRecvPacket creates a pair instead of sharing it. An upstream release that
// widens that branch -- a new denom prefix, an early return dropped -- leaves the
// gate answering false and delegating, so the callback creates a pair with the
// switch off. The delegation test above cannot see that: it only exercises packets
// the predicate already classifies as "will not register", so a shape upstream
// newly registers is delegated on both sides and compares equal.
//
// The property asserted here is the gate's whole job: with the switch off, no
// packet shape may bring a token pair into existence. Each shape is first run with
// the switch ON as a positive control, because "no pair" is trivially true for a
// shape the callback never registers anyway.
func TestGateCreatesNoPairForAnyShapeWhileTheSwitchIsOff(t *testing.T) {
	app, baseCtx := sharedTestApp(t)

	// The callback returns before the registration branch when the recipient is a
	// module account, which makes that recipient a packet shape of its own.
	moduleReceiver := app.AccountKeeper.GetModuleAccount(baseCtx, cosmoserc20types.ModuleName).
		GetAddress().String()

	// Likewise for the staking denom, which the callback excludes explicitly.
	bondDenom, err := app.StakingKeeper.BondDenom(baseCtx)
	require.NoError(t, err)

	shapes := []struct {
		name string
		// rawDenom is the denom path the counterparty puts in the packet.
		rawDenom string
		// receiver overrides the local recipient; empty means an ordinary account.
		receiver string
		// registers is what the switch-ON run must do: whether this shape is one
		// the callback's registration branch reaches at all.
		registers bool
	}{
		{"unseen voucher", "transfer/channel-0/uatom", "", true},
		{"unseen voucher, second denom", "transfer/channel-0/uosmo", "", true},
		{"token factory denom", "factory/osmo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a/utkn", "", false},
		{"module account recipient", "transfer/channel-0/uion", moduleReceiver, false},
		{"staking denom", transfertypes.PortID + "/" + inboundChannel + "/" + bondDenom, "", false},
	}

	for _, tc := range shapes {
		t.Run(tc.name, func(t *testing.T) {
			// run delivers this shape in the given switch state and reports what
			// the callback did with it. The acknowledgement is kept as the
			// interface both callers return, so the gate and the keeper stay
			// comparable without a type assertion.
			run := func(t *testing.T, permissionless, viaGate bool) (exported.Acknowledgement, string, bool) {
				t.Helper()

				ctx, _ := baseCtx.CacheContext()
				ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
				require.NoError(t, app.Erc20Keeper.SetParams(
					ctx, cosmoserc20types.NewParams(true, permissionless)))

				packet, denom := inboundPacketTo(t, tc.rawDenom, tc.receiver)
				var ack exported.Acknowledgement = channeltypes.NewResultAcknowledgement([]byte{1})
				if viaGate {
					ack = uptickkeepers.NewERC20IBCGate(&app.Erc20Keeper).OnRecvPacket(ctx, packet, ack)
				} else {
					ack = app.Erc20Keeper.OnRecvPacket(ctx, packet, ack)
				}

				return ack, denom, app.Erc20Keeper.IsDenomRegistered(ctx, denom)
			}

			// Positive control: does this shape reach the registration branch at
			// all? Without it the suppression assertion below could be satisfied
			// by a shape the callback ignores for an unrelated reason.
			onAck, onDenom, onRegistered := run(t, true, false)
			require.True(t, onAck.Success(), "the credit itself must survive the callback")
			require.Equal(t, tc.registers, onRegistered,
				"the shape of %s changed upstream, so this case no longer exercises what it claims to; "+
					"re-derive the gate's predicate against ibc_callbacks.go before trusting the assertion below",
				onDenom)

			offAck, _, offRegistered := run(t, false, true)
			require.True(t, offAck.Success(), "gating registration must not fail the transfer")

			if !tc.registers {
				// Nothing here for the gate to suppress. It suppresses these
				// anyway -- the predicate keys on the received denom alone, so a
				// factory/ denom, a module-account recipient and the staking denom
				// all get the suppressed event -- but the acknowledgement must not
				// move, and no pair may appear.
				require.Equal(t, onAck.Acknowledgement(), offAck.Acknowledgement(),
					"the acknowledgement must be identical whether or not the gate suppressed")
				require.False(t, offRegistered)
				return
			}

			// The bare keeper ignores the switch on this path, which is the whole
			// reason the gate exists. Assert it still does: if it ever stops, the
			// suppression assertion below would pass for want of a callback that
			// registers, and the canary would be dead.
			_, bareDenom, bareRegistered := run(t, false, false)
			require.True(t, bareRegistered,
				"the bare keeper no longer registers %s with the switch off, so the gate "+
					"assertion below cannot distinguish suppression from a no-op", bareDenom)

			require.False(t, offRegistered,
				"with PermissionlessRegistration off no shape may bring a token pair into existence")
		})
	}
}
