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
//	chain      last completed   this binary (v0.5.0)   next binary (v0.5.1)
//	mainnet    v0.3.3           served                 must be served
//	testnet    v0.4.1           NOT served             must be served
//
// Mainnet becomes served by v0.5.1 for a different reason than testnet: after it
// runs its own v0.5.0 plan, v0.5.0 is its LAST COMPLETED name, so the next
// binary owes it a handler for exactly the name this one executes. Testnet is not
// served here because it is not meant to be: it stays on v0.4.1 and upgrades
// straight to v0.5.1, so its last completed name is v0.4.1 all the way through.
//
// The two sub-cases are deliberately opposites. One asserts a chain is served,
// the other asserts a chain is not. Do not "fix" the second by registering
// v0.4.1 in THIS release; read the note on `router` instead.
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

	// The testnet, and the price of this release's scope. Testnet does not
	// upgrade in this release at all -- it waits on v0.4.1 and follows a plan
	// named v0.5.1 later -- so this one must FAIL here. If it ever starts
	// passing, a name came back into the router: check whether the fence in
	// app/upgrade.go came down with it, and whether v0.5.1 landed first. Height
	// is the testnet's real applied_plan/v0.4.1.
	t.Run("testnet is deliberately not served", func(t *testing.T) {
		err := preBlocker(t, 14_942_340, "v0.4.1")
		require.Error(t, err,
			"v0.4.1 must not be registered: this binary is scoped to mainnet, and v0.5.1 is the release that adds it back")
		require.Contains(t, err.Error(), "v0.4.1",
			"the failure has to name the missing plan; that string is the operator's only signal")
	})

	// The fence itself, stated as a boot-level fact rather than as an assertion
	// about the router's map: a chain whose LAST completed plan is v0.4.0 could
	// not start on this binary. No live chain is in that position (testnet's
	// last completed is v0.4.1, mainnet's is v0.3.3), which is why the absence of
	// v0.4.0 from the table costs nothing -- and it is the reason the name can be
	// withheld without breaking either chain.
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
// The control case is the state every node is in while a proposal is pending:
// the running binary has no handler for the plan's name, so nothing halts and
// the chain keeps producing blocks. It uses "v0.5.1" because that is the next
// plan name, and neither chain's current binary is the v0.5.1 release.
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
		err := schedule(t, "v0.5.0")
		require.Error(t, err,
			"a node that swaps in the new binary before the upgrade height must halt, not run")
		require.Contains(t, err.Error(), "BINARY UPDATED BEFORE TRIGGER",
			"the message is the operator's only clue that the fix is to put the old binary back")
	})

	t.Run("an unregistered plan leaves the running binary alone", func(t *testing.T) {
		require.NoError(t, schedule(t, "v0.5.1"),
			"while a plan is pending, the binary that does not carry its handler has to keep running normally")
	})
}
