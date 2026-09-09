package cli

import (
	"testing"
	"time"

	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

func TestResolvePacketTimeouts(t *testing.T) {
	t.Parallel()

	now := uint64(time.Now().UnixNano())

	t.Run("both zero falls back to 10 minute timestamp", func(t *testing.T) {
		t.Parallel()
		h, ts := resolvePacketTimeouts(clienttypes.Height{}, 0)
		// sample the upper bound after the call: the helper's internal
		// time.Now() always sits between `now` and this snapshot
		deadline := uint64(time.Now().Add(defaultPacketTimeout).UnixNano())
		require.True(t, h.IsZero())
		require.Greater(t, ts, now)
		require.LessOrEqual(t, ts, deadline)
	})

	t.Run("explicit height preserved, timestamp untouched", func(t *testing.T) {
		t.Parallel()
		h := clienttypes.Height{RevisionNumber: 1, RevisionHeight: 100}
		gotH, ts := resolvePacketTimeouts(h, 0)
		require.Equal(t, h, gotH)
		require.Zero(t, ts)
	})

	t.Run("explicit timestamp preserved", func(t *testing.T) {
		t.Parallel()
		gotH, ts := resolvePacketTimeouts(clienttypes.Height{}, 12345)
		require.True(t, gotH.IsZero())
		require.Equal(t, uint64(12345), ts)
	})
}

// TestTransferERC721Cmd_DefaultTimeouts is the regression guard for the G-15
// fix: with no timeout flags the CLI must fill in a fallback timestamp so the
// default invocation passes MsgTransferERC721.ValidateBasic instead of failing
// with "timeout height and timeout timestamp cannot both be zero".
func TestTransferERC721Cmd_DefaultTimeouts(t *testing.T) {
	clientCtx, addr := newCLIClient(t)
	out, err := clitestutil.ExecTestCLICmd(clientCtx, NewTransferERC721Cmd(), generateOnlyArgs("alice",
		"0x1111111111111111111111111111111111111111", "1",
		"nonfungibletokentransfer", "channel-0",
		addr.String(), "class-1", "nft-1",
	))
	require.NoError(t, err)

	msg := firstMsg[*types.MsgTransferERC721](t, clientCtx, out.Bytes())
	require.True(t, msg.TimeoutHeight.IsZero())
	require.NotZero(t, msg.TimeoutTimestamp)
	require.Greater(t, msg.TimeoutTimestamp, uint64(time.Now().UnixNano()))
	require.NoError(t, msg.ValidateBasic())
}
