package upgrades

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
)

type fakeCVModule struct{ v uint64 }

func (f fakeCVModule) ConsensusVersion() uint64 { return f.v }

// fakeNoCVModule models light-client / genesis-only modules (e.g. ibc-go's
// 06-solomachine and 07-tendermint AppModules) that do not expose a
// ConsensusVersion.
type fakeNoCVModule struct{}

// UpgradeAlreadyApplied must return true even when the module manager contains
// modules without a ConsensusVersion (light-client / genesis-only AppModules);
// those modules are skipped rather than blocking the guard.
func TestAudit_UpgradeGuardReachable(t *testing.T) {
	b := Toolbox{
		ModuleManager: &module.Manager{Modules: map[string]any{
			"withcv":         fakeCVModule{v: 3},
			"07-tendermint":  fakeNoCVModule{},
			"06-solomachine": fakeNoCVModule{},
		}},
	}

	// All versioned modules at their current consensus version — the guard
	// must be reachable and report "applied" even though two modules have no
	// consensus version at all.
	require.True(t, b.UpgradeAlreadyApplied(map[string]uint64{"withcv": 3}))

	// Any versioned module below its current version means the upgrade has
	// not run: guard must stay false.
	require.False(t, b.UpgradeAlreadyApplied(map[string]uint64{"withcv": 2}))

	// A versioned module missing from the incoming version map also means the
	// upgrade has not run.
	require.False(t, b.UpgradeAlreadyApplied(map[string]uint64{"other": 1}))

	// Empty map: nothing migrated yet.
	require.False(t, b.UpgradeAlreadyApplied(nil))
}
