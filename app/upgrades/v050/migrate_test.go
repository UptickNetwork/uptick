package v050

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/stretchr/testify/require"
)

const (
	// The voucher hash from the test-side analysis, whose source denom is auoc.
	voucherDenom  = "ibc/3D020BF987CDA6E232B092CB63F3BED461FB14C448481ABA943CDEED53A8183E"
	voucherPath   = "transfer/channel-1/auoc"
	voucherSource = "auoc"

	// A second voucher, micro-prefixed, so the two derivation branches are both
	// pinned.
	microVoucherDenom  = "ibc/9D35B24BF0F1B2C3D4E5F60718293A4B5C6D7E8F90A1B2C3D4E5F60718293A4B"
	microVoucherPath   = "transfer/channel-3/uusdc"
	microVoucherSource = "uusdc"
)

// ibcGoVoucherMetadata reproduces the metadata ibc-go v10.5.0 writes when a
// voucher first arrives (modules/apps/transfer/keeper/keeper.go:213-231): the
// full denom path in Display, and a single unit carrying the SOURCE denom with
// exponent 0. That single unit is why decimals() reads 0 for every voucher.
func ibcGoVoucherMetadata(voucherDenom, path, sourceDenom string) banktypes.Metadata {
	return banktypes.Metadata{
		Description: fmt.Sprintf("IBC token from %s", path),
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: sourceDenom, Exponent: 0},
		},
		Base:    voucherDenom,
		Display: path,
		Name:    fmt.Sprintf("%s IBC token", path),
		Symbol:  strings.ToUpper(sourceDenom),
	}
}

// fakeVoucherBank is the slice of the bank keeper the migrations use.
type fakeVoucherBank struct {
	supply   []sdk.Coin
	metadata map[string]banktypes.Metadata
	writes   []banktypes.Metadata
}

func newFakeVoucherBank(supply ...sdk.Coin) *fakeVoucherBank {
	return &fakeVoucherBank{supply: supply, metadata: map[string]banktypes.Metadata{}}
}

func (f *fakeVoucherBank) IterateTotalSupply(_ context.Context, cb func(sdk.Coin) bool) {
	for _, coin := range f.supply {
		if cb(coin) {
			return
		}
	}
}

func (f *fakeVoucherBank) GetDenomMetaData(_ context.Context, denom string) (banktypes.Metadata, bool) {
	metadata, found := f.metadata[denom]
	return metadata, found
}

func (f *fakeVoucherBank) SetDenomMetaData(_ context.Context, metadata banktypes.Metadata) {
	f.writes = append(f.writes, metadata)
	f.metadata[metadata.Base] = metadata
}

// fakeVoucherPairs is the slice of the erc20 keeper the backfill uses.
type fakeVoucherPairs struct {
	registered map[string]bool
	order      []string
	failOn     map[string]error
}

func newFakeVoucherPairs() *fakeVoucherPairs {
	return &fakeVoucherPairs{registered: map[string]bool{}, failOn: map[string]error{}}
}

func (f *fakeVoucherPairs) IsDenomRegistered(_ sdk.Context, denom string) bool {
	return f.registered[denom]
}

func (f *fakeVoucherPairs) RegisterERC20Extension(_ sdk.Context, denom string) (*erc20types.TokenPair, error) {
	if err := f.failOn[denom]; err != nil {
		return nil, err
	}
	f.registered[denom] = true
	f.order = append(f.order, denom)
	return &erc20types.TokenPair{Denom: denom, Erc20Address: "0x" + strings.Repeat("ab", 20)}, nil
}

// recordingLogger keeps every log line as text. The metadata rewrite discards the
// old DenomUnits, so the log line is the only place they survive -- asserting the
// audit trail needs a logger that remembers, not a nop one.
type recordingLogger struct {
	lines []string
}

func (l *recordingLogger) record(level, msg string, keyVals ...any) {
	l.lines = append(l.lines, fmt.Sprintf("%s %s %s", level, msg, formatKeyVals(keyVals)))
}

// formatKeyVals renders the pairs the way the real logger does (key=value), so
// the assertions read the same as the production log line an operator sees.
func formatKeyVals(keyVals []any) string {
	parts := make([]string, 0, len(keyVals)/2)
	for i := 0; i+1 < len(keyVals); i += 2 {
		parts = append(parts, fmt.Sprintf("%v=%v", keyVals[i], keyVals[i+1]))
	}
	return strings.Join(parts, " ")
}

func (l *recordingLogger) Info(msg string, keyVals ...any)  { l.record("info", msg, keyVals...) }
func (l *recordingLogger) Error(msg string, keyVals ...any) { l.record("error", msg, keyVals...) }
func (l *recordingLogger) Debug(msg string, keyVals ...any) { l.record("debug", msg, keyVals...) }
func (l *recordingLogger) Warn(msg string, keyVals ...any)  { l.record("warn", msg, keyVals...) }
func (l *recordingLogger) With(keyVals ...any) log.Logger   { return l }
func (l *recordingLogger) Impl() any                        { return l }

func (l *recordingLogger) text() string { return strings.Join(l.lines, "\n") }

