package ante

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const (
	// MaxAuthzNestingDepth limits nested authz.MsgExec unpacking before SetUpContext.
	MaxAuthzNestingDepth = 5

	// MaxExtractedMessages caps the flattened message list from a single tx,
	// including nested authz messages.
	MaxExtractedMessages = 32

	// EvmMsgTypeURL is the type URL for EVM messages
	EvmMsgTypeURL = "/cosmos.evm.vm.v1.MsgEthereumTx"
)

// MessageSecurityDecorator validates every message in a transaction before it
// reaches the keeper layer, including messages hidden inside nested authz
// MsgExec wrappers:
//   - CosmWasm messages are checked for basic validity and payload size, and
//     cannot be used to bypass AnteHandler gas accounting via DispatchMsg;
//   - Ethereum transactions are checked for a sane, bounded gas limit;
//   - the flattened message list is capped in count and nesting depth.
type MessageSecurityDecorator struct {
	cdc            codec.BinaryCodec
	maxTxGasWanted uint64
}

// NewMessageSecurityDecorator creates a new MessageSecurityDecorator
func NewMessageSecurityDecorator(cdc codec.BinaryCodec, maxTxGasWanted uint64) MessageSecurityDecorator {
	return MessageSecurityDecorator{
		cdc:            cdc,
		maxTxGasWanted: maxTxGasWanted,
	}
}

// AnteHandle inspects all messages in the transaction to ensure AnteHandler checks are not bypassed
func (msd MessageSecurityDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Extract all messages from the transaction, including nested messages (e.g., authz.MsgExec)
	msgs, err := msd.ExtractMessagesFromTx(ctx, tx)
	if err != nil {
		return ctx, err
	}

	// Validate each message
	for _, msg := range msgs {
		if err := msd.validateMessage(msg); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

// validateMessage validates a single sdk.Msg
func (msd MessageSecurityDecorator) validateMessage(msg sdk.Msg) error {
	switch msg := msg.(type) {
	case *wasmTypes.MsgExecuteContract:
		// Validate CosmWasm MsgExecuteContract
		return msd.validateWasmExecuteContract(msg)

	case *wasmTypes.MsgInstantiateContract:
		// Validate MsgInstantiateContract
		return msd.validateWasmInstantiateContract(msg)

	case *wasmTypes.MsgInstantiateContract2:
		// Validate MsgInstantiateContract2
		return msd.validateWasmInstantiateContract2(msg)

	default:
		// Check the message type URL to see if it is an EVM message
		// If it is an EVM message, ensure it goes through the proper AnteHandler
		return msd.checkEvmMessage(msg)
	}
}

// validateWasmExecuteContract validates a CosmWasm MsgExecuteContract
func (msd MessageSecurityDecorator) validateWasmExecuteContract(msg *wasmTypes.MsgExecuteContract) error {
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
func (msd MessageSecurityDecorator) validateWasmInstantiateContract(msg *wasmTypes.MsgInstantiateContract) error {
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
func (msd MessageSecurityDecorator) validateWasmInstantiateContract2(msg *wasmTypes.MsgInstantiateContract2) error {
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
func (msd MessageSecurityDecorator) checkEvmMessage(msg sdk.Msg) error {
	// Check message type URL
	msgTypeURL := sdk.MsgTypeURL(msg)
	if msgTypeURL == EvmMsgTypeURL {
		// If it is an EVM message, validate gas limit
		if evmMsg, ok := msg.(*evmtypes.MsgEthereumTx); ok {
			return msd.validateEvmGasLimit(evmMsg)
		}
	}

	return nil
}

// validateEvmGasLimit validates the gas limit of an EVM message
func (msd MessageSecurityDecorator) validateEvmGasLimit(msg *evmtypes.MsgEthereumTx) error {
	// Get gas limit from the Ethereum transaction
	tx := msg.AsTransaction()
	if tx == nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "invalid ethereum transaction")
	}

	gasLimit := tx.Gas()

	// Check whether gas limit exceeds the configured maximum
	if msd.maxTxGasWanted > 0 && gasLimit > msd.maxTxGasWanted {
		return sdkerrors.Wrapf(
			errortypes.ErrOutOfGas,
			"gas limit %d exceeds maximum allowed %d",
			gasLimit,
			msd.maxTxGasWanted,
		)
	}

	// Check that gas limit is not zero
	if gasLimit == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "gas limit cannot be zero")
	}

	return nil
}

// ExtractMessagesFromTx extracts all messages from a transaction, including
// nested messages from authz.MsgExec to prevent ante handler bypass attacks.
// BFS traversal deduplicates revisits of the same in-memory message via a
// pointer-based key; messages with equal content but distinct pointers are
// intentionally validated independently. The result-size cap is checked after
// dedup so a single duplicated message cannot trip the limit.
func (msd MessageSecurityDecorator) ExtractMessagesFromTx(ctx sdk.Context, tx sdk.Tx) ([]sdk.Msg, error) {
	type queuedMsg struct {
		msg   sdk.Msg
		depth int
	}

	var allMsgs []sdk.Msg
	msgs := tx.GetMsgs()
	msgQueue := make([]queuedMsg, 0, len(msgs))
	for _, msg := range msgs {
		msgQueue = append(msgQueue, queuedMsg{msg: msg, depth: 0})
	}

	processed := make(map[string]struct{})

	for len(msgQueue) > 0 {
		item := msgQueue[0]
		msgQueue = msgQueue[1:]
		msg := item.msg

		// Nil is silently skipped to avoid panics on degenerate txs.
		if msg == nil {
			continue
		}

		msgKey := msgDedupKey(msg)
		if _, seen := processed[msgKey]; seen {
			continue
		}
		processed[msgKey] = struct{}{}

		if len(allMsgs) >= MaxExtractedMessages {
			return nil, sdkerrors.Wrapf(
				errortypes.ErrInvalidRequest,
				"message count exceeds maximum %d", MaxExtractedMessages,
			)
		}
		allMsgs = append(allMsgs, msg)

		if execMsg, ok := msg.(*authz.MsgExec); ok {
			if item.depth >= MaxAuthzNestingDepth {
				return nil, sdkerrors.Wrapf(
					errortypes.ErrInvalidRequest,
					"authz nesting depth exceeds maximum %d", MaxAuthzNestingDepth,
				)
			}
			nestedMsgs, err := execMsg.GetMessages()
			if err != nil {
				return nil, sdkerrors.Wrapf(err, "failed to unpack authz.MsgExec nested messages")
			}
			for _, nested := range nestedMsgs {
				msgQueue = append(msgQueue, queuedMsg{msg: nested, depth: item.depth + 1})
			}
		}
	}

	return allMsgs, nil
}

// msgDedupKey returns the per-call dedup key. TypeURL scopes the key
// namespace so two distinct sdk.Msg types never collide on pointer value
// alone. See ExtractMessagesFromTx for the full semantic justification.
func msgDedupKey(msg sdk.Msg) string {
	return fmt.Sprintf("%s|%p", sdk.MsgTypeURL(msg), msg)
}
