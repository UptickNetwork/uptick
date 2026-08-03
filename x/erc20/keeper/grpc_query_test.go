package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

func (suite *KeeperTestSuite) TestTokenPairs() {
	coinMeta := banktypes.Metadata{
		Name:    "test-query",
		Symbol:  "TQ",
		Base:    "utestq",
		Display: "testq",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "utestq", Exponent: 0},
			{Denom: "testq", Exponent: 18},
		},
	}

	_ = suite.setupRegisterCoin(coinMeta)

	// Query all token pairs
	res, err := suite.queryClient.TokenPairs(suite.ctx, &types.QueryTokenPairsRequest{
		Pagination: &query.PageRequest{Limit: 10},
	})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(res.TokenPairs)
}

func (suite *KeeperTestSuite) TestTokenPair() {
	coinMeta := banktypes.Metadata{
		Name:    "test-pair",
		Symbol:  "TP",
		Base:    "utestp",
		Display: "testp",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "utestp", Exponent: 0},
			{Denom: "testp", Exponent: 18},
		},
	}

	pair := suite.setupRegisterCoin(coinMeta)

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
