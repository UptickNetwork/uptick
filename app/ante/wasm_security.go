package ante

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	ethante "github.com/evmos/ethermint/app/ante"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

const (
	// MaxWasmDispatchMsgCount limits the maximum number of nested messages in a CosmWasm DispatchMsg
	// to prevent DoS attacks via excessive nested messages
	MaxWasmDispatchMsgCount = 10

	// EvmMsgTypeURL is the type URL for EVM messages
	EvmMsgTypeURL = "/ethermint.evm.v1.MsgEthereumTx"
)

// WasmSecurityDecorator checks for security issues in CosmWasm messages
// and prevents bypassing AnteHandler gas checks via CosmWasm DispatchMsg
type WasmSecurityDecorator struct {
	cdc            codec.BinaryCodec
	evmKeeper      ethante.EVMKeeper
	maxTxGasWanted uint64
}

// NewWasmSecurityDecorator creates a new WasmSecurityDecorator
func NewWasmSecurityDecorator(cdc codec.BinaryCodec, evmKeeper ethante.EVMKeeper, maxTxGasWanted uint64) WasmSecurityDecorator {
	return WasmSecurityDecorator{
		cdc:            cdc,
		evmKeeper:      evmKeeper,
		maxTxGasWanted: maxTxGasWanted,
	}
}

// AnteHandle inspects CosmWasm messages in the tx to ensure AnteHandler checks are not bypassed
func (wsd WasmSecurityDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Get all messages from the transaction
	msgs := tx.GetMsgs()

	// Validate each message
	for _, msg := range msgs {
		if err := wsd.validateMessage(ctx, msg); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

// validateMessage validates a single sdk.Msg
func (wsd WasmSecurityDecorator) validateMessage(ctx sdk.Context, msg sdk.Msg) error {
	switch msg := msg.(type) {
	case *wasmTypes.MsgExecuteContract:
		// Validate CosmWasm MsgExecuteContract
		return wsd.validateWasmExecuteContract(ctx, msg)

	case *wasmTypes.MsgInstantiateContract:
		// Validate MsgInstantiateContract
		return wsd.validateWasmInstantiateContract(ctx, msg)

	case *wasmTypes.MsgInstantiateContract2:
		// Validate MsgInstantiateContract2
		return wsd.validateWasmInstantiateContract2(ctx, msg)

	default:
		// Check the message type URL to see if it is an EVM message
		// If it is an EVM message, ensure it goes through the proper AnteHandler
		return wsd.checkEvmMessage(ctx, msg)
	}
}

// validateWasmExecuteContract validates a CosmWasm MsgExecuteContract
func (wsd WasmSecurityDecorator) validateWasmExecuteContract(ctx sdk.Context, msg *wasmTypes.MsgExecuteContract) error {
	// Check basic validity of the message
	if err := msg.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "invalid wasm execute contract message")
	}
	// Check if the payload size is too large, which could cause DoS
	if len(msg.Msg) > 1024*1024 { // 1MB limit
		return sdkerrors.Wrapf(
			errortypes.ErrInvalidRequest,
			"wasm execute contract message too large: %d bytes",
			len(msg.Msg),
		)
	}

	return nil
}

// validateWasmInstantiateContract validates a CosmWasm MsgInstantiateContract
func (wsd WasmSecurityDecorator) validateWasmInstantiateContract(ctx sdk.Context, msg *wasmTypes.MsgInstantiateContract) error {
	if err := msg.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "invalid wasm instantiate contract message")
	}

	// Check message size
	if len(msg.Msg) > 1024*1024 { // 1MB limit
		return sdkerrors.Wrapf(
			errortypes.ErrInvalidRequest,
			"wasm instantiate contract message too large: %d bytes",
			len(msg.Msg),
		)
	}

	return nil
}

// validateWasmInstantiateContract2 validates a CosmWasm MsgInstantiateContract2
func (wsd WasmSecurityDecorator) validateWasmInstantiateContract2(ctx sdk.Context, msg *wasmTypes.MsgInstantiateContract2) error {
	if err := msg.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "invalid wasm instantiate contract2 message")
	}

	// Check message size
	if len(msg.Msg) > 1024*1024 { // 1MB limit
		return sdkerrors.Wrapf(
			errortypes.ErrInvalidRequest,
			"wasm instantiate contract2 message too large: %d bytes",
			len(msg.Msg),
		)
	}

	return nil
}

// checkEvmMessage checks whether the message is an EVM message and,
// if so, ensures it goes through the correct AnteHandler
func (wsd WasmSecurityDecorator) checkEvmMessage(ctx sdk.Context, msg sdk.Msg) error {
	// Check message type URL
	msgTypeURL := sdk.MsgTypeURL(msg)
	if msgTypeURL == EvmMsgTypeURL {
		// If it is an EVM message, validate gas limit
		if evmMsg, ok := msg.(*evmtypes.MsgEthereumTx); ok {
			return wsd.validateEvmGasLimit(ctx, evmMsg)
		}
	}

	return nil
}

// validateEvmGasLimit validates the gas limit of an EVM message
func (wsd WasmSecurityDecorator) validateEvmGasLimit(ctx sdk.Context, msg *evmtypes.MsgEthereumTx) error {
	// Get gas limit from the Ethereum transaction
	tx := msg.AsTransaction()
	if tx == nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "invalid ethereum transaction")
	}

	gasLimit := tx.Gas()

	// Check whether gas limit exceeds the configured maximum
	if wsd.maxTxGasWanted > 0 && gasLimit > wsd.maxTxGasWanted {
		return sdkerrors.Wrapf(
			errortypes.ErrOutOfGas,
			"gas limit %d exceeds maximum allowed %d",
			gasLimit,
			wsd.maxTxGasWanted,
		)
	}

	// Check that gas limit is not zero
	if gasLimit == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "gas limit cannot be zero")
	}

	return nil
}

// ExtractMessagesFromTx extracts all messages from a transaction
func (wsd WasmSecurityDecorator) ExtractMessagesFromTx(ctx sdk.Context, tx sdk.Tx) ([]sdk.Msg, error) {
	var allMsgs []sdk.Msg
	msgs := tx.GetMsgs()

	// Use a queue to process nested messages
	msgQueue := make([]sdk.Msg, 0, len(msgs))
	msgQueue = append(msgQueue, msgs...)

	processed := make(map[string]bool)

	for len(msgQueue) > 0 {
		msg := msgQueue[0]
		msgQueue = msgQueue[1:]

		// Avoid processing the same message multiple times
		msgKey := fmt.Sprintf("%s:%s", sdk.MsgTypeURL(msg), msg.String())
		if processed[msgKey] {
			continue
		}
		processed[msgKey] = true

		allMsgs = append(allMsgs, msg)

	}

	return allMsgs, nil
}
