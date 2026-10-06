package ibc

import (
	"context"
	"fmt"
	"strings"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmibc "github.com/cosmos/evm/ibc"
)

// VoucherPrefix marks the bank denominations that are ICS-20 vouchers: the
// denominations the ERC20 side of this file exists for.
const VoucherPrefix = "ibc/"

// VoucherBankStore is the slice of x/bank the voucher normalization needs. The
// SDK v0.53 bank keeper takes a plain context.Context (not sdk.Context), which
// sdk.Context satisfies, so both a live app and a test double fit this
// interface.
type VoucherBankStore interface {
	GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool)
	SetDenomMetaData(ctx context.Context, metadata banktypes.Metadata)
}

// NormalizationOutcome is what NormalizeVoucherDecimals did, or refused to do,
// with one denomination. It exists as a named type rather than a bool so the
// three "nothing was written" cases stay distinguishable: an operator reading a
// log line has to be able to tell "already correct" from "could not be repaired"
// from "nothing to repair", and only the middle one is a defect.
type NormalizationOutcome uint8

const (
	// NormalizationNoMetadata: the denom carries no bank metadata at all. The
	// ERC20 precompile then falls back to deriving decimals from the source denom
	// (precompiles/erc20/query.go:96-109), which is already the correct branch, so
	// there is nothing to repair.
	NormalizationNoMetadata NormalizationOutcome = iota
	// NormalizationAlreadyNormalized: the metadata is already shape C.
	NormalizationAlreadyNormalized
	// NormalizationApplied: the metadata was rewritten into shape C.
	NormalizationApplied
	// NormalizationSkipped: the record cannot be repaired without guessing an
	// exponent. Reason carries why; the record was left byte-identical.
	NormalizationSkipped
)

// String renders the outcome for logs and test failure messages.
func (o NormalizationOutcome) String() string {
	switch o {
	case NormalizationNoMetadata:
		return "no-metadata"
	case NormalizationAlreadyNormalized:
		return "already-normalized"
	case NormalizationApplied:
		return "applied"
	case NormalizationSkipped:
		return "skipped"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(o))
	}
}

// NormalizationResult is NormalizeVoucherDecimals' full answer. Decimals is only
// meaningful when the outcome is NormalizationApplied or
// NormalizationAlreadyNormalized: it is the exponent the ERC20 precompile will
// now report.
type NormalizationResult struct {
	Outcome  NormalizationOutcome
	Reason   string
	Decimals uint32
}

// Changed reports whether a state write happened. The callers log a repair only
// for this outcome, so that a replayed or crash-restarted pass stays silent.
func (r NormalizationResult) Changed() bool { return r.Outcome == NormalizationApplied }

