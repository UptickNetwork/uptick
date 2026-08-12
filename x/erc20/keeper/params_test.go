package keeper_test

func (suite *KeeperTestSuite) TestGetParams() {
	params := suite.app.Erc20Keeper.GetParams(suite.ctx)
	suite.Require().True(params.EnableErc20)
	suite.Require().True(params.EnableEVMHook)
}

func (suite *KeeperTestSuite) TestSetParams() {
	params := suite.app.Erc20Keeper.GetParams(suite.ctx)

	// Disable ERC20
	params.EnableErc20 = false
	suite.app.Erc20Keeper.SetParams(suite.ctx, params)

	// Verify update persisted
	updatedParams := suite.app.Erc20Keeper.GetParams(suite.ctx)
	suite.Require().False(updatedParams.EnableErc20)

	// Re-enable
	params.EnableErc20 = true
	params.EnableEVMHook = true
	suite.app.Erc20Keeper.SetParams(suite.ctx, params)

	restoredParams := suite.app.Erc20Keeper.GetParams(suite.ctx)
	suite.Require().True(restoredParams.EnableErc20)
	suite.Require().True(restoredParams.EnableEVMHook)
}
