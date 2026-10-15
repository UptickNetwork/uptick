package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestGrantGasPayerUniqueAcrossApps(t *testing.T) {
	e := setupTest(t)
	app1 := registerApp(t, e)
	app2 := hash32(0xBB)
	_, err := e.server.RegisterApplication(e.ctx, &types.MsgRegisterApplication{
		Authority: e.auth.String(), AppId: app2, Owner: e.owner.String(), AttestationSubmitter: e.sub.String(),
	})
	require.NoError(t, err)

	_, err = e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: app1, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)

	_, err = e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: app2, GasPayer: e.payer.String(),
	})
	require.ErrorIs(t, err, types.ErrGasPayerAlreadyBound)

	id, ok := e.k.GetAppIDByGasPayer(e.ctx, e.payer.String())
	require.True(t, ok)
	require.Equal(t, app1, id)
}

func TestVector_AUTO_CONSUME_ENROLLED_PAYER(t *testing.T) {
	TestAutoConsumeForTx(t)
}

func TestVector_AUTO_CONSUME_UNENROLLED_NOOP(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	e.k.SetGasCredit(e.ctx, types.GasCreditAccount{
		AppId: appID, Balance: "100", TotalIssued: "100", TotalConsumed: "0",
	})
	require.NoError(t, e.k.AutoConsumeForTx(e.ctx, e.payer, 10, hash32(0x91)))
	acct, _ := e.k.GetGasCredit(e.ctx, appID)
	require.Equal(t, "100", acct.Balance)
	require.False(t, e.k.HasGasCreditConsumption(e.ctx, appID, hash32(0x91)))
}

func TestVector_AUTO_CONSUME_INSUFFICIENT_SKIP(t *testing.T) {
	TestAutoConsumeInsufficientSkips(t)
}

func TestAutoConsumeForTx(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)
	e.k.SetGasCredit(e.ctx, types.GasCreditAccount{
		AppId: appID, Balance: "1000", TotalIssued: "1000", TotalConsumed: "0",
	})

	txHash := hash32(0x99)
	require.NoError(t, e.k.AutoConsumeForTx(e.ctx, e.payer, 42, txHash))

	acct, ok := e.k.GetGasCredit(e.ctx, appID)
	require.True(t, ok)
	require.Equal(t, "958", acct.Balance)
	require.Equal(t, "42", acct.TotalConsumed)
	require.True(t, e.k.HasGasCreditConsumption(e.ctx, appID, txHash))

	// replay is skipped (no error)
	require.NoError(t, e.k.AutoConsumeForTx(e.ctx, e.payer, 1, txHash))
	acct, _ = e.k.GetGasCredit(e.ctx, appID)
	require.Equal(t, "958", acct.Balance)

	// non-enrolled payer: no-op
	require.NoError(t, e.k.AutoConsumeForTx(e.ctx, e.owner, 10, hash32(0x98)))
}

func TestAutoConsumeInsufficientSkips(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)
	e.k.SetGasCredit(e.ctx, types.GasCreditAccount{
		AppId: appID, Balance: "5", TotalIssued: "5", TotalConsumed: "0",
	})

	require.NoError(t, e.k.AutoConsumeForTx(e.ctx, e.payer, 10, hash32(0x97)))
	acct, _ := e.k.GetGasCredit(e.ctx, appID)
	require.Equal(t, "5", acct.Balance)
	require.False(t, e.k.HasGasCreditConsumption(e.ctx, appID, hash32(0x97)))
}

func TestRevokeGasPayerClearsIndex(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)
	_, err = e.server.RevokeGasPayer(e.ctx, &types.MsgRevokeGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)
	_, ok := e.k.GetAppIDByGasPayer(e.ctx, e.payer.String())
	require.False(t, ok)
}

func TestGenesisRebuildsGasPayerIndex(t *testing.T) {
	e := setupTest(t)
	appID := registerApp(t, e)
	_, err := e.server.GrantGasPayer(e.ctx, &types.MsgGrantGasPayer{
		Owner: e.owner.String(), AppId: appID, GasPayer: e.payer.String(),
	})
	require.NoError(t, err)

	exported := e.k.ExportGenesis(e.ctx)
	e2 := setupTest(t)
	e2.k.InitGenesis(e2.ctx, *exported)
	id, ok := e2.k.GetAppIDByGasPayer(e2.ctx, e.payer.String())
	require.True(t, ok)
	require.Equal(t, appID, id)
}
