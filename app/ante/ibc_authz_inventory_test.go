package ante

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	cosmosante "github.com/cosmos/evm/ante/cosmos"

	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channelv2types "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

// This file turns "which IBC messages may be delegated through authz" from a
// hand-maintained list into a property that is checked against the protobuf
// descriptors.
//
// Why it exists: the list in disabled_authz.go had been reconciled by hand, and
// by hand it missed ibc.applications.nft_transfer.v1.MsgTransfer — the main
// asset exit for an NFT chain, and the only IBC message whose delegation had no
// scoped alternative (nft-transfer ships no Authorization type, so
// GenericAuthorization, with no limit and no allow-list, was the only grant that
// could be made). A table assembled from five modules cannot stay in sync by
// inspection; discovery can.
//
// The classification below is exhaustive by construction: every request message
// exposed by an "ibc.*" gRPC Msg service must appear in exactly one of
// ibcAuthzGrantable and the "/ibc." entries of DisabledAuthzMsgTypeURLs(). When
// an ibc-go bump adds or renames a message the enumeration grows, the partition
// stops matching, and this file fails with the name of the unclassified type
// instead of silently defaulting it to grantable.

// ibcAuthzGrantable is the deliberate complement of the "/ibc." entries in
// DisabledAuthzMsgTypeURLs(): every IBC request message that stays grantable on
// purpose, grouped by the reason it may stay.
//
// The test is what makes the deliberate part real. An entry here is a claim that
// the message carries no signer check — that executing it as the granter achieves
// nothing the grantee could not achieve by sending it from its own account. Each
// group cites the handler that was read to make that claim.
var ibcAuthzGrantable = []string{
	// Relayer packet flow (ibc-go modules/core/keeper/msg_server.go:230-430).
	// No signer check in any of them: a packet is authenticated by its proofs
	// and commitments, not by who submitted it. This is the surface an operator
	// actually delegates to a relaying bot, and it must keep working.
	sdk.MsgTypeURL(&channeltypes.MsgRecvPacket{}),
	sdk.MsgTypeURL(&channeltypes.MsgAcknowledgement{}),
	sdk.MsgTypeURL(&channeltypes.MsgTimeout{}),
	sdk.MsgTypeURL(&channeltypes.MsgTimeoutOnClose{}),

	// Channel handshake (msg_server.go:296-400). MsgChannelCloseInit is the
	// clearest case: it reads only msg.PortId and msg.ChannelId, never
	// msg.Signer (msg_server.go:374-400), so anyone can close any channel from
	// their own account. Blocking it would remove nothing.
	sdk.MsgTypeURL(&channeltypes.MsgChannelOpenInit{}),
	sdk.MsgTypeURL(&channeltypes.MsgChannelOpenTry{}),
	sdk.MsgTypeURL(&channeltypes.MsgChannelOpenAck{}),
	sdk.MsgTypeURL(&channeltypes.MsgChannelOpenConfirm{}),
	sdk.MsgTypeURL(&channeltypes.MsgChannelCloseInit{}),
	sdk.MsgTypeURL(&channeltypes.MsgChannelCloseConfirm{}),

	// IBC v2 packet flow. ibc-go's core module registers the v2 Msg services
	// unconditionally (modules/core/module.go:142 and :145), so these are
	// routable — Uptick simply never calls SetRouterV2, so the packet paths would
	// fail on a nil router. What matters for this list is who they read:
	// MsgSendPacket and MsgTimeout read no signer at all, while RecvPacket and
	// Acknowledgement call IsAllowedRelayer against the destination/source
	// client's config (modules/core/04-channel/v2/keeper/msg_server.go:56-60 and
	// :164-168). That allow-list is opt-in per client and empty by default, where
	// IsAllowedRelayer returns true (02-client/v2/types/config.go:19), so
	// unconfigured clients admit everybody — the same reasoning as MsgUpdateClient
	// above, and the same test pins the default.
	sdk.MsgTypeURL(&channelv2types.MsgSendPacket{}),
	sdk.MsgTypeURL(&channelv2types.MsgRecvPacket{}),
	sdk.MsgTypeURL(&channelv2types.MsgAcknowledgement{}),
	sdk.MsgTypeURL(&channelv2types.MsgTimeout{}),

	// Connection handshake (msg_server.go:176-290). No signer check.
	sdk.MsgTypeURL(&connectiontypes.MsgConnectionOpenInit{}),
	sdk.MsgTypeURL(&connectiontypes.MsgConnectionOpenTry{}),
	sdk.MsgTypeURL(&connectiontypes.MsgConnectionOpenAck{}),
	sdk.MsgTypeURL(&connectiontypes.MsgConnectionOpenConfirm{}),

	// Client creation and misbehaviour submission. MsgCreateClient does write
	// msg.Signer — it records the creator (msg_server.go:46-47) — but the
	// outcome it produces (a client exists) is one the grantee can obtain from
	// its own account; only the recorded creator differs, and that is the
	// granter's choice to make. MsgSubmitMisbehaviour routes to
	// ClientKeeper.UpdateClient exactly like MsgUpdateClient and checks nothing
	// (msg_server.go:127-140).
	sdk.MsgTypeURL(&ibcclienttypes.MsgCreateClient{}),
	sdk.MsgTypeURL(&ibcclienttypes.MsgSubmitMisbehaviour{}),

	// IBC client maintenance. Both used to be disabled and were removed in round
	// 25: neither satisfies the rule, so keeping them made the list look like it
	// only mostly followed its own predicate.
	//
	// MsgUpdateClient's relayer allow-list always runs (ibc-go constructs
	// ClientV2Keeper itself, modules/core/keeper/keeper.go:53) but admits everyone
	// until a client's creator configures one — GetConfig returns an empty Config
	// and IsAllowedRelayer reads that as "no restriction". ibc-go says so itself:
	// "DefaultConfig is empty and therefore permissionless"
	// (02-client/v2/types/config.go:19). Delegating an update for a client that
	// *has* opted into an allow-list is the list owner consenting to be relayed
	// for, not an escalation.
	//
	// MsgUpgradeClient has no signer check to find at all: its handler
	// (msg_server.go:108-122) never reads msg.Signer and the keeper it calls
	// (modules/core/02-client/keeper/client.go:88) takes no signer argument, so a
	// grant for it could never restrict anything — it only looked like it did.
	//
	// Both are pinned by ibcAuthzMustStayGrantable so a future round cannot
	// quietly re-add them, and the permissive default they rest on is pinned by
	// app/ibc_v2_authz_guard_test.go.
	sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
	sdk.MsgTypeURL(&ibcclienttypes.MsgUpgradeClient{}),

	// ICA host query relay. "This handler doesn't use the signer"
	// (apps/27-interchain-accounts/host/keeper/msg_server.go:30) and the
	// requests are restricted to the module_query_safe allow-list, so the
	// grantee gains nothing.
	sdk.MsgTypeURL(&icahosttypes.MsgModuleQuerySafe{}),
}

