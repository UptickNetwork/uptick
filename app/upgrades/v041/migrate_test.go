package v041

import (
	"encoding/hex"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
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

// ---------------------------------------------------------------------------
// feemarket base fee rescue
// ---------------------------------------------------------------------------

// legacyFeeMarketParamsHex is the exact protobuf the pre-upgrade chain has
// stored at feemarkettypes.ParamsKey. It was produced by ethermint
// v0.24.1-uptick's types.Params, which encodes base_fee as cosmossdk.io/math.Int:
//
//	10 08                                   field 2  varint  base_fee_change_denominator = 8
//	18 02                                   field 3  varint  elasticity_multiplier     = 2
//	28 00                                   field 5  varint  enable_height              = 0
//	32 0a 31 30 ... 30                      field 6  bytes   base_fee    = "1000000000"
//	3a 01 30                                field 7  bytes   min_gas_price = "0"
//	42 12 35 30 ... 30                      field 8  bytes   min_gas_multiplier = "5e17"
//
// The field-6 payload is the ASCII of the math.Int, which is also how
// math.LegacyDec marshals its raw big.Int - the two encodings are the same
// bytes and only the field's declared Go type says how to read them.
const legacyFeeMarketParamsHex = "100818022800320a313030303030303030303a01304212353030303030303030303030303030303030"

func legacyFeeMarketParams(t *testing.T) []byte {
	t.Helper()
	bz, err := hex.DecodeString(legacyFeeMarketParamsHex)
	require.NoError(t, err)
	return bz
}

// TestLegacyFeeMarketBaseFeeDecodesSmallerByOneE18 is the pin on WHY
// migrateFeeMarketBaseFee exists. It is deliberately written against the real
// cosmos/evm Params type rather than a restatement of the constants, so that if
// cosmos/evm ever restores math.Int (or changes LegacyDec's wire form) this test
// fails and tells the next reader the migration has become obsolete rather than
// merely unnecessary.
func TestLegacyFeeMarketBaseFeeDecodesSmallerByOneE18(t *testing.T) {
	var params feemarkettypes.Params
	require.NoError(t, params.Unmarshal(legacyFeeMarketParams(t)))

	// 1 gwei, the value the ethermint chain actually had, reads back as 10^-9.
	require.Equal(t, "0.000000001000000000", params.BaseFee.String())

	require.Equal(t, feemarkettypes.DefaultBaseFee.String(), "1000000000.000000000000000000",
		"the intended value must stay the module default; if this changes the "+
			"smaller-by-10^18 premise of the migration needs re-deriving")

	// The other two floats are math.LegacyDec in BOTH versions, so they are
	// unaffected - which is also why they cannot be used to detect the mismatch.
	require.Equal(t, "0.000000000000000000", params.MinGasPrice.String())
	require.Equal(t, "0.500000000000000000", params.MinGasMultiplier.String())

	// Nothing about the value makes GetParams fail; the mis-read is silent.
	require.True(t, params.BaseFee.IsPositive(), "the corrupt value is not an error state")
	require.True(t, params.BaseFee.LT(math.LegacyOneDec()), "it is below one whole wei")
}

func TestWithRepairedBaseFee_RescalesLegacyEncoding(t *testing.T) {
	// Exactly what the legacy bytes decode to.
	legacy := feemarkettypes.DefaultParams()
	legacy.BaseFee = math.LegacyNewDecWithPrec(1, 9) // 0.000000001

	repaired, changed := withRepairedBaseFee(legacy)

	require.True(t, changed)
	require.Equal(t, feemarkettypes.DefaultBaseFee.String(), repaired.BaseFee.String(),
		"1 gwei stored as math.Int must come back as 1 gwei in math.LegacyDec")
}

// TestWithRepairedBaseFee_IsIdempotent is the property the missing
// UpgradeAlreadyApplied guard depends on. The rescale is not idempotent on its
// own (a second application would multiply by another 10^18), so the guard is
// what makes a replayed plan safe.
func TestWithRepairedBaseFee_IsIdempotent(t *testing.T) {
	legacy := feemarkettypes.DefaultParams()
	legacy.BaseFee = math.LegacyNewDecWithPrec(1, 9)

	once, changed := withRepairedBaseFee(legacy)
	require.True(t, changed)

	twice, changedAgain := withRepairedBaseFee(once)
	require.False(t, changedAgain, "an already repaired value must not be rescaled again")
	require.Equal(t, once.BaseFee.String(), twice.BaseFee.String())
}

// TestWithRepairedBaseFee_LeavesPlausibleValuesAlone is the sentinel half of the
// guard: it must not rewrite state that is already correct, in particular the
// values a governance MsgUpdateParams or a fresh genesis can produce.
func TestWithRepairedBaseFee_LeavesPlausibleValuesAlone(t *testing.T) {
	for name, fee := range map[string]math.LegacyDec{
		"module default": feemarkettypes.DefaultBaseFee,
		"one whole wei":  math.LegacyOneDec(),
		"1 gwei again":   math.LegacyNewDec(1_000_000_000),
		"100 gwei":       math.LegacyNewDec(100_000_000_000),
		"zero":           math.LegacyZeroDec(),
		"nil":            {},
	} {
		t.Run(name, func(t *testing.T) {
			params := feemarkettypes.DefaultParams()
			params.BaseFee = fee

			repaired, changed := withRepairedBaseFee(params)

			require.False(t, changed, "%s must not be treated as legacy-encoded", name)
			require.Equal(t, fee.String(), repaired.BaseFee.String())
		})
	}
}

// recordingFeeMarketParamsStore records calls instead of only the final state,
// for the same reason as recordingICAParamsStore: writing the repaired value
// over the legacy value is indistinguishable from not writing at all when only
// the store contents are inspected.
type recordingFeeMarketParamsStore struct {
	params  feemarkettypes.Params
	writes  []feemarkettypes.Params
	failSet error
}

func (s *recordingFeeMarketParamsStore) GetParams(sdk.Context) feemarkettypes.Params {
	return s.params
}

// SetParams mirrors the real keeper's read-your-writes behavior so a second call
// observes the first one's write.
func (s *recordingFeeMarketParamsStore) SetParams(_ sdk.Context, params feemarkettypes.Params) error {
	if s.failSet != nil {
		return s.failSet
	}
	s.writes = append(s.writes, params)
	s.params = params
	return nil
}

func legacyFeeMarketStore(t *testing.T) *recordingFeeMarketParamsStore {
	t.Helper()
	var params feemarkettypes.Params
	require.NoError(t, params.Unmarshal(legacyFeeMarketParams(t)))
	return &recordingFeeMarketParamsStore{params: params}
}

func TestMigrateFeeMarketBaseFee_RepairsLegacyValue(t *testing.T) {
	store := legacyFeeMarketStore(t)
	require.Equal(t, "0.000000001000000000", store.params.BaseFee.String())

	require.NoError(t, migrateFeeMarketBaseFee(icaTestCtx(), store))

	require.Len(t, store.writes, 1)
	require.Equal(t, feemarkettypes.DefaultBaseFee.String(), store.params.BaseFee.String())
}

func TestMigrateFeeMarketBaseFee_NoWriteWhenAlreadyRepaired(t *testing.T) {
	// A fresh chain's genesis, or a second execution of the same plan.
	store := &recordingFeeMarketParamsStore{params: feemarkettypes.DefaultParams()}

	require.NoError(t, migrateFeeMarketBaseFee(icaTestCtx(), store))

	require.Empty(t, store.writes, "a plausible base fee is left untouched")
}

// TestMigrateFeeMarketBaseFee_SetParamsErrorPropagates pins that a rejected
// write fails the upgrade instead of being reported as a successful migration.
func TestMigrateFeeMarketBaseFee_SetParamsErrorPropagates(t *testing.T) {
	store := legacyFeeMarketStore(t)
	store.failSet = errSetFeeMarketParams

	err := migrateFeeMarketBaseFee(icaTestCtx(), store)

	require.Error(t, err)
	require.ErrorIs(t, err, errSetFeeMarketParams)
	require.Contains(t, err.Error(), "set feemarket params",
		"the wrap must name the step so an operator can tell which repair failed")
}

var errSetFeeMarketParams = errors.New("feemarket keeper rejected the params")
