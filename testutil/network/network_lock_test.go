package network_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/testutil/network"
)

// discardLogger is the minimal Logger the validation branches need: they log
// before validating, so it has to exist even when no network is ever built.
type discardLogger struct{}

func (discardLogger) Log(...interface{})          {}
func (discardLogger) Logf(string, ...interface{}) {}

// The package-level lock is released by Network.Cleanup, not by New, because a
// network owns the process for its whole lifetime. That makes every early
// return in New responsible for releasing it by hand. A rejected config used to
// leave the lock held forever, and the next New in the same process would block
// on lock.Lock() — a hang with no failing assertion, in a package that is on
// the uptickd binary's dependency graph.
//
// This asserts the lock is actually released by calling New a second time and
// requiring it to reach the same validation error within a bounded time. The
// bound is what turns a deadlock into a failure instead of a hung test binary.
func TestNewReleasesLockOnEarlyReturn(t *testing.T) {
	cfg := network.DefaultConfig()
	cfg.NumValidators = 2 // rejected before any network is built

	_, err := network.New(discardLogger{}, t.TempDir(), cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "1 validator")

	done := make(chan error, 1)
	go func() {
		// No require here: assertions must run on the test goroutine.
		_, err := network.New(discardLogger{}, t.TempDir(), cfg)
		done <- err
	}()

	select {
	case err := <-done:
		require.Error(t, err, "the same config must be rejected again")
	case <-time.After(5 * time.Second):
		t.Fatal("network.New blocked on the package lock: the earlier rejected call never released it")
	}
}
