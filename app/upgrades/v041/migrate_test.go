package v041

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
)

// failingEVMStore drives the SetParams failure branch of
// migrateActiveStaticPrecompiles. The real keeper cannot produce that error for
// this migration (it always writes a valid list), so without an injectable
// store the branch that decides whether a failed write is reported or swallowed
// would never run in CI.
type failingEVMStore struct {
	// populated is the stored precompile list GetParams reports. Empty means
	// "never configured", which is the case that triggers the write.
	populated []string
	written   evmtypes.Params
}

func (s *failingEVMStore) GetParams(sdk.Context) evmtypes.Params {
	return evmtypes.Params{ActiveStaticPrecompiles: slices.Clone(s.populated)}
}

func (s *failingEVMStore) SetParams(_ sdk.Context, params evmtypes.Params) error {
	s.written = params
	return errSetParams
}

var errSetParams = errors.New("evm keeper rejected the params")

func TestMigrateActiveStaticPrecompiles_SetParamsErrorPropagates(t *testing.T) {
	store := &failingEVMStore{}

	err := migrateActiveStaticPrecompiles(sdk.Context{}, store)

	require.Error(t, err, "a rejected write must not be reported as a successful migration")
	require.ErrorIs(t, err, errSetParams, "the keeper error must stay diagnosable through the wrap")
	require.Contains(t, err.Error(), "set evm params",
		"the wrap must name the step so an operator can tell which repair failed")
	require.Equal(t, defaultActiveStaticPrecompiles, store.written.ActiveStaticPrecompiles,
		"the migration must attempt to write the default list")
}

func TestMigrateActiveStaticPrecompiles_NoWriteWhenAlreadyPopulated(t *testing.T) {
	store := &failingEVMStore{}
	store.populated = []string{evmtypes.BankPrecompileAddress}

	require.NoError(t, migrateActiveStaticPrecompiles(sdk.Context{}, store),
		"an already-populated list is left alone, so the (failing) writer is never called")
	require.Empty(t, store.written.ActiveStaticPrecompiles)
}

func TestDefaultActiveStaticPrecompiles(t *testing.T) {
	require.Len(t, defaultActiveStaticPrecompiles, 8)

	// The vesting precompile (0x803) is not registered by Uptick and must not
	// be activated, otherwise GetStaticPrecompileInstance panics with
	// "precompiled contract not stored in memory".
	require.NotContains(t, defaultActiveStaticPrecompiles, evmtypes.VestingPrecompileAddress)

	for _, addr := range []string{
		evmtypes.P256PrecompileAddress,
		evmtypes.Bech32PrecompileAddress,
		evmtypes.StakingPrecompileAddress,
		evmtypes.DistributionPrecompileAddress,
		evmtypes.ICS20PrecompileAddress,
		evmtypes.BankPrecompileAddress,
		evmtypes.GovPrecompileAddress,
		evmtypes.SlashingPrecompileAddress,
	} {
		require.Contains(t, defaultActiveStaticPrecompiles, addr)
	}
}

func TestConfigureDefaultStaticPrecompiles(t *testing.T) {
	// ConfigureDefaultStaticPrecompiles writes to a package-level variable in
	// the EVM module - process-wide state shared with every other test in this
	// binary, including other packages' (evmtypes.DefaultParams() reads it).
	// Leaving it mutated makes failures depend on test ordering, so the
	// original value is captured and restored. slices.Clone preserves the
	// nil-vs-empty distinction instead of collapsing both to nil.
	original := slices.Clone(evmtypes.DefaultStaticPrecompiles)
	t.Cleanup(func() {
		evmtypes.DefaultStaticPrecompiles = original
	})

	// The app wires this explicitly instead of a package init(): the call
	// must set the EVM module default deterministically.
	ConfigureDefaultStaticPrecompiles()
	require.Equal(t, defaultActiveStaticPrecompiles, evmtypes.DefaultStaticPrecompiles)
}

