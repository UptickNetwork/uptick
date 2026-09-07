package ante

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

// Tests for the live validation path (validateMsg). The former
// involvesStakingMsg / involvesAuthzMsg helpers were dead code and have been
// removed — the AnteHandle switch dispatches directly.

func TestValidatorCommissionDecoratorValidateMsg(t *testing.T) {
	dec := ValidatorCommissionDecorator{}

	t.Run("create validator below min commission is rejected", func(t *testing.T) {
		msg, err := stakingtypes.NewMsgCreateValidator(
			"cosmosvaloper1validator",
			ed25519.GenPrivKey().PubKey(),
			sdk.NewCoin("stake", math.NewInt(1000)),
			stakingtypes.Description{Moniker: "test"},
			// 1% — below the 5% minimum.
			stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(1, 2), math.LegacyZeroDec(), math.LegacyZeroDec()),
			math.NewInt(1),
		)
		require.NoError(t, err)

		err = dec.validateMsg(sdk.Context{}, msg)
		require.Error(t, err)
		require.ErrorIs(t, err, errortypes.ErrInvalidRequest)
		require.Contains(t, err.Error(), "cannot be lower than minimum")
	})

	t.Run("create validator at exactly min commission passes", func(t *testing.T) {
		msg, err := stakingtypes.NewMsgCreateValidator(
			"cosmosvaloper1validator",
			ed25519.GenPrivKey().PubKey(),
			sdk.NewCoin("stake", math.NewInt(1000)),
			stakingtypes.Description{Moniker: "test"},
			// Exactly 5%.
			stakingtypes.NewCommissionRates(minCommission, math.LegacyZeroDec(), math.LegacyZeroDec()),
			math.NewInt(1),
		)
		require.NoError(t, err)
		require.NoError(t, dec.validateMsg(sdk.Context{}, msg))
	})

	t.Run("edit validator below min commission is rejected", func(t *testing.T) {
		rate := math.LegacyNewDecWithPrec(2, 2) // 2%
		msg := &stakingtypes.MsgEditValidator{
			ValidatorAddress: "cosmosvaloper1validator",
			CommissionRate:   &rate,
		}
		err := dec.validateMsg(sdk.Context{}, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be lower than minimum")
	})

	t.Run("edit validator with nil commission rate passes", func(t *testing.T) {
		msg := &stakingtypes.MsgEditValidator{
			ValidatorAddress: "cosmosvaloper1validator",
		}
		require.NoError(t, dec.validateMsg(sdk.Context{}, msg))
	})

	t.Run("non-staking msgs are ignored", func(t *testing.T) {
		bankMsg := banktypes.NewMsgSend(
			sdk.AccAddress([]byte("from")),
			sdk.AccAddress([]byte("to")),
			sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
		)
		require.NoError(t, dec.validateMsg(sdk.Context{}, bankMsg))

		voteMsg := govv1.NewMsgVote(sdk.AccAddress([]byte("addr")), 1, govv1.OptionYes, "metadata")
		require.NoError(t, dec.validateMsg(sdk.Context{}, voteMsg))
	})
}
