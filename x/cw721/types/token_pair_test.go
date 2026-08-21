package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewTokenPair(t *testing.T) {
	tp := NewTokenPair("0x1234567890abcdef1234567890abcdef12345678", "class-1")
	require.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", tp.Cw721Address)
	require.Equal(t, "class-1", tp.ClassId)
}

func TestTokenPairGetID(t *testing.T) {
	tp1 := NewTokenPair("0xAABBCCDDEEFF00112233445566778899AABBCCDD", "class-alpha")
	tp2 := NewTokenPair("0xAABBCCDDEEFF00112233445566778899AABBCCDD", "class-alpha")
	tp3 := NewTokenPair("0x1111111111111111111111111111111111111111", "class-alpha")

	// Same inputs produce same ID
	require.Equal(t, tp1.GetID(), tp2.GetID())

	// Different address produces different ID
	require.NotEqual(t, tp1.GetID(), tp3.GetID())

	// ID is non-empty
	require.NotEmpty(t, tp1.GetID())
}

func TestTokenPairGetCW721Contract(t *testing.T) {
	hexAddr := "0xDEADC0DE00000000000000000000000000DEADC0DE"
	tp := NewTokenPair(hexAddr, "test-class")
	addr := tp.GetCW721Contract()
	require.Equal(t, common.HexToAddress(hexAddr), addr)
}

func TestTokenPairValidate_Valid(t *testing.T) {
	addr1 := sdk.AccAddress([]byte("cw721contractaddrxx")).String()
	tp1 := NewTokenPair(addr1, "class001")
	err := tp1.Validate()
	require.NoError(t, err)
}

func TestTokenPairValidate_InvalidAddress(t *testing.T) {
	// address too short
	tp := NewTokenPair("0x123", "class001")
	err := tp.Validate()
	require.Error(t, err)
}

func TestTokenPairValidate_EmptyAddress(t *testing.T) {
	tp := NewTokenPair("", "class001")
	err := tp.Validate()
	require.Error(t, err)
}