// ibcAuthzMustStayDisabled pins the entries whose removal would restore an
// unbounded delegation: the two IBC transfers move assets, and the two ICA
// controller messages drive a whole account on the host chain. These are
// asserted by name so that deleting one from disabled_authz.go fails here
// rather than passing the partition check (an entry moved from "disabled" to
// "grantable" still partitions the surface).
var ibcAuthzMustStayDisabled = []string{
	sdk.MsgTypeURL(&transfertypes.MsgTransfer{}),
	sdk.MsgTypeURL(&nfttransfertypes.MsgTransfer{}),
	sdk.MsgTypeURL(&icacontrollertypes.MsgSendTx{}),
	sdk.MsgTypeURL(&icacontrollertypes.MsgRegisterInterchainAccount{}),
}

// ibcAuthzMustStayGrantable pins the relayer surface in the other direction, so
// that a future round tightening the list cannot quietly break delegated
// relaying by adding these. None of them carries a live signer check, so listing
// one would only remove an operator capability while adding no defence.
//
// MsgUpdateClient and MsgUpgradeClient belong here for the reason given on their
// entry above; they are pinned so that "put them back, it's safer" has to argue
// with a failing test rather than with a comment.
var ibcAuthzMustStayGrantable = []string{
	sdk.MsgTypeURL(&channeltypes.MsgRecvPacket{}),
	sdk.MsgTypeURL(&channeltypes.MsgAcknowledgement{}),
	sdk.MsgTypeURL(&channeltypes.MsgTimeout{}),
	sdk.MsgTypeURL(&channeltypes.MsgTimeoutOnClose{}),
	sdk.MsgTypeURL(&channeltypes.MsgChannelOpenInit{}),
	sdk.MsgTypeURL(&channeltypes.MsgChannelCloseInit{}),
	sdk.MsgTypeURL(&connectiontypes.MsgConnectionOpenInit{}),
	sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
	sdk.MsgTypeURL(&ibcclienttypes.MsgUpgradeClient{}),
}

