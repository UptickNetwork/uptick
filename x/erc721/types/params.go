package types

// Parameter store key
var (
	KeyPrefixParams = []byte("erc721/params/")
)

// NewParams creates a new Params object
func NewParams(
	enableErc721 bool,
	enableEVMHook bool,
) Params {
	return Params{
		EnableErc721:  enableErc721,
		EnableEVMHook: enableEVMHook,
	}
}

// DefaultParams returns default params
func DefaultParams() Params {
	return Params{
		EnableErc721:  true,
		EnableEVMHook: true,
	}
}

// Validate validates the params.
// EnableErc721 and EnableEVMHook are typed bool fields in the proto message;
// the former type-assertion checks (interface{}(bool).(bool)) were always true
// no-ops and have been removed.
//
// Read this before adding a rule. Validate is reached from BOTH the genesis
// path (GenesisState.Validate, called by module.go's
// AppModuleBasic.ValidateGenesis) and the governance path
// (MsgUpdateParams.ValidateBasic -> keeper.UpdateParams). A rejecting rule here
// is therefore not a local change: it decides what the chain will accept from
// its own genesis file, and a wrong one makes an operator unable to validate
// the backup they would restore from.
//
// Live-chain evidence, queried 2026-09-13:
//
//	mainnet (v0.3.3)  GET https://rest.uptick.network/uptick/erc721/v1/params
//	  -> {"enable_erc721":true,"enable_evm_hook":true}
//	testnet (v0.4.0)  GET https://rest.origin.uptick.network/uptick/erc721/v1/params
//	  -> {"enable_erc721":true,"enable_evm_hook":true}
//
// Both fields are plain bools, so unlike its twin x/cw721 there is not even a
// zero-value convention to pin: every one of the four combinations is a
// configuration the operator is entitled to choose, and EnableErc721=false is
// a supported way to switch conversions off (MsgConvertERC721 / MsgConvertNFT
// both gate on it alone). App/upgrades/v040/upgrades.go also copies these
// params forward verbatim, so a new rejection here would fire during an upgrade
// rather than at the point of the mistake.
//
// Kept as a documented no-op for the same reason as x/cw721: the module has no
// third field that could carry an invariant, and a gate with no invariant
// behind it only adds a way to break genesis. See
// TestParamsValidateAcceptsLiveChainParams.
func (p Params) Validate() error {
	return nil
}
