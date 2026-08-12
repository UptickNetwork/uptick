package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetQueryCmd(t *testing.T) {
	t.Parallel()
	cmd := GetQueryCmd()
	require.Equal(t, "erc721", cmd.Use)
	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	require.ElementsMatch(t, []string{
		"token-pairs",
		"token-pair",
		"params",
		"evm-contract",
	}, names)
}

func TestNewTxCmd(t *testing.T) {
	t.Parallel()
	cmd := NewTxCmd()
	require.Equal(t, "erc721", cmd.Use)
	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	require.ElementsMatch(t, []string{
		"convert-nft",
		"convert-erc721",
		"ibc-transfer-erc721",
	}, names)
}
