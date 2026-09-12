package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ConvertedNFTChecker reports whether a native NFT is paired with a token held
// by an ERC721 or CW721 contract.
//
// x/collection owns the native half of a conversion but not the contract-side
// bookkeeping, and it must not import x/erc721 or x/cw721 -- both of them
// depend on collection, so importing back would invert the module dependency.
// The check is therefore an interface, supplied by the app at wiring time.
type ConvertedNFTChecker interface {
	// IsConvertedNFT reports whether (classID, nftID) has a contract-side
	// counterpart, i.e. whether burning the native NFT would strand a token
	// that the conversion left escrowed in a module account.
	IsConvertedNFT(ctx sdk.Context, classID, nftID string) bool
}

// convertedNFTCheckerSlot holds the cross-module checker behind a pointer so
// that the wiring performed after this keeper has been copied into x/erc721,
// x/cw721 and x/internft stays visible on every copy. See the field comment on
// Keeper.convertedNFTs for why the indirection is required.
type convertedNFTCheckerSlot struct {
	checker ConvertedNFTChecker
}

// IsConvertedNFT reports whether the NFT is bound to a contract token,
// consulting the wiring-time checker. A nil slot or an unwired checker (a
// module-only setup with no contract side, such as the collection module's own
// unit tests) reports false, which preserves the unguarded behavior those
// callers expect.
func (k Keeper) IsConvertedNFT(ctx sdk.Context, classID, nftID string) bool {
	if k.convertedNFTs == nil || k.convertedNFTs.checker == nil {
		return false
	}
	return k.convertedNFTs.checker.IsConvertedNFT(ctx, classID, nftID)
}

// SetConvertedNFTChecker wires the cross-module burn guard.
//
// It is called by the application once x/erc721 and x/cw721 exist -- they are
// constructed *from* this keeper (their NewKeeper takes it by value), so the
// dependency cannot be a constructor parameter. app/keepers/keepers.go
// performs the wiring and TestConvertedNFTCheckerIsWired fails if it is ever
// dropped, because a silently unwired checker turns the burn guard back into
// the silent asset loss it exists to prevent.
//
// The write lands in the shared slot, so it is visible through every copy of
// this keeper taken before the call -- which is what makes it reach the refund
// path in x/erc721 (whose nftKeeper field is such a copy) and the ICS-721 path
// in x/internft.
func (k *Keeper) SetConvertedNFTChecker(checker ConvertedNFTChecker) {
	if k.convertedNFTs == nil {
		k.convertedNFTs = &convertedNFTCheckerSlot{}
	}
	k.convertedNFTs.checker = checker
}
