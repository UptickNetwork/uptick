package cli

import (
	"testing"

	"github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/stretchr/testify/require"
)

func TestGetQueryCmdStructure(t *testing.T) {
	cmd := GetQueryCmd()
	require.Equal(t, types.ModuleName, cmd.Use)
	require.True(t, cmd.DisableFlagParsing)

	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	require.True(t, names["denom"])
	require.True(t, names["denoms"])
	require.True(t, names["collection"])
	require.True(t, names["supply"])
	require.True(t, names["owner"])
	require.True(t, names["token"])
}

func TestNewTxCmdStructure(t *testing.T) {
	cmd := NewTxCmd()
	require.Equal(t, types.ModuleName, cmd.Use)
	require.True(t, cmd.DisableFlagParsing)

	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	require.True(t, names["issue"])
	require.True(t, names["mint"])
	require.True(t, names["edit"])
	require.True(t, names["transfer"])
	require.True(t, names["burn"])
	require.True(t, names["transfer-denom"])
}

func TestFlagSetsContainExpectedFlags(t *testing.T) {
	require.NotNil(t, FsIssueDenom.Lookup(FlagSchema))
	require.NotNil(t, FsIssueDenom.Lookup(FlagMintRestricted))
	require.NotNil(t, FsIssueDenom.Lookup(FlagUpdateRestricted))
	require.NotNil(t, FsMintNFT.Lookup(FlagRecipient))
	require.NotNil(t, FsMintNFT.Lookup(FlagTokenName))
	require.NotNil(t, FsQuerySupply.Lookup(FlagOwner))
	require.NotNil(t, FsQueryOwner.Lookup(FlagDenomID))
}
