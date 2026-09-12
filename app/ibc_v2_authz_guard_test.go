package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"

	"github.com/UptickNetwork/uptick/app/ante"
)

// TestAuthzClientMessagesRestOnAPermissiveClientV2Default is the tripwire behind
// MsgUpdateClient's removal from DisabledAuthzMsgTypeURLs.
//
// MsgUpdateClient's relayer allow-list always runs -- ibc-go's core keeper
// constructs ClientV2Keeper itself (modules/core/keeper/keeper.go:53), so the
// `if k.ClientV2Keeper != nil` guard at msg_server.go:87-93 is always taken -- but
// it is inert: GetConfig returns an empty Config for any client that never called
// MsgUpdateClientConfig, and an empty AllowedRelayers admits everyone
// ("DefaultConfig is empty and therefore permissionless",
// modules/core/02-client/v2/types/config.go:19).
//
// MsgUpgradeClient's removal does not rest on this default (its handler never reads
// a signer at all) and is pinned by ibc_authz_inventory_test.go instead.
//
// A future ibc-go release that flipped IsAllowedRelayer's empty-list answer to false
// would make every client allow-list-only. No chain has an allow-list for a client
// it did not create, so relaying would stop and MsgUpdateClient would become
// genuinely identity-gated; that must fail loudly here rather than silently invert
// the authz classification.
//
// The mechanism is asserted in both directions so the permissive case cannot pass
// vacuously -- an empty list admitting a stranger would also be "true" from an
// implementation that ignored the list entirely. The client id need not exist:
// GetConfig reads a per-client prefix and returns the default config when that
// prefix is empty (02-client/v2/keeper/keeper.go:47-52).
func TestAuthzClientMessagesRestOnAPermissiveClientV2Default(t *testing.T) {
	app, ctx := sharedTestApp(t)

	stranger := sdk.AccAddress([]byte("some-relayer-address"))
	onTheList := sdk.AccAddress([]byte("allowed-relayer-addr")).String()

	require.True(t, clientv2types.NewConfig().IsAllowedRelayer(stranger),
		"an empty AllowedRelayers must admit everyone — ibc-go: \"DefaultConfig is empty and therefore permissionless\" (02-client/v2/types/config.go:19)")
	require.False(t, clientv2types.NewConfig(onTheList).IsAllowedRelayer(stranger),
		"a populated AllowedRelayers must exclude addresses that are not on it, otherwise the empty-list case proves nothing")

	// The chain's clients sit on the empty branch until their own creator opts in.
	config := app.GetIBCKeeper().ClientV2Keeper.GetConfig(ctx, "07-tendermint-0")
	require.True(t, config.IsAllowedRelayer(stranger),
		`an unconfigured client no longer admits every relayer.

Round 25 removed MsgUpdateClient from DisabledAuthzMsgTypeURLs, and read the same
permissive default on the ibc.core.channel.v2 receipt and ack paths, because
IsAllowedRelayer returns true for an empty AllowedRelayers. This run says it no
longer does. Before shipping the ibc-go bump that caused that:

  - MsgUpdateClient becomes identity-gated chain-wide: only addresses listed in a
    client's AllowedRelayers could update it, and no client has one configured, so
    relaying would stop. The entry must go back into DisabledAuthzMsgTypeURLs and
    out of ibcAuthzMustStayGrantable, and the rationale in the ibcAuthzGrantable
    comment has to be rewritten around the new default.
  - ibc.core.channel.v2 MsgRecvPacket and MsgAcknowledgement are gated the same
    way (modules/core/04-channel/v2/keeper/msg_server.go:56-60 and :164-168), so
    their classification needs the same re-review.
  - Separately, decide whether the allow-list now has to be populated for existing
    clients before the upgrade height. That is an operational prerequisite, not an
    authz question, and this test cannot check it.`)

	require.NotContains(t, ante.DisabledAuthzMsgTypeURLs(),
		sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
		"MsgUpdateClient is permissionless while the client v2 default admits everyone, so it must stay delegatable")
}
