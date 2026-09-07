package keeper

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/evm/server/config"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	storetypes "cosmossdk.io/store/types"
	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// CallEVMWithData must never hand the EVM interpreter a
// budget larger than the SDK transaction's remaining gas. Before the fix the
// internal GasLimit was pinned to config.DefaultGasCap (25M) regardless of the
// tx gas, so a caller-supplied external contract (e.g. name() during
// RegisterERC721) could burn the full 25M inside the EVM before the post-hoc
// ConsumeGas ever ran — and consumed CPU cannot be rolled back with the state.
//
// The mock EVM keeper captures the core.Message actually delivered to
// ApplyMessage; the fix is correct only if the captured GasLimit equals
// min(ctx.GasMeter().GasRemaining(), config.DefaultGasCap) in every scenario.
type gasBudgetEvmKeeper struct {
	types.EVMKeeper // embedded (nil) — unused methods are never reached

	capturedMsg *core.Message
}

func (k *gasBudgetEvmKeeper) ApplyMessage(
	_ sdk.Context,
	_ *statedb.StateDB,
	msg core.Message,
	_ *tracing.Hooks,
	_ bool,
	_ bool,
	_ bool,
) (*evmtypes.MsgEthereumTxResponse, error) {
	m := msg
	k.capturedMsg = &m
	// The real EVM interpreter guarantees GasUsed <= GasLimit (an out-of-gas
	// call reports the limit as consumed, 0 for a 0 budget). Model that
	// invariant so the post-hoc ConsumeGas in CallEVMWithData behaves as it
	// does in production.
	gasUsed := uint64(100)
	if msg.GasLimit < gasUsed {
		gasUsed = msg.GasLimit
	}
	return &evmtypes.MsgEthereumTxResponse{GasUsed: gasUsed}, nil
}

// statedb.Keeper stubs — never invoked because ApplyMessage is stubbed, but
// required by the erc721StateDBKeeper wrapper built inside CallEVMWithData.
func (k *gasBudgetEvmKeeper) GetAccount(sdk.Context, common.Address) *statedb.Account {
	return nil
}
func (k *gasBudgetEvmKeeper) GetState(sdk.Context, common.Address, common.Hash) common.Hash {
	return common.Hash{}
}
func (k *gasBudgetEvmKeeper) GetCode(sdk.Context, common.Hash) []byte { return nil }
func (k *gasBudgetEvmKeeper) GetCodeHash(sdk.Context, common.Address) common.Hash {
	return common.Hash{}
}
func (k *gasBudgetEvmKeeper) ForEachStorage(sdk.Context, common.Address, func(key, value common.Hash) bool) {
}
func (k *gasBudgetEvmKeeper) SetAccount(sdk.Context, common.Address, statedb.Account) error {
	return nil
}
func (k *gasBudgetEvmKeeper) DeleteState(sdk.Context, common.Address, common.Hash) {}
func (k *gasBudgetEvmKeeper) SetState(sdk.Context, common.Address, common.Hash, []byte) {
}
func (k *gasBudgetEvmKeeper) DeleteCode(sdk.Context, []byte)      {}
func (k *gasBudgetEvmKeeper) SetCode(sdk.Context, []byte, []byte) {}
func (k *gasBudgetEvmKeeper) DeleteAccount(sdk.Context, common.Address) error {
	return nil
}
func (k *gasBudgetEvmKeeper) KVStoreKeys() map[string]*storetypes.KVStoreKey { return nil }

type gasBudgetAccountKeeper struct {
	types.AccountKeeper
}

func (k gasBudgetAccountKeeper) GetSequence(context.Context, sdk.AccAddress) (uint64, error) {
	return 1, nil
}

func gasBudgetTestKeeper(t *testing.T) (Keeper, *gasBudgetEvmKeeper) {
	t.Helper()
	evmMock := &gasBudgetEvmKeeper{}
	k := NewKeeper(nil, nil, gasBudgetAccountKeeper{}, nftkeeper.Keeper{}, evmMock, ibcnfttransferkeeper.Keeper{})
	return k, evmMock
}

func gasBudgetContext(gas storetypes.Gas) sdk.Context {
	key := storetypes.NewKVStoreKey("erc721-gas-budget")
	tkey := storetypes.NewTransientStoreKey("erc721-gas-budget-t")
	ctx := testutil.DefaultContext(key, tkey)
	return ctx.WithGasMeter(storetypes.NewGasMeter(gas))
}

// TestCallEVMWithData_BudgetCappedAtRemainingGas: a low-gas tx must deliver
// exactly its remaining gas to the EVM, not the 25M default cap.
func TestCallEVMWithData_BudgetCappedAtRemainingGas(t *testing.T) {
	k, evmMock := gasBudgetTestKeeper(t)
	ctx := gasBudgetContext(100_000)

	_, err := k.CallEVMWithData(ctx, types.ModuleAddress, nil, []byte{0x01}, true)
	require.NoError(t, err)
	require.NotNil(t, evmMock.capturedMsg)
	require.Equal(t, uint64(100_000), evmMock.capturedMsg.GasLimit,
		"EVM budget must equal the tx's remaining SDK gas")
}

// TestCallEVMWithData_BudgetKeepsDefaultCapWhenRemainingIsLarger: healthy
// transactions keep the full default cap.
func TestCallEVMWithData_BudgetKeepsDefaultCapWhenRemainingIsLarger(t *testing.T) {
	k, evmMock := gasBudgetTestKeeper(t)
	ctx := gasBudgetContext(config.DefaultGasCap + 5_000_000)

	_, err := k.CallEVMWithData(ctx, types.ModuleAddress, nil, []byte{0x01}, true)
	require.NoError(t, err)
	require.Equal(t, config.DefaultGasCap, evmMock.capturedMsg.GasLimit)
}

// TestCallEVMWithData_InfiniteMeterKeepsDefaultCap: query / genesis / simulate
// contexts run on an infinite meter and must keep the full DefaultGasCap
// (GasRemaining reports math.MaxUint64, so min() picks the cap without any
// unsigned underflow).
func TestCallEVMWithData_InfiniteMeterKeepsDefaultCap(t *testing.T) {
	k, evmMock := gasBudgetTestKeeper(t)
	key := storetypes.NewKVStoreKey("erc721-gas-budget-inf")
	tkey := storetypes.NewTransientStoreKey("erc721-gas-budget-inf-t")
	ctx := testutil.DefaultContext(key, tkey).WithGasMeter(storetypes.NewInfiniteGasMeter())

	_, err := k.CallEVMWithData(ctx, types.ModuleAddress, nil, []byte{0x01}, true)
	require.NoError(t, err)
	require.Equal(t, config.DefaultGasCap, evmMock.capturedMsg.GasLimit)
}

// TestCallEVMWithData_ZeroRemainingDeliversZeroBudget: an exhausted meter must
// not be able to trigger any EVM execution at all.
func TestCallEVMWithData_ZeroRemainingDeliversZeroBudget(t *testing.T) {
	k, evmMock := gasBudgetTestKeeper(t)
	ctx := gasBudgetContext(0)

	_, err := k.CallEVMWithData(ctx, types.ModuleAddress, nil, []byte{0x01}, true)
	require.NoError(t, err)
	require.Equal(t, uint64(0), evmMock.capturedMsg.GasLimit)
}
