package app

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	cosmosevmutils "github.com/cosmos/evm/utils"
	cosmoserc20types "github.com/cosmos/evm/x/erc20/types"
)

// The denom shapes this chain can store, and the fact that none of them may be
// case-folded, were previously pinned only by prose. This file makes them
// executable.
//
//	ibc/<UPPER-64-hex>  ibc-go renders Denom.Hash() through cmtbytes.HexBytes,
//	                    whose String() is strings.ToUpper(hex): the hash is
//	                    uppercase by construction, not by convention.
//	erc20:<EIP-55>      cosmos/go-ethereum's Address.String() is checksummed,
//	                    i.e. deliberately MIXED case.
//	auptick / auoc      native denoms, lowercase since genesis.
//
// v0.3.3 produced a fourth shape, "erc20/<addr>", which no longer exists; the
// first test below is the canary for it.
//
// Every lookup is byte-exact -- bank's SupplyKey (0x00||denom), the erc20 denom
// index, the IBC denom map -- so folding case on the way in normalises nothing
// and turns a hit into a silent miss. sdk.ValidateDenom will not object,
// because its regex accepts A-Za-z.

// TestErc20NativeDenomPrefixIsPinned is the canary for a prefix that has
// already moved once, silently.
//
// At tag v0.3.3 the self-developed module built this denom as
// ModuleName + "/" + addr, i.e. "erc20/0x..." (x/erc20/types/proposal.go).
// cosmos/evm builds it as Erc20NativeCoinDenomPrefix + addr, i.e. "erc20:0x...".
// The v0.3.3 -> v0.5.0 upgrade therefore changes what a NEW registration stores,
// and no migration exists for the old shape -- app/upgrades/v040/upgrades.go
// records that decision and the live-chain read that made it affordable.
//
// If a later cosmos/evm bump moves the prefix a third time, this must go red
// here rather than silently strand another generation of pairs.
func TestErc20NativeDenomPrefixIsPinned(t *testing.T) {
	require.Equal(t, "erc20:", cosmoserc20types.Erc20NativeCoinDenomPrefix)
	require.Equal(t, "erc20:0xabc", cosmoserc20types.CreateDenom("0xabc"))

	legacyShape := "erc20/" + "0xabc"
	require.NotEqual(t, legacyShape, cosmoserc20types.CreateDenom("0xabc"),
		"v0.3.3 used ModuleName+\"/\" (erc20/0x...); if CreateDenom returns that "+
			"shape again, every pair written under the current scheme becomes "+
			"unreachable by denom lookup, because the lookup is byte-exact")
}

// TestErc20NativeDenomIsStoredChecksummed pins the casing the ERC20-native denom
// actually carries in state, and shows the trap: the wrong casing is still a
// VALID denom, so nothing rejects it on the way in -- it simply is not the key
// that was written.
func TestErc20NativeDenomIsStoredChecksummed(t *testing.T) {
	const checksummed = "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"

	// Registration accepts any casing of the address because the entry point
	// validates with IsHexAddress and normalises through HexToAddress, then
	// builds the denom from Address.String().
	for _, tc := range []struct {
		name, in string
	}{
		{"canonical", checksummed},
		{"lowercase", strings.ToLower(checksummed)},
		{"uppercase", "0x" + strings.ToUpper(checksummed[2:])},
	} {
		t.Run(tc.name, func(t *testing.T) {
			addr := common.HexToAddress(tc.in)
			require.Equal(t, checksummed, addr.String(),
				"Address.String() is EIP-55 checksummed, so every spelling "+
					"converges on one state key")
			require.Equal(t, "erc20:"+checksummed, cosmoserc20types.CreateDenom(addr.String()))
		})
	}

	lowerDenom := "erc20:" + strings.ToLower(checksummed)
	require.NoError(t, sdk.ValidateDenom(lowerDenom),
		"sdk.ValidateDenom accepts A-Za-z, so a mis-cased denom passes validation")
	require.NoError(t, sdk.ValidateDenom(cosmoserc20types.CreateDenom(checksummed)))
	require.NotEqual(t, cosmoserc20types.CreateDenom(checksummed), lowerDenom,
		"two valid denoms, two different state keys -- which is why tooling must "+
			"never fold case on a denom")
}

