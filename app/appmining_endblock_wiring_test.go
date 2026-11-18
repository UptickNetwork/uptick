package app

import (
	"testing"

	appmodule "cosmossdk.io/core/appmodule"
	"github.com/stretchr/testify/require"

	appminingtypes "github.com/UptickNetwork/uptick/x/appmining/types"
)

// TestAppminingEndBlockerWired is the wiring half of the timeout-settlement
// guarantee: keeper.SweepExpiredChallenges must actually run every block.
//
// Two independent things have to hold, and neither is visible from the keeper
// tests: the module has to satisfy appmodule.HasEndBlocker (otherwise
// module.Manager skips it silently), and it has to appear in
// OrderEndBlockers (otherwise the manager never looks at it). Removing the
// method breaks compilation of the interface assertion in x/appmining/module.go;
// removing the name from the order list is caught here.
func TestAppminingEndBlockerWired(t *testing.T) {
	app, _ := sharedTestApp(t)

	mod, ok := app.mm.Modules[appminingtypes.ModuleName]
	require.True(t, ok, "x/appmining must be registered with the module manager")
	_, isEndBlocker := mod.(appmodule.HasEndBlocker)
	require.True(t, isEndBlocker,
		"x/appmining must implement EndBlock: without it an expired challenge only settles if somebody pays gas for it")

	require.Contains(t, app.mm.OrderEndBlockers, appminingtypes.ModuleName,
		"a module missing from SetOrderEndBlockers is never executed by the manager")
}
