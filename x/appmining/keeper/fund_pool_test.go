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
	e.bank.fundAccount(e.owner, denom, math.NewInt(5000))

	poolAddr := types.AccRewardPoolAddress
	poolBefore := e.bank.GetBalance(e.ctx, poolAddr, denom)

	_, err := e.server.FundRewardPool(e.ctx, &types.MsgFundRewardPool{
		Funder: e.owner.String(),
		Denom:  denom,
		Amount: "1000",
	})
	require.NoError(t, err)

	poolAfter := e.bank.GetBalance(e.ctx, poolAddr, denom)
	require.Equal(t, poolBefore.Amount.Add(math.NewInt(1000)), poolAfter.Amount)
	ownerAfter := e.bank.GetBalance(e.ctx, e.owner, denom)
	require.Equal(t, math.NewInt(4000), ownerAfter.Amount)
}

func TestFundRewardPoolDenomNotAllowlisted(t *testing.T) {
	e := setupTest(t)
	e.bank.fundAccount(e.auth, "ucoin", math.NewInt(1000))
	_, err := e.server.FundRewardPool(e.ctx, &types.MsgFundRewardPool{
		Funder: e.auth.String(),
		Denom:  "ucoin",
		Amount: "100",
	})
	require.ErrorIs(t, err, types.ErrInvalidDenom)
}

func TestFundRewardPoolZeroRejected(t *testing.T) {
	e := setupTest(t)
	msg := &types.MsgFundRewardPool{
		Funder: e.auth.String(),
		Denom:  types.DefaultChallengeBondDenom,
		Amount: "0",
	}
	require.Error(t, msg.ValidateBasic())
}

func TestChallengeBondDoesNotInflateRewardPool(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.NoError(t, err)

	denom := types.DefaultChallengeBondDenom
	e.bank.fundAccount(e.payer, denom, math.NewInt(10_000_000))
	poolBefore := e.bank.GetBalance(e.ctx, types.AccRewardPoolAddress, denom)
	escrowBefore := e.bank.GetBalance(e.ctx, types.AccChallengeEscrowAddress, denom)

	_, err = e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
		Challenger: e.payer.String(), AppId: appID, Epoch: 1, EvidenceHash: hash32(0x44),
	})
	require.NoError(t, err)

	poolAfter := e.bank.GetBalance(e.ctx, types.AccRewardPoolAddress, denom)
	escrowAfter := e.bank.GetBalance(e.ctx, types.AccChallengeEscrowAddress, denom)
	require.True(t, poolAfter.Amount.Equal(poolBefore.Amount), "challenge bond must not enter reward pool")
	require.True(t, escrowAfter.Amount.GT(escrowBefore.Amount), "challenge bond must enter escrow")
}
