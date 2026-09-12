package v041

import (
	"encoding/hex"
	"errors"
	"reflect"
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

// TestMigrateICAControllerParams_WritesBackEveryFieldRead is the fidelity check
// on the write migrateICAControllerParams performs. It fills EVERY field of
// icacontrollertypes.Params with a non-zero sentinel, sets the controller flag
// back to false, runs the migration, and requires the stored value to equal the
// read value with exactly that one field flipped. A whole-struct rebuild
// (icacontrollertypes.NewParams(true)) would instead zero every other field, so
// when ibc-go adds a second field this comparison fails and poses the real
// question -- "what should the new field be during this migration?" -- instead
// of a field count that a human would have to remember to bump.
//
// Sensitivity ceiling, stated honestly: today icacontrollertypes.Params has
// exactly ONE field, and that field is precisely the one the migration flips,
// so this assertion CANNOT fail today -- there is no other field to drop. Its
// teeth are on the SECOND and later fields. Until such a field appears, NO test
// in this package can tell a whole-struct rebuild apart from a read-modify-write
// (revert the write in upgrades.go to icacontrollertypes.NewParams(true) and the
// suite still passes). TestParamsWriteBackComparisonIsFieldSensitive proves only
// that the comparison predicate is not a tautology; it reads nothing from the
// production code, so it is not a guard.
//
// Two blind spots belong here rather than in a review comment. The assertion
// only sees fields that exist, so it says nothing at all until the field it
// would drop has been added. And it catches a pointer or non-[]byte slice field
// being dropped to nil or to a zero-length value, but not one being swapped for
// a freshly allocated zero-valued object -- a fresh pointer, or a fresh slice of
// the SAME length, compares equal to the sentinel.
//
// A field-count canary -- pinning
// reflect.TypeOf(icacontrollertypes.Params{}).NumField() -- was considered and
// dropped: once a second field exists this test already goes red if the write
// stops carrying fields through, so a count canary would add no detection, only
// a red build on the day ibc-go adds a field while the write is still correct. A
// gate that fires on correct code is how gates get bypassed.
func TestMigrateICAControllerParams_WritesBackEveryFieldRead(t *testing.T) {
	sentinel := fillNonZeroStruct(t, reflect.TypeOf(icacontrollertypes.Params{}))

	ctrl := sentinel.FieldByName("ControllerEnabled")
	require.True(t, ctrl.IsValid(),
		"icacontrollertypes.Params no longer has ControllerEnabled; the migration and this test must be revisited")
	ctrl.SetBool(false)

	store := &recordingICAParamsStore{params: sentinel.Interface().(icacontrollertypes.Params)}

	migrateICAControllerParams(icaTestCtx(), store)

	require.Len(t, store.writes, 1, "a disabled controller must be enabled")

	want := sentinel.Interface().(icacontrollertypes.Params)
	want.ControllerEnabled = true

	require.Equal(t, want, store.writes[0],
		"the write-back must be the value just read with exactly one field "+
			"flipped; failing here means the write did not carry through every "+
			"field it read -- a whole-struct construction "+
			"(icacontrollertypes.NewParams(true)) zeroes every field it does not "+
			"set, including any ibc-go adds later")
}

// fillNonZeroStruct returns a value of typ with every field set to a non-zero
// sentinel, so that a field silently dropped by a write shows up in an equality
// check. Whenever it cannot fill a field it FAILS rather than leaving that field
// zero: a silently-zero field would be a false negative on exactly the drift
// this helper exists to make visible, and a false negative is invisible while a
// failure is not.
//
// Failure is reachable two ways, and neither is a mechanical fix:
//
//   - a kind this helper cannot build a non-zero value for;
//   - an unexported field reached by recursing into a struct -- math.Int and
//     time.Time are both built that way, and no reflection can set them.
//
// The second case is a decision, not a missing case: that field has to be
// compared some other way (its own Equal or String, usually) and someone has to
// decide which. A failure names the field by a dotted path rooted at the type it
// was asked to fill, so a nested one reads e.g. "types.Params.Amount.i" rather
// than a bare "i".
func fillNonZeroStruct(t *testing.T, typ reflect.Type) reflect.Value {
	t.Helper()

	return fillNonZeroStructAt(t, typ, typ.String())
}

// fillNonZeroStructAt is fillNonZeroStruct with a running path. The root comes
// from typ.String(), which is the package NAME and not the import path: for
// icacontrollertypes.Params it reads "types.Params", and several packages in this
// module are named "types". Read the path as orientation for locating the field,
// not as a unique identifier.
func fillNonZeroStructAt(t *testing.T, typ reflect.Type, path string) reflect.Value {
	t.Helper()

	v := reflect.New(typ).Elem()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fv := v.Field(i)
		at := path + "." + field.Name
		if !fv.CanSet() {
			t.Fatalf("fillNonZeroStruct: %s is unexported, so no reflection can put a "+
				"sentinel in it; leaving it zero would make this comparison blind to "+
				"a write that drops it. This needs a decision rather than a new case: "+
				"choose how that wrapper should be compared (its own Equal or String, "+
				"usually) and teach this helper.", at)
		}
		switch fv.Kind() {
		case reflect.Bool:
			fv.SetBool(true)
		case reflect.String:
			fv.SetString("sentinel")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fv.SetInt(1)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fv.SetUint(1)
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.Uint8 {
				fv.SetBytes([]byte{0x01})
			} else {
				fv.Set(reflect.MakeSlice(field.Type, 1, 1))
			}
		case reflect.Struct:
			fv.Set(fillNonZeroStructAt(t, field.Type, at))
		case reflect.Ptr:
			fv.Set(reflect.New(field.Type.Elem()))
		default:
			t.Fatalf("fillNonZeroStruct: %s has kind %s, which this helper cannot build "+
				"a non-zero value for; leaving it zero would make this comparison blind "+
				"to a write that drops it. Teach this helper that kind.", at, fv.Kind())
		}
	}
	return v
}

