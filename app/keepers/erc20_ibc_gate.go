package keepers

import (
	"strings"

	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	evmibc "github.com/cosmos/evm/ibc"
	cosmoserc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	cosmoserc20types "github.com/cosmos/evm/x/erc20/types"

	uptickibc "github.com/UptickNetwork/uptick/ibc"
)

// EventTypeAutoRegistrationSuppressed is the observable counterpart of the gate
// below: emitted when an inbound ICS-20 transfer is left as a plain bank voucher
// because Params.PermissionlessRegistration is off, so a suppressed registration
// can be told apart from a packet that never reached the registration branch.
const EventTypeAutoRegistrationSuppressed = "erc20_auto_registration_suppressed"

// ERC20IBCGate sits between the ERC20 IBC middleware and the keeper so the inbound
// ICS-20 callback can be gated by Params.PermissionlessRegistration. It implements
// cosmoserc20types.Erc20Keeper, so NewIBCMiddleware can be handed it instead of the
// bare keeper.
//
// Why the gate exists. The callback creates a token pair - and with it a dynamic
// precompile account - for any previously unseen `ibc/` denom it is handed
// (cosmos/evm v0.6.2 x/erc20/keeper/ibc_callbacks.go:100):
//
//	case !found && strings.HasPrefix(coin.Denom, "ibc/"):
//	    tokenPair, err := k.RegisterERC20Extension(ctx, coin.Denom)
//
// Neither RegisterERC20Extension nor EnableDynamicPrecompile reads any governance
// parameter, whereas MsgRegisterERC20 goes through Params.PermissionlessRegistration
// (keeper/msg_server.go:181-185). Every denom a counterparty chain picks hashes to a
// distinct `ibc/<hash>` and the callback runs with a zeroed KV gas config
// (ibc_callbacks.go:53-56), so this branch can create unbounded pairs that are never
// charged to the relayer: governance can switch permissionless registration off and
// the IBC path keeps creating them.
//
// Scope. The gate suppresses exactly one branch - the registration above. Everything
// else is delegated verbatim, so under the default params (PermissionlessRegistration
// = true, see types.DefaultParams) behavior is unchanged. With the switch off an
// inbound packet for an unknown denom is still received and credited; it just stays a
// bank voucher instead of silently becoming an ERC20. Turning the switch back on
// restores the old behavior, and no state written by the gate needs undoing.
//
// Decimals. The gate also repairs the bank metadata that the registration branch is
// about to freeze into a dynamic precompile, for the same denom set the suppression
// guards. The precompile reads decimals() out of that metadata
// (precompiles/erc20/query.go:115-128), and ibc-go writes the shape whose last-segment
// match lands on the source denom's exponent-0 unit - so decimals() returns 0 for every
// voucher the callback registers. That is the same defect the v0.5.0 handler repairs
// for the vouchers that already existed, and it is repaired through the same
// implementation (ibc.NormalizeVoucherDecimals) so decimals() stops depending on
// whether a pair predates the upgrade. It runs only when the callback will in fact
// register, and it is idempotent, so a re-delivered packet costs reads.
//
// The repair is not a widening of the gate: it writes no token pair, and it is scoped
// to the one denomination the packet carries. It does add a store write per newly
// registered voucher - the same one the callback is about to make unbounded.
type ERC20IBCGate struct {
	keeper     *cosmoserc20keeper.Keeper
	bankKeeper uptickibc.VoucherBankStore
}

var _ cosmoserc20types.Erc20Keeper = ERC20IBCGate{}

// NewERC20IBCGate wraps the ERC20 keeper, which must be non-nil: a nil keeper
// would make every callback a no-op and silently disable auto registration. The
// bank keeper is required for the metadata repair and must be non-nil for the same
// reason - a nil one would leave every newly registered voucher at decimals()==0
// while looking like it works.
func NewERC20IBCGate(
	keeper *cosmoserc20keeper.Keeper,
	bankKeeper uptickibc.VoucherBankStore,
) ERC20IBCGate {
	if keeper == nil {
		panic("erc20 ibc gate: keeper cannot be nil")
	}
	if bankKeeper == nil {
		panic("erc20 ibc gate: bank keeper cannot be nil")
	}

	return ERC20IBCGate{keeper: keeper, bankKeeper: bankKeeper}
}

// Logger delegates so log output keeps the keeper's logger.
func (g ERC20IBCGate) Logger(ctx sdk.Context) log.Logger {
	return g.keeper.Logger(ctx)
}

// OnAcknowledgementPacket delegates: the refund path never registers a pair.
func (g ERC20IBCGate) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
	ack channeltypes.Acknowledgement,
) error {
	return g.keeper.OnAcknowledgementPacket(ctx, packet, data, ack)
}

// OnTimeoutPacket delegates: the refund path never registers a pair.
func (g ERC20IBCGate) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
) error {
	return g.keeper.OnTimeoutPacket(ctx, packet, data)
}

