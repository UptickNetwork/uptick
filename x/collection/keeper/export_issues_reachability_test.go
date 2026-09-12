package keeper

import (
	"cosmossdk.io/x/nft"
	nftkeeper "cosmossdk.io/x/nft/keeper"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ExportIssuesWithReport is the scan the app-level diagnostics sidecar
// (<home>/export-issues.json, app/export_diagnostics.go) reads, and it is the
// module's own GetCollectionsWithReport -- the only place the class-level AND
// the list-level checks are evaluated. This test measures the "complete report"
// claim against the real reachability of each kind.
//
// It pins TWO facts that the app-level sidecar test cannot see:
//
//  1. ExportIssueSupplyMismatch is reachable and reaches the full report. This
//     is the D-G1 half: the counter can only diverge through a historical
//     bypass or a decrTotalSupply wrap, so it is planted by writing the
//     upstream counter key directly (no public API can produce one).
//
//  2. ExportIssueNFTListFailed is NOT reachable, so it can be asserted nowhere
//     -- not at the app layer, not here. The branch that reports it
//     (collection.go, GetCollectionsWithReport: `if nftErr != nil`) fires only
//     when k.GetNFTs returns a non-nil error, but GetNFTs (nft.go) has returned
//     a nil error on every path since cbc0372 replaced its `return nil, err`
//     with a graceful downgrade (see TestGetNFTsSkipsUndecodableNFT). The enum
//     value, the branch and its comment are therefore dead code: no on-chain
//     shape can make a class' NFT list "unreadable". Writing a test that
//     asserted this kind would be asserting a value that cannot exist, so this
//     test instead proves the precondition cannot hold (step 4 below) and
//     records the finding. (This is intentionally NOT a t.Skip: a skip would be
//     indistinguishable from a pass, which is the failure mode this repository
//     keeps guarding against.)
func (s *KeeperTestSuite) TestExportIssuesWithReportSurfacesSupplyMismatchAndCannotReportNFTListFailure() {
	creator := sdk.AccAddress([]byte("reachability-creator"))

	// 1. A healthy-looking class whose stored supply counter is diverged.
	s.Require().NoError(s.keeper.SaveDenom(
		s.ctx, "supply", "Supply", "", "", creator, false, false, "", "", "", ""))
	s.Require().NoError(s.keeper.SaveNFT(
		s.ctx, "supply", "t1", "T1", "", "", "", creator))

	counterKey := append(append([]byte{}, nftkeeper.ClassTotalSupply...), []byte("supply")...)
	s.Require().NoError(s.storeSvc.OpenKVStore(s.ctx).Set(counterKey, sdk.Uint64ToBigEndian(42)))

	// 2. The worst "unreadable list" shape we can build: a class whose NFT
	//    carries an undecodable Data Any.
	s.Require().NoError(s.keeper.SaveDenom(
		s.ctx, "corruptnft", "Corrupt", "", "COR", creator, false, false, "", "", "", ""))
	s.Require().NoError(s.nftKpr.Mint(s.ctx, nft.NFT{
		ClassId: "corruptnft",
		Id:      "t1",
		Uri:     "ipfs://corrupt",
		Data: &codectypes.Any{
			TypeUrl: "/uptick.collection.NFTMetadata",
			Value:   []byte{0xde, 0xad, 0xbe, 0xef},
		},
	}, creator))

	// 3. Assert the reachable kind reaches the full report.
	issues := s.keeper.ExportIssuesWithReport(s.ctx)

	var supply *ExportIssue
	for i := range issues {
		if issues[i].Kind == ExportIssueSupplyMismatch {
			supply = &issues[i]
			break
		}
	}
	s.Require().NotNil(supply, "supply_mismatch must reach the full report; got %+v", issues)
	s.Require().Equal("supply", supply.ClassID)

	// 4. Prove the nft_list_failed precondition cannot hold: GetNFTs returns a
	//    nil error even for the corrupt-data class, so the `nftErr != nil`
	//    branch is unreachable and nft_list_failed can never be reported.
	_, err := s.keeper.GetNFTs(s.ctx, "corruptnft")
	s.Require().NoError(err, "GetNFTs downgrades corrupt Data; it never returns an error")

	for _, issue := range issues {
		s.Require().NotEqual(ExportIssueNFTListFailed, issue.Kind,
			"nft_list_failed is unreachable; a report containing it means GetNFTs regained an error path (see collection.go)")
	}
}
