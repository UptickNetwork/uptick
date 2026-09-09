package types

// Parameter store key
var (
	KeyPrefixParams = []byte("erc721/params/")
)

// NewParams creates a new Params object
func NewParams(
	enableErc721 bool,
	enableEVMHook bool,
) Params {
	return Params{
		EnableErc721:  enableErc721,
		EnableEVMHook: enableEVMHook,
	}
}

// DefaultParams returns default params
func DefaultParams() Params {
	return Params{
		EnableErc721:  true,
		EnableEVMHook: true,
	}
}

// Validate validates the params.
// EnableErc721 and EnableEVMHook are typed bool fields in the proto message;
// the former type-assertion checks (interface{}(bool).(bool)) were always true
// no-ops and have been removed.
func (p Params) Validate() error {
	return nil
}
