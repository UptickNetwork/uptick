package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// Required pause_state JSON keys when the object is present in a --full-params
// document. Missing keys would deserialize as false and silently resume.
var requiredPauseStateKeys = []string{
	"registration_paused",
	"attestation_paused",
	"challenge_paused",
	"settlement_paused",
	"gas_credit_paused",
	"entitlement_paused",
}

// deepMergeJSONObjects overlays patch onto base. Nested JSON objects are merged
// key-by-key; arrays and scalars in patch replace the corresponding base value.
func deepMergeJSONObjects(base, patch map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(base)+len(patch))
	for k, v := range base {
		out[k] = v
	}
	for k, pv := range patch {
		bv, ok := out[k]
		if !ok {
			out[k] = pv
			continue
		}
		merged, err := deepMergeRaw(bv, pv)
		if err != nil {
			return nil, fmt.Errorf("merge key %q: %w", k, err)
		}
		out[k] = merged
	}
	return out, nil
}

func deepMergeRaw(base, patch json.RawMessage) (json.RawMessage, error) {
	base = bytes.TrimSpace(base)
	patch = bytes.TrimSpace(patch)
	if len(base) == 0 {
		return patch, nil
	}
	if len(patch) == 0 {
		return base, nil
	}
	if base[0] == '{' && patch[0] == '{' {
		var baseObj, patchObj map[string]json.RawMessage
		if err := json.Unmarshal(base, &baseObj); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(patch, &patchObj); err != nil {
			return nil, err
		}
		merged, err := deepMergeJSONObjects(baseObj, patchObj)
		if err != nil {
			return nil, err
		}
		return json.Marshal(merged)
	}
	// Arrays / scalars / type mismatches: patch wins (explicit replace).
	return patch, nil
}

// flattenJSON returns dotted-path → canonical JSON value for leaf fields.
func flattenJSON(raw json.RawMessage, prefix string, out map[string]string) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		out[prefix] = "null"
		return nil
	}
	switch raw[0] {
	case '{':
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return err
		}
		if len(obj) == 0 {
			out[prefix] = "{}"
			return nil
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			if err := flattenJSON(obj[k], path, out); err != nil {
				return err
			}
		}
		return nil
	default:
		// Arrays and scalars: keep compact JSON as the leaf value.
		var compact bytes.Buffer
		if err := json.Compact(&compact, raw); err != nil {
			out[prefix] = string(raw)
			return nil
		}
		out[prefix] = compact.String()
		return nil
	}
}

