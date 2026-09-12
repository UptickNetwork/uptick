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

// DisabledAuthzMsgTypeURLs is the set of message type URLs that cannot be granted
// or executed through authz.MsgGrant / MsgExec.
//
// A message belongs here iff executing it requires the signer to *be* something the
// grantee is not: the owner of the assets being moved, the module authority, the
// client creator, or an allow-listed relayer. Delegating it hands over that identity.
//
// Permissionless messages do NOT belong here. authz never rewrites the inner
// message's signer -- x/authz/keeper/keeper.go:110-130 reads the signer off the
// message itself, treats it as the granter, and requires a grant from it -- so a
// message with no signer check is equally executable by anyone: listing it removes
// no capability, only makes the list longer. IBC's entire relayer surface therefore
// stays grantable, MsgChannelCloseInit included (it never reads the signer,
// ibc-go modules/core/keeper/msg_server.go:374). ibc_authz_inventory_test.go asserts
// this classification, permissionless half included, against the protobuf
// descriptors -- extend the list there and the test names the family you missed.
//
// gov v1 and v1beta1 are both listed because SDK 0.53 still registers the legacy
// vote/submit/deposit messages, and the limiter matches type URLs exactly.
//
// Two entries were removed; neither is derivable from the rule and neither is an
// escalation:
//
//   - MsgUpdateClient's relayer allow-list is real and always runs -- ibc-go's core
//     keeper constructs ClientV2Keeper itself (modules/core/keeper/keeper.go:53), so
//     the nil guard at msg_server.go:87-93 is always taken. It is inert because of
//     the *configuration*: GetConfig returns an empty Config for any client that
//     never called MsgUpdateClientConfig, and an empty list admits everyone
//     (modules/core/02-client/v2/types/config.go:19). A client is permissionless to
//     update until its creator opts in, and opting in is consenting.
//   - MsgUpgradeClient has no signer check to find: its handler
//     (msg_server.go:108-122) never reads msg.Signer, and the keeper it calls
//     (modules/core/02-client/keeper/client.go:88) takes no signer argument. Its
//     grant was a no-op that only appeared to restrict client upgrades.
//
// app/ibc_v2_authz_guard_test.go pins the permissive default the MsgUpdateClient
// removal relies on, so an ibc-go bump that flips it fails a test instead of quietly
// making that entry necessary again -- MsgUpgradeClient's removal does not depend on
// that default, and ibc_authz_inventory_test.go pins both. Removing an entry is itself
// a state change (a previously rejected MsgExec now succeeds), so it is recorded in
// the CHANGELOG.
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
		// wasm params are authority-gated (gov module account only), so they must not
		// be delegatable. MsgStoreCode / MsgMigrateContract / MsgUpdateAdmin /
		// MsgClearAdmin are only admin-gated, so they stay grantable.
		sdk.MsgTypeURL(&wasmtypes.MsgUpdateParams{}),

		// --- IBC: authority-gated -------------------------------------------
		//
		// Not a live fix: a module account can never be a granter, so none of these
		// is reachable through MsgExec today. They are listed so the guard can assert
		// an exact partition of the IBC surface, and stay load-bearing if the
		// authority ever moves to a normal account or multisig.
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
		// msg_server.go:669 and :684. Reachable through the creator arm: whoever
		// submitted MsgCreateClient is recorded as the client creator (:47).
		sdk.MsgTypeURL(&clientv2types.MsgUpdateClientConfig{}),
		sdk.MsgTypeURL(&ibcclienttypes.MsgDeleteClientCreator{}),

		// --- IBC: client creator only ---------------------------------------
		//
		// msg_server.go:57-60, and one-shot: the handler refuses to overwrite an
		// already-registered counterparty, so a delegated call is unrecoverable.
		sdk.MsgTypeURL(&clientv2types.MsgRegisterCounterparty{}),

		// --- IBC: owner-keyed (interchain accounts) -------------------------
		//
		// The port id derives from msg.Owner (controller/keeper/msg_server.go:32 and
		// :67), so the account belongs to its signer. MsgSendTx drives that account on
		// the host chain, making it the strongest delegation here: the grantee would
		// control a whole account, not a single transfer.
		sdk.MsgTypeURL(&icacontrollertypes.MsgRegisterInterchainAccount{}),
		sdk.MsgTypeURL(&icacontrollertypes.MsgSendTx{}),

		// --- IBC: moves the signer's own assets -----------------------------
		//
		// These two remove a live capability: the limiter matches on the message type
		// URL, and TransferAuthorization.MsgTypeURL() returns the same URL as
		// MsgTransfer, so no spend limit or allow-list can carve out an exception.
		// After this, IBC transfers are not delegatable at all.
		//
		// The NFT arm matters more: Uptick is an NFT chain, nft-transfer is the main
		// asset exit, and that module ships no Authorization type of its own, so
		// GenericAuthorization (no limit, no allow-list) was the only way to
		// delegate it.
		sdk.MsgTypeURL(&transfertypes.MsgTransfer{}),
		sdk.MsgTypeURL(&nfttransfertypes.MsgTransfer{}),
	}
}
