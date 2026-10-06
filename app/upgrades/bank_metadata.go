package upgrades

import (
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// HasNilDenomUnit reports whether a bank metadata denom-unit list contains a nil
// *DenomUnit element.
//
// A proto repeated message field can carry nil elements, and x/bank's
// Metadata.Validate dereferences every unit without a nil check
// (SDK v0.53 x/bank/types/metadata.go:42-58: `denomUnit.Denom`, `.Exponent`). A
// single already-stored nil unit therefore turns both the v0.5.0 metadata
// migration and the export diagnostic into a panic -- and those two paths exist
// precisely to cope with inconsistent historical state, so neither may assume the
// list is structurally sound.
//
// It lives in the shared upgrades package because both callers live outside it
// (`app` and `app/upgrades/v050`) and a guard that only one of them applies is
// the same asymmetry the denom-metadata bound extraction was made to remove.
func HasNilDenomUnit(units []*banktypes.DenomUnit) bool {
	for _, u := range units {
		if u == nil {
			return true
		}
	}
	return false
}
