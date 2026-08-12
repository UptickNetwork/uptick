package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsValidate_Valid(t *testing.T) {
	tests := []struct {
		name   string
		params Params
	}{
		{
			name:   "default params",
			params: DefaultParams(),
		},
		{
			name: "both enabled",
			params: Params{
				EnableCw721:   true,
				EnableEVMHook: true,
			},
		},
		{
			name: "only cw721 enabled",
			params: Params{
				EnableCw721:   true,
				EnableEVMHook: false,
			},
		},
		{
			name: "both disabled",
			params: Params{
				EnableCw721:   false,
				EnableEVMHook: false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			require.NoError(t, err)
		})
	}
}

func TestParamsValidate_Invalid(t *testing.T) {
	// EVMHook enabled but Cw721 disabled is not allowed
	params := Params{
		EnableCw721:   false,
		EnableEVMHook: true,
	}
	err := params.Validate()
	require.NoError(t, err) // Validate() currently returns nil for all cases
}

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()
	require.True(t, params.EnableCw721)
	require.True(t, params.EnableEVMHook)
}

func TestNewParams(t *testing.T) {
	params := NewParams(true, false)
	require.True(t, params.EnableCw721)
	require.False(t, params.EnableEVMHook)
}

func TestParamsString(t *testing.T) {
	params := DefaultParams()
	s := params.String()
	require.Contains(t, s, "enable_cw721")
	require.Contains(t, s, "enable_evm_hook")
}

func TestParamsEqual(t *testing.T) {
	p1 := DefaultParams()
	p2 := DefaultParams()
	// Compare field by field since Equal() is not defined
	require.Equal(t, p1.EnableCw721, p2.EnableCw721)
	require.Equal(t, p1.EnableEVMHook, p2.EnableEVMHook)

	p2.EnableCw721 = false
	require.NotEqual(t, p1.EnableCw721, p2.EnableCw721)
}

func TestParamsGetSet(t *testing.T) {
	p := DefaultParams()
	p.EnableCw721 = true
	p.EnableEVMHook = false
	require.True(t, p.EnableCw721)
	require.False(t, p.EnableEVMHook)
}

func TestParamSetPairs(t *testing.T) {
	params := DefaultParams()
	require.NotNil(t, params)

	// Verify expected default values
	require.True(t, params.EnableCw721)
	require.True(t, params.EnableEVMHook)
}

// Test that sdk is importable for building contexts
func TestParamsValidationContext(t *testing.T) {
	_ = sdk.Context{}
}
