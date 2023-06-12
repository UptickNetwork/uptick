package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// erc721 events
const (
	EventTypeTokenLock             = "token_lock"
	EventTypeTokenUnlock           = "token_unlock"
	EventTypeMint                  = "mint"
	EventTypeConvertNFT            = "convert_nft"
	EventTypeConvertERC721         = "convert_erc721"
	EventTypeBurn                  = "burn"
	EventTypeRegisterNFT           = "register_nft"
	EventTypeRegisterERC721        = "register_erc721"
	EventTypeToggleTokenConversion = "toggle_token_conversion" // #nosec

	AttributeKeyNFTClass      = "nft_class"
	AttributeKeyNFTID         = "nft_ids"
	AttributeKeyERC721Token   = "erc721_token"     // #nosec
	AttributeKeyERC721TokenID = "erc721_token_ids" // #nosec
	AttributeKeyReceiver      = "receiver"

	ERC721EventTransfer = "Transfer"
)

// LogTransfer type for Transfer(address from, address to, string tokenID)
type LogTransfer struct {
	From    common.Address
	To      common.Address
	TokenID string
}
