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
type ERC20IBCGate struct {
	keeper *cosmoserc20keeper.Keeper
}

var _ cosmoserc20types.Erc20Keeper = ERC20IBCGate{}

// NewERC20IBCGate wraps the ERC20 keeper, which must be non-nil: a nil keeper
// would make every callback a no-op and silently disable auto registration.
func NewERC20IBCGate(keeper *cosmoserc20keeper.Keeper) ERC20IBCGate {
	if keeper == nil {
		panic("erc20 ibc gate: keeper cannot be nil")
	}

	return ERC20IBCGate{keeper: keeper}
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
// default params it is true, so the common path costs one extra store read and
// nothing else.
func (g ERC20IBCGate) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ack exported.Acknowledgement,
) exported.Acknowledgement {
	if !ack.Success() ||
		g.permissionlessRegistrationEnabled(ctx) ||
		!wouldAutoRegisterTokenPair(ctx, g.keeper, packet) {
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
