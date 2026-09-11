// Package types provides compatibility shims for the cosmos/evm migration.
// This file will be progressively removed as each usage is migrated to the correct cosmos/evm package.
package types

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
)

// AttoPhoton is the base denomination for Uptick.
const AttoPhoton = "auptick"

// BaseDenomUnit is the number of decimal places in the base denom (18 for EVM chains).
const BaseDenomUnit = 18

// Bip44CoinType is the BIP-44 coin type for EVM chains.
const Bip44CoinType = 60

// BIP44HDPath is the default HD path for EVM keys.
const BIP44HDPath = "m/44'/60'/0'/0/0"

// PowerReduction is the power reduction for EVM chains (10^18).
var PowerReduction = math.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

// ProtoAccount is the default proto account type.
// In cosmos/evm, standard BaseAccount is used instead of EthAccount.
var ProtoAccount = authtypes.ProtoBaseAccount

// ValidateAddress validates that the given string is a valid hex (Ethereum) address.
func ValidateAddress(addr string) error {
	if !common.IsHexAddress(addr) {
		return fmt.Errorf("invalid hex address: %s", addr)
	}
	return nil
}

// IsValidChainID reports whether chainID is a Cosmos EIP-155 id of the form
// {name}_{eip155}-{revision}, e.g. uptick_117-1.
func IsValidChainID(chainID string) bool {
	_, err := ParseEIP155ChainID(chainID)
	return err == nil
}
