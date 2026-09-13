package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParamsValidateAcceptsLiveChainParams pins the parameters actually running
// on mainnet and testnet, plus the shape a genesis export of either chain
// carries.
//
// Params.Validate is not an ordinary sanity helper: GenesisState.Validate calls
// it, and AppModuleBasic.ValidateGenesis calls that, so `uptickd
// validate-genesis` runs it against the chain's own export. A rule that rejects
// a live configuration therefore fails validation on the exact backup an
// operator restores from — a new P0, not a hardening. This test makes such a
// rule fail here first.
//
// Values transcribed from these queries, run 2026-09-13:
//
//	GET https://rest.uptick.network/uptick/cw721/v1/params        (mainnet v0.3.3)
//	  {"params":{"enable_cw721":true,"enable_evm_hook":true}}
//	GET https://rest.origin.uptick.network/uptick/cw721/v1/params (testnet v0.4.0)
//	  {"params":{"enable_cw721":true,"enable_evm_hook":true,"wasm_code_id":"2"}}
//
// Mainnet carries no wasm_code_id at all: proto3 JSON omits a zero uint64, so
// mainnet's value is 0. "Reject the zero" is the tempting rule, and it is
// exactly the one that would break mainnet.
func TestParamsValidateAcceptsLiveChainParams(t *testing.T) {
	cases := []struct {
		name   string
		source string
		params Params
	}{
		{
			name:   "mainnet v0.3.3",
			source: "https://rest.uptick.network (wasm_code_id absent -> 0)",
			params: Params{EnableCw721: true, EnableEVMHook: true, WasmCodeId: 0},
		},
		{
			name:   "testnet v0.4.0",
			source: "https://rest.origin.uptick.network (wasm_code_id=2)",
			params: Params{EnableCw721: true, EnableEVMHook: true, WasmCodeId: 2},
		},
		{
			name:   "store has no params key",
			source: "keeper.GetParams falls back to DefaultParams()",
			params: DefaultParams(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.params.Validate(),
				"%s: a configuration the chain actually runs must never be rejected", tc.source)

			// The genesis gate is what `validate-genesis` hits, so assert it
			// too rather than trusting the wiring.
			gs := NewGenesisState(tc.params, nil)
			require.NoError(t, gs.Validate(),
				"%s: a genesis produced by the chain must pass its own validation", tc.source)
		})
	}
}

// TestParamsValidateHasNoRangeCheck pins the other half of the decision: not
// only are the live values accepted, there is no range or cross-field rule to
// trip over. Every boolean combination is a configuration an operator may
// choose (EnableCw721=false switches conversions off; MsgConvertCW721 and
// MsgConvertNFT both gate on that field alone), and WasmCodeId has no
// meaningful bound here — a code ID is a store-side fact this pure predicate
// cannot resolve.
//
// If this test ever needs changing, read the doc comment on Params.Validate
// first: the rule being added has to keep both live chains' exported genesis
// valid, which this test will check.
func TestParamsValidateHasNoRangeCheck(t *testing.T) {
	for _, enableCw721 := range []bool{false, true} {
		for _, enableEVMHook := range []bool{false, true} {
			// 0 is "not configured"; the others stand in for any assigned ID.
			for _, codeID := range []uint64{0, 1, 2, 1 << 40} {
				params := Params{
					EnableCw721:   enableCw721,
					EnableEVMHook: enableEVMHook,
					WasmCodeId:    codeID,
				}
				require.NoError(t, params.Validate(), "params %+v must stay valid", params)
			}
		}
	}
}
