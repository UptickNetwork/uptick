package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	t.Parallel()
	p := DefaultParams()
	require.True(t, p.EnableErc721)
	require.True(t, p.EnableEVMHook)
	require.NoError(t, p.Validate())
}

func TestNewParams(t *testing.T) {
	t.Parallel()
	p := NewParams(false, true)
	require.False(t, p.EnableErc721)
	require.True(t, p.EnableEVMHook)
	require.NoError(t, p.Validate())
}

func TestParamsValidate(t *testing.T) {
	t.Parallel()
	p := DefaultParams()
	require.NoError(t, p.Validate())
}
