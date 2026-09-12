package types

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// This package looks like a set of shims for the cosmos/evm migration but owns
// consensus-visible constants: the gas denom, the number of decimals the EVM
// treats it as, and the power reduction every staking weight is divided by. A
// silent edit to any of them moves voting power or splits balances.
func TestShimConstantsAreConsensusVisible(t *testing.T) {
	require.Equal(t, "auptick", AttoPhoton, "the base denom is spelled here and aliased in cmd/config")

	// 18 is not a display choice: cosmos/evm rejects an EvmCoinInfo whose Denom
	// and ExtendedDenom differ when Decimals == EighteenDecimals
	// (x/vm/types/denom_config.go, setEVMCoinInfo), and app/keepers/keepers.go
	// passes BaseDenom for both. Another value here therefore requires a different
	// extended denom too, which is a chain-wide migration.
	require.Equal(t, 18, BaseDenomUnit)

	// 10^18: app/app.go's init replaces sdk.DefaultPowerReduction with this, and
	// x/staking's keeper returns the global for every power conversion. The SDK
	// default is 10^6, so a lost override would divide every staking weight by
	// 10^12.
	expected := math.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	require.True(t, PowerReduction.Equal(expected), "power reduction must be 10^18, got %s", PowerReduction)

	// Coin type 60 (Ethereum) and the matching HD path must agree, or keys from
	// `uptickd keys add` would not match the same mnemonic in an EVM wallet.
	require.Equal(t, uint32(60), uint32(Bip44CoinType))
	require.Equal(t, "m/44'/60'/0'/0/0", BIP44HDPath)

	// cosmos/evm uses plain BaseAccounts; reverting to EthAccount would change the
	// account type written into genesis. Compared by pointer because function
	// values are not comparable.
	require.Equal(t,
		reflect.ValueOf(authtypes.ProtoBaseAccount).Pointer(),
		reflect.ValueOf(ProtoAccount).Pointer(),
		"ProtoAccount must stay authtypes.ProtoBaseAccount")
}

// ValidateAddress is the only hex-address gate in the repository: x/erc721's CLI
// and TokenPair.Validate both call it. It is deliberately a shape check, not an
// EIP-55 checksum check -- addresses are stored and compared lowercased
// elsewhere (see the A-4 findings), so enforcing case here would contradict how
// they are used.
func TestValidateAddressAcceptsHexRejectsBech32(t *testing.T) {
	require.NoError(t, ValidateAddress("0x0000000000000000000000000000000000000000"))
	require.NoError(t, ValidateAddress("0xAbCdEf0000000000000000000000000000000000"), "mixed case is accepted: no checksum is verified")
	require.NoError(t, ValidateAddress("AbCdEf0000000000000000000000000000000000"), "the 0x prefix is optional for common.IsHexAddress")

	require.Error(t, ValidateAddress(""), "an empty receiver must not pass")
	require.Error(t, ValidateAddress("0x1234"), "a short hex string must not pass")
	require.Error(t, ValidateAddress("uptick1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq5nmu6h"), "a bech32 address must not pass")
}
