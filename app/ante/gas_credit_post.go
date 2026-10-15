package ante

import (
	"strconv"

	"github.com/cometbft/cometbft/crypto/tmhash"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// GasCreditAutoConsumer is the subset of x/appmining keeper used by the
// PostHandler to debit gas credit for enrolled fee payers / eth senders.
type GasCreditAutoConsumer interface {
	AutoConsumeForTx(ctx sdk.Context, feePayer sdk.AccAddress, gasUsed uint64, sourceTxHash []byte) error
}

// GasCreditPostDecorator auto-consumes gas credit after txs when the payer is
// enrolled as an application gas_payer (Batch E / Q-09 P1).
//
// Paths:
//   - EVM: MsgEthereumTx → eth sender + ethereum tx hash + txGasUsed (events)
//   - Cosmos / CosmWasm / EIP-712: FeeTx fee payer + tmhash(txBytes) + gas meter
//
// Ledger debit only — does not replace bank / eth fee deduction.
type GasCreditPostDecorator struct {
	consumer GasCreditAutoConsumer
}

// NewGasCreditPostDecorator returns a PostDecorator. consumer may be nil (no-op).
func NewGasCreditPostDecorator(consumer GasCreditAutoConsumer) GasCreditPostDecorator {
	return GasCreditPostDecorator{consumer: consumer}
}

type gasCreditTarget struct {
	payer  sdk.AccAddress
	gas    uint64
	txHash []byte
}

// PostHandle implements sdk.PostDecorator.
func (d GasCreditPostDecorator) PostHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate, success bool,
	next sdk.PostHandler,
) (sdk.Context, error) {
	if d.consumer != nil && !simulate {
		for _, t := range resolveGasCreditTargets(ctx, tx) {
			if t.payer.Empty() || t.gas == 0 || len(t.txHash) != 32 {
				continue
			}
			if err := d.consumer.AutoConsumeForTx(ctx, t.payer, t.gas, t.txHash); err != nil {
				return ctx, err
			}
		}
	}
	return next(ctx, tx, simulate, success)
}

// resolveGasCreditTargets picks EVM-specific identity when MsgEthereumTx is
// present; otherwise falls back to Cosmos FeeTx + comet tx hash.
func resolveGasCreditTargets(ctx sdk.Context, tx sdk.Tx) []gasCreditTarget {
	ethGasByHash := ethGasUsedByTxHash(ctx)
	var out []gasCreditTarget
	for _, msg := range tx.GetMsgs() {
		ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			continue
		}
		ethTx := ethMsg.AsTransaction()
		if ethTx == nil {
			continue
		}
		hash := ethMsg.Hash().Bytes()
		gas := ethGasByHash[ethMsg.Hash().Hex()]
		if gas == 0 {
			// Fallback: meter after EVM ResetGasMeterAndConsumeGas, or tx gas limit.
			gas = ctx.GasMeter().GasConsumed()
			if gas == 0 {
				gas = ethTx.Gas()
			}
		}
		out = append(out, gasCreditTarget{
			payer:  ethMsg.GetFrom(),
			gas:    gas,
			txHash: hash,
		})
	}
	if len(out) > 0 {
		return out
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil
	}
	payer := sdk.AccAddress(feeTx.FeePayer())
	txBytes := ctx.TxBytes()
	gas := ctx.GasMeter().GasConsumed()
	if payer.Empty() || gas == 0 || len(txBytes) == 0 {
		return nil
	}
	return []gasCreditTarget{{
		payer:  payer,
		gas:    gas,
		txHash: tmhash.Sum(txBytes),
	}}
}

// ethGasUsedByTxHash reads ethereumTxHash → txGasUsed from EVM events emitted
// during RunMsgs (AttributeKeyEthereumTxHash / AttributeKeyTxGasUsed).
func ethGasUsedByTxHash(ctx sdk.Context) map[string]uint64 {
	out := make(map[string]uint64)
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != evmtypes.EventTypeEthereumTx {
			continue
		}
		var hashHex, gasStr string
		for _, attr := range ev.Attributes {
			switch attr.Key {
			case evmtypes.AttributeKeyEthereumTxHash:
				hashHex = attr.Value
			case evmtypes.AttributeKeyTxGasUsed:
				gasStr = attr.Value
			}
		}
		if hashHex == "" || gasStr == "" {
			continue
		}
		gas, err := strconv.ParseUint(gasStr, 10, 64)
		if err != nil {
			continue
		}
		out[hashHex] = gas
	}
	return out
}
