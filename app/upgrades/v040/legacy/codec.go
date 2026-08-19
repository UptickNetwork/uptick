package legacy

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the legacy Ethermint types (EthAccount and the
// eth_secp256k1 pubkey) solely for v0.3.3 -> v0.4.0 account migration:
//   - EthAccount  -> decoding legacy /ethermint.types.v1.EthAccount records
//   - PubKey      -> decoding BaseAccount.PubKey Any values stored under
//     /ethermint.crypto.v1.ethsecp256k1.PubKey, and verifying signatures of
//     migrated accounts after the upgrade.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.AccountI)(nil),
		&EthAccount{},
	)
	registry.RegisterImplementations(
		(*cryptotypes.PubKey)(nil),
		&EthSecp256k1PubKey{},
	)
}
