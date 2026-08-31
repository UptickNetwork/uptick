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
	eip712AnteHandler := newCosmosAnteHandlerEip712(options)

	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		// Route transactions based on their extension option so that EVM and
		// EIP-712 transactions receive the appropriate ante handling instead of
		// being rejected by the cosmos handler.
		switch extensionOptionTypeURL(tx) {
		case "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx":
			return ethAnteHandler(ctx, tx, simulate)
		case "/ethermint.types.v1.ExtensionOptionsWeb3Tx",
			"/cosmos.evm.eip712.v1.ExtensionOptionsWeb3Tx":
			return eip712AnteHandler(ctx, tx, simulate)
		}
		return cosmosAnteHandler(ctx, tx, simulate)
	}
}

// extensionOptionTypeURL returns the type URL of the tx's first extension
// option, or an empty string if it has none.
func extensionOptionTypeURL(tx sdk.Tx) string {
	txWithExtensions, ok := tx.(ante.HasExtensionOptionsTx)
	if !ok {
		return ""
	}
	opts := txWithExtensions.GetExtensionOptions()
	if len(opts) == 0 {
		return ""
	}
	return opts[0].GetTypeUrl()
}
