package types

import (
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	srvflags "github.com/cosmos/evm/server/flags"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
)

type mapOpts map[string]interface{}

func (m mapOpts) Get(key string) interface{} {
	return m[key]
}

func TestParseEIP155ChainID(t *testing.T) {
	id, err := ParseEIP155ChainID("uptick_117-1")
	require.NoError(t, err)
	require.Equal(t, uint64(117), id)

	id, err = ParseEIP155ChainID("uptick_1170-3")
	require.NoError(t, err)
	require.Equal(t, uint64(1170), id)

	id, err = ParseEIP155ChainID("uptick_9000-1")
	require.NoError(t, err)
	require.Equal(t, uint64(9000), id)

	_, err = ParseEIP155ChainID("uptick-117")
	require.Error(t, err)
	_, err = ParseEIP155ChainID("uptick_abc-1")
	require.Error(t, err)
}

func TestIsValidChainID(t *testing.T) {
	require.True(t, IsValidChainID("uptick_117-1"))
	require.True(t, IsValidChainID("uptick_1170-3"))
	require.False(t, IsValidChainID(""))
	require.False(t, IsValidChainID("uptick-117-1"))
	require.False(t, IsValidChainID("cosmoshub-4"))
}

func TestResolveEVMChainID(t *testing.T) {
	require.Equal(t, evmtypes.DefaultEVMChainID, ResolveEVMChainID(mapOpts{}, ""))

	require.Equal(t, uint64(117), ResolveEVMChainID(mapOpts{
		srvflags.EVMChainID: uint64(117),
	}, ""))

	require.Equal(t, uint64(1170), ResolveEVMChainID(mapOpts{
		srvflags.EVMChainID: evmtypes.DefaultEVMChainID,
		flags.FlagChainID:   "uptick_1170-1",
	}, ""))

	require.Equal(t, uint64(1170), ResolveEVMChainID(mapOpts{}, "uptick_1170-1"))
	require.Equal(t, uint64(117), ResolveEVMChainID(mapOpts{}, "uptick_117-1"))
	require.Equal(t, uint64(9000), ResolveEVMChainID(mapOpts{
		srvflags.EVMChainID: uint64(117),
	}, "uptick_9000-1"), "cosmos chain-id must win over app.toml evm-chain-id")
}

func TestChainIDFromGenesisFile(t *testing.T) {
	home := t.TempDir()
	require.Empty(t, ChainIDFromGenesisFile(home))

	require.NoError(t, os.MkdirAll(home+"/config", 0o755))
	require.NoError(t, os.WriteFile(home+"/config/genesis.json", []byte(`{"chain_id":"uptick_9000-1"}`), 0o644))
	require.Equal(t, "uptick_9000-1", ChainIDFromGenesisFile(home))
}
