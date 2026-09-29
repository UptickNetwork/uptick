package app

import (
	"encoding/binary"
	"testing"

	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/stretchr/testify/require"
)

// TestBootSelfCheckAgainstEveryServedChainsLastCompletedPlan runs x/upgrade's
// real startup self-check against the real production registry.
//
// This is the executable form of the decision documented on `router` in
// app/upgrade.go, and it is the only test that catches a future edit deleting a
// name the router has to keep. Asserting on the router's own map does not: the
// module never consults that map, it consults the keeper, and the check it
// performs is "does the chain's last completed upgrade have a handler" -- a
// no-handler answer aborts the node on startup, not just the upgrade.
//
// The two sub-cases are deliberately opposites. One asserts a chain is served,
// the other asserts a chain is not. Do not "fix" the second by registering
// v0.4.1; read the note on `router` instead.
func TestBootSelfCheckAgainstEveryServedChainsLastCompletedPlan(t *testing.T) {
	app, ctx := sharedTestApp(t)

	upgradeStore := ctx.KVStore(app.GetKey(upgradetypes.StoreKey))

	// preBlocker plants a completed-upgrade record and runs the real entry point
	// over it.
	//
	// The record is written by hand because x/upgrade's own setDone is
	// unexported, and the layout is part of the contract under test:
	// DoneByte 0x01 || height big-endian || name (keeper.go parseDoneKey).
	// GetLastCompletedUpgrade returns the record at the HIGHEST height, so each
	// sub-case plants exactly one and removes it on the way out.
	preBlocker := func(t *testing.T, height int64, name string) error {
		t.Helper()

		key := make([]byte, 9+len(name))
		key[0] = upgradetypes.DoneByte
		binary.BigEndian.PutUint64(key[1:9], uint64(height))
		copy(key[9:], name)
		upgradeStore.Set(key, []byte{0x01})
		t.Cleanup(func() { upgradeStore.Delete(key) })

		// DowngradeVerified is a process-lifetime flag on the keeper: once any
		// call has set it, every later call skips the whole self-check. Without
		// this reset the second sub-case would pass while testing nothing.
		app.UpgradeKeeper.SetDowngradeVerified(false)
		t.Cleanup(func() { app.UpgradeKeeper.SetDowngradeVerified(false) })

		_, err := upgrade.PreBlocker(ctx, app.UpgradeKeeper)
		return err
	}

	// Mainnet stopped on v0.3.3 with nothing scheduled -- the ordinary operator
	// window, and the case that has to boot. Height is mainnet's real
	// applied_plan/v0.3.3.
	t.Run("mainnet is served", func(t *testing.T) {
		require.NoError(t, preBlocker(t, 18_147_950, "v0.3.3"),
			"v0.3.3 is mainnet's last completed plan; with no handler every mainnet node aborts on start")
	})

	// The testnet, and the price of the table's scope: this binary is not built
	// to run there. This one must FAIL. If it ever starts passing, a name came
	// back into the router -- check whether the fence in app/upgrade.go came
	// down with it. Height is the testnet's real applied_plan/v0.4.1.
	t.Run("testnet is deliberately not served", func(t *testing.T) {
		err := preBlocker(t, 14_942_340, "v0.4.1")
		require.Error(t, err,
			"v0.4.1 must not be registered: this binary is scoped to mainnet")
		require.Contains(t, err.Error(), "v0.4.1",
			"the failure has to name the missing plan; that string is the operator's only signal")
	})
}
