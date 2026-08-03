package ante

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestExtractMessagesFromTxEmpty(t *testing.T) {
	mockTx := &mockTxWithMsgs{msgs: []sdk.Msg{}}
	msgs, err := extractMessagesFromTx(mockTx)
	require.NoError(t, err)
	require.Empty(t, msgs)
}

func TestExtractMessagesFromTxSimple(t *testing.T) {
	bankMsg := banktypes.NewMsgSend(
		"cosmos1sender",
		"cosmos1receiver",
		sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
	)

	mockTx := &mockTxWithMsgs{msgs: []sdk.Msg{bankMsg}}
	msgs, err := extractMessagesFromTx(mockTx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.IsType(t, &banktypes.MsgSend{}, msgs[0])
}

func TestExtractMessagesFromTxMultiple(t *testing.T) {
	msg1 := banktypes.NewMsgSend("cosmos1a", "cosmos1b", sdk.NewCoins(sdk.NewInt64Coin("uatom", 100)))
	msg2 := banktypes.NewMsgSend("cosmos1c", "cosmos1d", sdk.NewCoins(sdk.NewInt64Coin("uosmo", 200)))

	mockTx := &mockTxWithMsgs{msgs: []sdk.Msg{msg1, msg2}}
	msgs, err := extractMessagesFromTx(mockTx)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
}

// mockTxWithMsgs implements the sdk.Tx interface with configurable messages
type mockTxWithMsgs struct {
	msgs     []sdk.Msg
	gasLimit uint64
}

func (m *mockTxWithMsgs) GetMsgs() []sdk.Msg { return m.msgs }
func (m *mockTxWithMsgs) GetMsgsV2() ([]*sdk.Msg, error) {
	result := make([]*sdk.Msg, len(m.msgs))
	for i := range m.msgs {
		result[i] = &m.msgs[i]
	}
	return result, nil
}
func (m *mockTxWithMsgs) GetSigners() []sdk.AccAddress {
	signers := make([]sdk.AccAddress, len(m.msgs))
	for i, msg := range m.msgs {
		signers[i] = msg.GetSigners()[0]
	}
	return signers
}
func (m *mockTxWithMsgs) GetPubKeys() ([]sdk.PubKey, error)         { return nil, nil }
func (m *mockTxWithMsgs) GetSignaturesV2() ([]sdk.Signature, error) { return nil, nil }
func (m *mockTxWithMsgs) GetGasLimit() (uint64, error)              { return m.gasLimit, nil }
func (m *mockTxWithMsgs) GetFee() (sdk.Coins, error)                { return nil, nil }
func (m *mockTxWithMsgs) GetMemo() (string, error)                  { return "", nil }
func (m *mockTxWithMsgs) GetTimeoutHeight() (uint64, error)         { return 0, nil }
func (m *mockTxWithMsgs) GetSigningTxData() sdk.SigningTxData       { return sdk.SigningTxData{} }
