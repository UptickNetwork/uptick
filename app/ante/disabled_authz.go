package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"
)

// DisabledAuthzMsgTypeURLs is the set of message type URLs that cannot be
// granted or executed through authz.MsgExec / MsgGrant. Both gov v1 and
// v1beta1 are listed: SDK 0.53 still registers the legacy vote/submit/deposit
// messages, and the limiter matches type URLs exactly.
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
		sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
		sdk.MsgTypeURL(&ibcclienttypes.MsgUpgradeClient{}),
	}
}
