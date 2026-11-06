package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/keeper"
	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// UGE doc15: module must cover available+reserved, not reserved alone
// (Genesis / ledger skew without matching bank coins).
func TestFunding_UncoveredAvailableDisablesFlow(t *testing.T) {
	e := setupTest(t)
	params := e.k.GetParams(e.ctx)
	params.EntitlementEnableMinFunded = "100"
	require.NoError(t, e.k.SetParams(e.ctx, params))

	_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: "skew-pool", Denom: types.DefaultChallengeBondDenom, Version: 1,
	})
	require.NoError(t, err)

	// Ledger looks funded; module account has no coins (genesis/migration skew).
	e.k.SetEntitlementPool(e.ctx, types.Pool{
		PoolId: "skew-pool", Denom: types.DefaultChallengeBondDenom,
		Funded: "100", Available: "100", Reserved: "0", Paid: "0", Version: 1,
	})

	q, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: "skew-pool"})
	require.NoError(t, err)
	require.Equal(t, "0", q.ModuleBalance)
	require.False(t, q.EntitlementFlowEnabled)

	msg, broken := keeper.EntitlementPoolInvariant(e.k)(e.ctx)
	require.True(t, broken, msg)
}

// FundEntitlementPool is authority-gated (V1-AUDIT): a non-authority signer
// must be rejected before any pool lookup, and the authority path must still
// fund through the same msg (positive control for the gate).
func TestFunding_AuthorityGate(t *testing.T) {
	e := setupTest(t)
	params := e.k.GetParams(e.ctx)
	params.EntitlementEnableMinFunded = "50"
	require.NoError(t, e.k.SetParams(e.ctx, params))
	denom := types.DefaultChallengeBondDenom

	_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: "pool-gate", Denom: denom, Version: 1,
	})
	require.NoError(t, err)

	// Negative: a non-authority account is rejected even with balance.
	e.bank.fundAccount(e.owner, denom, math.NewInt(100))
	_, err = e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Authority: e.owner.String(), PoolId: "pool-gate", Amount: "100",
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	// Negative: authority name with a non-authority balance drains nothing.
	e.bank.fundAccount(e.auth, denom, math.NewInt(0))
	_, err = e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Authority: e.auth.String(), PoolId: "pool-gate", Amount: "100",
	})
	require.Error(t, err) // insufficient mock balance; gate itself passed

	// Positive: funded authority account funds the pool.
	e.bank.fundAccount(e.auth, denom, math.NewInt(100))
	_, err = e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Authority: e.auth.String(), PoolId: "pool-gate", Amount: "100",
	})
	require.NoError(t, err)

	q, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: "pool-gate"})
	require.NoError(t, err)
	require.Equal(t, "100", q.Pool.Funded)
	require.True(t, q.EntitlementFlowEnabled)
}

// Same module account + denom: per-pool reserved-only checks would pass while
// aggregate available+reserved exceeds balance.
func TestFunding_MultiPoolSameDenomAggregatesUnpaid(t *testing.T) {
	e := setupTest(t)
	params := e.k.GetParams(e.ctx)
	params.EntitlementEnableMinFunded = "50"
	require.NoError(t, e.k.SetParams(e.ctx, params))
	denom := types.DefaultChallengeBondDenom

	_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: "pool-a", Denom: denom, Version: 1,
	})
	require.NoError(t, err)
	_, err = e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(), PoolId: "pool-b", Denom: denom, Version: 1,
	})
	require.NoError(t, err)

	e.bank.fundAccount(e.auth, denom, math.NewInt(100))
	_, err = e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Authority: e.auth.String(), PoolId: "pool-a", Amount: "100",
	})
	require.NoError(t, err)

	// Inject second pool ledger unpaid without transferring coins (shared balance still 100).
	e.k.SetEntitlementPool(e.ctx, types.Pool{
		PoolId: "pool-b", Denom: denom,
		Funded: "50", Available: "50", Reserved: "0", Paid: "0", Version: 1,
	})

	qa, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: "pool-a"})
	require.NoError(t, err)
	require.Equal(t, "100", qa.ModuleBalance)
	require.False(t, qa.EntitlementFlowEnabled, "aggregate unpaid 150 must disable pool-a")

	qb, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: "pool-b"})
	require.NoError(t, err)
	require.False(t, qb.EntitlementFlowEnabled)

	msg, broken := keeper.EntitlementPoolInvariant(e.k)(e.ctx)
	require.True(t, broken, msg)
}