// paramsFieldDiffLines returns sorted "path: old → new" lines for changed leaves.
// Timing fields are annotated with Effective values so stored 0 is not mistaken
// for an upward change when the live Effective default is 604800/etc.
func paramsFieldDiffLines(beforeJSON, afterJSON []byte, beforeParams, afterParams types.Params) ([]string, error) {
	before := map[string]string{}
	after := map[string]string{}
	if err := flattenJSON(beforeJSON, "", before); err != nil {
		return nil, fmt.Errorf("flatten before: %w", err)
	}
	if err := flattenJSON(afterJSON, "", after); err != nil {
		return nil, fmt.Errorf("flatten after: %w", err)
	}
	annotateTimingLeaves(before, beforeParams)
	annotateTimingLeaves(after, afterParams)

	keys := make(map[string]struct{}, len(before)+len(after))
	for k := range before {
		keys[k] = struct{}{}
	}
	for k := range after {
		keys[k] = struct{}{}
	}
	// Ensure timing keys appear even when omitempty dropped stored 0.
	for _, k := range timingParamKeys {
		keys[k] = struct{}{}
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var lines []string
	for _, k := range sorted {
		b, bok := before[k]
		a, aok := after[k]
		if !bok {
			b = "<absent>"
		}
		if !aok {
			a = "<absent>"
		}
		if b == a {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s → %s", k, b, a))
	}
	return lines, nil
}

var timingParamKeys = []string{
	"challenge_window_seconds",
	"max_freeze_seconds",
	"max_wire_proof_validity_seconds",
	"nonce_retention_seconds",
}

func annotateTimingLeaves(leaves map[string]string, p types.Params) {
	eff := map[string]int64{
		"challenge_window_seconds":        p.EffectiveChallengeWindowSeconds(),
		"max_freeze_seconds":              p.EffectiveMaxFreezeSeconds(),
		"max_wire_proof_validity_seconds": p.EffectiveMaxWireProofValiditySeconds(),
		"nonce_retention_seconds":         p.EffectiveNonceRetentionSeconds(),
	}
	stored := map[string]uint64{
		"challenge_window_seconds":        p.ChallengeWindowSeconds,
		"max_freeze_seconds":              p.MaxFreezeSeconds,
		"max_wire_proof_validity_seconds": p.MaxWireProofValiditySeconds,
		"nonce_retention_seconds":         p.NonceRetentionSeconds,
	}
	for _, k := range timingParamKeys {
		leaves[k] = formatTimingDiffValue(stored[k], eff[k])
	}
}

func formatTimingDiffValue(stored uint64, effective int64) string {
	if stored == 0 {
		return fmt.Sprintf("0 (effective %d)", effective)
	}
	return fmt.Sprintf("%d (effective %d)", stored, effective)
}

// wireProofValidityDecreases reports whether after Effective max is lower.
func wireProofValidityDecreases(before, after types.Params) bool {
	return after.EffectiveMaxWireProofValiditySeconds() < before.EffectiveMaxWireProofValiditySeconds()
}

func formatWireProofShortenError(before, after types.Params) string {
	return fmt.Sprintf(
		"max_wire_proof_validity_seconds Effective decreases (%d → %d); "+
			"already-issued unused proofs will fail closed. Re-run with --%s to confirm",
		before.EffectiveMaxWireProofValiditySeconds(),
		after.EffectiveMaxWireProofValiditySeconds(),
		flagAllowShortenWireProofValidity,
	)
}

func formatParamsDiff(lines []string) string {
	if len(lines) == 0 {
		return "params diff: (no field changes)\n"
	}
	var b strings.Builder
	b.WriteString("params diff:\n")
	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// validateFullParamsPauseState rejects a partial pause_state object in a
// --full-params document (missing bool keys become false on unmarshal).
func validateFullParamsPauseState(raw []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return fmt.Errorf("decode full params json: %w", err)
	}
	psRaw, ok := top["pause_state"]
	if !ok || len(bytes.TrimSpace(psRaw)) == 0 || bytes.Equal(bytes.TrimSpace(psRaw), []byte("null")) {
		return nil // Validate() will still require a non-nil PauseState after unmarshal
	}
	var ps map[string]json.RawMessage
	if err := json.Unmarshal(psRaw, &ps); err != nil {
		return fmt.Errorf("decode pause_state: %w", err)
	}
	var missing []string
	for _, k := range requiredPauseStateKeys {
		if _, ok := ps[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("pause_state in --full-params must include all keys %v; missing %v",
			requiredPauseStateKeys, missing)
	}
	return nil
}

// pauseStateExplicit always emits all six pause bits so diffs show false
// instead of omitempty "<absent>".
type pauseStateExplicit struct {
	RegistrationPaused bool `json:"registration_paused"`
	AttestationPaused  bool `json:"attestation_paused"`
	ChallengePaused    bool `json:"challenge_paused"`
	SettlementPaused   bool `json:"settlement_paused"`
	GasCreditPaused    bool `json:"gas_credit_paused"`
	EntitlementPaused  bool `json:"entitlement_paused"`
}

// normalizeParamsJSONForDiff rewrites pause_state with explicit false values.
func normalizeParamsJSONForDiff(raw []byte) ([]byte, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	psRaw, ok := top["pause_state"]
	if !ok || len(bytes.TrimSpace(psRaw)) == 0 || bytes.Equal(bytes.TrimSpace(psRaw), []byte("null")) {
		return raw, nil
	}
	var ps pauseStateExplicit
	if err := json.Unmarshal(psRaw, &ps); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(ps)
	if err != nil {
		return nil, err
	}
	top["pause_state"] = encoded
	return json.Marshal(top)
}
