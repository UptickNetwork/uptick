package wirev1

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	idPattern   = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
	hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	uintPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
)

var evidenceTypes = map[string]struct{}{
	"TX_MISSING_OR_UNFINALIZED":      {},
	"RECEIPT_QBT_MISMATCH":           {},
	"DUPLICATE_OR_FAMILY_CONFLICT":   {},
	"CONFIRMED_REFUND_OR_REVERSAL":   {},
	"APP_OR_EMITTER_INELIGIBLE":      {},
	"REPRODUCIBLE_CALCULATION_ERROR": {},
}

func requireID(name, v string) error {
	if !idPattern.MatchString(v) {
		return fmt.Errorf("invalid %s", name)
	}
	return nil
}

func requireHash(name, v string) error {
	if !hashPattern.MatchString(v) {
		return fmt.Errorf("invalid %s", name)
	}
	return nil
}

func requireUint(name, v string) error {
	if !uintPattern.MatchString(v) {
		return fmt.Errorf("invalid %s", name)
	}
	return nil
}

func exactKeys(record map[string]string, expected []string) error {
	if len(record) != len(expected) {
		return fmt.Errorf("authorization fields missing or unexpected")
	}
	got := make([]string, 0, len(record))
	for k := range record {
		got = append(got, k)
	}
	sort.Strings(got)
	want := append([]string(nil), expected...)
	sort.Strings(want)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		return fmt.Errorf("authorization fields missing or unexpected")
	}
	return nil
}
