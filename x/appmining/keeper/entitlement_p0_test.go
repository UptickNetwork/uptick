package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestP0_OpenFundPoolAndEnableGate(t *testing.T) {
	e := setupTest(t)
	params := e.k.GetParams(e.ctx)
	params.EntitlementEnableMinFunded = "1000"
	require.NoError(t, e.k.SetParams(e.ctx, params))

	_, err := e.server.OpenEntitlementPool(e.ctx, &types.MsgOpenEntitlementPool{
		Authority: e.auth.String(),
		PoolId:    "ecosystem-v0",
		Denom:     types.DefaultChallengeBondDenom,
		Version:   1,
	})
	require.NoError(t, err)

	q, err := e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: "ecosystem-v0"})
	require.NoError(t, err)
	require.False(t, q.EntitlementFlowEnabled)
	require.Equal(t, "0", q.Pool.Funded)

	e.bank.fundAccount(e.owner, types.DefaultChallengeBondDenom, math.NewInt(5000))
	_, err = e.server.FundEntitlementPool(e.ctx, &types.MsgFundEntitlementPool{
		Funder: e.owner.String(),
		PoolId: "ecosystem-v0",
		Amount: "1000",
	})
	require.NoError(t, err)

	q, err = e.k.Pool(e.ctx, &types.QueryPoolRequest{PoolId: "ecosystem-v0"})
	require.NoError(t, err)
	require.True(t, q.EntitlementFlowEnabled)
	require.Equal(t, "1000", q.Pool.Funded)
	require.Equal(t, "1000", q.Pool.Available)
	require.Equal(t, "0", q.Pool.Reserved)
	require.Equal(t, "0", q.Pool.Paid)
}

func TestP0_RegisterRoleKey(t *testing.T) {
	e := setupTest(t)
	pk := make([]byte, 32)
	for i := range pk {
		pk[i] = byte(i + 1)
	}
	_, err := e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(),
		Role:      types.EntitlementRolePublisher,
		KeyId:     "pub-1",
		Pubkey:    pk,
		Version:   1,
	})
	require.NoError(t, err)

	// idempotent
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(),
		Role:      types.EntitlementRolePublisher,
		KeyId:     "pub-1",
		Pubkey:    pk,
		Version:   1,
	})
	require.NoError(t, err)

	other := make([]byte, 32)
	other[0] = 0xff
	_, err = e.server.RegisterRoleKey(e.ctx, &types.MsgRegisterRoleKey{
		Authority: e.auth.String(),
		Role:      types.EntitlementRolePublisher,
		KeyId:     "pub-1",
		Pubkey:    other,
		Version:   2,
	})
	require.ErrorIs(t, err, types.ErrRoleKeyConflict)

	roles, err := e.k.Roles(e.ctx, &types.QueryRolesRequest{Role: types.EntitlementRolePublisher})
	require.NoError(t, err)
	require.Len(t, roles.Keys, 1)
	require.Equal(t, types.RoleKeyStatusActive, roles.Keys[0].Status)
}
