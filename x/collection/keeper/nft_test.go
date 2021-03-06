package keeper_test

func (suite *KeeperSuite) TestGetNFT() {
	// MintNFT shouldn't fail when collection does not exist
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	// GetNFT should get the NFT
	receivedNFT, err := suite.app.CollectionKeeper.GetNFT(suite.ctx, denomID, tokenID)
	suite.NoError(err)
	suite.Equal(receivedNFT.GetID(), tokenID)
	suite.True(receivedNFT.GetOwner().Equals(address))
	suite.Equal(receivedNFT.GetURI(), tokenURI)

	// MintNFT shouldn't fail when collection exists
	err = suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID2, tokenNm2, tokenURI, tokenData, address, address)
	suite.NoError(err)

	// GetNFT should get the NFT when collection exists
	receivedNFT2, err := suite.app.CollectionKeeper.GetNFT(suite.ctx, denomID, tokenID2)
	suite.NoError(err)
	suite.Equal(receivedNFT2.GetID(), tokenID2)
	suite.True(receivedNFT2.GetOwner().Equals(address))
	suite.Equal(receivedNFT2.GetURI(), tokenURI)

	//msg, fail := keeper.SupplyInvariant(suite.keeper)(suite.ctx)
	//suite.False(fail, msg)
}

func (suite *KeeperSuite) TestGetNFTs() {
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID2, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	err = suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID2, tokenID2, tokenNm2, tokenURI, tokenData, address, address)
	suite.NoError(err)

	err = suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID2, tokenID3, tokenNm3, tokenURI, tokenData, address, address)
	suite.NoError(err)

	err = suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID3, tokenNm3, tokenURI, tokenData, address, address)
	suite.NoError(err)

	nfts, err := suite.app.CollectionKeeper.GetNFTs(suite.ctx, denomID2)
	suite.NoError(err)
	suite.Len(nfts, 3)
}

func (suite *KeeperSuite) TestAuthorize() {
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	err = suite.app.CollectionKeeper.Authorize(suite.ctx, denomID, tokenID, address2)
	suite.Error(err)

	err = suite.app.CollectionKeeper.Authorize(suite.ctx, denomID, tokenID, address)
	suite.NoError(err)
}

func (suite *KeeperSuite) TestHasNFT() {
	// IsNFT should return false
	isNFT := suite.app.CollectionKeeper.HasNFT(suite.ctx, denomID, tokenID)
	suite.False(isNFT)

	// MintNFT shouldn't fail when collection does not exist
	err := suite.app.CollectionKeeper.MintNFT(suite.ctx, denomID, tokenID, tokenNm, tokenURI, tokenData, address, address)
	suite.NoError(err)

	// IsNFT should return true
	isNFT = suite.app.CollectionKeeper.HasNFT(suite.ctx, denomID, tokenID)
	suite.True(isNFT)
}
