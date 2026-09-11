package ante

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"
)

// DisabledAuthzMsgTypeURLs is the set of message type URLs that cannot be
// granted or executed through authz.MsgGrant / MsgExec.
//
// # The rule
//
// A message belongs here iff executing it requires the signer to *be* something
// the grantee is not: the owner of the assets being moved, the module
// authority, the client creator, or an allow-listed relayer. Delegating such a
// message hands over that identity.
//
// Permissionless messages do NOT belong here. authz never rewrites the inner
// message's signer — x/authz/keeper/keeper.go:110-130 reads the signer off the
// message itself, treats it as the granter, and requires a grant from it — so a
// message with no signer check achieves exactly the same thing whoever sends
// it, and a grantee can obtain that outcome from its own account. Listing it
// removes no capability, it only makes the list longer.
//
// Concretely, IBC's whole relayer surface stays grantable: MsgRecvPacket,
// MsgAcknowledgement, MsgTimeout, MsgTimeoutOnClose, MsgSendPacket and every
// channel/connection handshake message carry no signer check (MsgChannelCloseInit
// does not even look at the signer — ibc-go modules/core/keeper/msg_server.go:374),
// so delegating relay work still works after this list is applied. The full
// classification, including the permissionless half, is asserted against the
// protobuf descriptors by ibc_authz_inventory_test.go; extend the list there and
// the test tells you if you missed a family.
//
// # Every entry in this list is now derivable from the rule
//
// Both gov v1 and v1beta1 are listed: SDK 0.53 still registers the legacy
// vote/submit/deposit messages, and the limiter matches type URLs exactly.
//
// Two IBC client messages that used to be here were removed in round 25:
// MsgUpdateClient and MsgUpgradeClient. Neither is derivable, and neither is a
// privilege escalation:
//
//   - MsgUpdateClient's relayer allow-list is real and always runs. ibc-go's core
//     keeper constructs ClientV2Keeper itself
//     (modules/core/keeper/keeper.go:53), so the `if k.ClientV2Keeper != nil`
//     guard at msg_server.go:87-93 is always taken. What makes the check inert is
//     the *configuration*, not the wiring: GetConfig returns an empty Config for
//     any client that never called MsgUpdateClientConfig, and an empty list
//     admits everyone — ibc-go states this outright
//     ("DefaultConfig is empty and therefore permissionless",
//     modules/core/02-client/v2/types/config.go:19). So every client is
//     permissionless to update until its own creator opts into an allow-list, and
//     when one does opt in, delegating the update is the allow-list owner
//     consenting to be relayed for — the same consent that adding the grantee to
//     the list would express.
//   - MsgUpgradeClient has no signer check to find: its handler
//     (msg_server.go:108-122) never reads msg.Signer, and the keeper it calls —
//     modules/core/02-client/keeper/client.go:88 — takes no signer argument at
//     all. A grant for it was a no-op that nonetheless appeared to restrict
//     client upgrades, which is worse than not restricting them.
//
// app/ibc_v2_authz_guard_test.go pins the permissive default that both removals
// depend on, so an ibc-go bump that flips it fails a test instead of quietly
// making these entries necessary again.
//
// Removing them is still a state change in the same sense as adding entries (a
// previously rejected MsgExec now succeeds), which is why it is recorded in the
// CHANGELOG rather than treated as a refactor.
func DisabledAuthzMsgTypeURLs() []string {
	return []string{
		sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
		sdk.MsgTypeURL(&vestingtypes.MsgCreateVestingAccount{}),
		sdk.MsgTypeURL(&govv1.MsgSubmitProposal{}),
		sdk.MsgTypeURL(&govv1.MsgVote{}),
		sdk.MsgTypeURL(&govv1.MsgVoteWeighted{}),
		sdk.MsgTypeURL(&govv1.MsgDeposit{}),
		sdk.MsgTypeURL(&govv1.MsgExecLegacyContent{}),
		sdk.MsgTypeURL(&govv1beta1.MsgSubmitProposal{}),
		sdk.MsgTypeURL(&govv1beta1.MsgVote{}),
		sdk.MsgTypeURL(&govv1beta1.MsgVoteWeighted{}),
		sdk.MsgTypeURL(&govv1beta1.MsgDeposit{}),
		sdk.MsgTypeURL(&upgradetypes.MsgSoftwareUpgrade{}),
		sdk.MsgTypeURL(&upgradetypes.MsgCancelUpgrade{}),
		// wasm params are authority-gated (gov module account only); they must
		// not be delegatable via authz. Note: wasm MsgStoreCode / MsgMigrateContract
		// / MsgUpdateAdmin / MsgClearAdmin are admin/permission-gated rather than
		// purely authority-gated, so they are intentionally left grantable.
		sdk.MsgTypeURL(&wasmtypes.MsgUpdateParams{}),

		// --- IBC: authority-gated -------------------------------------------
		//
		// Blocking these is not a live vulnerability fix: a module account can
		// never be a granter, so none of them is reachable through MsgExec
		// today. They are listed for uniformity — the same reason
		// upgradetypes.MsgSoftwareUpgrade and wasmtypes.MsgUpdateParams are —
		// and so that the guard can assert an exact partition of the IBC
		// surface instead of a hand-picked subset. The entries stay load-bearing
		// if the authority is ever moved to a normal account or multisig.
		sdk.MsgTypeURL(&ibcclienttypes.MsgRecoverClient{}),      // msg_server.go:143
		sdk.MsgTypeURL(&ibcclienttypes.MsgIBCSoftwareUpgrade{}), // msg_server.go:157
		sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateParams{}),       // msg_server.go:645 (UpdateClientParams)
		sdk.MsgTypeURL(&connectiontypes.MsgUpdateParams{}),      // msg_server.go:657 (UpdateConnectionParams)
		sdk.MsgTypeURL(&transfertypes.MsgUpdateParams{}),        // apps/transfer/keeper/msg_server.go:142
		sdk.MsgTypeURL(&nfttransfertypes.MsgUpdateParams{}),
		sdk.MsgTypeURL(&icacontrollertypes.MsgUpdateParams{}), // controller/keeper/msg_server.go:84
		sdk.MsgTypeURL(&icahosttypes.MsgUpdateParams{}),       // host/keeper/msg_server.go:65

		// --- IBC: authority OR client creator -------------------------------
		//
		// ibc-go messages/core/keeper/msg_server.go:669 and :684. The creator
		// arm makes these reachable: whoever submitted MsgCreateClient is
		// recorded as the client creator (:47) and can delegate that role.
		sdk.MsgTypeURL(&clientv2types.MsgUpdateClientConfig{}),
		sdk.MsgTypeURL(&ibcclienttypes.MsgDeleteClientCreator{}),

		// --- IBC: client creator only ---------------------------------------
		//
		// msg_server.go:57-60. Also one-shot: the handler refuses to overwrite
		// an already-registered counterparty, so a delegated call is
		// unrecoverable for the client's owner.
		sdk.MsgTypeURL(&clientv2types.MsgRegisterCounterparty{}),

		// --- IBC: owner-keyed (interchain accounts) -------------------------
		//
		// The port id is derived from msg.Owner
		// (controller/keeper/msg_server.go:32 and :67), so an interchain account
		// belongs to its signer. MsgSendTx drives that account on the host
		// chain, which makes it the strongest delegation in this block: the
		// grantee would control a whole account, not a single transfer.
		sdk.MsgTypeURL(&icacontrollertypes.MsgRegisterInterchainAccount{}),
		sdk.MsgTypeURL(&icacontrollertypes.MsgSendTx{}),

		// --- IBC: moves the signer's own assets -----------------------------
		//
		// These two are the entries that remove a live capability. The limiter
		// matches on the message type URL and TransferAuthorization.MsgTypeURL()
		// returns the same URL as MsgTransfer, so a spend-limited or
		// allow-listed grant cannot be carved out — after this, IBC transfers
		// (fungible and NFT) are not delegatable at all, and anyone who needs
		// to delegate one has to hand over the key or use a contract account.
		//
		// The NFT arm matters more than the fungible one here: Uptick is an NFT
		// chain, nft-transfer is the main asset exit, and that module ships no
		// Authorization type of its own, so GenericAuthorization (no limit, no
		// allow-list) was the only way to delegate it.
		sdk.MsgTypeURL(&transfertypes.MsgTransfer{}),
		sdk.MsgTypeURL(&nfttransfertypes.MsgTransfer{}),
	}
}