// TestParamsWriteBackComparisonIsFieldSensitive proves the fidelity comparison
// above is not vacuously true. It plays the same "sentinel + flip + compare"
// shape against a multi-field stand-in struct, twice: once as a read-modify-write
// and once as a whole-struct rebuild. The SAME comparison must accept the first
// and reject the second -- one predicate, two outcomes. What it does NOT claim
// is that the fidelity assertion above discriminates today: with a single field
// in icacontrollertypes.Params it passes under either write. The control only
// shows the comparison has a false outcome available to it, which is what makes
// the assertion worth having once a second field exists.
//
// The stand-in takes icahosttypes.Params' shape -- a bool plus a []string, the
// one other params message in ibc-go's ICA modules -- and adds a second scalar
// so that a String-branch field exists too. It is shaped for the branches it has
// to reach, not for convenience. Routing it through the SAME helper the fidelity
// test uses is the point: it exercises the multi-field path and the String and
// non-byte-Slice branches, which a one-field icacontrollertypes.Params cannot
// reach.
func TestParamsWriteBackComparisonIsFieldSensitive(t *testing.T) {
	type standInParams struct {
		ControllerEnabled bool
		Extra             string
		AllowMessages     []string
	}

	// Fill the stand-in through the SAME helper the fidelity test uses, so its
	// multi-field path and the String and Slice branches actually execute.
	sentinel := fillNonZeroStruct(t, reflect.TypeOf(standInParams{}))
	sentinel.FieldByName("ControllerEnabled").SetBool(false)

	want := sentinel.Interface().(standInParams)
	want.ControllerEnabled = true

	// same mirrors require.Equal's predicate for a non-[]byte struct
	// (reflect.DeepEqual), so the control exercises the real comparison.
	same := func(a, b standInParams) bool { return reflect.DeepEqual(a, b) }

	// (i) Read-modify-write: the value read, with only the bool flipped.
	readModifyWrite := sentinel.Interface().(standInParams)
	readModifyWrite.ControllerEnabled = true

	// (ii) Whole-struct rebuild: only the bool is set; the other two fields are
	// left zero, which is exactly what icacontrollertypes.NewParams(true) does to
	// the fields it does not know about.
	wholeStruct := standInParams{ControllerEnabled: true}

	require.True(t, same(want, readModifyWrite),
		"read-modify-write carries the fields it read, so it must equal the sentinel with the bool flipped")
	require.False(t, same(want, wholeStruct),
		"a whole-struct rebuild drops every field it does not set; it must NOT "+
			"equal, or the fidelity assertion would assert nothing")
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
