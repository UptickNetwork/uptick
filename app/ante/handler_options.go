package ante

import (
	sdkerrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	evmante "github.com/cosmos/evm/ante"
	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// HandlerOptions defines the list of module keepers required to run the Uptick
// AnteHandler decorators. It wraps cosmos/evm's HandlerOptions and adds
// Uptick-specific fields.
type HandlerOptions struct {
	AccountKeeper          anteinterfaces.AccountKeeper
	BankKeeper             anteinterfaces.BankKeeper
	IBCKeeper              *ibckeeper.Keeper
	FeeMarketKeeper        anteinterfaces.FeeMarketKeeper
	EvmKeeper              anteinterfaces.EVMKeeper
	FeegrantKeeper         ante.FeegrantKeeper
	SignModeHandler        *txsigning.HandlerMap
	SigGasConsumer         func(meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	TxCounterStoreKey      storetypes.StoreKey
	Cdc                    codec.BinaryCodec
	MaxTxGasWanted         uint64
	TxFeeChecker           ante.TxFeeChecker
	DisabledAuthzMsgs      []string
	MaxWasmDispatchMsgCount uint64
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

// toEvmHandlerOptions converts Uptick HandlerOptions to cosmos/evm HandlerOptions
func (options HandlerOptions) toEvmHandlerOptions() evmante.HandlerOptions {
	return evmante.HandlerOptions{
		Cdc:                    options.Cdc,
		AccountKeeper:          options.AccountKeeper,
		BankKeeper:             options.BankKeeper,
		IBCKeeper:              options.IBCKeeper,
		FeeMarketKeeper:        options.FeeMarketKeeper,
		EvmKeeper:              options.EvmKeeper,
		FeegrantKeeper:         options.FeegrantKeeper,
		SignModeHandler:        options.SignModeHandler,
		SigGasConsumer:         options.SigGasConsumer,
		MaxTxGasWanted:         options.MaxTxGasWanted,
		DynamicFeeChecker:      true,
		PendingTxListener:      nil,
	}
}

// newEthAnteHandler creates the ante handler for Ethereum transactions
// using cosmos/evm's monolithic EVM ante handler
func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return evmante.NewAnteHandler(options.toEvmHandlerOptions())
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions
// with Uptick-specific WasmSecurityDecorator and ValidatorCommissionDecorator added
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	evmOpts := options.toEvmHandlerOptions()
	return evmante.NewAnteHandler(evmOpts)
}

// newCosmosAnteHandlerEip712 creates the ante handler for transactions signed with EIP712
func newCosmosAnteHandlerEip712(options HandlerOptions) sdk.AnteHandler {
	// In cosmos/evm v0.6.1, EIP-712 is handled by the same cosmos ante handler
	return newCosmosAnteHandler(options)
}

// evmtypesAccountKeeper is a shim to satisfy evmtypes.AccountKeeper if needed
var _ evmtypes.AccountKeeper = (evmtypes.AccountKeeper)(nil)
