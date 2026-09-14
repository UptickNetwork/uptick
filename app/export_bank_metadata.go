package app

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Finding F-2. ibc-go v10.5.0's denom-trace migration writes IBC voucher
// metadata that bank itself rejects:
//
//	ibc-go v10.5.0 modules/apps/transfer/keeper/migrations.go
//	setDenomMetadataWithDenomTrace: DenomUnits[0] = {BaseDenom, 0} ("auoc")
//	while Base = IBCDenom() ("ibc/...") and Display = GetFullDenomPath().
//
// x/bank's Metadata.Validate() requires the first denomination unit to be the
// base denom (x/bank/types/metadata.go: "metadata's first denomination unit
// must be the one with base denom '%s'"), so every chain holding IBC vouchers
// failed its own `validate-genesis` on the exported file -- the export ->
// validate -> rebuild path was broken for exactly the chains that had received
// IBC tokens.
//
// The repair keeps what can be kept and drops nothing silently:
//
//   - the first unit is rewritten to {Base, 0} and Display falls back to Base.
//     The trace path CANNOT stay as the display unit: bank requires Display to
//     appear among the units with a strictly increasing exponent, and any
//     exponent > 0 would claim the path is a decimal multiple of the base, i.e.
//     invent a conversion factor that does not exist. The path remains readable
//     in Description / Name (both written by the upstream migration) and, as
//     data, in the IBC transfer module's own denom-trace store.
//   - metadata that still fails after the repair (e.g. a bank-unfixable field)
//     is dropped from the exported genesis entirely; balances are not touched
//     by metadata in any way.
//
// Both actions are recorded in the export diagnostics sidecar, per the export
// policy in app/export.go: an export degrades and reports, it does not fail.

const (
	// exportIssueDenomMetadataNormalized: the entry was rewritten into a form
	// bank accepts.
	exportIssueDenomMetadataNormalized = "denom_metadata_normalized"
	// exportIssueDenomMetadataDropped: the entry could not be repaired and was
	// removed from the exported genesis.
	exportIssueDenomMetadataDropped = "denom_metadata_dropped"
)

// normalizeExportedBankDenomMetadataEnabled is a test seam: the guard test
// flips it off to prove the normalisation is load-bearing (the exported
// genesis then still carries the metadata bank rejects).
var normalizeExportedBankDenomMetadataEnabled = true

// normalizeExportedBankDenomMetadata rewrites the bank denom metadata inside a
// freshly exported genesis so that every entry passes bank's own
// Metadata.Validate(). Entries that already validate are left byte-untouched,
// and when nothing changes the bank section is not re-encoded at all.
func normalizeExportedBankDenomMetadata(
	c codec.Codec, genState map[string]json.RawMessage,
) ([]ExportDiagnostic, error) {
	if !normalizeExportedBankDenomMetadataEnabled {
		return nil, nil
	}
	raw, ok := genState[banktypes.ModuleName]
	if !ok || len(raw) == 0 {
		return nil, nil
	}

	var bankGen banktypes.GenesisState
	if err := c.UnmarshalJSON(raw, &bankGen); err != nil {
		return nil, fmt.Errorf("decoding bank genesis for denom metadata normalisation: %w", err)
	}

	var (
		diags   []ExportDiagnostic
		changed bool
	)
	out := make([]banktypes.Metadata, 0, len(bankGen.DenomMetadata))
	for _, md := range bankGen.DenomMetadata {
		if md.Validate() == nil {
			out = append(out, md)
			continue
		}

		repaired := md
		repaired.DenomUnits = []*banktypes.DenomUnit{{Denom: md.Base, Exponent: 0}}
		repaired.Display = md.Base

		if err := repaired.Validate(); err != nil {
			changed = true
			diags = append(diags, ExportDiagnostic{
				Module: banktypes.ModuleName,
				Kind:   exportIssueDenomMetadataDropped,
				Key:    md.Base,
				Detail: fmt.Sprintf(
					"dropped denom metadata that bank itself rejects and that the export could not repair: %v (original: first unit=%q display=%q)",
					err, firstUnitDenom(md), md.Display),
			})
			continue
		}

		changed = true
		out = append(out, repaired)
		diags = append(diags, ExportDiagnostic{
			Module: banktypes.ModuleName,
			Kind:   exportIssueDenomMetadataNormalized,
			Key:    md.Base,
			Detail: fmt.Sprintf(
				"rewrote denom metadata that bank itself rejects (first unit=%q display=%q): the exported genesis now has first unit=%q display=%q; name and description keep the IBC path",
				firstUnitDenom(md), md.Display, repaired.Base, repaired.Display),
		})
	}

	if !changed {
		return nil, nil
	}
	bankGen.DenomMetadata = out
	reencoded, err := c.MarshalJSON(&bankGen)
	if err != nil {
		return nil, fmt.Errorf("re-encoding bank genesis after denom metadata normalisation: %w", err)
	}
	genState[banktypes.ModuleName] = reencoded
	return diags, nil
}

func firstUnitDenom(md banktypes.Metadata) string {
	if len(md.DenomUnits) > 0 {
		return md.DenomUnits[0].Denom
	}
	return ""
}
