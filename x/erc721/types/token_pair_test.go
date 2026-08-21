package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewTokenPair(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0xABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCD")
	tp := NewTokenPair(addr, "my/class/id")
	require.Equal(t, "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", tp.Erc721Address)
	require.Equal(t, "my/class/id", tp.ClassId)
}

func TestTokenPair_GetID_Deterministic(t *testing.T) {
	t.Parallel()
	tp := TokenPair{
		Erc721Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		ClassId:       "class/a",
	}
	id1 := tp.GetID()
	id2 := tp.GetID()
	require.Equal(t, id1, id2)
	require.Len(t, id1, 32)
}

func TestTokenPair_GetERC721Contract(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	tp := NewTokenPair(addr, "c")
	require.Equal(t, addr, tp.GetERC721Contract())
}

func TestTokenPair_Validate(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	tp := NewTokenPair(addr, "valid/denom/id")
	require.NoError(t, tp.Validate())

	badDenom := tp
	badDenom.ClassId = ""
	require.Error(t, badDenom.Validate())

	invalidHex := tp
	invalidHex.Erc721Address = "not-hex"
	require.Error(t, invalidHex.Validate())
}
