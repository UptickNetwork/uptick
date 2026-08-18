package ante

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestValidatorCommissionDecoratorValidateMsg(t *testing.T) {
	dec := ValidatorCommissionDecorator{}

	t.Run("involves staking msg", func(t *testing.T) {
		// MsgDelegate involves staking, should return true
		msg := stakingtypes.NewMsgDelegate("cosmos1addr", "cosmosvaloper1validator", sdk.NewInt64Coin("stake", 1000))
		require.True(t, dec.involvesStakingMsg(msg))
	})

	t.Run("staking CreateValidator", func(t *testing.T) {
		msg, err := stakingtypes.NewMsgCreateValidator(
			"cosmosvaloper1validator",
			ed25519.GenPrivKey().PubKey(),
			sdk.NewCoin("stake", math.NewInt(1000)),
			stakingtypes.Description{Moniker: "test"},
			stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
			math.NewInt(1),
		)
		require.NoError(t, err)
		require.True(t, dec.involvesStakingMsg(msg))
	})

	t.Run("non-staking msg returns false", func(t *testing.T) {
		msg := banktypes.NewMsgSend(sdk.AccAddress([]byte("from")), sdk.AccAddress([]byte("to")), sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)))
		require.False(t, dec.involvesStakingMsg(msg))
	})

	t.Run("gov vote msg is not staking", func(t *testing.T) {
		msg := govv1.NewMsgVote(sdk.AccAddress([]byte("addr")), 1, govv1.OptionYes, "metadata")
		require.False(t, dec.involvesStakingMsg(msg))
	})

	t.Run("nil msg returns false", func(t *testing.T) {
		require.False(t, dec.involvesStakingMsg(nil))
	})
}

func TestValidatorCommissionDecoratorValidateAuthz(t *testing.T) {
	dec := ValidatorCommissionDecorator{}

	t.Run("has staking exec msg returns true", func(t *testing.T) {
		delegateMsg := &stakingtypes.MsgDelegate{
			DelegatorAddress: "cosmos1addr",
			ValidatorAddress: "cosmosvaloper1validator",
			Amount:           sdk.NewInt64Coin("stake", 1000),
		}
		execMsg := sdk.Msg(delegateMsg)
		require.True(t, dec.involvesAuthzMsg(&execMsg))
	})

	t.Run("has non-staking exec msg returns false", func(t *testing.T) {
		bankMsg := &banktypes.MsgSend{
			FromAddress: "cosmos1addr",
			ToAddress:   "cosmos1recipient",
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
		}
		execMsg := sdk.Msg(bankMsg)
		require.False(t, dec.involvesAuthzMsg(&execMsg))
	})
}
