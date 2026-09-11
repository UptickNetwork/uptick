package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	upticktypes "github.com/UptickNetwork/uptick/types"
)

// app/app.go's init() replaces sdk.DefaultPowerReduction with Uptick's 10^18.
// The replacement is a package-level global write, so it is invisible from the
// call site and easy to lose in a refactor -- and losing it would not fail
// anything loudly: x/staking's keeper returns the global for every power
// conversion (x/staking/keeper/params.go), so staking weights, gentx
// validation and the genesis validator check would all silently start using
// the SDK's 10^6, i.e. every validator off by 10^12.
//
// This asserts the post-init state, which is the only observable that matters:
// importing this package is what installs it.
func TestDefaultPowerReductionIsUptickPowerReduction(t *testing.T) {
	require.True(t, sdk.DefaultPowerReduction.Equal(upticktypes.PowerReduction),
		"sdk.DefaultPowerReduction is %s but Uptick's power reduction is %s: app/app.go's init override is gone",
		sdk.DefaultPowerReduction, upticktypes.PowerReduction)

	// The override only applies to code that imports this package. Anything
	// that computes a package-level default from the SDK value before this
	// package is initialised keeps the old one; see
	// cmd/uptickd/power_reduction_order_test.go for the one place that happens
	// and why it is currently unreachable.
	require.Equal(t, int64(1000000000000000000), upticktypes.PowerReduction.Int64())
}