func TestWithDefaultActiveStaticPrecompiles(t *testing.T) {
	empty := evmtypes.Params{}
	filled, changed := withDefaultActiveStaticPrecompiles(empty)
	require.True(t, changed)
	require.Equal(t, defaultActiveStaticPrecompiles, filled.ActiveStaticPrecompiles)

	existing := []string{evmtypes.BankPrecompileAddress}
	params := evmtypes.Params{ActiveStaticPrecompiles: existing}
	unchanged, changed := withDefaultActiveStaticPrecompiles(params)
	require.False(t, changed)
	require.Equal(t, existing, unchanged.ActiveStaticPrecompiles)
}

// recordingICAParamsStore drives the guard in migrateICAControllerParams.
//
// The real ICA controller keeper cannot express the question these tests ask.
// ibc-go's SetParams returns nothing, and writing true over an already-true
// param leaves byte-identical state behind, so "the migration wrote when it
// should have skipped" is invisible in the store. Counting the calls is the
// only way to make dropping the guard fail a test.
type recordingICAParamsStore struct {
	params icacontrollertypes.Params
	writes []icacontrollertypes.Params
}

func (s *recordingICAParamsStore) GetParams(sdk.Context) icacontrollertypes.Params {
	return s.params
}

// SetParams mirrors the real keeper's read-your-writes behavior so a second
// call observes the first one's flip.
func (s *recordingICAParamsStore) SetParams(_ sdk.Context, params icacontrollertypes.Params) {
	s.writes = append(s.writes, params)
	s.params = params
}

// icaTestCtx is a context whose Logger() is safe to call.
// migrateICAControllerParams logs on the write path, and sdk.Context{}.Logger()
// returns a nil logger whose Info() panics; the real handler always runs on a
// context that has one.
func icaTestCtx() sdk.Context {
	return sdk.Context{}.WithLogger(log.NewNopLogger())
}

// TestMigrateICAControllerParams_EnablesWhenDisabled is the case the migration
// exists for: genesis templates derived from the legacy x/params defaults ship
// controller_enabled=false, and every ICA registration fails with "controller
// submodule is disabled" until it is flipped.
func TestMigrateICAControllerParams_EnablesWhenDisabled(t *testing.T) {
	store := &recordingICAParamsStore{params: icacontrollertypes.NewParams(false)}

	migrateICAControllerParams(icaTestCtx(), store)

	require.Len(t, store.writes, 1, "a disabled controller must be enabled")
	require.True(t, store.writes[0].ControllerEnabled)
}

// TestMigrateICAControllerParams_SkipsWriteWhenAlreadyEnabled is the guard.
// A fresh chain already stores controller_enabled=true. Deleting the early
// return in migrateICAControllerParams makes this test red - which it was not
// before, because the store cannot tell the two runs apart.
func TestMigrateICAControllerParams_SkipsWriteWhenAlreadyEnabled(t *testing.T) {
	store := &recordingICAParamsStore{params: icacontrollertypes.NewParams(true)}

	migrateICAControllerParams(icaTestCtx(), store)

	require.Empty(t, store.writes, "an already-enabled controller must be left untouched")
}

// TestMigrateICAControllerParams_IsIdempotent pins the property the missing
// UpgradeAlreadyApplied guard relies on: a replayed plan leaves the store where
// a single run would have. The second call goes through the real transition
// (disabled -> enabled on the first pass) rather than starting from the steady
// state, which is what a crash-restart of the upgrade actually does.
func TestMigrateICAControllerParams_IsIdempotent(t *testing.T) {
	store := &recordingICAParamsStore{params: icacontrollertypes.NewParams(false)}

	migrateICAControllerParams(icaTestCtx(), store)
	migrateICAControllerParams(icaTestCtx(), store)

	require.Len(t, store.writes, 1, "the second run must observe the flip and skip")
	require.True(t, store.params.ControllerEnabled)
}
