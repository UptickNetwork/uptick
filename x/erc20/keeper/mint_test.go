package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/UptickNetwork/uptick/x/erc20/types"
)

func (suite *KeeperTestSuite) TestMintingEnabled() {
	coinMeta := banktypes.Metadata{
		Name:    "test",
		Symbol:  "TST",
		Base:    "utest",
		Display: "test",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "utest", Exponent: 0},
			{Denom: "test", Exponent: 18},
		},
	}

	pair := suite.setupRegisterCoin(coinMeta)
	suite.Require().NotNil(pair)
	suite.Require().True(pair.Enabled)

	sender := sdk.AccAddress(suite.address.Bytes())
	receiver := sdk.AccAddress(suite.address.Bytes())

	// Test valid minting check
	resultPair, err := suite.app.Erc20Keeper.MintingEnabled(suite.ctx, sender, receiver, pair.Denom)
	suite.Require().NoError(err)
	suite.Require().Equal(pair.Denom, resultPair.Denom)
	suite.Require().True(resultPair.Enabled)
}

func (suite *KeeperTestSuite) TestMintingEnabledInvalidDenom() {
	sender := sdk.AccAddress(suite.address.Bytes())
	receiver := sdk.AccAddress(suite.address.Bytes())

	_, err := suite.app.Erc20Keeper.MintingEnabled(suite.ctx, sender, receiver, "nonexistent")
	suite.Require().Error(err)
}

func (suite *KeeperTestSuite) TestMintingEnabledUnregisteredCoin() {
	sender := sdk.AccAddress(suite.address.Bytes())
	receiver := sdk.AccAddress(suite.address.Bytes())

	// Get all pairs and find a non-existing one
	pairs := suite.app.Erc20Keeper.GetAllTokenPairs(suite.ctx)
	suite.Require().Empty(pairs)

	_, err := suite.app.Erc20Keeper.MintingEnabled(suite.ctx, sender, receiver, types.CreateDenom("utest"))
	suite.Require().Error(err)
}
