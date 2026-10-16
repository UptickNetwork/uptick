package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestManualConsumeDisabledByDefault(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)
	e.k.SetGasCredit(e.ctx, types.GasCreditAccount{
		AppId: appID, Balance: "50", TotalIssued: "50", TotalConsumed: "0",
	})
	_, err = e.server.ConsumeGasCredit(e.ctx, &types.MsgConsumeGasCredit{
		GasPayer: e.payer.String(), AppId: appID, Amount: "1", SourceTxHash: hash32(0x66),
	})
	require.ErrorIs(t, err, types.ErrManualConsumeDisabled)
	acct, _ := e.k.GetGasCredit(e.ctx, appID)
	require.Equal(t, "50", acct.Balance)
}

func TestSetAttestationSubmitterRotation(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	newSub := sdk.AccAddress(bytes20(0x33))

	_, err := e.server.SetAttestationSubmitter(e.ctx, &types.MsgSetAttestationSubmitter{
		Owner: e.owner.String(), AppId: appID, AttestationSubmitter: newSub.String(),
	})
	require.NoError(t, err)

	app, ok := e.k.GetApplication(e.ctx, appID)
	require.True(t, ok)
	require.Equal(t, newSub.String(), app.AttestationSubmitter)

	_, err = e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	_, err = e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: newSub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.NoError(t, err)
}

func TestRevokedIsPermanent(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)

	_, err := e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
		Authority: e.auth.String(), AppId: appID, Status: types.ApplicationStatusRevoked,
	})
	require.NoError(t, err)

	_, err = e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
		Authority: e.auth.String(), AppId: appID, Status: types.ApplicationStatusActive,
	})
	require.ErrorIs(t, err, types.ErrApplicationRevoked)

	_, err = e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.ErrorIs(t, err, types.ErrApplicationRevoked)

	_, err = e.server.SetAttestationSubmitter(e.ctx, &types.MsgSetAttestationSubmitter{
		Owner: e.owner.String(), AppId: appID, AttestationSubmitter: e.payer.String(),
	})
	require.ErrorIs(t, err, types.ErrApplicationRevoked)
}

func TestMaxGasPayersCap(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	p := e.k.GetParams(e.ctx)
	p.MaxGasPayers = 2
	require.NoError(t, e.k.SetParams(e.ctx, p))

	payers := []sdk.AccAddress{
		sdk.AccAddress(bytes20(0x10)),
		sdk.AccAddress(bytes20(0x11)),
		sdk.AccAddress(bytes20(0x12)),
	}
	for i, payer := range payers {
		_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
			Owner: e.owner.String(), AppId: appID, GasPayer: payer.String(),
		})
		if i < 2 {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, types.ErrGasPayerLimitExceeded)
		}
	}
	app, _ := e.k.GetApplication(e.ctx, appID)
	require.Len(t, app.GasPayers, 2)
}

func TestSetEpochBudgetRejectsDenomChange(t *testing.T) {
	e := setupTest(t)
	setBudget(t, e, 1, "100")
	params := e.k.GetParams(e.ctx)
	params.RewardDenomAllowlist = append(params.RewardDenomAllowlist, "uatom")
	require.NoError(t, e.k.SetParams(e.ctx, params))
	e.bank.fundModule(types.RewardPoolName, "uatom", math.NewInt(1_000_000))

	_, err := e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget: types.EpochBudget{
			Epoch: 1, Denom: "uatom", Limit: "100", Spent: "0",
		},
	})
	require.ErrorIs(t, err, types.ErrInvalidDenom)

	// same denom, raise limit — ok
	_, err = e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget: types.EpochBudget{
			Epoch: 1, Denom: types.DefaultChallengeBondDenom, Limit: "200", Spent: "0",
		},
	})
	require.NoError(t, err)
}

func TestRejectChallengeKeepsChallengedWhileOpenRemain(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.NoError(t, err)

	c1 := sdk.AccAddress(bytes20(0x51))
	c2 := sdk.AccAddress(bytes20(0x52))
	e.bank.fundAccount(c1, types.DefaultChallengeBondDenom, math.NewInt(10_000_000))
	e.bank.fundAccount(c2, types.DefaultChallengeBondDenom, math.NewInt(10_000_000))
	r1, err := e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
		Challenger: c1.String(), AppId: appID, Epoch: 1, EvidenceHash: hash32(0x41),
	})
	require.NoError(t, err)
	_, err = e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
		Challenger: c2.String(), AppId: appID, Epoch: 1, EvidenceHash: hash32(0x42),
	})
	require.NoError(t, err)

	_, err = e.server.ResolveChallenge(e.ctx, &types.MsgResolveChallenge{
		Authority: e.auth.String(), ChallengeId: r1.ChallengeId, Uphold: false, ResolutionHash: hash32(0x71),
	})
	require.NoError(t, err)

	att, ok := e.k.GetAttestation(e.ctx, appID, 1)
	require.True(t, ok)
	require.Equal(t, types.AttestationStatusChallenged, att.Status)
	require.True(t, e.k.HasOpenChallenge(e.ctx, appID, 1))
}

func TestFinalizePureCreditSettlement(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	att := sampleAttestation(appID, 1, "0")
	att.GasCreditRequested = "25"
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: att,
	})
	require.NoError(t, err)

	e = e.withHeight(2000)
	_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
		Caller: e.owner.String(), AppId: appID, Epoch: 1,
	})
	require.NoError(t, err)

	settlement, ok := e.k.GetSettlement(e.ctx, appID, 1)
	require.True(t, ok)
	require.Equal(t, "0", settlement.RewardReleased)
	require.Equal(t, "25", settlement.GasCreditIssued)
	acct, ok := e.k.GetGasCredit(e.ctx, appID)
	require.True(t, ok)
	require.Equal(t, "25", acct.Balance)
	budget, _ := e.k.GetEpochBudget(e.ctx, 1)
	require.Equal(t, "25", budget.Spent)
}

func TestSubmitRejectsWhenPendingExceedsBudget(t *testing.T) {
	e := setupTest(t)
	setBudget(t, e, 1, "100")
	app1 := registerApp(t, e)
	app2 := hash32(0xBB)
	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority: e.auth.String(), AppId: app2, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)

	_, err = e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(app1, 1, "60"),
	})
	require.NoError(t, err)

	_, err = e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(app2, 1, "50"),
	})
	require.ErrorIs(t, err, types.ErrBudgetExceeded)
}

func TestRegisterOptionalMetadataCommitment(t *testing.T) {
	e := setupTest(t)
	appID := hash32(0xCC)
	meta := hash32(0xDE)
	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority:            e.auth.String(),
		AppId:                appID,
		Owner:                e.owner.String(),
		AttestationSubmitter: e.sub.String(),
		MetadataCommitment:   meta,
	})
	require.NoError(t, err)
	app, ok := e.k.GetApplication(e.ctx, appID)
	require.True(t, ok)
	require.Equal(t, meta, app.MetadataCommitment)

	_, err = e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority:            e.auth.String(),
		AppId:                hash32(0xCD),
		Owner:                e.owner.String(),
		AttestationSubmitter: e.sub.String(),
		MetadataCommitment:   []byte{1, 2, 3},
	})
	require.ErrorIs(t, err, types.ErrInvalidHash)
}
