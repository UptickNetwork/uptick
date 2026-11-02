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
// The chain roll-out this table encodes:
//
//	chain      last completed   this binary (v0.5.1)
//	mainnet    v0.5.0           served
//	testnet    v0.4.1           served
//
// v0.5.1 is the first binary scoped to BOTH chains: mainnet reaches it from
// v0.5.0, testnet jumps v0.4.1 -> v0.5.1 without ever running a v0.5.0 plan.
// Both last-completed names therefore have to resolve to a handler here, or
// the respective nodes abort on start. v0.3.3 stays registered as well: it is
// mainnet's pre-v0.5.0 last-completed name and the router never drops a name a
// live chain may still carry in applied_plan. v0.4.0 remains unregistered on
// purpose: its legacy-pair deletion is not idempotent, and no live chain's last
// completed plan is v0.4.0 (testnet passed through it to v0.4.1 long ago).
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

	// Mainnet's last completed plan after its own v0.5.0 execution; the name
	// this binary must serve for mainnet to reach v0.5.1.
	t.Run("mainnet is served at v0.5.0", func(t *testing.T) {
		require.NoError(t, preBlocker(t, 19_500_000, "v0.5.0"),
			"v0.5.0 is mainnet's last completed plan; with no handler every mainnet node aborts on start")
	})

	// Historical mainnet name: nodes that recorded v0.3.3 as their last
	// completed plan (pre-v0.5.0 window) must still boot this binary.
	t.Run("mainnet historical v0.3.3 stays served", func(t *testing.T) {
		require.NoError(t, preBlocker(t, 18_147_950, "v0.3.3"),
			"v0.3.3 is a done record mainnet nodes still carry; the router never drops a served name")
	})

	// Testnet jumps v0.4.1 -> v0.5.1 without ever running a v0.5.0 plan, so
	// v0.4.1 is its last completed name and MUST boot this binary. This
	// sub-case used to assert the opposite under the v0.5.0 binary's scope.
	// Height is the testnet's real applied_plan/v0.4.1.
	t.Run("testnet is served at v0.4.1", func(t *testing.T) {
		require.NoError(t, preBlocker(t, 14_942_340, "v0.4.1"),
			"v0.4.1 is the testnet's last completed plan; v0.5.1 upgrades testnet straight from it")
	})

	// The fence itself, stated as a boot-level fact rather than as an assertion
	// about the router's map: a chain whose LAST completed plan is v0.4.0 could
	// not start on this binary. No live chain is in that position (testnet's
	// last completed is v0.4.1, mainnet's is v0.5.0), which is why the absence
	// of v0.4.0 from the table costs nothing -- and it is the reason the name
	// can be withheld without breaking either chain.
	t.Run("v0.4.0 is not a served name either", func(t *testing.T) {
		err := preBlocker(t, 14_607_000, "v0.4.0")
		require.Error(t, err)
		require.Contains(t, err.Error(), "v0.4.0")
	})
}

// TestBootSelfCheckRefusesToRunTheNewBinaryBeforeItsHeight pins the operational
// rule that governs how this binary reaches a validator, which no document in
// the repo stated before this test.
//
// x/upgrade's PreBlocker does not merely apply a plan at its height: on every
// block where a plan is PENDING AND NOT YET DUE, it refuses to run at all if the
// binary already carries that plan's handler (cosmossdk.io/x/upgrade@v0.2.0/
// abci.go:117-125, "BINARY UPDATED BEFORE TRIGGER"). So the new binary cannot be
// started early: the old one has to reach the upgrade height first, and the swap
// happens there -- which is exactly what cosmovisor automates. Swapping by hand
// ahead of the height halts the node, and the recovery is to put the old binary
// back.
//
// The control case is the state every node is in while the v0.5.1 proposal is
// pending: the running binary (v0.5.0) has no handler for that name, so nothing
// halts and the chain keeps producing blocks. It uses "v0.4.0" as the
// unregistered name because that name is deliberately absent from this binary's
// router (see `router` in app/upgrade.go); "v0.5.1" no longer qualifies --
// THIS binary is the one that carries it.
func TestBootSelfCheckRefusesToRunTheNewBinaryBeforeItsHeight(t *testing.T) {
	app, ctx := sharedTestApp(t)

	// schedule plans a plan that is not due yet, the way a passed governance
	// proposal leaves the chain until its height arrives.
	schedule := func(t *testing.T, name string) error {
		t.Helper()

		require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{
			Name:   name,
			Height: ctx.BlockHeight() + 1_000_000,
			Info:   "test: pending, not due",
		}))
		t.Cleanup(func() { require.NoError(t, app.UpgradeKeeper.ClearUpgradePlan(ctx)) })

		// Same process-lifetime flag as above: without the reset the self-check
		// block is skipped and the case below would pass while testing nothing.
		app.UpgradeKeeper.SetDowngradeVerified(false)
		t.Cleanup(func() { app.UpgradeKeeper.SetDowngradeVerified(false) })

		_, err := upgrade.PreBlocker(ctx, app.UpgradeKeeper)
		return err
	}

	t.Run("a registered plan refuses an early swap", func(t *testing.T) {
		err := schedule(t, "v0.5.1")
		require.Error(t, err,
			"a node that swaps in this binary before the v0.5.1 upgrade height must halt, not run")
		require.Contains(t, err.Error(), "BINARY UPDATED BEFORE TRIGGER",
			"the message is the operator's only clue that the fix is to put the old binary back")
	})

	t.Run("an unregistered plan leaves the running binary alone", func(t *testing.T) {
		require.NoError(t, schedule(t, "v0.4.0"),
			"while a plan is pending, the binary that does not carry its handler has to keep running normally")
	})
}
