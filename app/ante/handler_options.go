package ante

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	evmante "github.com/cosmos/evm/ante"
	cosmosante "github.com/cosmos/evm/ante/cosmos"
	evmevm "github.com/cosmos/evm/ante/evm"
	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	antetypes "github.com/cosmos/evm/ante/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	corestoretypes "cosmossdk.io/core/store"
	sdkerrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"
)

// HandlerOptions defines the list of module keepers required to run the Uptick
// AnteHandler decorators. It wraps cosmos/evm's HandlerOptions and adds
// Uptick-specific fields.
type HandlerOptions struct {
	AccountKeeper         anteinterfaces.AccountKeeper
	BankKeeper            anteinterfaces.BankKeeper
	IBCKeeper             *ibckeeper.Keeper
	FeeMarketKeeper       anteinterfaces.FeeMarketKeeper
	EvmKeeper             anteinterfaces.EVMKeeper
	FeegrantKeeper        ante.FeegrantKeeper
	SignModeHandler       *txsigning.HandlerMap
	SigGasConsumer        func(meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	Cdc                   codec.BinaryCodec
	MaxTxGasWanted        uint64
	TxFeeChecker          ante.TxFeeChecker
	DisabledAuthzMsgs     []string
	WasmKeeper            *wasmkeeper.Keeper
	WasmNodeConfig        *wasmtypes.NodeConfig
	TXCounterStoreService corestoretypes.KVStoreService
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
	if options.Cdc == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "cdc is required for AnteHandler")
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
		ExtensionOptionChecker: antetypes.HasDynamicFeeExtensionOption,
		PendingTxListener:      nil,
	}
}

func (options HandlerOptions) disabledAuthzMsgs() []string {
	if len(options.DisabledAuthzMsgs) > 0 {
		return options.DisabledAuthzMsgs
	}
	return []string{
		sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
		sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}),
	}
}

// newEthAnteHandler creates the ante handler for Ethereum transactions
// using cosmos/evm's monolithic EVM ante handler
func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return evmante.NewAnteHandler(options.toEvmHandlerOptions())
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions.
// Wasm CountTX / GasRegister / TxContracts run before SetUpContext.
// LimitSimulationGas runs immediately after SetUpContext so the simulation gas
// meter is not overwritten. AuthzLimiter uses DisabledAuthzMsgs from app.go.
//
// cosmos/evm v0.6.1's NewAuthzLimiterDecorator recursively descends into
// authz.MsgExec (see ante/cosmos/authz.go), so nested MsgExec is not a bypass.
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		feemarketParams := options.FeeMarketKeeper.GetParams(ctx)
		txFeeChecker := evmevm.NewDynamicFeeChecker(&feemarketParams)

		var simGasLimit *storetypes.Gas
		if options.WasmNodeConfig != nil {
			simGasLimit = options.WasmNodeConfig.SimulationGasLimit
		}

		decorators := []sdk.AnteDecorator{
			NewMessageSecurityDecorator(options.Cdc, options.MaxTxGasWanted),
			NewValidatorCommissionDecorator(options.Cdc),
		}
		if options.TXCounterStoreService != nil {
			decorators = append(decorators, wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService))
		}
		if options.WasmKeeper != nil {
			decorators = append(decorators, wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()))
		}

		extChecker := antetypes.HasDynamicFeeExtensionOption

		decorators = append(decorators,
			wasmkeeper.NewTxContractsDecorator(),
			cosmosante.NewRejectMessagesDecorator(),
			cosmosante.NewAuthzLimiterDecorator(options.disabledAuthzMsgs()...),
			ante.NewSetUpContextDecorator(),
			wasmkeeper.NewLimitSimulationGasDecorator(simGasLimit),
			ante.NewExtensionOptionsDecorator(extChecker),
			ante.NewValidateBasicDecorator(),
			ante.NewTxTimeoutHeightDecorator(),
			ante.NewValidateMemoDecorator(options.AccountKeeper),
			cosmosante.NewMinGasPriceDecorator(&feemarketParams),
			ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
			ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, txFeeChecker),
			ante.NewSetPubKeyDecorator(options.AccountKeeper),
			ante.NewValidateSigCountDecorator(options.AccountKeeper),
			ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
			ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
			ante.NewIncrementSequenceDecorator(options.AccountKeeper),
			ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
			evmevm.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper, &feemarketParams),
		)

		return sdk.ChainAnteDecorators(decorators...)(ctx, tx, simulate)
	}
}

// newCosmosAnteHandlerEip712 creates the ante handler for Cosmos transactions
// signed with the legacy ethermint EIP-712 scheme (e.g. Keplr). It differs from
// newCosmosAnteHandler in that it skips the extension-option rejection decorator
// (the EIP-712 Web3 extension is handled here) and verifies the EIP-712
// signature through Eip712SigVerificationDecorator instead of the standard
// signature verification decorator.
func newCosmosAnteHandlerEip712(options HandlerOptions) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		feemarketParams := options.FeeMarketKeeper.GetParams(ctx)
		txFeeChecker := evmevm.NewDynamicFeeChecker(&feemarketParams)

		var simGasLimit *storetypes.Gas
		if options.WasmNodeConfig != nil {
			simGasLimit = options.WasmNodeConfig.SimulationGasLimit
		}

		decorators := []sdk.AnteDecorator{
			NewMessageSecurityDecorator(options.Cdc, options.MaxTxGasWanted),
			NewValidatorCommissionDecorator(options.Cdc),
		}
		if options.TXCounterStoreService != nil {
			decorators = append(decorators, wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService))
		}
		if options.WasmKeeper != nil {
			decorators = append(decorators, wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()))
		}

		decorators = append(decorators,
			wasmkeeper.NewTxContractsDecorator(),
			cosmosante.NewRejectMessagesDecorator(),
			cosmosante.NewAuthzLimiterDecorator(options.disabledAuthzMsgs()...),
			ante.NewSetUpContextDecorator(),
			wasmkeeper.NewLimitSimulationGasDecorator(simGasLimit),
			// accept exactly the Web3 extension the EIP-712 signature
			// verifier requires. The previous DynamicFee-only checker rejected
			// valid Keplr txs with "unknown extension options" before they
			// could reach Eip712SigVerificationDecorator below.
			ante.NewExtensionOptionsDecorator(HasWeb3ExtensionOption),
			ante.NewValidateBasicDecorator(),
			ante.NewTxTimeoutHeightDecorator(),
			ante.NewValidateMemoDecorator(options.AccountKeeper),
			cosmosante.NewMinGasPriceDecorator(&feemarketParams),
			ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
			ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, txFeeChecker),
			ante.NewSetPubKeyDecorator(options.AccountKeeper),
			ante.NewValidateSigCountDecorator(options.AccountKeeper),
			ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
			NewEip712SigVerificationDecorator(options.AccountKeeper, options.Cdc),
			ante.NewIncrementSequenceDecorator(options.AccountKeeper),
			ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
			evmevm.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper, &feemarketParams),
		)

		return sdk.ChainAnteDecorators(decorators...)(ctx, tx, simulate)
	}
}

// evmtypesAccountKeeper is a shim to satisfy evmtypes.AccountKeeper if needed
var _ evmtypes.AccountKeeper = (evmtypes.AccountKeeper)(nil)
