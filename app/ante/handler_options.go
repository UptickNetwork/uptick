package ante

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	evmante "github.com/cosmos/evm/ante"
	cosmosante "github.com/cosmos/evm/ante/cosmos"
	evmevm "github.com/cosmos/evm/ante/evm"
	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	antetypes "github.com/cosmos/evm/ante/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
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
	// Fall back to the full disable list (single source of truth in
	// DisabledAuthzMsgTypeURLs) so a second construction point that forgets to
	// set DisabledAuthzMsgs still denies gov/upgrade/IBC-client messages rather
	// than silently dropping that protection.
	return DisabledAuthzMsgTypeURLs()
}

// newEthAnteHandler creates the ante handler for Ethereum transactions
// using cosmos/evm's monolithic EVM ante handler
func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return evmante.NewAnteHandler(options.toEvmHandlerOptions())
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions.
// SetUpContext runs before the wasm CountTX / GasRegister / TxContracts so their
// KV writes (CountTX counter) are metered by the tx gas meter rather than the
// infinite meter BaseApp presets. LimitSimulationGas must run after SetUpContext
// so the simulation gas meter is not overwritten. AuthzLimiter uses
// DisabledAuthzMsgs from app.go.
//
// cosmos/evm v0.6.1's NewAuthzLimiterDecorator recursively descends into
// authz.MsgExec (see ante/cosmos/authz.go), so nested MsgExec is not a bypass.
// cosmosAnteDecorators returns the AnteDecorator chain shared by
// newCosmosAnteHandler and newCosmosAnteHandlerEip712. Extracting the chain
// here (rather than inlining it in each handler) serves two purposes:
//
//  1. The two handlers must stay in lock-step on every decorator they have in
//     common; refactors to one without the other would otherwise silently
//     regress on one transaction path.
//  2. Tests can pin the relative order of decorators (e.g. SetUpContext before
//     CountTX) by reflecting over the returned slice.
//
// SetUpContext is intentionally placed BEFORE the wasm CountTX / GasRegister /
// TxContracts decorators so their KV writes are metered by the tx gas meter
// rather than the infinite meter BaseApp presets before ante. LimitSimulationGas
// must run after SetUpContext so the simulation gas meter is not overwritten.
//
// cosmos/evm v0.6.1's NewAuthzLimiterDecorator recursively descends into
// authz.MsgExec (see ante/cosmos/authz.go), so nested MsgExec is not a bypass.
func cosmosAnteDecorators(
	options HandlerOptions,
	feemarketParams *feemarkettypes.Params,
	extChecker func(*codectypes.Any) bool,
	sigVerify sdk.AnteDecorator,
	txFeeChecker ante.TxFeeChecker,
) []sdk.AnteDecorator {
	var simGasLimit *storetypes.Gas
	if options.WasmNodeConfig != nil {
		simGasLimit = options.WasmNodeConfig.SimulationGasLimit
	}

	decorators := []sdk.AnteDecorator{
		NewMessageSecurityDecorator(options.Cdc, options.MaxTxGasWanted),
		NewValidatorCommissionDecorator(options.Cdc),
		// SetUpContext must precede the wasm decorators so their KV writes
		// (CountTX counter) are metered by the tx gas meter, not the
		// infinite meter BaseApp presets before ante.
		ante.NewSetUpContextDecorator(),
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
		wasmkeeper.NewLimitSimulationGasDecorator(simGasLimit),
		ante.NewExtensionOptionsDecorator(extChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		cosmosante.NewMinGasPriceDecorator(feemarketParams),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, txFeeChecker),
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		sigVerify,
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		evmevm.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper, feemarketParams),
	)
	return decorators
}

func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		feemarketParams := options.FeeMarketKeeper.GetParams(ctx)
		txFeeChecker := evmevm.NewDynamicFeeChecker(&feemarketParams)

		decorators := cosmosAnteDecorators(
			options,
			&feemarketParams,
			antetypes.HasDynamicFeeExtensionOption,
			ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
			txFeeChecker,
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

		// accept exactly the Web3 extension the EIP-712 signature
		// verifier requires. The previous DynamicFee-only checker rejected
		// valid Keplr txs with "unknown extension options" before they
		// could reach Eip712SigVerificationDecorator below.
		decorators := cosmosAnteDecorators(
			options,
			&feemarketParams,
			HasWeb3ExtensionOption,
			NewEip712SigVerificationDecorator(options.AccountKeeper, options.Cdc),
			txFeeChecker,
		)

		return sdk.ChainAnteDecorators(decorators...)(ctx, tx, simulate)
	}
}
