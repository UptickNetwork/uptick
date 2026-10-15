package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func registerApp(t *testing.T, e testEnv) []byte {
	t.Helper()
	appID := hash32(0xAA)
	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority:            e.auth.String(),
		AppId:                appID,
		Owner:                e.owner.String(),
		AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)
	return appID
}

func setBudget(t *testing.T, e testEnv, epoch uint64, limit string) {
	t.Helper()
	_, err := e.server.SetEpochBudget(e.ctx, &types.MsgSetEpochBudget{
		Authority: e.auth.String(),
		Budget: types.EpochBudget{
			Epoch:  epoch,
			Denom:  types.DefaultChallengeBondDenom,
			Limit:  limit,
			Spent:  "0",
			Closed: false,
		},
	})
	require.NoError(t, err)
}

func sampleAttestation(appID []byte, epoch uint64, reward string) types.AttestationCommitment {
	return types.AttestationCommitment{
		AppId:              appID,
		Epoch:              epoch,
		Profile:            types.DefaultUGEProfile,
		ProfileVersion:     types.DefaultUGEProfileVersion,
		AttestationHash:    hash32(0x11),
		EvidenceRoot:       hash32(0x22),
		ResultCommitment:   hash32(0x33),
		RewardRequested:    reward,
		GasCreditRequested: "0",
		Submitter:          "",
	}
}

func TestVector_ZERO_BUDGET(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "0")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter:   e.sub.String(),
		Attestation: sampleAttestation(appID, 1, "1"),
	})
	require.NoError(t, err)
	e = e.withHeight(2000)
	_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
		Caller: e.owner.String(),
		AppId:  appID,
		Epoch:  1,
	})
	require.ErrorIs(t, err, types.ErrBudgetUnavailable)
	_, found := e.k.GetSettlement(e.ctx, appID, 1)
	require.False(t, found)
}

func TestVector_OVER_BUDGET(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter:   e.sub.String(),
		Attestation: sampleAttestation(appID, 1, "101"),
	})
	require.NoError(t, err)
	e = e.withHeight(2000)
	_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
		Caller: e.owner.String(),
		AppId:  appID,
		Epoch:  1,
	})
	require.ErrorIs(t, err, types.ErrBudgetExceeded)
	b, _ := e.k.GetEpochBudget(e.ctx, 1)
	require.Equal(t, "0", b.Spent)
}

func TestVector_DUPLICATE_ATTESTATION(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	msg := &types.MsgSubmitAttestation{Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10")}
	_, err := e.server.SubmitAttestation(e.ctx, msg)
	require.NoError(t, err)
	_, err = e.server.SubmitAttestation(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrAttestationDuplicate)
}

func TestVector_OPEN_CHALLENGE(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.NoError(t, err)
	e.bank.fundAccount(e.payer, types.DefaultChallengeBondDenom, math.NewInt(10_000_000))
	_, err = e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
		Challenger:   e.payer.String(),
		AppId:        appID,
		Epoch:        1,
		EvidenceHash: hash32(0x44),
	})
	require.NoError(t, err)
	e = e.withHeight(2000)
	_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
		Caller: e.owner.String(), AppId: appID, Epoch: 1,
	})
	require.ErrorIs(t, err, types.ErrOpenChallenge)
}

func TestVector_UPHELD_CHALLENGE(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.NoError(t, err)
	e.bank.fundAccount(e.payer, types.DefaultChallengeBondDenom, math.NewInt(10_000_000))
	chalRes, err := e.server.ChallengeAttestation(e.ctx, &types.MsgChallengeAttestation{
		Challenger: e.payer.String(), AppId: appID, Epoch: 1, EvidenceHash: hash32(0x44),
	})
	require.NoError(t, err)
	_, err = e.server.ResolveChallenge(e.ctx, &types.MsgResolveChallenge{
		Authority: e.auth.String(), ChallengeId: chalRes.ChallengeId, Uphold: true, ResolutionHash: hash32(0x55),
	})
	require.NoError(t, err)
	att, _ := e.k.GetAttestation(e.ctx, appID, 1)
	require.Equal(t, types.AttestationStatusRejected, att.Status)
	e = e.withHeight(2000)
	_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
		Caller: e.owner.String(), AppId: appID, Epoch: 1,
	})
	require.Error(t, err)
	_, found := e.k.GetSettlement(e.ctx, appID, 1)
	require.False(t, found)
}

