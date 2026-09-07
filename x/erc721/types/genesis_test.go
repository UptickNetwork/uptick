package types

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func validPair(t *testing.T, ercHex, class string) TokenPair {
	t.Helper()
	return NewTokenPair(common.HexToAddress(ercHex), class)
}

func TestGenesisState_Validate(t *testing.T) {
	t.Parallel()
	p := validPair(t, "0x3333333333333333333333333333333333333333", "pair/class/a")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	require.NoError(t, gs.Validate())

	dupErc := NewGenesisState(DefaultParams(), []TokenPair{
		validPair(t, "0x4444444444444444444444444444444444444444", "c1"),
		validPair(t, "0x4444444444444444444444444444444444444444", "c2"),
	})
	require.Error(t, dupErc.Validate())

	dupClass := NewGenesisState(DefaultParams(), []TokenPair{
		validPair(t, "0x5555555555555555555555555555555555555555", "same/class"),
		validPair(t, "0x6666666666666666666666666666666666666666", "same/class"),
	})
	require.Error(t, dupClass.Validate())
}

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()
	gs := DefaultGenesisState()
	require.NotNil(t, gs)
	require.Empty(t, gs.TokenPairs)
	require.NoError(t, gs.Validate())
}

// TestGenesisState_ValidatePerTokenState: a genesis carrying valid per-token
// bindings and refund receivers must validate cleanly now that Validate() runs
// ValidateGenesisPairs.
func TestGenesisState_ValidatePerTokenState(t *testing.T) {
	p := validPair(t, "0x3333333333333333333333333333333333333333", "pair/class/a")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID(p.Erc721Address, "1"), NftUid: CreateNFTUID(p.ClassId, "nft-1")},
	}
	gs.RefundReceivers = []RefundReceiver{
		{EvmContractAddress: p.Erc721Address, TokenId: "1", EvmAddress: "0x2222222222222222222222222222222222222222"},
	}

	require.NoError(t, gs.Validate())
}

// TestGenesisState_ValidateRejectsDupTokenUID: one-to-one must hold across the
// whole batch; Validate() must surface it (previously only InitGenesis panicked).
func TestGenesisState_ValidateRejectsDupTokenUID(t *testing.T) {
	p := validPair(t, "0x3333333333333333333333333333333333333333", "pair/class/a")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID(p.Erc721Address, "1"), NftUid: CreateNFTUID(p.ClassId, "nft-1")},
		{TokenUid: CreateTokenUID(p.Erc721Address, "1"), NftUid: CreateNFTUID(p.ClassId, "nft-2")},
	}

	require.Error(t, gs.Validate())
}

// TestGenesisState_ValidateRejectsDupNFTUID.
func TestGenesisState_ValidateRejectsDupNFTUID(t *testing.T) {
	p := validPair(t, "0x3333333333333333333333333333333333333333", "pair/class/a")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID(p.Erc721Address, "1"), NftUid: CreateNFTUID(p.ClassId, "nft-1")},
		{TokenUid: CreateTokenUID(p.Erc721Address, "2"), NftUid: CreateNFTUID(p.ClassId, "nft-1")},
	}

	require.Error(t, gs.Validate())
}

// TestGenesisState_ValidateRejectsOrphanRefund: a refund receiver whose
// contract is not a registered pair must be rejected by Validate().
func TestGenesisState_ValidateRejectsOrphanRefund(t *testing.T) {
	p := validPair(t, "0x3333333333333333333333333333333333333333", "pair/class/a")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.RefundReceivers = []RefundReceiver{
		{EvmContractAddress: "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", TokenId: "7", EvmAddress: "0x2222222222222222222222222222222222222222"},
	}

	require.Error(t, gs.Validate())
}

// TestGenesisState_ValidateRejectsEmptyUID.
func TestGenesisState_ValidateRejectsEmptyUID(t *testing.T) {
	p := validPair(t, "0x3333333333333333333333333333333333333333", "pair/class/a")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: "", NftUid: CreateNFTUID(p.ClassId, "nft-1")},
	}

	require.Error(t, gs.Validate())
}

func TestGenesisState_ValidateRejectsOrphanUID(t *testing.T) {
	p := validPair(t, "0x3333333333333333333333333333333333333333", "pair/class/a")
	gs := NewGenesisState(DefaultParams(), []TokenPair{p})
	gs.NftUidPairs = []NFTUIDPair{
		{TokenUid: CreateTokenUID("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "1"), NftUid: CreateNFTUID("other-class", "nft-1")},
	}

	require.Error(t, gs.Validate())
}

func TestGenesisState_ValidateRejectsChecksumDuplicate(t *testing.T) {
	addr := "0x3333333333333333333333333333333333333333"
	gs := NewGenesisState(DefaultParams(), []TokenPair{
		validPair(t, addr, "c1"),
		{Erc721Address: strings.ToUpper(addr), ClassId: "c2"},
	})
	require.Error(t, gs.Validate())
}
