package app

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/app/upgrades"
)

// Regression: setupUpgradeStoreLoaders used to swallow *every* error from
// ReadUpgradeInfoFromDisk. A corrupt or half-written upgrade-info.json would
// then be treated as "no upgrade scheduled", the store loader would never be
// installed and the node would boot straight into an inconsistent state.
// Only "file does not exist" is benign now.
func TestIsMissingUpgradeInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "plain not-exist",
			err:  os.ErrNotExist,
			want: true,
		},
		{
			name: "wrapped not-exist",
			err:  fmt.Errorf("reading %s: %w", "/root/.uptickd/data/upgrade-info.json", os.ErrNotExist),
			want: true,
		},
		{
			name: "raw path error",
			err:  &os.PathError{Op: "open", Path: "/root/.uptickd/data/upgrade-info.json", Err: os.ErrNotExist},
			want: true,
		},
		{
			name: "corrupt json",
			err:  errors.New("invalid character 'x' looking for beginning of value"),
			want: false,
		},
		{
			name: "permission denied",
			err:  &os.PathError{Op: "open", Path: "upgrade-info.json", Err: os.ErrPermission},
			want: false,
		},
		{
			name: "wrapped parse error",
			err:  fmt.Errorf("unmarshal plan: %w", errors.New("unexpected EOF")),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isMissingUpgradeInfo(tc.err))
		})
	}
}

// The router must expose every registered upgrade, and duplicates must be
// rejected rather than silently overwriting an existing handler.
func TestUpgradeRouter_Registration(t *testing.T) {
	t.Parallel()

	router := upgrades.NewUpgradeRouter()
	require.Empty(t, router.Routers())

	names := []string{"v0.3.3", "v0.4.0", "v0.4.1"}
	for _, name := range names {
		u := upgrades.Upgrade{UpgradeName: name}
		router.Register(u)
		require.Equal(t, u, router.UpgradeInfo(name))
	}

	require.Len(t, router.Routers(), len(names))
	require.Empty(t, router.UpgradeInfo("v9.9.9").UpgradeName)

	require.PanicsWithValue(t, "v0.4.1 already registered", func() {
		router.Register(upgrades.Upgrade{UpgradeName: "v0.4.1"})
	}, "re-registering an upgrade name must be loud, not a silent overwrite")
}

// The app's own router must carry exactly the plans this binary is meant to
// serve. The two halves below fail in opposite directions on purpose: one
// catches a name that went missing, the other catches a name that came back.
func TestAppRouterRegistration(t *testing.T) {
	t.Parallel()

	// Present: a binary that cannot name the last completed upgrade of the chain
	// it runs on fails x/upgrade's startup self-check and never starts, and a
	// plan name it cannot name is a plan it refuses to execute. A missing entry
	// is not a missing migration -- it is a node that will not boot.
	for _, name := range []string{"v0.3.3", "v0.5.0"} {
		require.Equal(t, name, router.UpgradeInfo(name).UpgradeName,
			"the app router must register %s", name)
	}

	// Absent: this is the fence, not an oversight. v0.4.0 and v0.4.1 are both
	// still schedulable on mainnet (neither has a done record there) and neither
	// handler is safe against v0.3.3 state -- v0.4.0's legacy-pair deletion is
	// not idempotent, and v0.4.1 reads cosmos/evm EVM params out of a legacy
	// ethermint store. Unregistered, such a plan halts at the upgrade height
	// instead of reaching ApplyUpgrade. Registering either one again is not a
	// fix; it re-opens the hole.
	for _, name := range []string{"v0.4.0", "v0.4.1"} {
		require.Empty(t, router.UpgradeInfo(name).UpgradeName,
			"%s must not be registered: it is schedulable on mainnet and its handler is "+
				"destructive (v0.4.0) or unrunnable (v0.4.1) on a v0.3.3 chain", name)
	}

	require.Len(t, router.Routers(), 2,
		"the table is scoped to the chains this release serves, not to the whole bloodline")

	// The capability-store deletion travels with the plan name: a mainnet node
	// follows "v0.5.0" and consults nothing else, so a v0.5.0 entry without the
	// deletion would boot the node with the legacy store still mounted.
	require.Equal(t, []string{"capability"}, router.UpgradeInfo("v0.5.0").StoreUpgrades.Deleted)
}
