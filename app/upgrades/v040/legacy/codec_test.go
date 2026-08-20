package legacy

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func TestLegacyEthAccountRegisteredAsGenesisAccount(t *testing.T) {
	reg := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(reg)

	acc := &EthAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address: "uptick1n6ak2gnzsxgg6q6ankz53zwc2zacrhvrxt4433",
		},
		CodeHash: "0xabc",
	}

	// The legacy account must be unpackable against both sdk.AccountI (store
	// decoding) and authtypes.GenesisAccount (genesis InitGenesis), since a
	// v0.3.3-era genesis or pre-upgrade export still contains EthAccount Any
	// records.
	anyAcc, err := codectypes.NewAnyWithValue(acc)
	require.NoError(t, err)

	var accountI sdk.AccountI
	require.NoError(t, reg.UnpackAny(anyAcc, &accountI))
	require.IsType(t, &EthAccount{}, accountI)

	var genesisAccount authtypes.GenesisAccount
	require.NoError(t, reg.UnpackAny(anyAcc, &genesisAccount))
	require.IsType(t, &EthAccount{}, genesisAccount)
	require.NoError(t, genesisAccount.Validate())

	// The type URL must remain the legacy ethermint URL so that the app hash
	// produced from a v0.3.3 genesis is preserved.
	require.Equal(t, "/ethermint.types.v1.EthAccount", anyAcc.TypeUrl)

	// The legacy pubkey must stay unpackable as a crypto PubKey.
	pubAny, err := codectypes.NewAnyWithValue(&EthSecp256k1PubKey{Key: make([]byte, 33)})
	require.NoError(t, err)
	var pubKey cryptotypes.PubKey
	require.NoError(t, reg.UnpackAny(pubAny, &pubKey))
	require.IsType(t, &EthSecp256k1PubKey{}, pubKey)
}

func TestLegacyEthAccountTypeURL(t *testing.T) {
	require.Equal(t, "ethermint.types.v1.EthAccount", proto.MessageName(&EthAccount{}))
}
