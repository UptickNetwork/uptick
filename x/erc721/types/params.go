package types

import (
	fmt "fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store key
var (
	ParamStoreKeyEnableErc721  = []byte("EnableErc721")
	ParamStoreKeyEnableEVMHook = []byte("EnableEVMHook")
)

var _ paramtypes.ParamSet = &Params{}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

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

func DefaultParams() Params {
	return Params{
		EnableErc721:  true,
		EnableEVMHook: true,
	}
}

func validateBool(i interface{}) error {
	if _, ok := i.(bool); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyEnableErc721, &p.EnableErc721, validateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyEnableEVMHook, &p.EnableEVMHook, validateBool),
	}
}

func (p Params) Validate() error { return nil }
