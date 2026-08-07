// Package types provides compatibility shims for the ethermint → cosmos/evm migration.
// This file will be progressively removed as each usage is migrated to the correct cosmos/evm package.
package types

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
)

// AttoPhoton is the base denomination for Uptick (replaces ethermint.AttoPhoton).
const AttoPhoton = "auptick"

// BaseDenomUnit is the number of decimal places in the base denom (18 for EVM chains).
const BaseDenomUnit = 18

// Bip44CoinType is the BIP-44 coin type for EVM chains.
const Bip44CoinType = 60

// BIP44HDPath is the default HD path for EVM keys.
const BIP44HDPath = "m/44'/60'/0'/0/0"

// PowerReduction is the power reduction for EVM chains (10^18).
var PowerReduction = math.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

// ProtoAccount is the default proto account type (replaces ethermint.ProtoAccount).
// In cosmos/evm, standard BaseAccount is used instead of EthAccount.
var ProtoAccount = authtypes.ProtoBaseAccount

// ValidateAddress validates that the given string is a valid hex (Ethereum) address.
// Replaces ethermint.ValidateAddress.
func ValidateAddress(addr string) error {
	if !common.IsHexAddress(addr) {
		return fmt.Errorf("invalid hex address: %s", addr)
	}
	return nil
}

// IsValidChainID checks if the chain ID follows the EIP-155 format: <chain>-<id>-<seq>.
// Replaces ethermint.IsValidChainID.
func IsValidChainID(chainID string) bool {
	if len(chainID) == 0 {
		return false
	}
	dashCount := 0
	for _, c := range chainID {
		if c == '-' {
			dashCount++
		}
	}
	return dashCount >= 2
}

// EthAccount wraps BaseAccount with a CodeHash for EVM compatibility.
// In cosmos/evm, EthAccount is no longer used — standard BaseAccount is used instead.
// This type is provided as a temporary shim during migration.
type EthAccount struct {
	*authtypes.BaseAccount
	CodeHash string
}
