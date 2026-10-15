package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestCanSubmitAttestationBlocksResubmit(t *testing.T) {
	app := types.Application{Status: types.ApplicationStatusActive}
	existing := &types.AttestationCommitment{Status: types.AttestationStatusChallenged}
	err := types.CanSubmitAttestation(app, existing, 1)
	require.ErrorIs(t, err, types.ErrAttestationDuplicate)

	err = types.CanSubmitAttestation(app, nil, 1)
	require.NoError(t, err)

	app.Status = types.ApplicationStatusSuspended
	err = types.CanSubmitAttestation(app, nil, 1)
	require.ErrorIs(t, err, types.ErrApplicationNotActive)
}

func TestComputeSettlementAmountsRejectsOverBudget(t *testing.T) {
	att := types.AttestationCommitment{RewardRequested: "101"}
	budget := types.EpochBudget{Limit: "100", Spent: "0"}
	_, _, _, err := types.ComputeSettlementAmounts(att, budget, 0)
	require.ErrorIs(t, err, types.ErrBudgetExceeded)
}

func TestComputeSettlementAmountsRejectsZeroBudget(t *testing.T) {
	att := types.AttestationCommitment{RewardRequested: "1"}
	budget := types.EpochBudget{Limit: "0", Spent: "0"}
	_, _, _, err := types.ComputeSettlementAmounts(att, budget, 0)
	require.ErrorIs(t, err, types.ErrBudgetUnavailable)
}

func TestCanFinalizeOpenChallenge(t *testing.T) {
	att := types.AttestationCommitment{
		Status:                  types.AttestationStatusChallengeWindow,
		ChallengeDeadlineHeight: 10,
	}
	err := types.CanFinalize(att, 11, true)
	require.ErrorIs(t, err, types.ErrOpenChallenge)

	err = types.CanFinalize(att, 11, false)
	require.NoError(t, err)

	err = types.CanFinalize(att, 10, false)
	require.ErrorIs(t, err, types.ErrChallengeWindow)
}

func TestCanConsumeCredit(t *testing.T) {
	_, err := types.CanConsumeCredit("50", "51")
	require.ErrorIs(t, err, types.ErrCreditInsufficient)

	bal, err := types.CanConsumeCredit("50", "20")
	require.NoError(t, err)
	require.Equal(t, "30", bal)
}

func TestValidateAmountString(t *testing.T) {
	require.NoError(t, types.ValidateAmountString("0"))
	require.NoError(t, types.ValidateAmountString("100"))
	require.Error(t, types.ValidateAmountString("01"))
	require.Error(t, types.ValidateAmountString("1.5"))
	require.Error(t, types.ValidateAmountString("-1"))
}

func TestValidateAppID(t *testing.T) {
	require.Error(t, types.ValidateAppID([]byte("short")))
	require.NoError(t, types.ValidateAppID(make([]byte, 32)))
}
