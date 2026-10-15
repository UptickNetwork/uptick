package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestListQueriesPagination(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.NoError(t, err)

	apps, err := e.k.Applications(e.ctx, &types.QueryApplicationsRequest{})
	require.NoError(t, err)
	require.Len(t, apps.Applications, 1)

	atts, err := e.k.Attestations(e.ctx, &types.QueryAttestationsRequest{})
	require.NoError(t, err)
	require.Len(t, atts.Attestations, 1)

	e = e.withHeight(2000)
	_, err = e.server.FinalizeAttestation(e.ctx, &types.MsgFinalizeAttestation{
		Caller: e.owner.String(), AppId: appID, Epoch: 1,
	})
	require.NoError(t, err)

	setts, err := e.k.Settlements(e.ctx, &types.QuerySettlementsRequest{})
	require.NoError(t, err)
	require.Len(t, setts.Settlements, 1)
}

func TestSuspendedApplicationBlocksAll(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SetApplicationStatus(e.ctx, &types.MsgSetApplicationStatus{
		Authority: e.auth.String(), AppId: appID, Status: types.ApplicationStatusSuspended,
	})
	require.NoError(t, err)

	_, err = e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.ErrorIs(t, err, types.ErrApplicationNotActive)

	// seed an attestation before suspend path for finalize/consume checks
	e2 := setupTest(t)
	appID2 := registerApp(t, e2)
	setBudget(t, e2, 1, "100")
	_, err = e2.server.SubmitAttestation(e2.ctx, &types.MsgSubmitAttestation{
		Submitter: e2.sub.String(), Attestation: sampleAttestation(appID2, 1, "10"),
	})
	require.NoError(t, err)
	_, err = e2.server.GrantGasPayer(e2.ctx, &types.MsgGrantGasPayer{
		Owner: e2.owner.String(), AppId: appID2, GasPayer: e2.payer.String(),
	})
	require.NoError(t, err)
	e2.k.SetGasCredit(e2.ctx, types.GasCreditAccount{
		AppId: appID2, Balance: "50", TotalIssued: "50", TotalConsumed: "0",
	})
	_, err = e2.server.SetApplicationStatus(e2.ctx, &types.MsgSetApplicationStatus{
		Authority: e2.auth.String(), AppId: appID2, Status: types.ApplicationStatusSuspended,
	})
	require.NoError(t, err)

	e2 = e2.withHeight(2000)
	_, err = e2.server.FinalizeAttestation(e2.ctx, &types.MsgFinalizeAttestation{
		Caller: e2.owner.String(), AppId: appID2, Epoch: 1,
	})
	require.ErrorIs(t, err, types.ErrApplicationNotActive)

	_, err = e2.server.ConsumeGasCredit(e2.ctx, &types.MsgConsumeGasCredit{
		GasPayer: e2.payer.String(), AppId: appID2, Amount: "1", SourceTxHash: hash32(0x88),
	})
	require.ErrorIs(t, err, types.ErrApplicationNotActive)

	// history preserved
	att, ok := e2.k.GetAttestation(e2.ctx, appID2, 1)
	require.True(t, ok)
	require.Equal(t, types.AttestationStatusChallengeWindow, att.Status)
	acct, ok := e2.k.GetGasCredit(e2.ctx, appID2)
	require.True(t, ok)
	require.Equal(t, "50", acct.Balance)
}

func TestPausePreservesHistory(t *testing.T) {
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

	_, err = e.server.Pause(e.ctx, &types.MsgPause{
		PauseAuthority: e.auth.String(),
		PauseBits:      types.PauseState{SettlementPaused: true, AttestationPaused: true},
	})
	require.NoError(t, err)

	sett, err := e.k.Settlement(e.ctx, &types.QuerySettlementRequest{AppId: appID, Epoch: 1})
	require.NoError(t, err)
	require.Equal(t, "10", sett.Settlement.RewardReleased)

	atts, err := e.k.Attestations(e.ctx, &types.QueryAttestationsRequest{})
	require.NoError(t, err)
	require.Len(t, atts.Attestations, 1)
	require.Equal(t, types.AttestationStatusFinalized, atts.Attestations[0].Status)
}

func TestRestartRecoveryIdenticalState(t *testing.T) {
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

	before, err := e.k.Settlement(e.ctx, &types.QuerySettlementRequest{AppId: appID, Epoch: 1})
	require.NoError(t, err)

	exported := e.k.ExportGenesis(e.ctx)
	e2 := setupTest(t)
	e2.k.InitGenesis(e2.ctx, *exported)

	after, err := e2.k.Settlement(e2.ctx, &types.QuerySettlementRequest{AppId: appID, Epoch: 1})
	require.NoError(t, err)
	require.Equal(t, before.Settlement.RewardReleased, after.Settlement.RewardReleased)
	require.Equal(t, before.Settlement.GasCreditIssued, after.Settlement.GasCreditIssued)

	_, err = e2.server.FinalizeAttestation(e2.ctx, &types.MsgFinalizeAttestation{
		Caller: e2.owner.String(), AppId: appID, Epoch: 1,
	})
	require.ErrorIs(t, err, types.ErrSettlementExists)
}
