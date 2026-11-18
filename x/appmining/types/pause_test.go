package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestApplyPauseOR_PreservesEntitlementPaused(t *testing.T) {
	cur := types.PauseState{
		AttestationPaused: true,
		EntitlementPaused: true,
	}
	req := types.PauseState{ChallengePaused: true}
	next := types.ApplyPauseOR(cur, req)
	require.True(t, next.AttestationPaused)
	require.True(t, next.ChallengePaused)
	require.True(t, next.EntitlementPaused)
	require.False(t, next.RegistrationPaused)
}

func TestPauseBitsHasAnyTrue_IncludesEntitlement(t *testing.T) {
	require.True(t, types.PauseBitsHasAnyTrue(types.PauseState{EntitlementPaused: true}))
	require.False(t, types.PauseBitsHasAnyTrue(types.PauseState{}))
}
