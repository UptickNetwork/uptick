package config

import wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

const (
	defaultWasmQueryGasLimit      uint64 = 50_000_000
	defaultWasmSimulationGasLimit uint64 = 25_000_000
	defaultWasmMemoryCacheSize    uint32 = 512
)

// DefaultWasmNodeConfig returns Uptick's default wasm node settings.
// Query gas is raised above wasmd's 3M default so CW721 smart queries do not
// fail on larger metadata. Memory cache is 512 MiB.
func DefaultWasmNodeConfig() wasmtypes.NodeConfig {
	cfg := wasmtypes.DefaultNodeConfig()
	cfg.SmartQueryGasLimit = defaultWasmQueryGasLimit
	cfg.MemoryCacheSize = defaultWasmMemoryCacheSize
	simLimit := defaultWasmSimulationGasLimit
	cfg.SimulationGasLimit = &simLimit
	return cfg
}

// WasmConfigTemplate returns the [wasm] app.toml snippet with Uptick defaults.
func WasmConfigTemplate() string {
	return wasmtypes.ConfigTemplate(DefaultWasmNodeConfig())
}