// OnRecvPacket returns the ICS-20 acknowledgement untouched when permissionless
// registration is off and the callback's only remaining effect would be to create a
// token pair; otherwise it delegates. The switch is read first on purpose: under the
// default params it is true, and the common path then costs one extra store read plus
// the registration predicate the metadata repair needs, and nothing else.
//
// The predicate is what makes the repair safe to run on every inbound packet: only a
// packet the callback is about to register for has metadata worth rewriting, and the
// rewrite is idempotent, so an established denom is a pair lookup plus nothing.
func (g ERC20IBCGate) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ack exported.Acknowledgement,
) exported.Acknowledgement {
	if ack.Success() && g.permissionlessRegistrationEnabled(ctx) {
		g.normalizeVoucherDecimals(ctx, packet)
		return g.keeper.OnRecvPacket(ctx, packet, ack)
	}

	if !ack.Success() || !wouldAutoRegisterTokenPair(ctx, g.keeper, packet) {
		return g.keeper.OnRecvPacket(ctx, packet, ack)
	}

	// Registration is off: the callback would mint a token pair nobody asked for, so
	// return the acknowledgement unchanged - credit stays committed, voucher stays plain.
	denom, _ := receivedDenom(packet)

	g.keeper.Logger(ctx).Info(
		"erc20 auto-registration suppressed by PermissionlessRegistration",
		"denom", denom,
		"source_port", packet.SourcePort,
		"source_channel", packet.SourceChannel,
	)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeAutoRegistrationSuppressed,
			sdk.NewAttribute(cosmoserc20types.AttributeKeyCosmosCoin, denom),
			sdk.NewAttribute(cosmoserc20types.AttributeCoinSourceChannel, packet.SourceChannel),
		),
	)

	return ack
}

// normalizeVoucherDecimals rewrites the inbound voucher's bank metadata into the shape
// that makes the dynamic precompile report the source denom's own exponent, when - and
// only when - the callback is about to register this denomination.
//
// The guard is the gate's own predicate rather than a second copy of it, so the two
// cannot drift apart about which packets reach the registration branch.
//
// Not failing the packet is deliberate. A record that cannot be repaired is skipped by
// NormalizeVoucherDecimals with a reason, and leaving that pair at decimals()==0 is the
// status quo the v0.5.0 handler already accepts for unreadable vouchers; rejecting the
// acknowledgement over a cosmetic field would instead strand a real transfer.
func (g ERC20IBCGate) normalizeVoucherDecimals(ctx sdk.Context, packet channeltypes.Packet) {
	if !wouldAutoRegisterTokenPair(ctx, g.keeper, packet) {
		return
	}

	denom, ok := receivedDenom(packet)
	if !ok {
		return
	}

	result := uptickibc.NormalizeVoucherDecimals(ctx, g.bankKeeper, denom)
	switch result.Outcome {
	case uptickibc.NormalizationApplied:
		g.keeper.Logger(ctx).Info(
			"normalized inbound ibc voucher erc20 decimals",
			"denom", denom,
			"decimals", result.Decimals,
			"source_port", packet.SourcePort,
			"source_channel", packet.SourceChannel,
		)
	case uptickibc.NormalizationSkipped:
		// The denom keeps decimals()==0 rather than getting a guessed exponent; the
		// reason has to reach the log, because nothing else about the transfer differs.
		g.keeper.Logger(ctx).Info(
			"skipping inbound ibc voucher metadata",
			"denom", denom,
			"reason", result.Reason,
		)
	}
}

func (g ERC20IBCGate) permissionlessRegistrationEnabled(ctx sdk.Context) bool {
	return g.keeper.GetParams(ctx).PermissionlessRegistration
}

// wouldAutoRegisterTokenPair reports whether the upstream callback would take its
// RegisterERC20Extension branch for this packet.
//
// The predicate mirrors ibc_callbacks.go:95-100 - no pair for the received denom and
// that denom carries the `ibc/` prefix - and is deliberately loose in the safe
// direction: answering true requires the received denom to have no token pair, which is
// the registration branch's precondition. The callback's other state-writing branch
// (ibc_callbacks.go:119) requires a pair to exist, and the remaining branches (module
// account receiver, `factory/` denom, bond denom) write no state at all, so in each case
// answering true returns the same acknowledgement the callback would have returned.
//
// The only way it can be wrong is by answering false when the callback would in fact
// register, which leaks the gate rather than breaking a legitimate path.
//
// Answering true when the callback would have returned early anyway is allowed, and
// does happen: the predicate deliberately does not replicate the callback's other
// early returns, so a `factory/` denom, a module-account recipient and the staking
// denom all answer true. The acknowledgement is identical either way, so nothing
// behaves differently -- but the suppressed event and log line are emitted for those
// packets too, and an operator counting them is counting packets that were never
// going to register. TestGateCreatesNoPairForAnyShapeWhileTheSwitchIsOff pins both
// halves: no pair for any shape, and identical acknowledgements for the shapes the
// callback would have ignored.
func wouldAutoRegisterTokenPair(
	ctx sdk.Context,
	keeper *cosmoserc20keeper.Keeper,
	packet channeltypes.Packet,
) bool {
	denom, ok := receivedDenom(packet)
	if !ok {
		// Undecodable payload: delegate so the upstream callback keeps emitting
		// the error acknowledgement it has always emitted.
		return false
	}

	if _, found := keeper.GetTokenPair(ctx, keeper.GetTokenPairID(ctx, denom)); found {
		return false
	}

	return strings.HasPrefix(denom, "ibc/")
}

// receivedDenom mirrors the upstream callback's denom computation
// (ibc_callbacks.go:72-77) and reuses the helpers it calls, so the two cannot drift.
func receivedDenom(packet channeltypes.Packet) (string, bool) {
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return "", false
	}

	coin := evmibc.GetReceivedCoin(packet, transfertypes.Token{
		Denom:  transfertypes.ExtractDenomFromPath(data.Denom),
		Amount: data.Amount,
	})

	return coin.Denom, true
}