// TestIBCDenomHashIsUppercaseByConstruction pins where the uppercase hashes on
// both live chains come from, and shows the lowercase spelling is accepted by
// sdk but is a different key.
func TestIBCDenomHashIsUppercaseByConstruction(t *testing.T) {
	require.Equal(t, "ibc", transfertypes.DenomPrefix)

	denom := transfertypes.NewDenom("uatom", transfertypes.NewHop("transfer", "channel-1"))
	ibcDenom := denom.IBCDenom()
	require.True(t, strings.HasPrefix(ibcDenom, "ibc/"), "got %q", ibcDenom)

	hash := strings.TrimPrefix(ibcDenom, "ibc/")
	require.Len(t, hash, 64)
	require.Equal(t, strings.ToUpper(hash), hash,
		"ibc-go renders Denom.Hash() via cmtbytes.HexBytes.String() = "+
			"strings.ToUpper(hex); the uppercase hash is a protocol invariant")

	require.NoError(t, sdk.ValidateDenom(ibcDenom))
	require.NoError(t, sdk.ValidateDenom("ibc/"+strings.ToLower(hash)),
		"the lowercase spelling is not rejected either -- it is simply a different key")
}

// TestIBCVoucherDerivationUsesTheLastTwentyHashBytes pins the STRv2 derivation
// this chain maps an inbound voucher to, from pure inputs.
//
// Worth pinning because the pre-upgrade module did NOT derive anything: it
// deployed a real ERC20 contract per voucher and stored that address, which is
// why the four pairs mainnet carries today disagree with the derivation -- and
// why the v0.4.0 migration deletes every OWNER_MODULE pair instead of keeping
// it (see app/upgrades/v040/erc20_legacy_pairs_test.go, which pins exactly that
// disagreement on the live addresses). Do not "fix" this test to match a
// pre-upgrade pair.
func TestIBCVoucherDerivationUsesTheLastTwentyHashBytes(t *testing.T) {
	denom := transfertypes.NewDenom("uatom", transfertypes.NewHop("transfer", "channel-1")).IBCDenom()

	hashBytes, err := transfertypes.ParseHexHash(strings.TrimPrefix(denom, "ibc/"))
	require.NoError(t, err)
	require.Len(t, hashBytes, 32)

	derived, err := cosmosevmutils.GetIBCDenomAddress(denom)
	require.NoError(t, err)
	require.Equal(t, common.BytesToAddress(hashBytes[12:]), derived,
		"the derivation is the last 20 bytes of the parsed ICS-20 hash")

	// And the rendered form is the EIP-55 checksum, not the lowercase one: that
	// string is what ends up in state whenever an address is turned into a denom.
	require.Equal(t, derived.String(), derived.Hex(),
		"in this go-ethereum fork both String() and Hex() render the checksum "+
			"form; upstream's Hex() is lowercase, so code copied from upstream "+
			"may assume the wrong casing")
	require.Equal(t, common.BytesToAddress(hashBytes[12:]).String(), derived.String())
}

// TestUpstreamStillStripsTheLegacyPrefixFromNames documents that cosmos/evm
// still recognises "erc20/": residue of the prefix change, in
// removeInvalidPrefixes (x/erc20/types/utils.go). It sanitises the ERC20 token
// DISPLAY name only -- it normalises no denom and is not a migration. If this
// goes red, the shim is gone and the legacy shape is no longer referenced
// anywhere upstream.
func TestUpstreamStillStripsTheLegacyPrefixFromNames(t *testing.T) {
	require.Equal(t, "USDC", cosmoserc20types.SanitizeERC20Name("erc20/USDC"))
	require.Equal(t, "USDC", cosmoserc20types.SanitizeERC20Name("erc20:USDC"))
}
