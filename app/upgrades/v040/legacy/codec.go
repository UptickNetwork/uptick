package legacy

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// RegisterInterfaces registers the legacy Ethermint types (EthAccount and the
// eth_secp256k1 pubkey) and the legacy uptick x/erc20 governance proposal types
// solely for v0.3.3 -> v0.4.0 compatibility:
//   - EthAccount  -> decoding legacy /ethermint.types.v1.EthAccount records
//   - PubKey      -> decoding BaseAccount.PubKey Any values stored under
//     /ethermint.crypto.v1.ethsecp256k1.PubKey, and verifying signatures of
//     migrated accounts after the upgrade.
//   - ERC20 proposals -> decoding legacy x/gov proposal contents so post-upgrade
//     state export can still serialize proposals created before the upgrade.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.AccountI)(nil),
		&EthAccount{},
	)
	registry.RegisterImplementations(
		(*cryptotypes.PubKey)(nil),
		&EthSecp256k1PubKey{},
	)
	registry.RegisterImplementations(
		(*govv1beta1.Content)(nil),
		&RegisterCoinProposal{},
		&RegisterERC20Proposal{},
		&ToggleTokenRelayProposal{},
		&UpdateTokenPairERC20Proposal{},
	)
}
