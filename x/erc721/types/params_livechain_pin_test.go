package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParamsValidateAcceptsLiveChainParams is the x/cw721 test's twin, required
// because x/erc721 is a copy of the same module and the two must not drift
// apart on what consensus will accept.
//
// Params.Validate is on the genesis path: GenesisState.Validate calls it and
// AppModuleBasic.ValidateGenesis calls that, so it runs against the chain's own
// export. A rule that rejects a live configuration makes `uptickd
// validate-genesis` fail on the backup an operator restores from.
//
// Values transcribed from these queries, run 2026-09-13:
//
//	GET https://rest.uptick.network/uptick/erc721/v1/params        (mainnet v0.3.3)
//	  {"params":{"enable_erc721":true,"enable_evm_hook":true}}
//	GET https://rest.origin.uptick.network/uptick/erc721/v1/params (testnet v0.4.0)
//	  {"params":{"enable_erc721":true,"enable_evm_hook":true}}
func TestParamsValidateAcceptsLiveChainParams(t *testing.T) {
	cases := []struct {
		name   string
		source string
		params Params
	}{
		{
			name:   "mainnet v0.3.3",
			source: "https://rest.uptick.network",
			params: Params{EnableErc721: true, EnableEVMHook: true},
		},
		{
			name:   "testnet v0.4.0",
			source: "https://rest.origin.uptick.network",
			params: Params{EnableErc721: true, EnableEVMHook: true},
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

			gs := NewGenesisState(tc.params, nil)
			require.NoError(t, gs.Validate(),
				"%s: a genesis produced by the chain must pass its own validation", tc.source)
		})
	}
}

// TestParamsValidateHasNoRangeCheck pins that every boolean combination is a
// configuration an operator may choose — EnableErc721=false switches
// conversions off, and MsgConvertERC721 / MsgConvertNFT both gate on that field
// alone. There is no third field here, so there is nothing else a rule could
// constrain.
func TestParamsValidateHasNoRangeCheck(t *testing.T) {
	for _, enableErc721 := range []bool{false, true} {
		for _, enableEVMHook := range []bool{false, true} {
			params := Params{
				EnableErc721:  enableErc721,
				EnableEVMHook: enableEVMHook,
			}
			require.NoError(t, params.Validate(), "params %+v must stay valid", params)
		}
	}
}
