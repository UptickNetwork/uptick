package ante

import (
	"testing"

	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	ethante "github.com/evmos/ethermint/app/ante"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"
)

// stubAcctKeeper is a non-nil stub satisfying evmtypes.AccountKeeper
type stubAcctKeeper struct{ evmtypes.AccountKeeper }

// stubBankKeeper is a non-nil stub satisfying evmtypes.BankKeeper
type stubBankKeeper struct{ evmtypes.BankKeeper }

// stubEvmKeeper is a non-nil stub satisfying ethante.EVMKeeper
type stubEvmKeeper struct{ ethante.EVMKeeper }

// stubFeeMarketKeeper is a non-nil stub satisfying ethante.FeeMarketKeeper
type stubFeeMarketKeeper struct{ ethante.FeeMarketKeeper }

func TestHandlerOptionsValidate(t *testing.T) {
	validOptions := func() HandlerOptions {
		return HandlerOptions{
			AccountKeeper:   &stubAcctKeeper{},
			BankKeeper:      &stubBankKeeper{},
			SignModeHandler: &txsigning.HandlerMap{},
			FeeMarketKeeper: &stubFeeMarketKeeper{},
			EvmKeeper:       &stubEvmKeeper{},
			WasmConfig:      wasmTypes.DefaultWasmConfig(),
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

	t.Run("nil WasmConfig (zero value) passes", func(t *testing.T) {
		opts := validOptions()
		opts.WasmConfig = wasmTypes.WasmConfig{}
		require.NoError(t, opts.Validate())
	})
}

func TestNewAnteHandler(t *testing.T) {
	opts := HandlerOptions{
		AccountKeeper:   &stubAcctKeeper{},
		BankKeeper:      &stubBankKeeper{},
		SignModeHandler: &txsigning.HandlerMap{},
		FeeMarketKeeper: &stubFeeMarketKeeper{},
		EvmKeeper:       &stubEvmKeeper{},
		WasmConfig:      wasmTypes.DefaultWasmConfig(),
	}

	t.Run("returns non-nil handler", func(t *testing.T) {
		handler := NewAnteHandler(opts)
		require.NotNil(t, handler)
	})
}
