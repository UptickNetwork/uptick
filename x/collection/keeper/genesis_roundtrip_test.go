package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/exported"
	"github.com/UptickNetwork/uptick/x/collection/types"
)

func (s *KeeperTestSuite) TestInitExportGenesisRoundTrip() {
	creator := sdk.AccAddress([]byte("creator"))
	owner := sdk.AccAddress([]byte("owner"))

	denom := types.Denom{
		Id:               "denom1",
		Name:             "Denom One",
		Symbol:           "ONE",
		Schema:           "",
		Creator:          creator.String(),
		MintRestricted:   true,
		UpdateRestricted: false,
		Description:      "test denom",
		Uri:              "ipfs://class",
		UriHash:          "",
		Data:             "",
	}
	nft := types.NewBaseNFT("nft1", "NFT One", owner, "ipfs://nft", "", "")
	collection := types.NewCollection(denom, []exported.NFT{nft})

	s.keeper.InitGenesis(s.ctx, *types.NewGenesisState([]types.Collection{collection}))
	exportedState := s.keeper.ExportGenesis(s.ctx)

	s.Require().Len(exportedState.Collections, 1)
	s.Require().Equal("denom1", exportedState.Collections[0].Denom.Id)
	s.Require().Len(exportedState.Collections[0].NFTs, 1)
	s.Require().Equal("nft1", exportedState.Collections[0].NFTs[0].GetID())
	s.Require().Equal(owner.String(), exportedState.Collections[0].NFTs[0].GetOwner().String())

	got, err := s.keeper.GetNFT(s.ctx, "denom1", "nft1")
	s.Require().NoError(err)
	s.Require().Equal("nft1", got.GetID())
	s.Require().Equal(owner.String(), got.GetOwner().String())
}
