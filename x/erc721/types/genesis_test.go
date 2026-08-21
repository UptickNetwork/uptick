package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func validPair(t *testing.T, ercHex, class string) TokenPair {
	t.Helper()
	return NewTokenPair(common.HexToAddress(ercHex), class)
}

func TestGenesisState_Validate(t *testing.T) {
	t.Parallel()
	p := validPair(t, "0x3333333333333333333333333333333333333333", "pair/class/a")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	require.NoError(t, gs.Validate())

	dupErc := NewGenesisState(DefaultParams(), []TokenPair{
		validPair(t, "0x4444444444444444444444444444444444444444", "c1"),
		validPair(t, "0x4444444444444444444444444444444444444444", "c2"),
	})
	require.Error(t, dupErc.Validate())

	dupClass := NewGenesisState(DefaultParams(), []TokenPair{
		validPair(t, "0x5555555555555555555555555555555555555555", "same/class"),
		validPair(t, "0x6666666666666666666666666666666666666666", "same/class"),
	})
	require.Error(t, dupClass.Validate())
}

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()
	gs := DefaultGenesisState()
	require.NotNil(t, gs)
	require.Empty(t, gs.TokenPairs)
	require.NoError(t, gs.Validate())
}
