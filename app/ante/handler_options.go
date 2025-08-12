package ante

import (
	sdkerrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ethante "github.com/evmos/ethermint/app/ante"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// HandlerOptions defines the list of module keepers required to run the Evmos
// AnteHandler decorators.
type HandlerOptions struct {
	AccountKeeper evmtypes.AccountKeeper
	BankKeeper    evmtypes.BankKeeper
	IBCKeeper     *ibckeeper.Keeper
	//FeeMarketKeeper evmtypes.FeeMarketKeeper
	FeeMarketKeeper   ethante.FeeMarketKeeper
	EvmKeeper         ethante.EVMKeeper
	FeegrantKeeper    ante.FeegrantKeeper
	SignModeHandler   *txsigning.HandlerMap
	SigGasConsumer    func(meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	TxCounterStoreKey storetypes.StoreKey
	WasmConfig        wasmTypes.WasmConfig
	Cdc               codec.BinaryCodec
	MaxTxGasWanted    uint64
	TxFeeChecker      ante.TxFeeChecker
}

// Validate checks if the keepers are defined
func (options HandlerOptions) Validate() error {
	if options.AccountKeeper == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if options.SignModeHandler == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "sign mode handler is required for ante builder")
	}
	if options.FeeMarketKeeper == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "fee market keeper is required for AnteHandler")
	}
	if options.EvmKeeper == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "evm keeper is required for AnteHandler")
	}
	return nil
}

// newCosmosAnteHandler creates the default ante handler for Ethereum transactions
func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ethante.NewEthSetUpContextDecorator(options.EvmKeeper),                         // outermost AnteDecorator. SetUpContext must be called first
		ethante.NewEthMempoolFeeDecorator(options.EvmKeeper),                           // Check eth effective gas price against the node's minimal-gas-prices config
		ethante.NewEthMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper), // Check eth effective gas price against the global MinGasPrice
		ethante.NewEthValidateBasicDecorator(options.EvmKeeper),
		ethante.NewEthSigVerificationDecorator(options.EvmKeeper),
		ethante.NewEthAccountVerificationDecorator(options.AccountKeeper, options.EvmKeeper),
		ethante.NewCanTransferDecorator(options.EvmKeeper),
		ethante.NewEthGasConsumeDecorator(options.EvmKeeper, options.MaxTxGasWanted),
		ethante.NewEthIncrementSenderSequenceDecorator(options.AccountKeeper),
		ethante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
		ethante.NewEthEmitEventDecorator(options.EvmKeeper), // emit eth tx hash and index at the very last ante handler.
	)
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ethante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		ante.NewSetUpContextDecorator(),
		//	ante.NewRejectExtensionOptionsDecorator(),
		ante.NewValidateBasicDecorator(),
		ethante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		NewValidatorCommissionDecorator(options.Cdc),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		ethante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	)
}

// newCosmosAnteHandlerEip712 creates the ante handler for transactions signed with EIP712
func newCosmosAnteHandlerEip712(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ethante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		ante.NewSetUpContextDecorator(),
		ante.NewValidateBasicDecorator(),
		ethante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		NewValidatorCommissionDecorator(options.Cdc),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		ethante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	)
}