func TestVector_CHALLENGE_WINDOW_EXPIRED(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.NoError(t, err)
	e = e.withHeight(2000)
	_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
		Caller: e.owner.String(), AppId: appID, Epoch: 1,
	})
	require.NoError(t, err)
	_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
		Caller: e.owner.String(), AppId: appID, Epoch: 1,
	})
	require.ErrorIs(t, err, types.ErrSettlementExists)
}

func TestVector_UNAUTHORIZED_SUBMITTER(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.payer.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)
}

func TestVector_UNAUTHORIZED_GAS_PAYER(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	e.k.SetGasCredit(e.ctx, types.GasCreditAccount{
		AppId: appID, Balance: "100", TotalIssued: "100", TotalConsumed: "0",
	})
	_, err := e.server.ConsumeGasCredit(e.ctx, &types.MsgConsumeGasCredit{
		GasPayer: e.payer.String(), AppId: appID, Amount: "1", SourceTxHash: hash32(0x66),
	})
	require.ErrorIs(t, err, types.ErrGasPayerUnauthorized)
	acct, _ := e.k.GetGasCredit(e.ctx, appID)
	require.Equal(t, "100", acct.Balance)
}

func TestVector_CREDIT_OVERSPEND(t *testing.T) {
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
		GasPayer: e.payer.String(), AppId: appID, Amount: "51", SourceTxHash: hash32(0x66),
	})
	require.ErrorIs(t, err, types.ErrCreditInsufficient)
	acct, _ := e.k.GetGasCredit(e.ctx, appID)
	require.Equal(t, "50", acct.Balance)
}

func TestVector_UNSUPPORTED_PROFILE(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	att := sampleAttestation(appID, 1, "10")
	att.Profile = "unknown"
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: att,
	})
	require.ErrorIs(t, err, types.ErrUnsupportedProfile)
}

func TestVector_SOURCE_TX_REPLAY(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)
	e.k.SetGasCredit(e.ctx, types.GasCreditAccount{
		AppId: appID, Balance: "50", TotalIssued: "50", TotalConsumed: "0",
	})
	msg := &types.MsgConsumeGasCredit{
		GasPayer: e.payer.String(), AppId: appID, Amount: "10", SourceTxHash: hash32(0x66),
	}
	_, err = e.server.ConsumeGasCredit(e.ctx, msg)
	require.NoError(t, err)
	_, err = e.server.ConsumeGasCredit(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrSourceTxReplay)
}

func TestGenesisRoundTrip(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 7, "100")
	_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)
	e.k.SetGasCredit(e.ctx, types.GasCreditAccount{
		AppId: appID, Balance: "20", TotalIssued: "20", TotalConsumed: "0",
	})
	e.k.SetGasCreditConsumption(e.ctx, types.GasCreditConsumption{
		AppId: appID, GasPayer: e.payer.String(), SourceTxHash: hash32(0x77), Amount: "5", ConsumedHeight: 1,
	})

	exported := e.k.ExportGenesis(e.ctx)
	require.NoError(t, types.ValidateGenesis(*exported))
	require.Len(t, exported.Applications, 1)
	require.Len(t, exported.Applications[0].GasPayers, 1)
	require.Len(t, exported.GasCreditConsumptions, 1)

	e2 := setupTest(t)
	e2.k.InitGenesis(e2.ctx, *exported)
	exported2 := e2.k.ExportGenesis(e2.ctx)
	require.Equal(t, len(exported.Applications), len(exported2.Applications))
	require.Equal(t, len(exported.GasCreditConsumptions), len(exported2.GasCreditConsumptions))
	require.True(t, e2.k.HasGasCreditConsumption(e2.ctx, appID, hash32(0x77)))
}
