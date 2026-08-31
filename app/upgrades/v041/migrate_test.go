package v041

import (
	"testing"

	"github.com/stretchr/testify/require"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func TestDefaultActiveStaticPrecompiles(t *testing.T) {
	require.Len(t, defaultActiveStaticPrecompiles, 8)

	// The vesting precompile (0x803) is not registered by Uptick and must not
	// be activated, otherwise GetStaticPrecompileInstance panics with
	// "precompiled contract not stored in memory".
	require.NotContains(t, defaultActiveStaticPrecompiles, evmtypes.VestingPrecompileAddress)

	for _, addr := range []string{
		evmtypes.P256PrecompileAddress,
		evmtypes.Bech32PrecompileAddress,
		evmtypes.StakingPrecompileAddress,
		evmtypes.DistributionPrecompileAddress,
		evmtypes.ICS20PrecompileAddress,
		evmtypes.BankPrecompileAddress,
		evmtypes.GovPrecompileAddress,
		evmtypes.SlashingPrecompileAddress,
	} {
		require.Contains(t, defaultActiveStaticPrecompiles, addr)
	}
}

func TestInitPopulatesDefaultStaticPrecompiles(t *testing.T) {
	require.Equal(t, defaultActiveStaticPrecompiles, evmtypes.DefaultStaticPrecompiles)
}