// NormalizeVoucherDecimals rewrites an IBC voucher's bank metadata into the only
// shape that satisfies both of the constraints it lives under, so that
// cosmos/evm's ERC20 precompile reports the right decimals.
//
// # What is wrong without it
//
// ibc-go writes a voucher's metadata as:
//
//	Base       = "ibc/3D02…"               (the voucher hash)
//	Display    = "transfer/channel-1/auoc"  (the full denom path)
//	DenomUnits = [{Denom: "auoc", Exponent: 0}]
//
// cosmos/evm's precompile (precompiles/erc20/query.go:115-128) reads decimals
// differently for these: with an "ibc/" base it matches the LAST SEGMENT of
// Display against DenomUnits. That segment is "auoc", it matches the only unit,
// and that unit's exponent is 0 -- so decimals() returns 0. Wallets, explorers
// and DEX front-ends then read a 1 auoc balance as 10^18.
//
// THE SHAPE THAT WORKS ("shape C" in the test-side analysis)
//
//	Display    = "auoc"                    (the source denom, no path)
//	DenomUnits = [{Base, 0}, {"auoc", 18}]
//
// which satisfies:
//
//   - x/bank's Metadata.Validate (first unit is the base with exponent 0, units
//     sorted ascending, Display present among them), so the metadata stops
//     being illegal state; and
//   - the precompile's rule above, whose last-segment match now lands on the
//     unit carrying the real exponent.
//
// # Why this lives here and not in the upgrade handler
//
// The v0.5.0 handler repairs what the chain already holds (one pass, bounded by
// what the chain accepted before the upgrade). The inbound auto-registration
// path keeps minting NEW vouchers after the upgrade, and it writes ibc-go's
// shape every time -- so a handler-only repair leaves two populations on one
// chain and decimals() depends on whether a pair predates the upgrade. Both
// callers therefore share this one implementation instead of each carrying a
// copy that can drift.
//
// The counterparty picks the source denom, so a wrong exponent here silently
// rescales a real asset. Everything that cannot be read is therefore skipped
// with a reason rather than guessed.
//
// CONTRACT: the caller passes a denomination that carries the ibc/ prefix; the
// entry point enforces it. NormalizeVoucherMetadata on its own assumes it.
func NormalizeVoucherDecimals(
	ctx context.Context,
	bank VoucherBankStore,
	denom string,
) NormalizationResult {
	if !strings.HasPrefix(denom, VoucherPrefix) {
		// Refuse rather than repair: this rewrite is only correct for a voucher
		// (the precompile's "ibc/" branch keys on the base denom), and applying
		// it to a native denom would drop its intermediate DenomUnits.
		return NormalizationResult{
			Outcome: NormalizationSkipped,
			Reason:  fmt.Sprintf("denom %q is not an ibc voucher", denom),
		}
	}

	metadata, found := bank.GetDenomMetaData(ctx, denom)
	if !found {
		return NormalizationResult{Outcome: NormalizationNoMetadata}
	}

	updated, reason, ok := NormalizeVoucherMetadata(metadata)
	if !ok {
		return NormalizationResult{Outcome: NormalizationSkipped, Reason: reason}
	}

	decimals := lastUnitExponent(updated.DenomUnits)
	if updated.Display == metadata.Display && SameDenomUnits(updated.DenomUnits, metadata.DenomUnits) {
		return NormalizationResult{Outcome: NormalizationAlreadyNormalized, Decimals: decimals}
	}

	bank.SetDenomMetaData(ctx, updated)
	return NormalizationResult{Outcome: NormalizationApplied, Decimals: decimals}
}

// NormalizeVoucherMetadata returns the shape-C rewrite of a voucher's metadata,
// or the reason it cannot be repaired. It never mutates its input.
//
// It is pure: the caller decides whether to store the result. That is what lets
// the upgrade handler and the inbound registration path share the decision while
// writing from their own keepers.
func NormalizeVoucherMetadata(metadata banktypes.Metadata) (banktypes.Metadata, string, bool) {
	if HasNilDenomUnit(metadata.DenomUnits) {
		// A nil unit means the stored record is structurally corrupt. Refuse to
		// rewrite it: the shape-C rewrite reuses nothing but Base/Display here, so
		// the exponent would have to be guessed, and a wrong exponent silently
		// rescales a real asset. Skip it with a reason and leave it for an
		// operator. (x/bank's own Validate would panic on this record, so every
		// walk below must stay nil-safe too.)
		return metadata, "metadata carries a nil denom unit", false
	}

	if metadata.Display == metadata.Base {
		// The source denom is not recoverable, and deriving one from the hash
		// tail would invent an asset name.
		return metadata, "display is the voucher hash, source denom unknown", false
	}

	sourceDenom := metadata.Display
	if i := strings.LastIndex(sourceDenom, "/"); i >= 0 {
		sourceDenom = sourceDenom[i+1:]
	}
	if sourceDenom == "" || sourceDenom == metadata.Base {
		return metadata, "display carries no usable source denom", false
	}

	decimals, ok := NonZeroUnitExponent(metadata.DenomUnits, sourceDenom)
	if !ok {
		derived, err := evmibc.DeriveDecimalsFromDenom(sourceDenom)
		if err != nil {
			return metadata, fmt.Sprintf("cannot derive decimals for source denom %q: %v", sourceDenom, err), false
		}
		decimals = uint32(derived)
	}

	updated := metadata
	updated.Display = sourceDenom
	updated.DenomUnits = []*banktypes.DenomUnit{
		{Denom: metadata.Base, Exponent: 0},
		{Denom: sourceDenom, Exponent: decimals},
	}

	// Refuse to write metadata the bank module itself would reject: once such a
	// record is on chain, the export-side normalization rewrites Display to the
	// base denom and decimals() goes from 0 to a revert -- worse than the
	// starting point.
	if err := updated.Validate(); err != nil {
		return metadata, fmt.Sprintf("repaired metadata is invalid: %v", err), false
	}
	return updated, "", true
}