// ibcMsgServiceRequestTypeURLs discovers every request message exposed by an
// "ibc.*" gRPC Msg service, read straight out of the protobuf file descriptors.
//
// Nothing about the result is written down ahead of time, which is the point:
// the first run returned 38 types, eleven more than the hand-written table this
// replaced, including the whole 27-interchain-accounts family and the ibc-go v2
// channel messages.
func ibcMsgServiceRequestTypeURLs(t *testing.T) map[string]bool {
	t.Helper()

	// Registering the modules is what puts the app-shaped set of modules under
	// test; it mirrors app.go's ModuleBasics. The descriptors themselves come
	// from the gogoproto registry, which every linked package populates in init.
	registry := codectypes.NewInterfaceRegistry()
	for _, register := range []func(codectypes.InterfaceRegistry){
		ibcclienttypes.RegisterInterfaces,
		clientv2types.RegisterInterfaces,
		connectiontypes.RegisterInterfaces,
		channeltypes.RegisterInterfaces,
		channelv2types.RegisterInterfaces,
		transfertypes.RegisterInterfaces,
		nfttransfertypes.RegisterInterfaces,
		icacontrollertypes.RegisterInterfaces,
		icahosttypes.RegisterInterfaces,
	} {
		register(registry)
	}

	found := map[string]bool{}
	registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			if !strings.HasSuffix(string(svc.FullName()), ".Msg") {
				continue
			}
			methods := svc.Methods()
			for j := 0; j < methods.Len(); j++ {
				name := string(methods.Get(j).Input().FullName())
				if !strings.HasPrefix(name, "ibc.") {
					continue
				}
				found["/"+name] = true
			}
		}
		return true
	})

	// A descriptor walk that silently returned nothing would make the partition
	// assertion below pass for the wrong reason.
	require.NotEmpty(t, found, "the protobuf walk found no ibc.* Msg services at all")
	return found
}

