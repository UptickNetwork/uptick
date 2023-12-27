package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// cw721 events
const (
	EventTypeTokenLock             = "token_lock"
	EventTypeTokenUnlock           = "token_unlock"
	EventTypeMint                  = "mint"
	EventTypeConvertNFT            = "convert_nft"
	EventTypeConvertCW721          = "convert_cw721"
	EventTypeBurn                  = "burn"
	EventTypeRegisterNFT           = "register_nft"
	EventTypeRegisterCW721         = "register_cw721"
	EventTypeToggleTokenConversion = "toggle_token_conversion" // #nosec

	AttributeKeyNFTClass     = "nft_class"
	AttributeKeyNFTID        = "nft_ids"
	AttributeKeyCW721Token   = "cw721_token"     // #nosec
	AttributeKeyCW721TokenID = "cw721_token_ids" // #nosec
	AttributeKeyReceiver     = "receiver"

	CW721EventTransfer = "Transfer"
)

// LogTransfer type for Transfer(address from, address to, string tokenID)
type LogTransfer struct {
	From    common.Address
	To      common.Address
	TokenID string
}
