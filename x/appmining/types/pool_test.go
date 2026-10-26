package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestEntitlementFlowEnabled_RequiresAvailablePlusReserved(t *testing.T) {
	params := types.DefaultParams()
	params.EntitlementEnableMinFunded = "1"
	pool := types.Pool{
		PoolId:    "p1",
		Denom:     "auptick",
		Funded:    "100",
		Available: "90",
		Reserved:  "10",
		Paid:      "0",
	}
	require.NoError(t, types.ValidatePoolInvariant(pool))

	// Old bug: reserved-only coverage would pass with balance==10.
	require.False(t, types.EntitlementFlowEnabled(pool, params, "10"))
	require.False(t, types.EntitlementFlowEnabled(pool, params, "99"))
	require.True(t, types.EntitlementFlowEnabled(pool, params, "100"))
}

func TestUnpaidLiability(t *testing.T) {
	sum, err := types.UnpaidLiability(types.Pool{Available: "40", Reserved: "15"})
	require.NoError(t, err)
	require.Equal(t, "55", sum)
}
