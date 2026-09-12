package types

// cw721 events.
//
// This list used to mirror Evmos x/erc20's event skeleton; every inherited
// identifier was unused here: conversion emits convert_cw721 (the contract call
// itself registers the token, so register_* had no emitter), cross-chain escrow
// and release belong to nft-transfer (the module only emits
// refund_packet_token(_skip)), and transfers are carried by CosmWasm
// execute/query messages (keeper/wasm_adapter.go) rather than decoded from EVM
// logs, so a hand-rolled LogTransfer had no caller. The inherited convert_nft
// was never emitted either: the skeleton predates this module settling on
// convert_cw721 as its conversion event.
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
