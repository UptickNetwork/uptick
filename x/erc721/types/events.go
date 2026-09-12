package types

// erc721 events.
//
// This list used to mirror Evmos x/erc20's event skeleton; every inherited
// identifier was unused here, because registration and locking are done by
// ConvertNFT / ConvertERC721 (which emit convert_nft / convert_erc721
// themselves), cross-chain escrow and release belong to nft-transfer (the
// module only emits refund_packet_token(_skip)), and transfer logs are decoded
// with go-ethereum's abi in the keeper.
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
	// states: erc721 checks HasNFT before its EVM owner query, so it cannot yet
	// know whether an escrowed ERC721 is still held and must keep the mapping
	// for triage, whereas cw721 cleans up only after the refund side is settled
	// and so converges. Without this attribute an operator cannot tell, from the
	// reason alone, whether a `nft_already_gone` alert needs action.
	//
	// Only erc721 sets it; absence means false, which is why x/cw721 does not
	// need a constant of its own.
	AttributeKeyPairsRetained = "pairs_retained"
)