// TestNormalizeIBCVoucherDecimalsRepairsTheShapeIBCGoWrites is the A-01 case: a
// voucher carrying ibc-go's metadata must come out in shape C, and that shape has
// to be legal bank metadata (the input is not: its first unit is the source denom,
// not the base).
func TestNormalizeIBCVoucherDecimalsRepairsTheShapeIBCGoWrites(t *testing.T) {
	t.Parallel()

	bank := newFakeVoucherBank(sdk.NewInt64Coin(voucherDenom, 1_000_000))
	bank.metadata[voucherDenom] = ibcGoVoucherMetadata(voucherDenom, voucherPath, voucherSource)
	require.Error(t, bank.metadata[voucherDenom].Validate(),
		"the fixture must reproduce the illegal on-chain shape, otherwise it proves nothing")

	logger := &recordingLogger{}
	normalizeIBCVoucherERC20Decimals(sdk.Context{}, bank, logger)

	require.Len(t, bank.writes, 1, "exactly one denom needed repair")
	got := bank.metadata[voucherDenom]
	require.Equal(t, voucherSource, got.Display,
		"Display must become the source denom, so the precompile's last-segment rule lands on the exponent unit")
	require.Equal(t, []*banktypes.DenomUnit{
		{Denom: voucherDenom, Exponent: 0},
		{Denom: voucherSource, Exponent: 18},
	}, got.DenomUnits, "auoc is atto-prefixed, so the unit must carry exponent 18")
	require.NoError(t, got.Validate(),
		"the repaired metadata must be legal bank state, not merely readable by the precompile")
	require.Equal(t, "AUOC", got.Symbol, "Name/Symbol/Description are not part of the defect and must survive")

	// The old units are dropped, so the block log has to carry them.
	require.Contains(t, logger.text(), voucherSource+":0",
		"the discarded unit list must be in the audit trail")
	require.Contains(t, logger.text(), "normalized=1")

	// Idempotency: the second run is the one a crash-restart performs.
	normalizeIBCVoucherERC20Decimals(sdk.Context{}, bank, logger)
	require.Len(t, bank.writes, 1, "a second run must not write again")
}

// TestNormalizeIBCVoucherDecimalsDerivesFromTheMicroPrefix covers the other
// derivation branch: u… means 6 decimals, and getting it wrong rescales a real
// asset by 10^12.
func TestNormalizeIBCVoucherDecimalsDerivesFromTheMicroPrefix(t *testing.T) {
	t.Parallel()

	bank := newFakeVoucherBank(sdk.NewInt64Coin(microVoucherDenom, 5))
	bank.metadata[microVoucherDenom] = ibcGoVoucherMetadata(microVoucherDenom, microVoucherPath, microVoucherSource)

	normalizeIBCVoucherERC20Decimals(sdk.Context{}, bank, &recordingLogger{})

	require.Len(t, bank.writes, 1)
	require.Equal(t, []*banktypes.DenomUnit{
		{Denom: microVoucherDenom, Exponent: 0},
		{Denom: microVoucherSource, Exponent: 6},
	}, bank.metadata[microVoucherDenom].DenomUnits)
}

// TestNormalizeIBCVoucherDecimalsKeepsARecordedExponent: when the metadata
// already states an exponent for the source denom, that value wins over the
// prefix convention. The prefix rule is a fallback, not a correction.
func TestNormalizeIBCVoucherDecimalsKeepsARecordedExponent(t *testing.T) {
	t.Parallel()

	metadata := ibcGoVoucherMetadata(voucherDenom, voucherPath, voucherSource)
	metadata.DenomUnits = []*banktypes.DenomUnit{
		{Denom: voucherDenom, Exponent: 0},
		{Denom: voucherSource, Exponent: 12},
	}
	bank := newFakeVoucherBank(sdk.NewInt64Coin(voucherDenom, 1))
	bank.metadata[voucherDenom] = metadata

	normalizeIBCVoucherERC20Decimals(sdk.Context{}, bank, &recordingLogger{})

	require.Len(t, bank.writes, 1)
	require.Equal(t, uint32(12), bank.metadata[voucherDenom].DenomUnits[1].Exponent)
}

// TestNormalizeIBCVoucherDecimalsSkipsUnderivableSource: DeriveDecimalsFromDenom
// only knows u… and a…. Anything else must be left exactly as it is, with a
// reason in the log -- guessing an exponent silently rescales an asset.
func TestNormalizeIBCVoucherDecimalsSkipsUnderivableSource(t *testing.T) {
	t.Parallel()

	original := ibcGoVoucherMetadata(voucherDenom, "transfer/channel-9/wrapped-weth", "wrapped-weth")
	bank := newFakeVoucherBank(sdk.NewInt64Coin(voucherDenom, 7))
	bank.metadata[voucherDenom] = original

	logger := &recordingLogger{}
	normalizeIBCVoucherERC20Decimals(sdk.Context{}, bank, logger)

	require.Empty(t, bank.writes, "an underivable source denom must not be rewritten")
	require.Equal(t, original, bank.metadata[voucherDenom])
	require.Contains(t, logger.text(), "cannot derive decimals")
	require.Contains(t, logger.text(), "skipped=1")
}

