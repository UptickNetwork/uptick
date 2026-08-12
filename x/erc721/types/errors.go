package types

import (
	sdkerrors "cosmossdk.io/errors"
)

// errors
var (
	ErrERC721Disabled            = sdkerrors.Register(ModuleName, 2, "erc721 module is disabled")
	ErrClassNotExist             = sdkerrors.Register(ModuleName, 3, "nft class not exist")
	ErrNFTNotExist               = sdkerrors.Register(ModuleName, 4, "nft not exist")
	ErrInternalTokenPair         = sdkerrors.Register(ModuleName, 5, "internal nft token mapping error")
	ErrTokenPairNotFound         = sdkerrors.Register(ModuleName, 6, "token pair not found")
	ErrTokenPairAlreadyExists    = sdkerrors.Register(ModuleName, 7, "token pair already exists")
	ErrUndefinedOwner            = sdkerrors.Register(ModuleName, 8, "undefined owner of contract pair")
	ErrUnexpectedEvent           = sdkerrors.Register(ModuleName, 9, "unexpected event")
	ErrABIPack                   = sdkerrors.Register(ModuleName, 10, "contract ABI pack failed")
	ErrABIUnpack                 = sdkerrors.Register(ModuleName, 11, "contract ABI unpack failed")
	ErrEVMCall                   = sdkerrors.Register(ModuleName, 12, "EVM call unexpected error")
	ErrERC721TokenPairDisabled   = sdkerrors.Register(ModuleName, 13, "erc721 token pair is disabled")
	ErrNftIdNotCorrect           = sdkerrors.Register(ModuleName, 14, "nft id is not correct")
	ErrClassIdNotCorrect         = sdkerrors.Register(ModuleName, 15, "nft class is not correct")
	ErrContractAddressNotCorrect = sdkerrors.Register(ModuleName, 16, "contract address is not correct")
	ErrTokenIdNotCorrect         = sdkerrors.Register(ModuleName, 17, "token id is not correct")
)