func typeURLSet(urls []string) map[string]bool {
	out := make(map[string]bool, len(urls))
	for _, u := range urls {
		out[u] = true
	}
	return out
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// disabledIBCTypeURLs returns the "/ibc." entries of the production list.
func disabledIBCTypeURLs() []string {
	var out []string
	for _, u := range DisabledAuthzMsgTypeURLs() {
		if strings.HasPrefix(u, "/ibc.") {
			out = append(out, u)
		}
	}
	return out
}

// TestIBCAuthzInventoryPartitionsTheIBCMsgSurface is the guard that makes the
// list mechanical. It fails in three distinct ways, each with its own diagnostic:
// a registered IBC message that nobody classified, a classified URL that no
// longer exists (a rename or a drop), and an entry claimed as both disabled and
// grantable.
func TestIBCAuthzInventoryPartitionsTheIBCMsgSurface(t *testing.T) {
	enumerated := ibcMsgServiceRequestTypeURLs(t)
	disabled := typeURLSet(disabledIBCTypeURLs())
	grantable := typeURLSet(ibcAuthzGrantable)

	classified := map[string]bool{}
	for u := range disabled {
		classified[u] = true
	}
	for u := range grantable {
		classified[u] = true
	}

	var unclassified []string
	for u := range enumerated {
		if !classified[u] {
			unclassified = append(unclassified, u)
		}
	}
	sort.Strings(unclassified)
	require.Empty(t, unclassified,
		"these IBC Msg types are registered but classified nowhere; add each to disabled_authz.go (identity-bearing) or ibcAuthzGrantable (permissionless)")

	var stale []string
	for u := range classified {
		if !enumerated[u] {
			stale = append(stale, u)
		}
	}
	sort.Strings(stale)
	require.Empty(t, stale,
		"these classified type URLs are no longer registered by any ibc.* Msg service; drop them or fix the type")

	var both []string
	for u := range disabled {
		if grantable[u] {
			both = append(both, u)
		}
	}
	sort.Strings(both)
	require.Empty(t, both, "a message cannot be both disabled and deliberately grantable")

	t.Logf("IBC Msg surface: %d registered, %d disabled, %d deliberately grantable",
		len(enumerated), len(disabled), len(grantable))
}

// TestIBCAuthzInventoryPinsTheCriticalEntries is the subjective half, in both
// directions: the entries whose loss would be a security regression must be
// disabled, and the relayer surface that must keep working must stay grantable.
// The partition check cannot see either property on its own — moving an entry
// from one list to the other keeps the partition valid.
func TestIBCAuthzInventoryPinsTheCriticalEntries(t *testing.T) {
	disabled := typeURLSet(disabledIBCTypeURLs())
	grantable := typeURLSet(ibcAuthzGrantable)

	for _, u := range ibcAuthzMustStayDisabled {
		require.True(t, disabled[u],
			"%s moves assets or drives an account, so it must not be delegatable", u)
		require.False(t, grantable[u], "%s must not also be listed as deliberately grantable", u)
	}

	for _, u := range ibcAuthzMustStayGrantable {
		require.True(t, grantable[u],
			"%s carries no live signer check, so blocking it removes an operator capability without adding defence", u)
		require.False(t, disabled[u], "%s must not be disabled", u)
	}
}

// TestDisabledAuthzMsgTypeURLsHasNoDuplicates guards the list itself: the
// limiter compares type URLs with ==, so a duplicate is invisible at runtime and
// only shows up as a confusing diff later.
func TestDisabledAuthzMsgTypeURLsHasNoDuplicates(t *testing.T) {
	seen := map[string]int{}
	for _, u := range DisabledAuthzMsgTypeURLs() {
		seen[u]++
	}

	var dupes []string
	for u, n := range seen {
		if n > 1 {
			dupes = append(dupes, u)
		}
	}
	sort.Strings(dupes)
	require.Empty(t, dupes, "duplicate entries in DisabledAuthzMsgTypeURLs")
}

// TestAuthzLimiterBlocksIBCIdentityMessagesInMsgExec is the behavioural half of
// the guard: it runs the real decorator over a real MsgExec and requires a
// rejection. A type URL that appears in the list but is not the URL the
// decorator computes would pass the partition check above and fail here.
//
// Zero-value messages are enough because the decorator only ever reads
// sdk.MsgTypeURL(msg).
func TestAuthzLimiterBlocksIBCIdentityMessagesInMsgExec(t *testing.T) {
	cases := []struct {
		name string
		msg  sdk.Msg
	}{
		{"ICS-20 transfer moves the granter's tokens", &transfertypes.MsgTransfer{}},
		{"ICS-721 transfer moves the granter's NFTs", &nfttransfertypes.MsgTransfer{}},
		{"ICA SendTx drives the granter's interchain account", &icacontrollertypes.MsgSendTx{}},
		{"ICA registration binds the account to the granter", &icacontrollertypes.MsgRegisterInterchainAccount{}},
		{"client recovery is authority-gated", &ibcclienttypes.MsgRecoverClient{}},
		{"connection params are authority-gated", &connectiontypes.MsgUpdateParams{}},
		{"deleting a client creator is creator-gated", &ibcclienttypes.MsgDeleteClientCreator{}},
		{"registering a counterparty is creator-gated", &clientv2types.MsgRegisterCounterparty{}},
	}

	grantee := sdk.AccAddress([]byte("grantee-address-bytes"))
	dec := cosmosante.NewAuthzLimiterDecorator(DisabledAuthzMsgTypeURLs()...)
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exec := authz.NewMsgExec(grantee, []sdk.Msg{tc.msg})
			_, err := dec.AnteHandle(sdk.Context{}, &mockTxWithMsgs{msgs: []sdk.Msg{&exec}}, false, next)
			require.Error(t, err, "%s must not be executable inside MsgExec", tc.name)
			require.Contains(t, err.Error(), sdk.MsgTypeURL(tc.msg))
		})
	}
}

