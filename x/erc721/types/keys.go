package types

import (
	"github.com/ethereum/go-ethereum/common"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// constants
const (
	// module name
	ModuleName = "erc721"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for message routing
	RouterKey = ModuleName
)

// ModuleAddress is the native module address for EVM
var ModuleAddress common.Address

func init() {
	ModuleAddress = common.BytesToAddress(authtypes.NewModuleAddress(ModuleName).Bytes())
}

// prefix bytes for the EVM persistent store
const (
	prefixTokenPair = iota + 1
	prefixTokenPairByERC721
	prefixTokenPairByClass
	prefixNFTPairByNFTID
	prefixNFTPairByTokenID
)

// KVStore key prefixes
var (
	KeyPrefixTokenPair         = []byte{prefixTokenPair}
	KeyPrefixTokenPairByERC721 = []byte{prefixTokenPairByERC721}
	KeyPrefixTokenPairByClass  = []byte{prefixTokenPairByClass}
	KeyPrefixNFTPairByNFTID    = []byte{prefixNFTPairByNFTID}
	KeyPrefixNFTPairByTokenID  = []byte{prefixNFTPairByTokenID}
)
