package types

// erc721 events.
//
// This list used to mirror Evmos x/erc20's event skeleton, including
// token_lock, token_unlock, mint, burn, register_nft, register_erc721 and
// toggle_token_conversion, along with an ERC721EventTransfer constant and a
// hand-rolled LogTransfer decoder struct. None of them was ever referenced
// here:
//
//   - Registration and locking in this module are performed by ConvertNFT /
//     ConvertERC721, which emit convert_nft / convert_erc721 themselves.
//   - Cross-chain escrow and release are owned by nft-transfer, not by these
//     events; the module only emits refund_packet_token(_skip).
//   - Transfer logs are decoded with go-ethereum's abi in the keeper, so
//     LogTransfer had no caller.
//
// Unused exported identifiers are not free in a module that ships as a public
// Go package: they read as supported surface and invite new code to adopt a
// shape that nothing exercises or maintains.
const (
	EventTypeConvertNFT            = "convert_nft"
	EventTypeConvertERC721         = "convert_erc721"
	EventTypeRefundPacketToken     = "refund_packet_token"
	EventTypeRefundPacketTokenSkip = "refund_packet_token_skip"

	AttributeKeyNFTClass      = "nft_class"
	AttributeKeyNFTID         = "nft_ids"
	AttributeKeyNFTOwner      = "nft_owner"
	AttributeKeyERC721Token   = "erc721_token"     // #nosec
	AttributeKeyERC721TokenID = "erc721_token_ids" // #nosec
	AttributeKeyReceiver      = "receiver"

	// AttributeKeyPairsRetained marks the skip events whose pair mappings were
	// deliberately kept. It exists because this module and x/cw721 emit the
	// same event type with the same reason `nft_already_gone` for opposite
	// situations: erc721 checks HasNFT before its EVM owner query, so it cannot
	// yet know whether an escrowed ERC721 is still held and must keep the
	// mapping for triage, while cw721 cleans up only after the refund side is
	// settled and so converges. Without this attribute an operator has to know
	// that asymmetry to decide whether a `nft_already_gone` alert needs action.
	//
	// Only erc721 sets it; absence means false, which is why x/cw721 does not
	// need a constant of its own.
	AttributeKeyPairsRetained = "pairs_retained"
)
