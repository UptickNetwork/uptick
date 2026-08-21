package keepers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetWasmCapabilities(t *testing.T) {
	caps := GetWasmCapabilities()
	require.NotEmpty(t, caps)

	seen := make(map[string]struct{}, len(caps))
	for _, cap := range caps {
		seen[cap] = struct{}{}
	}

	require.Contains(t, seen, "iterator")
	require.Contains(t, seen, "staking")
	require.Contains(t, seen, "ibc2")
	require.NotContains(t, seen, "stargaze")
	require.NotContains(t, seen, "token_factory")
}
