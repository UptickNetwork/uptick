package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// fakeConvertedNFTs stands in for the app-injected cross-module checker.
type fakeConvertedNFTs map[string]bool

func (f fakeConvertedNFTs) IsConvertedNFT(_ sdk.Context, classID, nftID string) bool {
	return f[classID+"/"+nftID]
}

// Burning the native half of a converted NFT leaves the contract half escrowed
// in a module account with no on-chain record of it -- x/erc721's own comment
// calls that state "undiscoverable and unrecoverable". RemoveNFT must refuse it
// instead, and tell the user to un-wrap first.
func (s *KeeperTestSuite) TestRemoveNFTRefusesConvertedToken() {
	creator := sdk.AccAddress([]byte("creator-bound-01"))
	owner := sdk.AccAddress([]byte("owner-bound-0001"))
	s.Require().NoError(s.keeper.SaveDenom(s.ctx, "bound", "Bound", "", "B", creator, false, false, "", "", "", ""))
	s.Require().NoError(s.keeper.SaveNFT(s.ctx, "bound", "nft1", "One", "ipfs://one", "", "", owner))
	s.Require().NoError(s.keeper.SaveNFT(s.ctx, "bound", "nft2", "Two", "ipfs://two", "", "", owner))

	// A module-only setup has no contract side, so the guard stays inert and
	// burning works exactly as before.
	s.Require().False(s.keeper.IsConvertedNFT(s.ctx, "bound", "nft1"))
	s.Require().NoError(s.keeper.RemoveNFT(s.ctx, "bound", "nft1", owner))
	s.Require().False(s.keeper.NFTkeeper().HasNFT(s.ctx, "bound", "nft1"))

	// Once the checker reports a contract binding, the burn is refused.
	s.keeper.SetConvertedNFTChecker(fakeConvertedNFTs{"bound/nft2": true})

	err := s.keeper.RemoveNFT(s.ctx, "bound", "nft2", owner)
	s.Require().ErrorIs(err, types.ErrNFTBoundToContract)
	s.Require().True(s.keeper.NFTkeeper().HasNFT(s.ctx, "bound", "nft2"),
		"the native NFT must survive a refused burn, so the pair stays repairable")

	// Unrelated tokens of the same class are still burnable.
	s.Require().NoError(s.keeper.SaveNFT(s.ctx, "bound", "nft3", "Three", "ipfs://three", "", "", owner))
	s.Require().NoError(s.keeper.RemoveNFT(s.ctx, "bound", "nft3", owner))
}

// The guard must not become a way to bypass authorisation: a caller who does
// not own the token still gets ErrUnauthorized, and a bound token owned by
// someone else is never burned by a stranger.
func (s *KeeperTestSuite) TestRemoveNFTKeepsAuthorizationBeforeConversionCheck() {
	creator := sdk.AccAddress([]byte("creator-auth-001"))
	owner := sdk.AccAddress([]byte("owner-auth-00001"))
	stranger := sdk.AccAddress([]byte("stranger-auth-01"))
	s.Require().NoError(s.keeper.SaveDenom(s.ctx, "authd", "Authd", "", "A", creator, false, false, "", "", "", ""))
	s.Require().NoError(s.keeper.SaveNFT(s.ctx, "authd", "nft1", "One", "ipfs://one", "", "", owner))

	s.keeper.SetConvertedNFTChecker(fakeConvertedNFTs{"authd/nft1": true})

	err := s.keeper.RemoveNFT(s.ctx, "authd", "nft1", stranger)
	s.Require().ErrorIs(err, types.ErrUnauthorized)
	s.Require().True(s.keeper.NFTkeeper().HasNFT(s.ctx, "authd", "nft1"))
}
