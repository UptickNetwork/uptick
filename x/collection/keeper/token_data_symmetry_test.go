package keeper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cosmossdk.io/x/nft"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// TestOversizedTokenDataRoundTripsThroughGenesis is the guard that the token
// metadata bound added to TokenBuilder.Build did NOT become a validate/import
// asymmetry -- the failure mode that produced F-001 on the denom side.
//
// The bound sits on the ICS-721 receive path only. A token whose
// NFTMetadata.Data already exceeds it -- written before the gate existed, or
// through MsgMintNFT / MsgEditNFT, which never call Build -- must still be
// exportable AND importable. The two halves of that claim:
//
//  1. ExportGenesis emits the blob verbatim, so ValidateGenesis has to keep
//     accepting it. If someone adds the same predicate to the validate/import
//     side, the chain can no longer validate its own backup, which is F-001
//     with the signs reversed.
//  2. InitGenesis restores it without panicking, so the node can start from
//     that export.
//
// Reverse control: adding `ValidateTokenMetadataBounds(nft.GetData())` to the
// per-NFT loop in types/genesis.go leaves the export-length assertions green
// and turns the ValidateGenesis assertion red -- the asymmetry, made visible.
// Deleting the mint below instead makes the length assertion fail, so neither
// half can pass vacuously.
func (s *KeeperTestSuite) TestOversizedTokenDataRoundTripsThroughGenesis() {
	creator := sdk.AccAddress([]byte("creator-tokendata"))
	s.Require().NoError(s.keeper.SaveDenom(
		s.ctx, "tokendenom", "Token Denom", "", "TD", creator, false, false, "", "", "", ""))

	oversized := strings.Repeat("z", types.MaxTokenDataLen+1)

	// The write path that bypasses Build: the underlying nft keeper, the same
	// way an ICS-721 voucher token was written before the receive-side gate
	// existed.
	anyMeta, err := codectypes.NewAnyWithValue(&types.NFTMetadata{Data: oversized})
	s.Require().NoError(err)
	s.Require().NoError(s.nftKpr.Mint(s.ctx, nft.NFT{
		ClassId: "tokendenom",
		Id:      "nft-token-data",
		Uri:     "ipfs://token-data",
		Data:    anyMeta,
	}, creator))

	gs := s.keeper.ExportGenesis(s.ctx)
	s.Require().Len(gs.Collections, 1)
	s.Require().Len(gs.Collections[0].NFTs, 1)

	// 1a. The export carries the blob verbatim: the token bound is not an
	// export-side truncation, so nothing is lost on the way out.
	exportedData := gs.Collections[0].NFTs[0].GetData()
	s.Require().Len(exportedData, len(oversized),
		"the export must neither drop nor truncate token data")
	s.Require().Equal(oversized, exportedData)

	// 1b. And the chain's own export still satisfies its own validate-genesis.
	s.Require().NoError(types.ValidateGenesis(*gs),
		"the token bound must not reach ValidateGenesis: a genesis the export "+
			"produces has to pass validation, or the backup is unstartable")

	// 2. Import a copy under a fresh class id (genesis init targets an empty
	// store, and upstream SaveClass answers ErrClassExists otherwise -- a panic
	// unrelated to the bound).
	denomForImport := gs.Collections[0].Denom
	denomForImport.Id = "tokendenomimport"
	importGS := &types.GenesisState{
		Collections: []types.Collection{{
			Denom: denomForImport,
			NFTs:  gs.Collections[0].NFTs,
		}},
	}
	s.Require().False(s.importPanics(*importGS),
		"a node must be able to start from a genesis holding pre-existing oversized token data")

	// Import does not silently clamp either: the bytes survive the round trip.
	restored, err := s.keeper.GetNFT(s.ctx, "tokendenomimport", "nft-token-data")
	s.Require().NoError(err)
	s.Require().Len(restored.GetData(), len(oversized),
		"import must preserve the bytes it was given, not clamp them")
}

// TestKeeperTokenDataPathsDoNotBound is the keeper-side companion to
// x/collection/types's TestTokenMetadataBoundStaysOnTheReceiveSide.
//
// SaveNFT sits on BOTH the genesis import path (InitGenesis -> SaveCollection)
// and the Msg path (MsgMintNFT); GetNFTs sits on the export path. Bounding
// either of them would be the exact F-001 mistake in a new place -- import
// would reject state the export is obliged to emit, and InitGenesis would
// panic on the chain's own backup. The gate belongs on the receive path
// (TokenBuilder.Build) and nowhere else.
//
// Reverse control: adding `types.ValidateTokenMetadataBounds(tokenData)` to
// SaveNFT turns this test red. It was exercised when the guard was written.
func (s *KeeperTestSuite) TestKeeperTokenDataPathsDoNotBound() {
	sources := readKeeperSources(s.T(), ".")

	for _, file := range []string{"nft.go", "collection.go", "genesis.go"} {
		src, ok := sources[file]
		s.Require().True(ok, "sanity: keeper/%s must be readable", file)
		s.Require().NotContains(src, "ValidateTokenMetadataBounds",
			"keeper/%s is on the export or import path; the token bound must not "+
				"be enforced there or the export/validate/import pair desynchronises", file)
		s.Require().NotContains(src, "MaxTokenDataLen",
			"keeper/%s names the token metadata bound; the receive-path gate must "+
				"stay the only enforcement point", file)
	}
}

// readKeeperSources returns filename -> source for the non-test .go files in
// dir, with line comments stripped so prose about a bound cannot trip a
// source-level assertion. Same idiom as
// x/collection/types/metadata_bounds_test.go's readNonTestGoFiles, repeated
// here because the two tests live in different packages.
func readKeeperSources(t *testing.T, dir string) map[string]string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	out := make(map[string]string)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)

		var b strings.Builder
		for _, line := range strings.Split(string(raw), "\n") {
			if idx := strings.Index(line, "//"); idx != -1 {
				line = line[:idx]
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		out[name] = b.String()
	}

	require.NotEmpty(t, out, "no Go sources read from %s", dir)
	return out
}