// HasNilDenomUnit reports whether a bank metadata denom-unit list contains a nil
// *DenomUnit element.
//
// A proto repeated message field can carry nil elements, and x/bank's
// Metadata.Validate dereferences every unit without a nil check
// (SDK v0.53 x/bank/types/metadata.go:42-58: `denomUnit.Denom`, `.Exponent`). A
// single already-stored nil unit therefore turns the metadata repair and the
// export diagnostic into a panic -- and those paths exist precisely to cope with
// inconsistent historical state, so neither may assume the list is structurally
// sound.
func HasNilDenomUnit(units []*banktypes.DenomUnit) bool {
	for _, u := range units {
		if u == nil {
			return true
		}
	}
	return false
}

// NonZeroUnitExponent returns the exponent the metadata already records for
// denom, if it is non-zero. ibc-go's shape carries the source denom with
// exponent 0, which is the value being repaired, so zero is not a usable answer
// here.
//
// Nil elements are skipped rather than dereferenced: proto repeated message
// fields can carry them, and this repair's job is to survive inconsistent
// historical state, not to panic on it.
func NonZeroUnitExponent(units []*banktypes.DenomUnit, denom string) (uint32, bool) {
	for _, unit := range units {
		if unit == nil {
			continue
		}
		if unit.Denom == denom && unit.Exponent > 0 {
			return unit.Exponent, true
		}
	}
	return 0, false
}

// SameDenomUnits reports whether two unit lists are equal in order and content.
// A nil element compares as such (never dereferenced), so a corrupt list is
// reported as changed rather than crashing the caller.
func SameDenomUnits(a, b []*banktypes.DenomUnit) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if (a[i] == nil) != (b[i] == nil) {
			return false
		}
		if denomUnitDenom(a[i]) != denomUnitDenom(b[i]) || denomUnitExponent(a[i]) != denomUnitExponent(b[i]) {
			return false
		}
	}
	return true
}

// FormatDenomUnits renders a unit list for the audit log, marking nil elements
// instead of dereferencing them.
func FormatDenomUnits(units []*banktypes.DenomUnit) string {
	parts := make([]string, 0, len(units))
	for _, unit := range units {
		if unit == nil {
			parts = append(parts, "<nil>")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%d", unit.Denom, unit.Exponent))
	}
	return strings.Join(parts, ",")
}

// lastUnitExponent reads the exponent shape C always writes last: the intended
// decimals. A list that is empty or ends in a nil unit (only reachable on a
// corrupt record, which NormalizeVoucherMetadata refuses before this is called)
// reports 0.
func lastUnitExponent(units []*banktypes.DenomUnit) uint32 {
	if len(units) == 0 {
		return 0
	}
	return denomUnitExponent(units[len(units)-1])
}

// denomUnitDenom / denomUnitExponent read a possibly-nil denom unit. They exist
// because x/bank's Metadata.Validate dereferences every unit unconditionally, so
// any path that walks a stored Metadata must tolerate nil first.
func denomUnitDenom(u *banktypes.DenomUnit) string {
	if u == nil {
		return ""
	}
	return u.Denom
}

func denomUnitExponent(u *banktypes.DenomUnit) uint32 {
	if u == nil {
		return 0
	}
	return u.Exponent
}
