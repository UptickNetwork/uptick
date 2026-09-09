package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// TestConvertCW721_RejectsEmptyTokenIds pins the length-mismatch guard at
// msg_server.go:107 — an empty TokenIds slice must be rejected before the
// conversion loop touches any array index. (NftIds may be rewritten by the
// upstream GetClassIDAndNFTID query, so this guards the user-supplied
// TokenIds length directly.)
func TestConvertCW721_RejectsEmptyTokenIds(t *testing.T) {
	k, ctx, owner, contract, _ := setupConvertKeeper(t)
	_, err := k.ConvertCW721(ctx, &types.MsgConvertCW721{
		ContractAddress: contract,
		TokenIds:        []string{},
		NftIds:          []string{"nft1"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
		ClassId:         "kitty",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "length mismatch")
}

// TestConvertCW721_RejectsMismatchedArrays pins that a TokenIds/NftIds length
// mismatch is rejected (either at the explicit 107 guard or earlier during
// GetClassIDAndNFTID) — never a panic / out-of-range access.
func TestConvertCW721_RejectsMismatchedArrays(t *testing.T) {
	k, ctx, owner, contract, _ := setupConvertKeeper(t)
	_, err := k.ConvertCW721(ctx, &types.MsgConvertCW721{
		ContractAddress: contract,
		TokenIds:        []string{"1", "2"},
		NftIds:          []string{"nft1"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
		ClassId:         "kitty",
	})
	require.Error(t, err, "mismatched arrays must be rejected, not panic")
}

// TestConvertNFT_RejectsNonExistentClass pins that converting against a class
// that was never registered yields an error rather than a panic or a silently
// orphan contract.
func TestConvertNFT_RejectsNonExistentClass(t *testing.T) {
	k, ctx, owner, contract, _ := setupConvertKeeper(t)
	_, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:         "nope-not-registered",
		NftIds:          []string{"nft1"},
		ContractAddress: contract,
		TokenIds:        []string{"1"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
	})
	require.Error(t, err, "non-existent class must be rejected")
}

// TestConvertNFT_RejectsMismatchedArrays pins the ConvertNFT length-mismatch
// guard (convertCosmos2Wasm:315) — or an earlier rejection — never a panic.
func TestConvertNFT_RejectsMismatchedArrays(t *testing.T) {
	k, ctx, owner, contract, _ := setupConvertKeeper(t)
	_, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:         "kitty",
		NftIds:          []string{"nft1", "nft2"},
		ContractAddress: contract,
		TokenIds:        []string{"1"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
	})
	require.Error(t, err, "mismatched arrays must be rejected, not panic")
}

// TestConvertNFT_RejectsEmptyNftIds pins that an empty NftIds batch is
// rejected up front (the batch-size / mismatch checks) before any contract
// deployment, avoiding an orphan contract.
func TestConvertNFT_RejectsEmptyNftIds(t *testing.T) {
	k, ctx, owner, contract, _ := setupConvertKeeper(t)
	_, err := k.ConvertNFT(ctx, &types.MsgConvertNFT{
		ClassId:         "kitty",
		NftIds:          []string{},
		ContractAddress: contract,
		TokenIds:        []string{"1"},
		Sender:          owner.String(),
		Receiver:        owner.String(),
	})
	require.Error(t, err, "empty NftIds must be rejected")
}
