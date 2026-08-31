package v041

import (
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/evm/ethereum/eip712"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/UptickNetwork/uptick/app/upgrades/v040/legacy"
)

// RegisterCompatInterfaces registers the runtime interface compatibility needed
// by Keplr on the main registry. The legacy ethermint pubkey and EIP-712
// extension option type URLs are mapped onto the complete cosmos/evm types
// (their protobuf wire formats are identical), and the v0.3.3-era EthAccount and
// governance proposal types remain registered for genesis bootstrap and state
// export. This deliberately does not register the legacy v0.3.3 pubkey type on
// the main registry, since it is incomplete and only needed for the v0.3.3 ->
// v0.4.0 state migration.
func RegisterCompatInterfaces(registry codectypes.InterfaceRegistry) {
	// v0.3.3-era account / proposal records carried over for genesis bootstrap
	// and state export.
	registry.RegisterImplementations((*sdk.AccountI)(nil), &legacy.EthAccount{})
	registry.RegisterImplementations((*authtypes.GenesisAccount)(nil), &legacy.EthAccount{})
	registry.RegisterImplementations(
		(*govv1beta1.Content)(nil),
		&legacy.RegisterCoinProposal{},
		&legacy.RegisterERC20Proposal{},
		&legacy.ToggleTokenRelayProposal{},
		&legacy.UpdateTokenPairERC20Proposal{},
	)

	// Legacy Keplr type URLs -> complete cosmos/evm types.
	registerCustomTypeURL(registry, (*cryptotypes.PubKey)(nil), "/ethermint.crypto.v1.ethsecp256k1.PubKey", &ethsecp256k1.PubKey{})
	registerCustomTypeURL(registry, (*txtypes.TxExtensionOptionI)(nil), "/ethermint.types.v1.ExtensionOptionsWeb3Tx", &eip712.ExtensionOptionsWeb3Tx{})
}

// customTypeURLRegistrar is the subset of the SDK's unexported interface
// registry implementation that supports registering a concrete type under an
// arbitrary (legacy) type URL.
type customTypeURLRegistrar interface {
	RegisterCustomTypeURL(iface any, typeURL string, impl proto.Message)
}

func registerCustomTypeURL(registry codectypes.InterfaceRegistry, iface any, typeURL string, impl proto.Message) {
	r, ok := registry.(customTypeURLRegistrar)
	if !ok {
		panic("interface registry does not support RegisterCustomTypeURL")
	}
	r.RegisterCustomTypeURL(iface, typeURL, impl)
}
