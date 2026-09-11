package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"

	upticktypes "github.com/UptickNetwork/uptick/types"
)

// MEASURED, NOT ASSUMED. app/app.go's init() rewrites sdk.DefaultPowerReduction
// to 10^18, but that happens at package-initialisation time: any package-level
// variable in the dependency graph that read the SDK default is frozen at
// whatever value it had when *its own* package was initialised. Measured in
// this binary, x/staking/client/cli got there first:
//
//	x/staking/client/cli.DefaultTokens = 100000000            (10^8)
//	Uptick's equivalent                = 100000000000000000000 (10^20)
//
// so the upstream "100 tokens" bundle is short by 10^12, and its sibling
// defaultAmount ("100000000" + sdk.DefaultBondDenom, itself still the SDK's
// "stake") is short in a way that also carries a denom this chain does not
// issue.
//
// It is NOT a live defect, and that is the whole reason this test asserts
// rather than fixes: nothing in this repository reaches either value.
// uptickd's only create-validator paths are `tx staking create-validator`,
// which leaves --amount empty (the SDK's own flag default is "" and
// newBuildCreateValidatorMsg does not substitute defaultAmount) and lets
// validation fail with a clear message, and `uptickd testnet`, which builds
// MsgCreateValidator by hand with upticktypes.PowerReduction and
// cmdcfg.BaseDenom (cmd/uptickd/testnet.go:381-390). The frozen values only
// reachable through the SDK's own CreateValidatorMsgFlagSet /
// PrepareConfigForTxCreateValidator helpers, which we never call.
//
// The pin exists so that a future command which *does* use those helpers meets
// this comment instead of a wrong gentx. If the NotEqual below ever fails, the
// initialisation order changed: delete the pin and this paragraph together.
func TestUpstreamStakingCLIDefaultsAreFrozenBeforeAppInit(t *testing.T) {
	require.True(t, sdk.DefaultPowerReduction.Equal(upticktypes.PowerReduction),
		"app's init override must have run by now: got %s", sdk.DefaultPowerReduction)

	uptickHundred := sdk.TokensFromConsensusPower(100, upticktypes.PowerReduction)

	t.Logf("frozen x/staking/client/cli.DefaultTokens=%s; Uptick's 100-token bundle=%s",
		stakingcli.DefaultTokens, uptickHundred)

	require.NotEqual(t, uptickHundred.String(), stakingcli.DefaultTokens.String(),
		"upstream's package-level default is no longer frozen at the SDK reduction; re-read the comment above")
}
