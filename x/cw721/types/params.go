package types

import (
	"fmt"
)

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

func validateBool(i interface{}) error {
	if _, ok := i.(bool); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func (p Params) Validate() error { return nil }
