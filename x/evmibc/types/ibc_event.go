package types

// IBCERC20Status mirrors the Status enum from the legacy (removed) uptick
// x/erc20 IBC events. Retained as a local type because the self-developed erc20
// module was replaced by cosmos/evm v0.6.1 in v0.4.0.
type IBCERC20Status int32

const (
	// IBCERC20Status_UNKNOWN is the default state.
	IBCERC20Status_UNKNOWN IBCERC20Status = 0
	// IBCERC20Status_FAILED indicates the IBC transfer failed.
	IBCERC20Status_FAILED IBCERC20Status = 1
	// IBCERC20Status_SUCCESS indicates the IBC transfer succeeded.
	IBCERC20Status_SUCCESS IBCERC20Status = 2
)
