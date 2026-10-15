package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestFundRewardPool(t *testing.T) {
	e := setupTest(t)
	denom := types.DefaultChallengeBondDenom
	e.bank.fundAccount(e.auth, denom, math.NewInt(5000))

	modAddr := types.AccModuleAddress
	modBefore := e.bank.GetBalance(e.ctx, modAddr, denom)

	_, err := e.server.FundRewardPool(e.ctx, &types.MsgFundRewardPool{
		Authority: e.auth.String(),
		Denom:     denom,
		Amount:    "1000",
	})
	require.NoError(t, err)

	modAfter := e.bank.GetBalance(e.ctx, modAddr, denom)
	require.Equal(t, modBefore.Amount.Add(math.NewInt(1000)), modAfter.Amount)
	authAfter := e.bank.GetBalance(e.ctx, e.auth, denom)
	require.Equal(t, math.NewInt(4000), authAfter.Amount)
}

func TestFundRewardPoolUnauthorized(t *testing.T) {
	e := setupTest(t)
	e.bank.fundAccount(e.owner, types.DefaultChallengeBondDenom, math.NewInt(1000))
	_, err := e.server.FundRewardPool(e.ctx, &types.MsgFundRewardPool{
		Authority: e.owner.String(),
		Denom:     types.DefaultChallengeBondDenom,
		Amount:    "100",
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)
}

func TestFundRewardPoolDenomNotAllowlisted(t *testing.T) {
	e := setupTest(t)
	e.bank.fundAccount(e.auth, "ucoin", math.NewInt(1000))
	_, err := e.server.FundRewardPool(e.ctx, &types.MsgFundRewardPool{
		Authority: e.auth.String(),
		Denom:     "ucoin",
		Amount:    "100",
	})
	require.ErrorIs(t, err, types.ErrInvalidDenom)
}

func TestFundRewardPoolZeroRejected(t *testing.T) {
	e := setupTest(t)
	msg := &types.MsgFundRewardPool{
		Authority: e.auth.String(),
		Denom:     types.DefaultChallengeBondDenom,
		Amount:    "0",
	}
	require.Error(t, msg.ValidateBasic())
}