// TestAuthzLimiterBlocksGenericGrantForIBCAssetMessages covers the other half of
// the limiter: MsgGrant. It matters here more than for the other entries,
// because nft-transfer defines no Authorization of its own, so a
// GenericAuthorization for ICS-721 MsgTransfer — no spend limit, no allow-list —
// was the only grant that could ever be written. If this check regresses, the
// delegated NFT exit reopens silently.
func TestAuthzLimiterBlocksGenericGrantForIBCAssetMessages(t *testing.T) {
	cases := []struct {
		name string
		msg  sdk.Msg
	}{
		{"ICS-20", &transfertypes.MsgTransfer{}},
		{"ICS-721", &nfttransfertypes.MsgTransfer{}},
	}

	granter := sdk.AccAddress([]byte("granter-address-bytes"))
	grantee := sdk.AccAddress([]byte("grantee-address-bytes"))
	dec := cosmosante.NewAuthzLimiterDecorator(DisabledAuthzMsgTypeURLs()...)
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			generic := authz.NewGenericAuthorization(sdk.MsgTypeURL(tc.msg))
			msgGrant, err := authz.NewMsgGrant(granter, grantee, generic, nil)
			require.NoError(t, err)

			_, err = dec.AnteHandle(sdk.Context{}, &mockTxWithMsgs{msgs: []sdk.Msg{msgGrant}}, false, next)
			require.Error(t, err, "a GenericAuthorization for %s must be refused at grant time", sdk.MsgTypeURL(tc.msg))
			require.Contains(t, err.Error(), sdk.MsgTypeURL(tc.msg))
		})
	}
}

// TestAuthzLimiterAllowsIBCRelayerMessagesInMsgExec is the reverse sentinel for
// the block above: the relayer surface must remain delegatable. Without it, the
// cheapest way to make the block test pass — listing the whole IBC surface —
// would look green while quietly breaking delegated relaying.
func TestAuthzLimiterAllowsIBCRelayerMessagesInMsgExec(t *testing.T) {
	cases := []struct {
		name string
		msg  sdk.Msg
	}{
		{"v1 packet receipt", &channeltypes.MsgRecvPacket{}},
		{"v1 packet ack", &channeltypes.MsgAcknowledgement{}},
		{"v1 packet timeout", &channeltypes.MsgTimeout{}},
		{"v2 packet send", &channelv2types.MsgSendPacket{}},
		{"channel open", &channeltypes.MsgChannelOpenInit{}},
		{"one-sided channel close", &channeltypes.MsgChannelCloseInit{}},
		{"connection open", &connectiontypes.MsgConnectionOpenInit{}},
		{"client creation", &ibcclienttypes.MsgCreateClient{}},
		{"client update", &ibcclienttypes.MsgUpdateClient{}},
		{"client upgrade", &ibcclienttypes.MsgUpgradeClient{}},
		{"ica host query relay", &icahosttypes.MsgModuleQuerySafe{}},
	}

	grantee := sdk.AccAddress([]byte("grantee-address-bytes"))
	dec := cosmosante.NewAuthzLimiterDecorator(DisabledAuthzMsgTypeURLs()...)
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exec := authz.NewMsgExec(grantee, []sdk.Msg{tc.msg})
			_, err := dec.AnteHandle(sdk.Context{}, &mockTxWithMsgs{msgs: []sdk.Msg{&exec}}, false, next)
			require.NoError(t, err, "%s carries no signer check and must stay delegatable", tc.name)
		})
	}
}

// TestAuthzLimiterStillAllowsUnrelatedMessages keeps the broadest sentinel in
// place: the list gained fifteen IBC entries this round and lost two, and a
// predicate that over-matched (a prefix check instead of an equality check, say)
// would start rejecting messages that have nothing to do with IBC.
func TestAuthzLimiterStillAllowsUnrelatedMessages(t *testing.T) {
	grantee := sdk.AccAddress([]byte("grantee-address-bytes"))
	// bank MsgSend is the canary because it is the message most likely to be
	// delegated in practice, and the whole point of authz on this chain.
	send := banktypes.NewMsgSend(
		sdk.AccAddress([]byte("from-address-bytes")),
		sdk.AccAddress([]byte("to-address-bytesxx")),
		sdk.NewCoins(sdk.NewInt64Coin("auptick", 1)),
	)
	exec := authz.NewMsgExec(grantee, []sdk.Msg{send})

	dec := cosmosante.NewAuthzLimiterDecorator(DisabledAuthzMsgTypeURLs()...)
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }

	_, err := dec.AnteHandle(sdk.Context{}, &mockTxWithMsgs{msgs: []sdk.Msg{&exec}}, false, next)
	require.NoError(t, err, "ordinary Cosmos messages must stay delegatable")
}
