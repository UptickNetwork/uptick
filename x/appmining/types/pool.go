package types

import "fmt"

// ValidatePoolInvariant checks funded = available + reserved + paid.
func ValidatePoolInvariant(p Pool) error {
	if err := ValidateWireID(p.PoolId); err != nil {
		return fmt.Errorf("pool_id: %w", err)
	}
	if err := ValidateAmountString(p.Funded); err != nil {
		return fmt.Errorf("funded: %w", err)
	}
	if err := ValidateAmountString(p.Available); err != nil {
		return fmt.Errorf("available: %w", err)
	}
	if err := ValidateAmountString(p.Reserved); err != nil {
		return fmt.Errorf("reserved: %w", err)
	}
	if err := ValidateAmountString(p.Paid); err != nil {
		return fmt.Errorf("paid: %w", err)
	}
	sum, err := AddAmountStrings(p.Available, p.Reserved)
	if err != nil {
		return err
	}
	sum, err = AddAmountStrings(sum, p.Paid)
	if err != nil {
		return err
	}
	cmp, err := CompareAmountStrings(sum, p.Funded)
	if err != nil {
		return err
	}
	if cmp != 0 {
		return fmt.Errorf("%w: funded=%s available+reserved+paid=%s", ErrPoolInvariant, p.Funded, sum)
	}
	return nil
}

// EmptyPool returns a zeroed ledger for a newly opened pool.
func EmptyPool(poolID, denom string, version uint64) Pool {
	return Pool{
		PoolId:    poolID,
		Denom:     denom,
		Funded:    "0",
		Available: "0",
		Reserved:  "0",
		Paid:      "0",
		Paused:    false,
		Version:   version,
	}
}

// EntitlementFlowEnabled reports whether the V1 entitlement flow may accept
// business msgs for this pool given params and module account balance.
func EntitlementFlowEnabled(p Pool, params Params, moduleBalance string) bool {
	if p.Paused {
		return false
	}
	if params.PauseState != nil && params.PauseState.EntitlementPaused {
		return false
	}
	min := params.EntitlementEnableMinFunded
	if min == "" || IsZeroAmount(min) {
		return false
	}
	if err := ValidateAmountString(min); err != nil {
		return false
	}
	cmpFunded, err := CompareAmountStrings(p.Funded, min)
	if err != nil || cmpFunded < 0 {
		return false
	}
	// reserved must be coverable by module balance
	if err := ValidateAmountString(moduleBalance); err != nil {
		return false
	}
	cmpBal, err := CompareAmountStrings(moduleBalance, p.Reserved)
	if err != nil || cmpBal < 0 {
		return false
	}
	return true
}
