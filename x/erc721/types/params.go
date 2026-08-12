package types

import (
	fmt "fmt"
)

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

// Validate validates the params
func (p Params) Validate() error {
	if _, ok := interface{}(p.EnableErc721).(bool); !ok {
		return fmt.Errorf("invalid parameter type for EnableErc721: %T", p.EnableErc721)
	}
	if _, ok := interface{}(p.EnableEVMHook).(bool); !ok {
		return fmt.Errorf("invalid parameter type for EnableEVMHook: %T", p.EnableEVMHook)
	}
	return nil
}
