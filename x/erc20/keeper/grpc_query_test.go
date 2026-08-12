package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

func (suite *KeeperTestSuite) TestTokenPairs() {
	_, _ = suite.setupRegisterCoin()

	// Query all token pairs
	res, err := suite.queryClient.TokenPairs(suite.ctx, &types.QueryTokenPairsRequest{
		Pagination: &query.PageRequest{Limit: 10},
	})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(res.TokenPairs)
}

func (suite *KeeperTestSuite) TestTokenPair() {
	_, pair := suite.setupRegisterCoin()

	// Query by denom
	res, err := suite.queryClient.TokenPair(suite.ctx, &types.QueryTokenPairRequest{
		Token: pair.Denom,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(pair.Denom, res.TokenPair.Denom)
}

func (suite *KeeperTestSuite) TestParams() {
	res, err := suite.queryClient.Params(suite.ctx, &types.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().True(res.Params.EnableErc20)
	suite.Require().True(res.Params.EnableEVMHook)
}
