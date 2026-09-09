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
func (p Params) Validate() error { return nil }
