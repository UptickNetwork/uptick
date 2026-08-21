package ante

import (
	"testing"

	txsigning "cosmossdk.io/x/tx/signing"
	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
)

// stubAcctKeeper is a non-nil stub satisfying evmtypes.AccountKeeper
type stubAcctKeeper struct{ evmtypes.AccountKeeper }

// stubBankKeeper is a non-nil stub satisfying evmtypes.BankKeeper
type stubBankKeeper struct{ evmtypes.BankKeeper }

// stubEvmKeeper is a non-nil stub satisfying anteinterfaces.EVMKeeper
type stubEvmKeeper struct{ anteinterfaces.EVMKeeper }

// stubFeeMarketKeeper is a non-nil stub satisfying anteinterfaces.FeeMarketKeeper
type stubFeeMarketKeeper struct{ anteinterfaces.FeeMarketKeeper }

func TestHandlerOptionsValidate(t *testing.T) {
	validOptions := func() HandlerOptions {
		return HandlerOptions{
			AccountKeeper:   &stubAcctKeeper{},
			BankKeeper:      &stubBankKeeper{},
			SignModeHandler: &txsigning.HandlerMap{},
			FeeMarketKeeper: &stubFeeMarketKeeper{},
			EvmKeeper:       &stubEvmKeeper{},
		}
	}

	t.Run("valid options pass", func(t *testing.T) {
		opts := validOptions()
		require.NoError(t, opts.Validate())
	})

	t.Run("nil AccountKeeper fails", func(t *testing.T) {
		opts := validOptions()
		opts.AccountKeeper = nil
		require.ErrorContains(t, opts.Validate(), "account keeper")
	})

	t.Run("nil BankKeeper fails", func(t *testing.T) {
		opts := validOptions()
		opts.BankKeeper = nil
		require.ErrorContains(t, opts.Validate(), "bank keeper")
	})

	t.Run("nil SignModeHandler fails", func(t *testing.T) {
		opts := validOptions()
		opts.SignModeHandler = nil
		require.ErrorContains(t, opts.Validate(), "sign mode handler")
	})

	t.Run("nil FeeMarketKeeper fails", func(t *testing.T) {
		opts := validOptions()
		opts.FeeMarketKeeper = nil
		require.ErrorContains(t, opts.Validate(), "fee market keeper")
	})

	t.Run("nil EvmKeeper fails", func(t *testing.T) {
		opts := validOptions()
		opts.EvmKeeper = nil
		require.ErrorContains(t, opts.Validate(), "evm keeper")
	})
}

func TestNewAnteHandler(t *testing.T) {
	opts := HandlerOptions{
		AccountKeeper:   &stubAcctKeeper{},
		BankKeeper:      &stubBankKeeper{},
		SignModeHandler: &txsigning.HandlerMap{},
		FeeMarketKeeper: &stubFeeMarketKeeper{},
		EvmKeeper:       &stubEvmKeeper{},
	}

	t.Run("returns non-nil handler", func(t *testing.T) {
		handler := NewAnteHandler(opts)
		require.NotNil(t, handler)
	})

	t.Run("wasm fields optional at construction", func(t *testing.T) {
		opts.WasmKeeper = nil
		opts.WasmNodeConfig = nil
		opts.TXCounterStoreService = nil
		require.NotNil(t, NewAnteHandler(opts))
	})
}
