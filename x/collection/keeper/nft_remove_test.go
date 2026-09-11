package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// Burning a token that does not exist has to say so. Authorize compares against
// GetOwner, which reports an empty address for a missing token, so the previous
// check order surfaced this as ErrUnauthorized against a blank address --
// misleading for callers that branch on ErrUnknownNFT, which is what UpdateNFT
// and TransferOwnership already report for the same situation.
func (s *KeeperTestSuite) TestRemoveNFTUnknownTokenReportsUnknownNFT() {
	creator := sdk.AccAddress([]byte("creator-unknown-1"))
	owner := sdk.AccAddress([]byte("owner-unknown-001"))
	stranger := sdk.AccAddress([]byte("stranger-unknown1"))

	s.Require().NoError(s.keeper.SaveDenom(s.ctx, "unk", "Unk", "", "U", creator, false, false, "", "", "", ""))
	s.Require().NoError(s.keeper.SaveNFT(s.ctx, "unk", "nft1", "One", "ipfs://one", "", "", owner))

	// Missing token: ErrUnknownNFT regardless of who asks, because existence is
	// settled before authorization.
	for _, addr := range []sdk.AccAddress{owner, stranger} {
		err := s.keeper.RemoveNFT(s.ctx, "unk", "nosuch", addr)
		s.Require().ErrorIs(err, types.ErrUnknownNFT,
			"a missing token must be reported as unknown, not as an authorization failure")
	}

	// A token that exists but belongs to someone else is still ErrUnauthorized.
	// Reordering the checks must not turn authorization failures into not-found
	// (which would also leak that the token does exist).
	err := s.keeper.RemoveNFT(s.ctx, "unk", "nft1", stranger)
	s.Require().ErrorIs(err, types.ErrUnauthorized)
	s.Require().True(s.keeper.NFTkeeper().HasNFT(s.ctx, "unk", "nft1"),
		"a refused burn must not remove the token")

	// And the owner can still burn it.
	s.Require().NoError(s.keeper.RemoveNFT(s.ctx, "unk", "nft1", owner))
	s.Require().False(s.keeper.NFTkeeper().HasNFT(s.ctx, "unk", "nft1"))
}
