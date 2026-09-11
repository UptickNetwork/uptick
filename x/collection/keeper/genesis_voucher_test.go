package keeper

import (
	"cosmossdk.io/x/nft"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// Regression for the P0 export/init asymmetry (第 22 轮审查):
//
// An ICS-721 voucher class carries no metadata blob, so the export produced it
// with an EMPTY creator. ValidateGenesis accepted that, but InitGenesis called
// sdk.AccAddressFromBech32("") and panicked -- meaning a chain that had ever
// received a cross-chain NFT could export a genesis and then fail to start
// from it. The three layers must agree:
//
//   - export keeps the class (a voucher class is not corruption),
//   - validation accepts an empty creator and rejects a malformed one,
//   - InitGenesis imports the class instead of panicking on it.
//
// This test walks the whole chain: seed a voucher-shaped class in the
// underlying nft store (exactly what nft-transfer's OnRecvPacket leaves
// behind), export, validate, and import again.
func (s *KeeperTestSuite) TestVoucherClassExportValidateImportRoundTrip() {
	const classID = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"

	owner := sdk.AccAddress([]byte("voucher-owner"))
	s.Require().NoError(s.nftKpr.SaveClass(s.ctx, nft.Class{
		Id:     classID,
		Name:   "Voucher",
		Symbol: "V",
	}))
	s.Require().NoError(s.nftKpr.Mint(s.ctx, nft.NFT{
		ClassId: classID,
		Id:      "1",
		Uri:     "ipfs://voucher-1",
	}, owner))

	// The degradation is reported, not silent: a class with no metadata blob is
	// listed so an operator can tell the export is not a byte-for-byte copy.
	issues := s.keeper.ExportIssues(s.ctx)
	s.Require().Len(issues, 1)
	s.Require().Equal(classID, issues[0].ClassID)
	s.Require().Equal(ExportIssueClassMetadataMissing, issues[0].Kind)

	// Export keeps the class, with the empty creator it actually has on chain.
	exported := s.keeper.ExportGenesis(s.ctx)
	s.Require().Len(exported.Collections, 1)
	s.Require().Equal(classID, exported.Collections[0].Denom.Id)
	s.Require().Empty(exported.Collections[0].Denom.Creator)
	s.Require().Len(exported.Collections[0].NFTs, 1)
	s.Require().Equal("1", exported.Collections[0].NFTs[0].GetID())

	// Validation must accept what the export produced...
	s.Require().NoError(types.ValidateGenesis(*exported))

	// ...and the importer must accept it too, without panicking. A node
	// starting from genesis imports into an EMPTY store, so rebuild the
	// fixture instead of re-importing on top of the live data.
	s.SetupTest()

	s.Require().NotPanics(func() {
		s.keeper.InitGenesis(s.ctx, *exported)
	})

	// The class and its NFT survive a full round trip.
	denom, err := s.keeper.GetDenomInfo(s.ctx, classID)
	s.Require().NoError(err)
	s.Require().Equal(classID, denom.Id)
	s.Require().Empty(denom.Creator)

	got, err := s.keeper.GetNFT(s.ctx, classID, "1")
	s.Require().NoError(err)
	s.Require().Equal("1", got.GetID())
	s.Require().Equal(owner.String(), got.GetOwner().String())

	// A second export no longer reports the class: the import wrote a (empty)
	// metadata blob, so the round trip has converged to a representable state.
	s.Require().Empty(s.keeper.ExportIssues(s.ctx))
}

// A malformed NON-empty creator is real corruption and must keep aborting the
// import -- loosening the empty case must not loosen this one.
func (s *KeeperTestSuite) TestInitGenesisRejectsMalformedCreator() {
	gs := types.NewGenesisState([]types.Collection{{
		Denom: types.Denom{
			Id:      "broken",
			Name:    "Broken",
			Creator: "not-a-bech32-address",
		},
	}})

	s.Require().Error(types.ValidateGenesis(*gs))
	s.Require().Panics(func() {
		s.keeper.InitGenesis(s.ctx, *gs)
	})
}
