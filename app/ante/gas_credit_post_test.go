package ante_test

import (
	"math/big"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/UptickNetwork/uptick/app/ante"
)

type mockGasCreditConsumer struct {
	calls     int
	lastPayer sdk.AccAddress
	lastGas   uint64
	lastHash  []byte
}

func (m *mockGasCreditConsumer) AutoConsumeForTx(_ sdk.Context, feePayer sdk.AccAddress, gasUsed uint64, sourceTxHash []byte) error {
	m.calls++
	m.lastPayer = feePayer
	m.lastGas = gasUsed
	m.lastHash = append([]byte{}, sourceTxHash...)
	return nil
}

type stubTx struct{}

func (stubTx) GetMsgs() []sdk.Msg                    { return nil }
func (stubTx) ValidateBasic() error                  { return nil }
func (stubTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

type feeTxAdapter struct {
	stubTx
	payer sdk.AccAddress
	gas   uint64
	msgs  []sdk.Msg
}

func (t feeTxAdapter) GetMsgs() []sdk.Msg {
	if t.msgs != nil {
		return t.msgs
	}
	return nil
}
func (t feeTxAdapter) GetGas() uint64     { return t.gas }
func (t feeTxAdapter) GetFee() sdk.Coins  { return nil }
func (t feeTxAdapter) FeePayer() []byte   { return t.payer }
func (t feeTxAdapter) FeeGranter() []byte { return nil }

func TestGasCreditPostDecoratorCosmosPath(t *testing.T) {
	consumer := &mockGasCreditConsumer{}
	dec := ante.NewGasCreditPostDecorator(consumer)

	payer := sdk.AccAddress(make([]byte, 20))
	for i := range payer {
		payer[i] = 0xAB
	}
	txBytes := []byte("test-tx-bytes-for-hash")
	ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewGasMeter(1_000_000)).
		WithTxBytes(txBytes)
	ctx.GasMeter().ConsumeGas(123, "test")

	ftx := feeTxAdapter{payer: payer, gas: 200_000}
	next := func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	}

	_, err := dec.PostHandle(ctx, ftx, false, true, next)
	require.NoError(t, err)
	require.Equal(t, 1, consumer.calls)
	require.Equal(t, payer.String(), consumer.lastPayer.String())
	require.Equal(t, uint64(123), consumer.lastGas)
	require.Len(t, consumer.lastHash, 32)

	consumer.calls = 0
	_, err = dec.PostHandle(ctx, ftx, true, true, next)
	require.NoError(t, err)
	require.Equal(t, 0, consumer.calls)
}

func TestGasCreditPostDecoratorNilConsumer(t *testing.T) {
	dec := ante.NewGasCreditPostDecorator(nil)
	ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())
	next := func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	}
	_, err := dec.PostHandle(ctx, feeTxAdapter{payer: sdk.AccAddress(make([]byte, 20))}, false, true, next)
	require.NoError(t, err)
}

func TestGasCreditPostDecoratorEVMPath(t *testing.T) {
	consumer := &mockGasCreditConsumer{}
	dec := ante.NewGasCreditPostDecorator(consumer)

	from := make([]byte, 20)
	for i := range from {
		from[i] = 0xCD
	}
	to := common.Address{}
	ethTx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    1,
		GasPrice: big.NewInt(1),
		Gas:      21000,
		To:       &to,
		Value:    big.NewInt(0),
	})
	ethMsg := &evmtypes.MsgEthereumTx{}
	ethMsg.FromEthereumTx(ethTx)
	ethMsg.From = from

	ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewGasMeter(1_000_000)).
		WithEventManager(sdk.NewEventManager())
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		evmtypes.EventTypeEthereumTx,
		sdk.NewAttribute(evmtypes.AttributeKeyEthereumTxHash, ethMsg.Hash().Hex()),
		sdk.NewAttribute(evmtypes.AttributeKeyTxGasUsed, "777"),
	))

	ftx := feeTxAdapter{
		payer: sdk.AccAddress(from),
		gas:   21000,
		msgs:  []sdk.Msg{ethMsg},
	}
	next := func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) { return ctx, nil }

	_, err := dec.PostHandle(ctx, ftx, false, true, next)
	require.NoError(t, err)
	require.Equal(t, 1, consumer.calls)
	require.Equal(t, sdk.AccAddress(from).String(), consumer.lastPayer.String())
	require.Equal(t, uint64(777), consumer.lastGas)
	require.Equal(t, ethMsg.Hash().Bytes(), consumer.lastHash)
}
