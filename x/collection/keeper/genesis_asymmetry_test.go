package keeper

import (
	"fmt"
	"strings"

	"cosmossdk.io/x/nft"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/exported"
	"github.com/UptickNetwork/uptick/x/collection/types"
)

// importPanics reports whether InitGenesis panics on the given genesis. It is
// the other half of the validate/import contract: ValidateGenesis has no side
// channel, so the only way to prove the two agree is to actually import what
// validate accepted.
func (s *KeeperTestSuite) importPanics(gs types.GenesisState) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	s.keeper.InitGenesis(s.ctx, gs)
	return false
}

// TestGenesisValidationIsSymmetricWithImport pins the invariant stated in
// x/collection/types/genesis.go: "The rules here must accept exactly what
// InitGenesis can import: a genesis that passes validation but makes InitGenesis
// panic is worse than one that is rejected up front."
//
// v0.4.1 broke it for the schema/data bounds: keeper.SaveDenom gained the bound
// while ValidateGenesis did not, so an oversized class stored on chain (an
// ICS-721 voucher class written straight through the nft keeper) was exported
// verbatim, accepted by `uptickd validate-genesis`, and then panicked the node
// restarting from that export.
//
// This is a differential test, not two independent assertions on each side: it
// derives the expected verdict from the size and then requires validate and
// import to agree with it. Either half drifting -- validate becoming laxer
// (panic returns) or stricter by accident (valid genesis rejected) -- fails it.
//
// Reverse control: with the ValidateDenomMetadataBounds call in
// types/genesis.go commented out, the over-bound rows below report
// "ValidateGenesis accepted ... but InitGenesis panicked" and this test fails.
// That is how the guard was shown to catch the real bug rather than pass
// vacuously.
func (s *KeeperTestSuite) TestGenesisValidationIsSymmetricWithImport() {
	owner := sdk.AccAddress([]byte("owner"))
	creator := sdk.AccAddress([]byte("creator"))

	sizes := []int{
		0,
		1,
		types.MaxDenomSchemaLen - 1,
		types.MaxDenomSchemaLen,
		types.MaxDenomSchemaLen + 1,
		types.MaxDenomSchemaLen + 1000,
	}

	i := 0
	for _, schemaLen := range sizes {
		for _, dataLen := range sizes {
			i++

			// A fresh denom id per row: a panicking import can leave the store
			// half-written, and reuse would let one row's damage decide the
			// next row's verdict.
			denomID := fmt.Sprintf("denom%03d", i)

			denom := types.Denom{
				Id:      denomID,
				Name:    "Denom",
				Symbol:  "SYM",
				Schema:  strings.Repeat("s", schemaLen),
				Data:    strings.Repeat("d", dataLen),
				Creator: creator.String(),
			}
			one := types.NewBaseNFT("nft1", "NFT One", owner, "ipfs://nft", "", "")
			gs := types.NewGenesisState([]types.Collection{types.NewCollection(denom, []exported.NFT{one})})

			// The bound is inclusive: exactly MaxDenomSchemaLen is legal.
			wantAccept := schemaLen <= types.MaxDenomSchemaLen && dataLen <= types.MaxDenomDataLen
			label := fmt.Sprintf("schema=%d data=%d", schemaLen, dataLen)

			validateErr := types.ValidateGenesis(*gs)

			if !wantAccept {
				s.Require().Error(validateErr,
					"%s: over the bound but ValidateGenesis accepted -- import would then panic", label)
				// A rejected genesis is never imported in production, so there
				// is nothing to assert on the import side for this row.
				continue
			}

			s.Require().NoError(validateErr, "%s: within the bound but ValidateGenesis rejected", label)
			s.Require().False(s.importPanics(*gs),
				"%s: ValidateGenesis accepted but InitGenesis panicked -- the validate/import "+
					"asymmetry is back; the two sides must share ValidateDenomMetadataBounds", label)
		}
	}
}

