package keeper

import (
	"testing"

	errorsmod "cosmossdk.io/errors"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Pair resolution must route by call semantics, not string shape: a denom
// whose id is a 40-nibble hex string (issuable via MsgIssueDenom) would
// otherwise resolve into the contract-address namespace and mint into another
// pair's contract. The lookups below must keep the two namespaces separated.
func TestAudit_PairNamespaceSeparation(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	victimContract := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	victimPair := erc721types.NewTokenPair(common.HexToAddress(victimContract), "victimclass")
	victimID := victimPair.GetID()
	k.SetTokenPair(ctx, victimPair)
	k.SetClassMap(ctx, victimPair.ClassId, victimID)
	k.SetERC721Map(ctx, victimPair.GetERC721Contract(), victimID)

	// Hex-shaped denom identical to the victim contract (the MsgIssueDenom
	// attack input): starts with 'a', 40 nibbles, passes the denom regex.
	hexDenom := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	t.Run("class lookup must not hit the contract namespace", func(t *testing.T) {
		_, err := k.GetPairByClass(ctx, hexDenom)
		require.Error(t, err)
		require.True(t, isNotFound(err))

		// The combined shape-routed lookup WOULD have resolved it (documenting
		// the old behaviour that must never be reintroduced in handlers).
		require.NotEmpty(t, k.GetTokenPairID(ctx, hexDenom))
	})

	t.Run("contract lookup must not hit the class namespace", func(t *testing.T) {
		// A class id equal to the contract address must not resolve the
		// victim pair through the EVM lookup.
		_, err := k.GetPairByEVM(ctx, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		require.Error(t, err)
		require.True(t, isNotFound(err))
	})

	t.Run("semantic lookups still resolve registered pairs", func(t *testing.T) {
		got, err := k.GetPairByClass(ctx, "victimclass")
		require.NoError(t, err)
		require.Equal(t, victimID, got.GetID())

		got, err = k.GetPairByEVM(ctx, victimContract)
		require.NoError(t, err)
		require.Equal(t, victimID, got.GetID())
	})
}

// MsgIssueDenom must reject denom ids shaped like an EVM address.
func TestAudit_IssueDenomRejectsHexAddressShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		denom string
		valid bool
	}{
		{"aaaa0000000000000000000000000000000aaaa1", false},  // 40 hex nibbles
		{"0xaaaa0000000000000000000000000000000aaa1", false}, // 0x-prefixed
		{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", false},  // uppercase hex
		{"aaaa0000000000000000000000000000000aaa", true},     // 39 nibbles: not an address
		{"kitty", true},
		{"uptick1234", true},
	}

	for _, tc := range cases {
		err := collectiontypes.ValidateIssueDenomID(tc.denom)
		if tc.valid {
			require.NoError(t, err, "denom %s", tc.denom)
		} else {
			require.Error(t, err, "denom %s", tc.denom)
		}
	}
}

func isNotFound(err error) bool {
	return errorsmod.IsOf(err, erc721types.ErrTokenPairNotFound)
}

// RegisterNFT must reject a contract that is already registered under another
// class: a duplicate binding would overwrite the first pair's ERC721Map entry
// and orphan its class.
func TestAudit_RegisterNFTRejectsExistingContract(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	contract := common.HexToAddress("0x3333333333333333333333333333333333333333")
	first := "first-class"
	second := "different-class-same-contract"

	// First registration with (contract, first) must succeed.
	_, err := k.RegisterNFT(ctx, &erc721types.MsgConvertNFT{
		ClassId:            first,
		EvmContractAddress: contract.Hex(),
	})
	require.NoError(t, err)
	require.True(t, k.IsERC721Registered(ctx, contract))
	require.True(t, k.IsClassRegistered(ctx, first))
	require.False(t, k.IsClassRegistered(ctx, second),
		"second class must not be registered at this point")

	// A SECOND pair that tries to bind the SAME contract under a different
	// class must be rejected.
	_, err = k.RegisterNFT(ctx, &erc721types.MsgConvertNFT{
		ClassId:            second,
		EvmContractAddress: contract.Hex(),
	})
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, erc721types.ErrTokenPairAlreadyExists),
		"contract collision must surface ErrTokenPairAlreadyExists, got %v", err)

	// The second class must still NOT be registered (state must not be
	// partially mutated by the failed registration).
	require.False(t, k.IsClassRegistered(ctx, second),
		"rejected registration must not create a partial state")

	// The first pair must remain intact and lookup-able through the
	// contract axis (no overwrite).
	boundClass := k.GetClassMap(ctx, first)
	require.NotEmpty(t, boundClass)
	boundID := k.GetERC721Map(ctx, contract)
	require.Equal(t, boundClass, boundID)
}
