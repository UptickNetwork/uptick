package v032

import (
	"testing"

	"cosmossdk.io/math"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	"github.com/stretchr/testify/require"

	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

func TestApplyEVMForkParams(t *testing.T) {
	params := evmtypes.DefaultParams()
	negOne := math.NewInt(-1)
	params.ChainConfig.ShanghaiBlock = &negOne
	params.ChainConfig.CancunBlock = &negOne
	params.ChainConfig.PragueBlock = &negOne
	params.ExtraEIPs = []int64{}

	got := applyEVMForkParams(params)

	require.NotNil(t, got.ChainConfig.ShanghaiBlock)
	require.NotNil(t, got.ChainConfig.CancunBlock)
	require.NotNil(t, got.ChainConfig.PragueBlock)
	require.Equal(t, int64(0), got.ChainConfig.ShanghaiBlock.Int64())
	require.Equal(t, int64(0), got.ChainConfig.CancunBlock.Int64())
	require.Equal(t, int64(0), got.ChainConfig.PragueBlock.Int64())
	require.Contains(t, got.ExtraEIPs, eip3855Extra)
}

func TestApplyEVMForkParamsKeepsExistingEIP3855Unique(t *testing.T) {
	params := evmtypes.DefaultParams()
	params.ExtraEIPs = []int64{eip3855Extra}

	got := applyEVMForkParams(params)

	count := 0
	for _, eip := range got.ExtraEIPs {
		if eip == eip3855Extra {
			count++
		}
	}
	require.Equal(t, 1, count)
}

func TestNeedsV031Bootstrap(t *testing.T) {
	require.True(t, needsV031Bootstrap(module.VersionMap{}))

	require.True(t, needsV031Bootstrap(module.VersionMap{
		icatypes.ModuleName: 0,
	}))

	require.False(t, needsV031Bootstrap(module.VersionMap{
		icatypes.ModuleName:             1,
		ibcnfttransfertypes.ModuleName: 1,
	}))
}
