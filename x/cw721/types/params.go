package types

// Parameter store key
var (
	KeyPrefixParams = []byte("cw721/params/")
)

// NewParams creates a new Params object
func NewParams(
	enableCw721 bool,
	enableEVMHook bool,
) Params {
	return Params{
		EnableCw721:   enableCw721,
		EnableEVMHook: enableEVMHook,
	}
}

func DefaultParams() Params {
	return Params{
		EnableCw721:   true,
		EnableEVMHook: true,
	}
}

// Validate performs basic sanity checks on the module parameters.
//
// There is deliberately nothing to reject today:
//   - EnableCw721 / EnableEVMHook are booleans — every value is meaningful.
//   - WasmCodeId is a uint64 code ID. Zero is a LEGAL value meaning "not
//     configured": keeper.GetParams back-fills it from the store-backed wasm
//     code ID, and keeper.SetParams only persists it when non-zero. Whether a
//     non-zero ID actually resolves to an uploaded contract is a state
//     question that cannot be answered here (this package has no keeper/state
//     access); it surfaces as a runtime error at contract instantiation.
//
// The check is kept (rather than inlined) so a future param with real
// constraints has an obvious home.
//
// Read this before adding a rule. Validate is reached from BOTH the genesis
// path (GenesisState.Validate, called by module.go's
// AppModuleBasic.ValidateGenesis) and the governance path
// (MsgUpdateParams.ValidateBasic -> keeper.UpdateParams). A rejecting rule here
// is therefore not a local change: it is a statement about what the chain will
// accept from its own genesis file, and a wrong one makes an operator unable to
// validate the backup they would restore from.
//
// Live-chain evidence, queried 2026-09-13:
//
//	mainnet (v0.3.3)  GET https://rest.uptick.network/uptick/cw721/v1/params
//	  -> {"enable_cw721":true,"enable_evm_hook":true}            (wasm_code_id 0)
//	testnet (v0.4.0)  GET https://rest.origin.uptick.network/uptick/cw721/v1/params
//	  -> {"enable_cw721":true,"enable_evm_hook":true,"wasm_code_id":"2"}
//
// So: do NOT reject WasmCodeId == 0. Mainnet runs without one, and a non-zero
// requirement would reject mainnet's own exported genesis. Do not reject a
// boolean combination either — app/upgrades/v040/upgrades.go copies these
// params forward verbatim, and EnableCw721=false is a supported way to switch
// conversions off (MsgConvertCW721 / MsgConvertNFT both gate on it alone).
//
// The concern that motivated looking at this — governance pointing WasmCodeId
// at a contract that is not cw721 — cannot be checked here at all: this package
// has no state access, so it cannot resolve a code ID. That check belongs at
// the deploy site (keeper/wasm.go), with a rollback, not in a pure predicate
// that is also the genesis gate. Keeping this function a documented no-op is
// what makes the "zero means not configured" contract testable; see
// TestParamsValidateAcceptsLiveChainParams.
func (p Params) Validate() error { return nil }
