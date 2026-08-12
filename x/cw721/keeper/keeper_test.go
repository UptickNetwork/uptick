package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint64ToBytes_RoundTrip(t *testing.T) {
	tests := []uint64{0, 1, 42, 256, 65535, 1<<32 - 1, 1<<63 - 1, 1<<64 - 1}
	for _, n := range tests {
		bz := Uint64ToBytes(n)
		require.Len(t, bz, 8, "uint64 must encode to 8 bytes")
		got := BytesToUint64(bz)
		require.Equal(t, n, got, "round-trip failed for %d", n)
	}
}

func TestBytesToUint64_Zero(t *testing.T) {
	bz := make([]byte, 8)
	got := BytesToUint64(bz)
	require.Equal(t, uint64(0), got)
}

func TestUint64ToBytes_Deterministic(t *testing.T) {
	bz1 := Uint64ToBytes(42)
	bz2 := Uint64ToBytes(42)
	require.Equal(t, bz1, bz2)
}

func TestBytesToUint64_SmallArrayPanics(t *testing.T) {
	// BytesToUint64 with less than 8 bytes will read out of bounds
	// (this is the big-endian Uint64 contract; it expects 8 bytes)
	require.Panics(t, func() {
		BytesToUint64([]byte{0x01})
	})
}
