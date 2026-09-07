package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// testAddr returns a valid bech32 address for whatever prefix the SDK config
// is using (defaults to "cosmos" in a bare unit test). Using AccAddress.String()
// keeps the test prefix-agnostic and avoids mutating the global SDK config.
func testAddr(seed string) string {
	return sdk.AccAddress([]byte(seed)).String()
}

func validCW721Pair(t *testing.T, class string) (TokenPair, string, string) {
	t.Helper()
	contract := testAddr("cw721-contract-addr")
	owner := testAddr("cw721-owner-addr")
	return NewTokenPair(contract, class), contract, owner
}

func TestCW721GenesisState_ValidatePerTokenState(t *testing.T) {
	p, contract, owner := validCW721Pair(t, "kitty")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID(p.Cw721Address, "1"), NftUid: CreateNFTUID(p.ClassId, "nft-1")},
	}
	gs.RefundReceivers = []RefundReceiver{
		{ContractAddress: contract, TokenId: "1", Owner: owner},
	}

	require.NoError(t, gs.Validate())
}

func TestCW721GenesisState_ValidateRejectsDupTokenUID(t *testing.T) {
	p, _, _ := validCW721Pair(t, "kitty")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID(p.Cw721Address, "1"), NftUid: CreateNFTUID(p.ClassId, "nft-1")},
		{TokenUid: CreateTokenUID(p.Cw721Address, "1"), NftUid: CreateNFTUID(p.ClassId, "nft-2")},
	}

	require.Error(t, gs.Validate())
}

func TestCW721GenesisState_ValidateRejectsDupNFTUID(t *testing.T) {
	p, _, _ := validCW721Pair(t, "kitty")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID(p.Cw721Address, "1"), NftUid: CreateNFTUID(p.ClassId, "nft-1")},
		{TokenUid: CreateTokenUID(p.Cw721Address, "2"), NftUid: CreateNFTUID(p.ClassId, "nft-1")},
	}

	require.Error(t, gs.Validate())
}

func TestCW721GenesisState_ValidateRejectsOrphanRefund(t *testing.T) {
	p, _, owner := validCW721Pair(t, "kitty")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.RefundReceivers = []RefundReceiver{
		{ContractAddress: testAddr("cw721-other-unregistered-contract"), TokenId: "7", Owner: owner},
	}

	require.Error(t, gs.Validate())
}

func TestCW721GenesisState_ValidateRejectsEmptyUID(t *testing.T) {
	p, _, _ := validCW721Pair(t, "kitty")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID(p.Cw721Address, "1"), NftUid: ""},
	}

	require.Error(t, gs.Validate())
}

func TestCW721GenesisState_ValidateRejectsOrphanUID(t *testing.T) {
	p, _, _ := validCW721Pair(t, "kitty")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID(testAddr("cw721-other-unregistered-contract"), "1"), NftUid: CreateNFTUID("other-class", "nft-1")},
	}

	require.Error(t, gs.Validate())
}
