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
		case extOptionEthereumTx:
			return ethAnteHandler(ctx, tx, simulate)
		case extOptionWeb3Tx, extOptionEip712Tx:
			return eip712AnteHandler(ctx, tx, simulate)
		}
		return cosmosAnteHandler(ctx, tx, simulate)
	}
}

// Extension option type URLs recognized by the ante router.
const (
	extOptionEthereumTx = "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx"
	extOptionWeb3Tx     = "/ethermint.types.v1.ExtensionOptionsWeb3Tx"
	extOptionEip712Tx   = "/cosmos.evm.eip712.v1.ExtensionOptionsWeb3Tx"
)

// extensionOptionTypeURL scans ALL of the tx's extension options and returns
// the first recognized routing URL, or an empty string if none matches.
// Audit P3-10: the previous implementation read only opts[0], so a transaction
// carrying an EVM option at any later position was mis-routed to the plain
// cosmos ante handler and rejected. Unknown option URLs are skipped rather
// than short-circuiting the scan. Downstream handlers still enforce their own
// tx-type validation, so a spurious EVM option on a plain cosmos tx is
// rejected by the EVM handlers themselves.
func extensionOptionTypeURL(tx sdk.Tx) string {
	txWithExtensions, ok := tx.(ante.HasExtensionOptionsTx)
	if !ok {
		return ""
	}
	for _, opt := range txWithExtensions.GetExtensionOptions() {
		switch opt.GetTypeUrl() {
		case extOptionEthereumTx, extOptionWeb3Tx, extOptionEip712Tx:
			return opt.GetTypeUrl()
		}
	}
	return ""
}
