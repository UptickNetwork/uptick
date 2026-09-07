package ante

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/evm/ethereum/eip712"
)

// HasWeb3ExtensionOption is the extension-option checker for the EIP-712
// (Keplr) ante handler. It accepts exactly the extension that
// verifyEip712Signature requires (see eip712.go: the tx must carry exactly one
// *eip712.ExtensionOptionsWeb3Tx), so the pre-check here and the signature
// verifier's own "exactly one Web3 option" rule can never disagree.
//
// Both the legacy "/ethermint.types.v1.ExtensionOptionsWeb3Tx" and the current
// "/cosmos.evm.eip712.v1.ExtensionOptionsWeb3Tx" type URLs unpack to this type:
// the legacy URL is registered onto *eip712.ExtensionOptionsWeb3Tx by
// app/upgrades/v041.RegisterCompatInterfaces, so a single type assertion covers
// both wallet generations.
//
// M-01: the EIP-712 handler previously installed
// antetypes.HasDynamicFeeExtensionOption, which only accepts
// *ExtensionOptionDynamicFeeTx. A perfectly valid Keplr EIP-712 tx therefore
// failed with "unknown extension options" before ever reaching
// Eip712SigVerificationDecorator. DynamicFee options belong to the Ethereum tx
// path and remain rejected here; unknown extensions are rejected by the
// sdk ante.ExtensionOptionsDecorator wrapper around this checker.
func HasWeb3ExtensionOption(anyType *codectypes.Any) bool {
	_, ok := anyType.GetCachedValue().(*eip712.ExtensionOptionsWeb3Tx)
	return ok
}
