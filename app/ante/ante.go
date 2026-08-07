package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewAnteHandler returns the AnteHandler for the EVM
func NewAnteHandler(options HandlerOptions) sdk.AnteHandler {
	if err := options.Validate(); err != nil {
		panic(err)
	}

	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		// Determine which ante handler to use based on the transaction type.
		// If the transaction contains an Ethereum tx, use the Ethereum-specific ante handler.
		// Otherwise, use the Cosmos ante handler (which may include EIP-712 wrapping).
		return newCosmosAnteHandler(options)(ctx, tx, simulate)
	}
}