// TestNormalizeIBCVoucherDecimalsSkipsHashDisplay covers the degenerate record
// where Display is the voucher hash itself: the source denom is unknowable, and
// taking the hash tail as an asset name would invent one.
func TestNormalizeIBCVoucherDecimalsSkipsHashDisplay(t *testing.T) {
	t.Parallel()

	metadata := ibcGoVoucherMetadata(voucherDenom, voucherPath, voucherSource)
	metadata.Display = voucherDenom
	bank := newFakeVoucherBank(sdk.NewInt64Coin(voucherDenom, 1))
	bank.metadata[voucherDenom] = metadata

	logger := &recordingLogger{}
	normalizeIBCVoucherERC20Decimals(sdk.Context{}, bank, logger)

	require.Empty(t, bank.writes)
	require.Contains(t, logger.text(), "display is the voucher hash")
}

// TestNormalizeIBCVoucherDecimalsLeavesNativeDenomsAlone: only ibc/-prefixed
// denominations are in scope, whatever metadata they carry.
func TestNormalizeIBCVoucherDecimalsLeavesNativeDenomsAlone(t *testing.T) {
	t.Parallel()

	bank := newFakeVoucherBank(sdk.NewInt64Coin("auoc", 1))
	bank.metadata["auoc"] = banktypes.Metadata{
		Description: "Native",
		DenomUnits:  []*banktypes.DenomUnit{{Denom: "auoc", Exponent: 0}, {Denom: "uoc", Exponent: 18}},
		Base:        "auoc",
		Display:     "uoc",
		Name:        "Uptick",
		Symbol:      "UOC",
	}
	before := bank.metadata["auoc"]

	normalizeIBCVoucherERC20Decimals(sdk.Context{}, bank, &recordingLogger{})

	require.Empty(t, bank.writes)
	require.Equal(t, before, bank.metadata["auoc"])
}

// TestBackfillRegistersEveryVoucherWithoutAPair is the A-02 case. Registration
// order is pinned too: the migration's writes have to be deterministic.
func TestBackfillRegistersEveryVoucherWithoutAPair(t *testing.T) {
	t.Parallel()

	bank := newFakeVoucherBank(
		sdk.NewInt64Coin(microVoucherDenom, 5),
		sdk.NewInt64Coin("auoc", 9),
		sdk.NewInt64Coin(voucherDenom, 1_000_000),
	)
	pairs := newFakeVoucherPairs()
	logger := &recordingLogger{}

	backfillIBCVoucherTokenPairs(sdk.Context{}, bank, pairs, logger)

	require.Equal(t, []string{voucherDenom, microVoucherDenom}, pairs.order,
		"every voucher gets a pair, native denoms do not, and the order is sorted")
	require.Contains(t, logger.text(), "registered=2")
	require.Contains(t, logger.text(), "failed=0")

	// Idempotency: the pairs written by the first run are left alone.
	backfillIBCVoucherTokenPairs(sdk.Context{}, bank, pairs, logger)
	require.Len(t, pairs.order, 2)
	require.Contains(t, logger.text(), "already_registered=2")
}

// TestBackfillScopesToDenomsWithSupply pins the scope decision: a voucher that
// only exists as metadata (no supply, therefore no holder) is not given an EVM
// representation. Without this the migration would be a metadata walk instead of
// a holdings walk, and the two differ on every chain that ever burned a voucher.
func TestBackfillScopesToDenomsWithSupply(t *testing.T) {
	t.Parallel()

	bank := newFakeVoucherBank(sdk.NewInt64Coin(voucherDenom, 1))
	bank.metadata[microVoucherDenom] = ibcGoVoucherMetadata(microVoucherDenom, microVoucherPath, microVoucherSource)
	pairs := newFakeVoucherPairs()

	backfillIBCVoucherTokenPairs(sdk.Context{}, bank, pairs, &recordingLogger{})

	require.Equal(t, []string{voucherDenom}, pairs.order)
}

// TestBackfillContinuesPastAFailingDenom: one unreachable voucher must not stop
// the others, and must not abort the upgrade. The failure is still loud.
func TestBackfillContinuesPastAFailingDenom(t *testing.T) {
	t.Parallel()

	bank := newFakeVoucherBank(
		sdk.NewInt64Coin(microVoucherDenom, 5),
		sdk.NewInt64Coin(voucherDenom, 1),
	)
	pairs := newFakeVoucherPairs()
	pairs.failOn[microVoucherDenom] = fmt.Errorf("token already exists for token 0xdead")
	logger := &recordingLogger{}

	backfillIBCVoucherTokenPairs(sdk.Context{}, bank, pairs, logger)

	require.Equal(t, []string{voucherDenom}, pairs.order,
		"the denom after the failure must still be registered")
	require.Contains(t, logger.text(), "failed to backfill ibc voucher token pair")
	require.Contains(t, logger.text(), "registered=1")
	require.Contains(t, logger.text(), "failed=1")
}
