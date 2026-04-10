package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultPageRequest(t *testing.T) {
	got := newDefaultPageRequest()
	require.NotNil(t, got)
	require.Equal(t, uint64(0), got.Offset)
	require.Equal(t, paginationDefaultLimit, got.Limit)
	require.False(t, got.CountTotal)
	require.False(t, got.Reverse)
	require.Nil(t, got.Key)
}

func TestShapePageRequest(t *testing.T) {
	t.Run("nil request returns default", func(t *testing.T) {
		got := shapePageRequest(nil)
		require.Equal(t, newDefaultPageRequest(), got)
	})

	t.Run("valid limit/key/reverse are preserved", func(t *testing.T) {
		req := &query.PageRequest{
			Key:     []byte("next-key"),
			Limit:   50,
			Reverse: true,
			Offset:  999, // should be ignored
		}
		got := shapePageRequest(req)
		require.Equal(t, []byte("next-key"), got.Key)
		require.Equal(t, uint64(50), got.Limit)
		require.True(t, got.Reverse)
		require.Equal(t, uint64(0), got.Offset)
		require.False(t, got.CountTotal)
	})

	t.Run("too large limit falls back to default", func(t *testing.T) {
		req := &query.PageRequest{Limit: paginationMaxLimit + 1}
		got := shapePageRequest(req)
		require.Equal(t, paginationDefaultLimit, got.Limit)
	})
}
