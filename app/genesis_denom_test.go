package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cmdcfg "github.com/UptickNetwork/uptick/cmd/config"
)

// TestPrepareDefaultGenesisDenomDrivesDerivedDefaults pins the bootstrap step
// that `uptickd init` and `uptickd testnet init-files` now share.
//
// mint's Params.MintDenom, crisis' ConstantFee and gov's deposit minimums are
// copied out of the package-level sdk.DefaultBondDenom while DefaultGenesis
// runs, and the SDK zero value for it is "stake". A genesis entry point that
// forgets to override it produces a chain whose staking denom and minting denom
// disagree: the chain mints "stake" that no account ever holds, while every
// bank balance, deposit and delegation is denominated in the chain denom.
//
// The test asserts both directions on purpose. The baseline half documents that
// the problem is real rather than theoretical; if it ever stops holding, the
// bootstrap step can be reconsidered instead of being kept on faith.
func TestPrepareDefaultGenesisDenomDrivesDerivedDefaults(t *testing.T) {
	uptickApp, _ := sharedTestApp(t)
	cdc := uptickApp.AppCodec()

	// Other tests in this binary (app/sim_test.go, app/sim_staking_ops_test.go)
	// read sdk.DefaultBondDenom directly, so leave it as we found it.
	original := sdk.DefaultBondDenom
	t.Cleanup(func() { sdk.DefaultBondDenom = original })

	// The SDK zero value: the state a process that never bootstrapped is in.
	const untouchedDenom = "stake"
	require.NotEqual(t, cmdcfg.BaseDenom, untouchedDenom,
		"the test only makes sense while the SDK zero denom differs from the chain denom")

	sdk.DefaultBondDenom = untouchedDenom

	type denoms struct {
		mint    string
		crisis  string
		staking string
		gov     string
	}

	readDenoms := func() denoms {
		t.Helper()

		genesis := uptickApp.DefaultGenesis()

		var mintGen minttypes.GenesisState
		cdc.MustUnmarshalJSON(genesis[minttypes.ModuleName], &mintGen)

		var crisisGen crisistypes.GenesisState
		cdc.MustUnmarshalJSON(genesis[crisistypes.ModuleName], &crisisGen)

		var stakingGen stakingtypes.GenesisState
		cdc.MustUnmarshalJSON(genesis[stakingtypes.ModuleName], &stakingGen)

		var govGen govv1.GenesisState
		cdc.MustUnmarshalJSON(genesis[govtypes.ModuleName], &govGen)
		require.NotEmpty(t, govGen.Params.MinDeposit, "gov default genesis has no MinDeposit to inspect")

		return denoms{
			mint:    mintGen.Params.MintDenom,
			crisis:  crisisGen.ConstantFee.Denom,
			staking: stakingGen.Params.BondDenom,
			gov:     govGen.Params.MinDeposit[0].Denom,
		}
	}

	// Baseline: without the bootstrap the derived defaults are the SDK zero
	// value, exactly the malformed genesis the fix removes.
	before := readDenoms()
	require.Equal(t, untouchedDenom, before.mint, "mint derives MintDenom from sdk.DefaultBondDenom")
	require.Equal(t, untouchedDenom, before.crisis, "crisis derives ConstantFee from sdk.DefaultBondDenom")
	require.Equal(t, untouchedDenom, before.staking, "staking derives BondDenom from sdk.DefaultBondDenom")
	require.Equal(t, untouchedDenom, before.gov, "gov derives MinDeposit[0].Denom from sdk.DefaultBondDenom")

	// After the bootstrap every derived default agrees on the chain denom.
	PrepareDefaultGenesisDenom(cmdcfg.BaseDenom)

	after := readDenoms()
	require.Equal(t, cmdcfg.BaseDenom, after.mint)
	require.Equal(t, cmdcfg.BaseDenom, after.crisis)
	require.Equal(t, cmdcfg.BaseDenom, after.staking)
	require.Equal(t, cmdcfg.BaseDenom, after.gov)
}

// TestPrepareDefaultGenesisDenomIgnoresEmpty pins the flag-passthrough
// behaviour `uptickd init` depends on: an unset --default-denom is passed
// straight through and must not wipe the package-level default.
func TestPrepareDefaultGenesisDenomIgnoresEmpty(t *testing.T) {
	original := sdk.DefaultBondDenom
	t.Cleanup(func() { sdk.DefaultBondDenom = original })

	const sentinel = "sentinel-denom"
	sdk.DefaultBondDenom = sentinel

	PrepareDefaultGenesisDenom("")
	require.Equal(t, sentinel, sdk.DefaultBondDenom,
		"an empty denom must be ignored so an unset flag cannot blanket the default")

	PrepareDefaultGenesisDenom(cmdcfg.BaseDenom)
	require.Equal(t, cmdcfg.BaseDenom, sdk.DefaultBondDenom)
}