// TestExportTruncatesOversizedClassMetadataAndKeepsGenesisImportable is the
// export-side half of the same contract.
//
// A class can hold metadata past the bounds only through a write path that
// bypasses SaveDenom, so the fixture writes it the way the ICS-721 receive path
// used to: straight through the underlying nft keeper.
//
// ExportGenesis must not emit a record ValidateGenesis rejects -- an export no
// node can start from is the failure mode the module's own doc comment calls
// out. It truncates to the same bound the validate/import side enforces and
// records the loss as an ExportIssue (which reaches both the Error log and
// <home>/export-issues.json).
//
// Reverse control: without classMetadataBoundsIssue in GetCollectionsWithReport
// the exported data keeps its full length, ValidateGenesis fails, and both the
// length assertion and the ValidateGenesis assertion below fail.
func (s *KeeperTestSuite) TestExportTruncatesOversizedClassMetadataAndKeepsGenesisImportable() {
	// A legal ICS-721 voucher class id shape, so nothing but the size bound is
	// in play.
	classID := "ibc/" + strings.Repeat("a", 64)

	oversizedData := strings.Repeat("z", types.MaxDenomDataLen+1)
	oversizedSchema := strings.Repeat("y", types.MaxDenomSchemaLen+1)

	meta := &types.DenomMetadata{
		Creator:          sdk.AccAddress([]byte("moduleaddrbytes1")).String(),
		MintRestricted:   true,
		UpdateRestricted: true,
		Schema:           oversizedSchema,
		Data:             oversizedData,
	}
	anyMeta, err := codectypes.NewAnyWithValue(meta)
	s.Require().NoError(err)

	// The write path that bypasses SaveDenom.
	s.Require().NoError(s.nftKpr.SaveClass(s.ctx, nft.Class{
		Id:   classID,
		Data: anyMeta,
	}))

	gs := s.keeper.ExportGenesis(s.ctx)
	s.Require().Len(gs.Collections, 1)

	exportedDenom := gs.Collections[0].Denom
	s.Require().Len(exportedDenom.Data, types.MaxDenomDataLen, "exported data must be clamped to the bound")
	s.Require().Len(exportedDenom.Schema, types.MaxDenomSchemaLen, "exported schema must be clamped to the bound")
	s.Require().Equal(oversizedData[:types.MaxDenomDataLen], exportedDenom.Data, "the prefix is preserved, not cleared")
	s.Require().Equal(oversizedSchema[:types.MaxDenomSchemaLen], exportedDenom.Schema)

	// The point of the whole exercise: the export is importable.
	s.Require().NoError(types.ValidateGenesis(*gs),
		"the chain's own export must satisfy its own validate-genesis")

	// Import a copy under a fresh class id. Genesis init is for an empty store,
	// and upstream SaveClass answers ErrClassExists when the class is already
	// there -- which InitGenesis turns into a panic for a reason that has
	// nothing to do with the bounds. The real restore target is empty, so
	// renaming the fixture's class is faithful isolation, not a workaround.
	denomForImport := gs.Collections[0].Denom
	denomForImport.Id = "denomimport"
	importGS := types.NewGenesisState([]types.Collection{
		types.NewCollection(denomForImport, []exported.NFT{}),
	})
	s.Require().False(s.importPanics(*importGS),
		"a node must be able to start from the chain's own export")

	// And the loss is loud, not silent.
	var found *ExportIssue
	for _, issue := range s.keeper.ExportIssuesWithReport(s.ctx) {
		if issue.Kind == ExportIssueClassMetadataTooLarge {
			issue := issue
			found = &issue
			break
		}
	}
	s.Require().NotNil(found, "the truncation must be reported as an ExportIssue")
	s.Require().Equal(classID, found.ClassID)
	s.Require().Contains(found.Detail, fmt.Sprintf("%d -> %d", len(oversizedData), types.MaxDenomDataLen))
	s.Require().Contains(found.Detail, fmt.Sprintf("%d -> %d", len(oversizedSchema), types.MaxDenomSchemaLen))
}
