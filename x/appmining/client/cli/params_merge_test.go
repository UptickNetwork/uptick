package cli

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestDeepMergeJSONObjects_PauseStatePartial(t *testing.T) {
	baseRaw := []byte(`{
		"authority":"a",
		"pause_state":{
			"registration_paused":false,
			"attestation_paused":true,
			"challenge_paused":true,
			"settlement_paused":true,
			"gas_credit_paused":true,
			"entitlement_paused":false
		}
	}`)
	patchRaw := []byte(`{"pause_state":{"entitlement_paused":true}}`)

	var base, patch map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(baseRaw, &base))
	require.NoError(t, json.Unmarshal(patchRaw, &patch))

	merged, err := deepMergeJSONObjects(base, patch)
	require.NoError(t, err)

	var pause map[string]bool
	require.NoError(t, json.Unmarshal(merged["pause_state"], &pause))
	require.False(t, pause["registration_paused"])
	require.True(t, pause["attestation_paused"])
	require.True(t, pause["challenge_paused"])
	require.True(t, pause["settlement_paused"])
	require.True(t, pause["gas_credit_paused"])
	require.True(t, pause["entitlement_paused"])
}

func TestParamsFieldDiffLines_PauseResume(t *testing.T) {
	before := []byte(`{"pause_state":{"attestation_paused":true,"entitlement_paused":true}}`)
	after := []byte(`{"pause_state":{"attestation_paused":true}}`) // entitlement omitted (omitempty false)
	beforeNorm, err := normalizeParamsJSONForDiff(before)
	require.NoError(t, err)
	afterNorm, err := normalizeParamsJSONForDiff(after)
	require.NoError(t, err)
	lines, err := paramsFieldDiffLines(beforeNorm, afterNorm, types.Params{}, types.Params{})
	require.NoError(t, err)
	require.Contains(t, lines, `pause_state.entitlement_paused: true → false`)
}

func TestParamsFieldDiffLines_TimingEffectiveAnnotation(t *testing.T) {
	beforeP := types.Params{} // stored 0 → effective defaults
	afterP := types.Params{MaxWireProofValiditySeconds: 3600, NonceRetentionSeconds: 3600}
	beforeJSON := []byte(`{}`)
	afterJSON := []byte(`{"max_wire_proof_validity_seconds":"3600","nonce_retention_seconds":"3600"}`)
	lines, err := paramsFieldDiffLines(beforeJSON, afterJSON, beforeP, afterP)
	require.NoError(t, err)
	require.Contains(t, lines,
		fmt.Sprintf("max_wire_proof_validity_seconds: 0 (effective %d) → 3600 (effective 3600)",
			types.DefaultMaxWireProofValiditySeconds))
}

func TestWireProofValidityDecreases(t *testing.T) {
	cases := []struct {
		name   string
		before types.Params
		after  types.Params
		want   bool
	}{
		{
			name:   "stored zero to explicit lower",
			before: types.Params{}, // Effective default 604800
			after:  types.Params{MaxWireProofValiditySeconds: 3600, NonceRetentionSeconds: 3600},
			want:   true,
		},
		{
			name:   "raise",
			before: types.Params{MaxWireProofValiditySeconds: 3600, NonceRetentionSeconds: 3600},
			after:  types.Params{MaxWireProofValiditySeconds: 7200, NonceRetentionSeconds: 7200},
			want:   false,
		},
		{
			name:   "unchanged effective",
			before: types.Params{},
			after:  types.Params{MaxWireProofValiditySeconds: types.DefaultMaxWireProofValiditySeconds},
			want:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, wireProofValidityDecreases(tc.before, tc.after))
		})
	}
}

func TestFormatWireProofShortenError(t *testing.T) {
	before := types.Params{}
	after := types.Params{MaxWireProofValiditySeconds: 3600, NonceRetentionSeconds: 3600}
	msg := formatWireProofShortenError(before, after)
	require.Contains(t, msg, "604800")
	require.Contains(t, msg, "3600")
	require.Contains(t, msg, flagAllowShortenWireProofValidity)
}

func TestValidateFullParamsPauseState_RequiresAllKeys(t *testing.T) {
	require.NoError(t, validateFullParamsPauseState([]byte(`{"authority":"a"}`)))
	require.Error(t, validateFullParamsPauseState([]byte(`{"pause_state":{"entitlement_paused":true}}`)))
	require.NoError(t, validateFullParamsPauseState([]byte(`{
		"pause_state":{
			"registration_paused":false,
			"attestation_paused":true,
			"challenge_paused":true,
			"settlement_paused":true,
			"gas_credit_paused":true,
			"entitlement_paused":false
		}
	}`)))
}
