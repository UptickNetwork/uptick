package v041

import (
	"strings"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/evm/ethereum/eip712"

	"github.com/stretchr/testify/require"
)

// TestRegisterCompatInterfaces verifies that the real SDK registry accepts all
// legacy Keplr type URL registrations (happy path).
func TestRegisterCompatInterfaces(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	require.NoError(t, RegisterCompatInterfaces(registry))

	// The custom type URLs must resolve to the complete cosmos/evm types.
	pubAny, err := codectypes.NewAnyWithValue(&ethsecp256k1.PubKey{})
	require.NoError(t, err)
	var pk cryptotypes.PubKey
	require.NoError(t, registry.UnpackAny(pubAny, &pk))
	require.IsType(t, &ethsecp256k1.PubKey{}, pk)

	extAny, err := codectypes.NewAnyWithValue(&eip712.ExtensionOptionsWeb3Tx{})
	require.NoError(t, err)
	var ext txtypes.TxExtensionOptionI
	require.NoError(t, registry.UnpackAny(extAny, &ext))
	require.IsType(t, &eip712.ExtensionOptionsWeb3Tx{}, ext)
}

// failingRegistry embeds the codectypes.InterfaceRegistry *interface* (not the
// concrete implementation), so RegisterCustomTypeURL is not promoted and the
// registry does not satisfy customTypeURLRegistrar — the Keplr type URL
// registration must fail with an error (M-7: error instead of panic).
type failingRegistry struct {
	codectypes.InterfaceRegistry
}

func TestRegisterCompatInterfacesRejectsUnsupportedRegistry(t *testing.T) {
	registry := failingRegistry{codectypes.NewInterfaceRegistry()}
	err := RegisterCompatInterfaces(registry)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "/ethermint.crypto.v1.ethsecp256k1.PubKey"), "error should name the failing type URL, got: %v", err)
}
