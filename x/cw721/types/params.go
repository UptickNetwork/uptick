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

func (p Params) Validate() error { return nil }
