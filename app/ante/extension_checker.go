package ante

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/evm/ethereum/eip712"
)

// HasWeb3ExtensionOption is the extension-option checker for the EIP-712 (Keplr)
// ante handler. It accepts exactly *eip712.ExtensionOptionsWeb3Tx, the extension
// verifyEip712Signature requires. Both the legacy ethermint and the current
// cosmos.evm type URLs unpack to this type (see app/upgrades/v041
// RegisterCompatInterfaces). DynamicFee options belong to the Ethereum tx path
// and are rejected here.
func HasWeb3ExtensionOption(anyType *codectypes.Any) bool {
	_, ok := anyType.GetCachedValue().(*eip712.ExtensionOptionsWeb3Tx)
	return ok
}
