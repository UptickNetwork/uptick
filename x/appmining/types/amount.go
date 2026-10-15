package types

import (
	"math/big"
	"regexp"
	"strings"

	sdkmath "cosmossdk.io/math"
)

// amountPattern accepts denom-base-unit non-negative integers without leading zeros
// (except a lone "0").
var amountPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

const maxAmountDigits = 78

// ValidateAmountString checks NFR-01 integer amount encoding.
func ValidateAmountString(s string) error {
	if s == "" || !amountPattern.MatchString(s) {
		return ErrInvalidAmount
	}
	if len(s) > maxAmountDigits {
		return ErrInvalidAmount
	}
	return nil
}

// ParseAmount parses a validated amount string into sdkmath.Int.
func ParseAmount(s string) (sdkmath.Int, error) {
	if err := ValidateAmountString(s); err != nil {
		return sdkmath.ZeroInt(), err
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return sdkmath.ZeroInt(), ErrInvalidAmount
	}
	return sdkmath.NewIntFromBigInt(n), nil
}

// MustParseAmount panics on invalid amount; for trusted genesis defaults only.
func MustParseAmount(s string) sdkmath.Int {
	v, err := ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return v
}

// AmountString returns the canonical decimal string for an Int.
func AmountString(v sdkmath.Int) string {
	if v.IsNil() || v.IsNegative() {
		return "0"
	}
	return v.String()
}

// AddAmountStrings adds two validated amount strings.
func AddAmountStrings(a, b string) (string, error) {
	ai, err := ParseAmount(a)
	if err != nil {
		return "", err
	}
	bi, err := ParseAmount(b)
	if err != nil {
		return "", err
	}
	return AmountString(ai.Add(bi)), nil
}

// SubAmountStrings subtracts b from a; fails if result would be negative.
func SubAmountStrings(a, b string) (string, error) {
	ai, err := ParseAmount(a)
	if err != nil {
		return "", err
	}
	bi, err := ParseAmount(b)
	if err != nil {
		return "", err
	}
	if ai.LT(bi) {
		return "", ErrCreditInsufficient
	}
	return AmountString(ai.Sub(bi)), nil
}

// CompareAmountStrings returns -1/0/1 for a<b / a==b / a>b.
func CompareAmountStrings(a, b string) (int, error) {
	ai, err := ParseAmount(a)
	if err != nil {
		return 0, err
	}
	bi, err := ParseAmount(b)
	if err != nil {
		return 0, err
	}
	return ai.BigInt().Cmp(bi.BigInt()), nil
}

// SplitRewardAndCredit splits total by gas_credit_share_bps.
// credit = total * bps / 10000; reward = total - credit.
func SplitRewardAndCredit(total string, bps uint32) (reward, credit string, err error) {
	if bps > 10000 {
		return "", "", ErrInvalidAmount
	}
	ti, err := ParseAmount(total)
	if err != nil {
		return "", "", err
	}
	creditInt := ti.MulRaw(int64(bps)).QuoRaw(10000)
	rewardInt := ti.Sub(creditInt)
	return AmountString(rewardInt), AmountString(creditInt), nil
}

// IsZeroAmount reports whether s is "0".
func IsZeroAmount(s string) bool {
	return strings.TrimSpace(s) == "0"
}
