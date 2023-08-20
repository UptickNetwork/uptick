package app

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

const (
	// DefaultUptickInstanceCost is initially set the same as in wasmd
	DefaultUptickInstanceCost uint64 = 60_000
	// DefaultUptickCompileCost set to a large number for testing
	DefaultUptickCompileCost uint64 = 100
)

// UptickGasRegisterConfig is defaults plus a custom compile amount
func UptickGasRegisterConfig() wasmkeeper.WasmGasRegisterConfig {
	gasConfig := wasmkeeper.DefaultGasRegisterConfig()
	gasConfig.InstanceCost = DefaultUptickInstanceCost
	gasConfig.CompileCost = DefaultUptickCompileCost

	return gasConfig
}

func NewUptickWasmGasRegister() wasmkeeper.WasmGasRegister {
	return wasmkeeper.NewWasmGasRegister(UptickGasRegisterConfig())
}
