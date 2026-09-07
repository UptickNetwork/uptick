package ante

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	cosmosante "github.com/cosmos/evm/ante/cosmos"
	"github.com/stretchr/testify/require"
)

func TestDisabledAuthzMsgTypeURLsIncludesGovV1Beta1(t *testing.T) {
	urls := map[string]struct{}{}
	for _, u := range DisabledAuthzMsgTypeURLs() {
		urls[u] = struct{}{}
	}
	require.Contains(t, urls, sdk.MsgTypeURL(&govv1.MsgVote{}))
	require.Contains(t, urls, sdk.MsgTypeURL(&govv1beta1.MsgVote{}))
	require.Contains(t, urls, sdk.MsgTypeURL(&govv1beta1.MsgSubmitProposal{}))
	require.Contains(t, urls, sdk.MsgTypeURL(&govv1beta1.MsgVoteWeighted{}))
	require.Contains(t, urls, sdk.MsgTypeURL(&govv1beta1.MsgDeposit{}))
	require.Contains(t, urls, sdk.MsgTypeURL(&govv1.MsgExecLegacyContent{}))
}

func TestAuthzLimiterRejectsGovV1Beta1VoteInMsgExec(t *testing.T) {
	voter := sdk.AccAddress([]byte("voter-address-bytes"))
	vote := govv1beta1.NewMsgVote(voter, 1, govv1beta1.OptionYes)
	exec := authz.NewMsgExec(voter, []sdk.Msg{vote})

	dec := cosmosante.NewAuthzLimiterDecorator(DisabledAuthzMsgTypeURLs()...)
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}

	_, err := dec.AnteHandle(sdk.Context{}, &mockTxWithMsgs{msgs: []sdk.Msg{&exec}}, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), sdk.MsgTypeURL(&govv1beta1.MsgVote{}))
}

func TestAuthzLimiterAllowsBankSendInMsgExec(t *testing.T) {
	from := sdk.AccAddress([]byte("from-address-bytes"))
	to := sdk.AccAddress([]byte("to-address-bytesxx"))
	send := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin("auptick", 1)))
	exec := authz.NewMsgExec(from, []sdk.Msg{send})

	dec := cosmosante.NewAuthzLimiterDecorator(DisabledAuthzMsgTypeURLs()...)
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}

	_, err := dec.AnteHandle(sdk.Context{}, &mockTxWithMsgs{msgs: []sdk.Msg{&exec}}, false, next)
	require.NoError(t, err)
}
