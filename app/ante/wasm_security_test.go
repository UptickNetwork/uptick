package ante

import (
	"encoding/json"
	"testing"

	txsigning "cosmossdk.io/x/tx/signing"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestExtractMessagesFromTxEmpty(t *testing.T) {
	mockTx := &mockTxWithMsgs{msgs: []sdk.Msg{}}
	msgs, err := extractMessagesFromTx(mockTx)
	require.NoError(t, err)
	require.Empty(t, msgs)
}

func TestExtractMessagesFromTxSimple(t *testing.T) {
	bankMsg := banktypes.NewMsgSend(
		sdk.AccAddress([]byte("from")),
		sdk.AccAddress([]byte("to")),
		sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
	)

	mockTx := &mockTxWithMsgs{msgs: []sdk.Msg{bankMsg}}
	msgs, err := extractMessagesFromTx(mockTx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.IsType(t, &banktypes.MsgSend{}, msgs[0])
}

func TestExtractMessagesFromTxMultiple(t *testing.T) {
	msg1 := banktypes.NewMsgSend(sdk.AccAddress([]byte("a")), sdk.AccAddress([]byte("b")), sdk.NewCoins(sdk.NewInt64Coin("uatom", 100)))
	msg2 := banktypes.NewMsgSend(sdk.AccAddress([]byte("c")), sdk.AccAddress([]byte("d")), sdk.NewCoins(sdk.NewInt64Coin("uosmo", 200)))

	mockTx := &mockTxWithMsgs{msgs: []sdk.Msg{msg1, msg2}}
	msgs, err := extractMessagesFromTx(mockTx)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
}

func TestExtractMessagesFromTxAuthzNested(t *testing.T) {
	grantee := sdk.AccAddress([]byte("grantee"))
	inner := banktypes.NewMsgSend(
		sdk.AccAddress([]byte("from")),
		sdk.AccAddress([]byte("to")),
		sdk.NewCoins(sdk.NewInt64Coin("stake", 1)),
	)
	exec := authz.NewMsgExec(grantee, []sdk.Msg{inner})

	msgs, err := extractMessagesFromTx(&mockTxWithMsgs{msgs: []sdk.Msg{&exec}})
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.IsType(t, &authz.MsgExec{}, msgs[0])
	require.IsType(t, &banktypes.MsgSend{}, msgs[1])
}

func TestExtractMessagesFromTxAuthzDepthLimit(t *testing.T) {
	grantee := sdk.AccAddress([]byte("grantee"))
	var current sdk.Msg = banktypes.NewMsgSend(
		sdk.AccAddress([]byte("from")),
		sdk.AccAddress([]byte("to")),
		sdk.NewCoins(sdk.NewInt64Coin("stake", 1)),
	)
	for i := 0; i < MaxAuthzNestingDepth+1; i++ {
		exec := authz.NewMsgExec(grantee, []sdk.Msg{current})
		current = &exec
	}

	_, err := extractMessagesFromTx(&mockTxWithMsgs{msgs: []sdk.Msg{current}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "authz nesting depth exceeds maximum")
}

func TestExtractMessagesFromTxCountLimit(t *testing.T) {
	msgs := make([]sdk.Msg, MaxExtractedMessages+1)
	for i := range msgs {
		msgs[i] = banktypes.NewMsgSend(
			sdk.AccAddress([]byte("from")),
			sdk.AccAddress([]byte("to")),
			sdk.NewCoins(sdk.NewInt64Coin("stake", 1)),
		)
	}

	_, err := extractMessagesFromTx(&mockTxWithMsgs{msgs: msgs})
	require.Error(t, err)
	require.Contains(t, err.Error(), "message count exceeds maximum")
}

func TestCountWasmDispatchMsgsJSONDepthLimit(t *testing.T) {
	var v interface{} = map[string]interface{}{"wasm": map[string]interface{}{}}
	for i := 0; i < maxWasmJSONDepth+1; i++ {
		v = map[string]interface{}{"nested": v}
	}
	bz, err := json.Marshal(v)
	require.NoError(t, err)
	n := countWasmDispatchMsgs(bz, MaxWasmDispatchMsgCount)
	require.Greater(t, n, uint64(MaxWasmDispatchMsgCount))
}

// mockTxWithMsgs implements the sdk.Tx interface with configurable messages
type mockTxWithMsgs struct {
	msgs     []sdk.Msg
	gasLimit uint64
}

func (m *mockTxWithMsgs) GetMsgs() []sdk.Msg { return m.msgs }
func (m *mockTxWithMsgs) GetMsgsV2() ([]proto.Message, error) {
	return nil, nil
}
func (m *mockTxWithMsgs) GetSigners() []sdk.AccAddress {
	return nil
}
func (m *mockTxWithMsgs) GetPubKeys() ([]cryptotypes.PubKey, error)       { return nil, nil }
func (m *mockTxWithMsgs) GetSignaturesV2() ([]signing.SignatureV2, error) { return nil, nil }
func (m *mockTxWithMsgs) GetGasLimit() (uint64, error)                    { return m.gasLimit, nil }
func (m *mockTxWithMsgs) GetFee() (sdk.Coins, error)                      { return nil, nil }
func (m *mockTxWithMsgs) GetMemo() (string, error)                        { return "", nil }
func (m *mockTxWithMsgs) GetTimeoutHeight() (uint64, error)               { return 0, nil }
func (m *mockTxWithMsgs) GetSigningTxData() txsigning.TxData              { return txsigning.TxData{} }

// extractMessagesFromTx is a test helper that delegates to the production
// WasmSecurityDecorator.ExtractMessagesFromTx.
func extractMessagesFromTx(tx sdk.Tx) ([]sdk.Msg, error) {
	return WasmSecurityDecorator{}.ExtractMessagesFromTx(sdk.Context{}, tx)
}
