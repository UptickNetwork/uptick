package keeper_test

import (
	gocontext "context"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

func (suite *KeeperSuite) TestSupply() {
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	response, err := suite.queryClient.Supply(
		gocontext.Background(),
		&types.QuerySupplyRequest{
			DenomId: denomID,
			Owner:   address.String(),
		},
	)

	suite.NoError(err)
	suite.Equal(1, int(response.Amount))
}

func (suite *KeeperSuite) TestOwner() {
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	response, err := suite.queryClient.NFTsOfOwner(
		gocontext.Background(),
		&types.QueryNFTsOfOwnerRequest{
			DenomId: denomID,
			Owner:   address.String(),
		},
	)

	suite.NoError(err)
	suite.NotNil(response.Owner)
	suite.Contains(response.Owner.IDCollections[0].TokenIDs, tokenID)
}

func (suite *KeeperSuite) TestCollection() {
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	response, err := suite.queryClient.Collection(
		gocontext.Background(),
		&types.QueryCollectionRequest{
			DenomId: denomID,
		},
	)

	suite.NoError(err)
	suite.NotNil(response.Collection)
	suite.Len(response.Collection.NFTs, 1)
	suite.Equal(response.Collection.NFTs[0].ID, tokenID)
}

func (suite *KeeperSuite) TestDenom() {
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	response, err := suite.queryClient.Denom(
		gocontext.Background(),
		&types.QueryDenomRequest{
			DenomId: denomID,
		},
	)

	suite.NoError(err)
	suite.NotNil(response.Denom)
	suite.Equal(response.Denom.ID, denomID)
}

func (suite *KeeperSuite) TestDenoms() {
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	response, err := suite.queryClient.Denoms(
		gocontext.Background(),
		&types.QueryDenomsRequest{},
	)

	suite.NoError(err)
	suite.NotEmpty(response.Denoms)
	suite.Equal(response.Denoms[0].ID, denomID)
}

func (suite *KeeperSuite) TestNFT() {
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	response, err := suite.queryClient.NFT(
		gocontext.Background(),
		&types.QueryNFTRequest{
			DenomId: denomID,
			TokenId: tokenID,
		},
	)

	suite.NoError(err)
	suite.NotEmpty(response.NFT)
	suite.Equal(response.NFT.ID, tokenID)
}
