package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWasmConfigTemplate(t *testing.T) {
	tmpl := WasmConfigTemplate()
	require.Contains(t, tmpl, "[wasm]")
	require.Contains(t, tmpl, "query_gas_limit = 50000000")
	require.Contains(t, tmpl, "memory_cache_size = 512")
	require.Contains(t, tmpl, "simulation_gas_limit = 25000000")
}

func TestDefaultWasmNodeConfig(t *testing.T) {
	cfg := DefaultWasmNodeConfig()
	require.Equal(t, uint64(50_000_000), cfg.SmartQueryGasLimit)
	require.Equal(t, uint32(512), cfg.MemoryCacheSize)
	require.NotNil(t, cfg.SimulationGasLimit)
	require.Equal(t, uint64(25_000_000), *cfg.SimulationGasLimit)
	require.True(t, strings.Contains(WasmConfigTemplate(), "query_gas_limit"))
}
