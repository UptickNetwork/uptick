package keeper

import (
	"testing"

	errorsmod "cosmossdk.io/errors"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Regression for P0-1: erc721 pair resolution used to route by string shape
// (common.IsHexAddress) instead of by call semantics. A denom whose id is a
// 40-nibble hex string ("aaaa...aa", allowed by the denom regex and
// permissionlessly issuable via MsgIssueDenom) was routed into the
// contract-address map, so converting that denom resolved — and minted into —
// the VICTIM pair's contract. The semantics-specific lookups below must keep
// the two namespaces strictly separated.
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

// Regression: MsgIssueDenom must reject denom ids shaped like an EVM address.
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
