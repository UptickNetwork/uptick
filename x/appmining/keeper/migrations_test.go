package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/keeper"
	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestMigrate1to2_PausesLegacyFlow(t *testing.T) {
	e := setupTest(t)
	// setupTest unpauses legacy for POC vectors; simulate a pre-V1 chain.
	params := e.k.GetParams(e.ctx)
	require.False(t, params.PauseState.AttestationPaused)

	m := keeper.NewMigrator(e.k)
	require.NoError(t, m.Migrate1to2(e.ctx))

	params = e.k.GetParams(e.ctx)
	require.True(t, params.PauseState.AttestationPaused)
	require.True(t, params.PauseState.ChallengePaused)
	require.True(t, params.PauseState.SettlementPaused)
	require.True(t, params.PauseState.GasCreditPaused)
	require.False(t, params.PauseState.EntitlementPaused)

	appID := registerApp(t, e)
	setBudget(t, e, 1, "100")
	_, err := e.server.SubmitAttestation(e.ctx, &types.MsgSubmitAttestation{
		Submitter: e.sub.String(), Attestation: sampleAttestation(appID, 1, "10"),
	})
	require.ErrorIs(t, err, types.ErrAttestationPaused)
}

func TestDefaultParams_PausesLegacyAttestationFlow(t *testing.T) {
	p := types.DefaultParams()
	require.True(t, p.PauseState.AttestationPaused)
	require.True(t, p.PauseState.SettlementPaused)
	require.True(t, p.PauseState.GasCreditPaused)
	require.False(t, p.PauseState.EntitlementPaused)
}

func TestE6_TimeRollbackDoesNotSkipChallengeWindow(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)

	// Peek past the window via block time only — watermark stays at publish time.
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+types.ChallengeWindowSeconds, 0))
	// Roll clock back before the window elapses.
	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+1, 0))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "961"))
	require.ErrorIs(t, err, types.ErrChallengeWindow)
}

func TestE6_WatermarkKeepsWindowElapsedAfterRollback(t *testing.T) {
	e, _, _, aprPriv, c := setupP2(t)
	cand, _ := e.k.GetCandidate(e.ctx, c.EntitlementID)

	// Prior observation past the window (e.g. earlier successful tx path).
	future := cand.PublishedAt + types.ChallengeWindowSeconds
	cand.MaxObservedTime = future
	e.k.SetCandidate(e.ctx, cand)

	e.ctx = e.ctx.WithBlockTime(time.Unix(cand.PublishedAt+1, 0))
	_, err := e.server.ApproveEntitlement(e.ctx, approveMsg(t, e, aprPriv, c, "962"))
	require.NoError(t, err)
}
