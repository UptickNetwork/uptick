package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyPrefixTokenPair(t *testing.T) {
	require.NotNil(t, KeyPrefixTokenPair)
	require.Len(t, KeyPrefixTokenPair, 1)
	require.Equal(t, byte(100), KeyPrefixTokenPair[0])
}

func TestKeyPrefixTokenPairByCW721(t *testing.T) {
	require.NotNil(t, KeyPrefixTokenPairByCW721)
	require.Len(t, KeyPrefixTokenPairByCW721, 1)
	require.Equal(t, byte(101), KeyPrefixTokenPairByCW721[0])
}

func TestKeyPrefixTokenPairByClass(t *testing.T) {
	require.NotNil(t, KeyPrefixTokenPairByClass)
	require.Len(t, KeyPrefixTokenPairByClass, 1)
	require.Equal(t, byte(102), KeyPrefixTokenPairByClass[0])
}

func TestKeyPrefixNFTUIDPairByNFTUID(t *testing.T) {
	require.NotNil(t, KeyPrefixNFTUIDPairByNFTUID)
	require.Len(t, KeyPrefixNFTUIDPairByNFTUID, 1)
}

func TestKeyPrefixNFTUIDPairByTokenUID(t *testing.T) {
	require.NotNil(t, KeyPrefixNFTUIDPairByTokenUID)
	require.Len(t, KeyPrefixNFTUIDPairByTokenUID, 1)
}

func TestKeyPrefixWasmCode(t *testing.T) {
	require.NotNil(t, KeyPrefixWasmCode)
	require.Len(t, KeyPrefixWasmCode, 1)
}

func TestKeyPrefixCwAddressByContractTokenId(t *testing.T) {
	require.NotNil(t, KeyPrefixCwAddressByContractTokenId)
	require.Len(t, KeyPrefixCwAddressByContractTokenId, 1)
}

func TestStoreKeyConstant(t *testing.T) {
	require.Equal(t, "cw721", StoreKey)
	require.Equal(t, "cw721", ModuleName)
	require.Equal(t, "cw721", RouterKey)
}

func TestDefaultPrefix(t *testing.T) {
	require.Equal(t, "uptick", DefaultPrefix)
}

func TestTransferCW721Memo(t *testing.T) {
	require.Equal(t, ":IBCTransferFromCW721", TransferCW721Memo)
}

func TestModuleAddress_Init(t *testing.T) {
	// ModuleAddress should be initialized by init()
	require.NotEmpty(t, AccModuleAddress)
	require.NotEmpty(t, ModuleAddress)
}
