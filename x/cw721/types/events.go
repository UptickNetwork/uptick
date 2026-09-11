package types

// cw721 events.
//
// This list used to mirror Evmos x/erc20's event skeleton, including
// token_lock, token_unlock, mint, burn, register_nft, register_cw721,
// toggle_token_conversion and even convert_nft — the last one because the
// skeleton was copied before this module settled on convert_cw721 for its own
// conversion event. Alongside them sat a CW721EventTransfer constant and a
// hand-rolled LogTransfer decoder struct. None of them was ever referenced
// here:
//
//   - Conversion emits convert_cw721; the contract call itself is what
//     registers the token, so register_* had no emitter.
//   - Cross-chain escrow and release are owned by nft-transfer, not by these
//     events; the module only emits refund_packet_token(_skip).
//   - Transfer logs are decoded with go-ethereum's abi in the keeper, so
//     LogTransfer had no caller.
//
// Unused exported identifiers are not free in a module that ships as a public
// Go package: they read as supported surface and invite new code to adopt a
// shape that nothing exercises or maintains.
const (
	EventTypeConvertCW721          = "convert_cw721"
	EventTypeRefundPacketToken     = "refund_packet_token"
	EventTypeRefundPacketTokenSkip = "refund_packet_token_skip"

	AttributeKeyNFTClass     = "nft_class"
	AttributeKeyNFTID        = "nft_ids"
	AttributeKeyNFTOwner     = "nft_owner"
	AttributeKeyCW721Token   = "cw721_token"     // #nosec
	AttributeKeyCW721TokenID = "cw721_token_ids" // #nosec
	AttributeKeyReceiver     = "receiver"
)
