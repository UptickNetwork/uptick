package ante

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
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

// HandlerOptions defines the module keepers required to run the Uptick
// AnteHandler decorators: cosmos/evm's HandlerOptions plus Uptick-specific
// fields.
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

// Validate checks if the keepers are defined.
//
// This is the only startup gate for the ante wire-up: cosmos/evm's own
// NewAnteHandler does not validate its options (see ante/ante.go:74), so a
// missing keeper is not caught until the first transaction reaches the
// decorator that dereferences it. Every field checked below is one that
// cosmosAnteDecorators passes to a decorator unconditionally — IBCKeeper to
// NewRedundantRelayDecorator and SigGasConsumer to NewSigGasConsumeDecorator —
// so nil there is a guaranteed panic on the first tx rather than a degraded
// feature. The required set matches cosmos/evm@v0.6.2's own
// ante.HandlerOptions.Validate except for PendingTxListener, which this package
// pins to nil in toEvmHandlerOptions.
//
// Deliberately NOT required:
//
//   - FeegrantKeeper: cosmos-sdk's DeductFeeDecorator reads nil as "feegrant
//     disabled", a supported configuration (upstream's Validate omits it too),
//     so rejecting nil here would turn a supported wire-up into a startup
//     failure.
//   - WasmKeeper / WasmNodeConfig / TXCounterStoreService: each guards an
//     optional decorator inside cosmosAnteDecorators, and
//     TestWasmDecoratorsOmittedWhenKeepersNil pins the nil form as supported
//     input. Requiring them would break that contract.
func (options HandlerOptions) Validate() error {
	if options.AccountKeeper == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if options.IBCKeeper == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "ibc keeper is required for AnteHandler")
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
	if options.SigGasConsumer == nil {
		return sdkerrors.Wrap(errortypes.ErrLogic, "signature gas consumer is required for AnteHandler")
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

// cosmosAnteDecorators returns the AnteDecorator chain shared by
// newCosmosAnteHandler and newCosmosAnteHandlerEip712. Extracting it here (rather
// than inlining it in each handler) keeps the two in lock-step -- a refactor to
// one without the other would otherwise silently regress one tx path -- and lets
// tests pin the relative order of decorators by reflecting over the slice.
//
// Two orderings are load-bearing:
//
//   - SetUpContext precedes the wasm CountTX / GasRegister / TxContracts
//     decorators, so their KV writes (the CountTX counter) are metered by the tx
//     gas meter rather than the infinite meter BaseApp presets before ante.
//   - LimitSimulationGas follows SetUpContext, so the simulation gas meter is not
//     overwritten.
//
// AuthzLimiter uses DisabledAuthzMsgs from app.go. cosmos/evm@v0.6.2's
// NewAuthzLimiterDecorator recursively descends into authz.MsgExec
// (ante/cosmos/authz.go), so a nested MsgExec is not a bypass.
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
		// SetUpContext first: the wasm decorators' KV writes must hit the tx gas
		// meter, not BaseApp's infinite preset meter.
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

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions.
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
// newCosmosAnteHandler in the extension-option checker it accepts
// (HasWeb3ExtensionOption, so the EIP-712 Web3 extension passes) and in the
// signature verifier it uses (Eip712SigVerificationDecorator instead of the
// standard one).
func newCosmosAnteHandlerEip712(options HandlerOptions) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		feemarketParams := options.FeeMarketKeeper.GetParams(ctx)
		txFeeChecker := evmevm.NewDynamicFeeChecker(&feemarketParams)

		// Accept exactly the Web3 extension the EIP-712 verifier requires: a
		// DynamicFee-only checker rejects valid Keplr txs with "unknown extension
		// options" before they reach Eip712SigVerificationDecorator.
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
