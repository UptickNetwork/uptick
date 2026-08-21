package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// NewAnteHandler returns the AnteHandler for the EVM
func NewAnteHandler(options HandlerOptions) sdk.AnteHandler {
	if err := options.Validate(); err != nil {
		panic(err)
	}

	ethAnteHandler := newEthAnteHandler(options)
	cosmosAnteHandler := newCosmosAnteHandler(options)

	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		// Route Ethereum (ExtensionOptionsEthereumTx) transactions to the EVM
		// ante handler so that EVM-specific validation errors are properly
		// surfaced, instead of being silently swallowed by the cosmos handler.
		if isEthTx(tx) {
			return ethAnteHandler(ctx, tx, simulate)
		}
		return cosmosAnteHandler(ctx, tx, simulate)
	}
}

// isEthTx reports whether the tx carries the Ethereum extension option.
func isEthTx(tx sdk.Tx) bool {
	txWithExtensions, ok := tx.(ante.HasExtensionOptionsTx)
	if !ok {
		return false
	}
	opts := txWithExtensions.GetExtensionOptions()
	if len(opts) == 0 {
		return false
	}
	return opts[0].GetTypeUrl() == "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx"
}
