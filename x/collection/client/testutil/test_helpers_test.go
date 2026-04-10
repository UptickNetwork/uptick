package testutil

import (
	"fmt"
	"testing"

	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/stretchr/testify/require"
)

func TestMakeFromArgs(t *testing.T) {
	got := makeFromArgs("alice", "denom", "token")
	require.Equal(t, []string{
		"denom",
		"token",
		fmt.Sprintf("--%s=%s", flags.FlagFrom, "alice"),
	}, got)
}

func TestMakeJSONQueryArgs(t *testing.T) {
	got := makeJSONQueryArgs("denom", "token")
	require.Equal(t, []string{
		"denom",
		"token",
		fmt.Sprintf("--%s=json", cli.OutputFlag),
	}, got)
}

func TestWithExtraArgs(t *testing.T) {
	base := []string{"a", "b"}
	got := withExtraArgs(base, "--gas=200000", "--fees=10auptick")
	require.Equal(t, []string{"a", "b", "--gas=200000", "--fees=10auptick"}, got)
}
