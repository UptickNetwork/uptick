package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseEIP155ChainID(t *testing.T) {
	id, err := ParseEIP155ChainID("uptick_117-1")
	require.NoError(t, err)
	require.Equal(t, uint64(117), id)

	id, err = ParseEIP155ChainID("uptick_1170-3")
	require.NoError(t, err)
	require.Equal(t, uint64(1170), id)

	_, err = ParseEIP155ChainID("uptick-117")
	require.Error(t, err)
	_, err = ParseEIP155ChainID("uptick_abc-1")
	require.Error(t, err)
}
