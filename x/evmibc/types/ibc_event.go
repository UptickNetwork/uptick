package types

// IBCERC20EventStatus mirrors the Status enum from the legacy ERC20 IBC events
// (github.com/UptickNetwork/uptick/x/erc20/types). Retained as a local type
// because the ERC20 module has been migrated to cosmos/evm v0.6.1.
type IBCERC20Status int32

const (
	// IBCERC20Status_UNKNOWN is the default state.
	IBCERC20Status_UNKNOWN IBCERC20Status = 0
	// IBCERC20Status_FAILED indicates the IBC transfer failed.
	IBCERC20Status_FAILED IBCERC20Status = 1
	// IBCERC20Status_SUCCESS indicates the IBC transfer succeeded.
	IBCERC20Status_SUCCESS IBCERC20Status = 2
)
