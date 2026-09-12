package keeper

import (
	"fmt"

	"cosmossdk.io/x/nft"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// SaveCollection saves every NFT in the collection, returning the first error
// (an NFT that already exists is one: upstream Mint rejects duplicates).
func (k Keeper) SaveCollection(ctx sdk.Context, collection types.Collection) error {
	for _, nft := range collection.NFTs {
		if err := k.SaveNFT(
			ctx,
			collection.Denom.Id,
			nft.GetID(),
			nft.GetName(),
			nft.GetURI(),
			nft.GetURIHash(),
			nft.GetData(),
			nft.GetOwner(),
		); err != nil {
			return err
		}
	}
	return nil
}

// GetCollection returns the collection by the specified denom ID
func (k Keeper) GetCollection(ctx sdk.Context, denomID string) (types.Collection, error) {
	denom, err := k.GetDenomInfo(ctx, denomID)
	if err != nil {
		return types.Collection{}, err
	}

	nfts, err := k.GetNFTs(ctx, denomID)
	if err != nil {
		return types.Collection{}, err
	}
	return types.NewCollection(*denom, nfts), nil
}

// ExportIssueKind classifies a recoverable problem found while exporting
// module state.
type ExportIssueKind string

const (
	// ExportIssueNFTListFailed means the NFT list of a class could not be read,
	// so neither the class nor its NFTs could be exported.
	ExportIssueNFTListFailed ExportIssueKind = "nft_list_failed"
	// ExportIssueClassMetadata means a class' metadata blob could not be
	// decoded: only its base class fields could be exported.
	ExportIssueClassMetadata ExportIssueKind = "class_metadata_undecodable"
	// ExportIssueClassMetadataMissing means a class carries no metadata blob at
	// all, so its collection-level fields (creator, schema, data, restrictions)
	// are absent on chain. That is the shape nft-transfer leaves on ICS-721
	// voucher classes (it calls the underlying nft keeper directly), and of
	// classes predating the metadata wrapper. Nothing the chain held is dropped,
	// but a round-tripped genesis carries those classes with empty metadata.
	ExportIssueClassMetadataMissing ExportIssueKind = "class_metadata_missing"
	// ExportIssueSupplyMismatch means the class' stored total-supply counter
	// disagrees with the number of NFTs actually stored under it. The counter is
	// maintained by the upstream nft keeper's mint/burn, and every write path in
	// this repo goes through them, so a mismatch means a historical bypass
	// diverged it -- or decrTotalSupply wrapped a zero counter to 2^64-1 -- and
	// every supply query for the class is now wrong. It is the one degradation a
	// round-tripped genesis does NOT preserve: the counter is not in
	// GenesisState and is recomputed on import, so the mismatch would quietly
	// vanish. Reported so it cannot vanish unobserved.
	ExportIssueSupplyMismatch ExportIssueKind = "supply_mismatch"
)

// ExportIssue is one recoverable problem encountered during genesis export. It
// is surfaced to the caller so a degradation is reported explicitly instead of
// hiding behind a successful-looking export.
type ExportIssue struct {
	ClassID string
	Kind    ExportIssueKind
	Detail  string
}

func (i ExportIssue) String() string {
	return fmt.Sprintf("%s: class %q: %s", i.Kind, i.ClassID, i.Detail)
}

// GetCollections returns all collections, logging each degradation at Error
// level; the machine-readable list is GetCollectionsWithReport.
//
// A class with undecodable metadata keeps its base fields -- never dropped
// (skipping it used to delete the class and all its NFTs from the genesis
// while the export still reported success).
//
// No error return, deliberately: the iteration cannot fail as a whole (every
// degradation is per class, already an ExportIssue), and the error this used to
// return was never non-nil, leaving a branch in SupplyInvariant unreachable.
func (k Keeper) GetCollections(ctx sdk.Context) []types.Collection {
	cs, issues := k.GetCollectionsWithReport(ctx)
	for _, issue := range issues {
		k.Logger(ctx).Error("genesis export degraded", "issue", issue.String())
	}
	return cs
}

// GetCollectionsWithReport is GetCollections plus the explicit list of
// degradations that happened during the iteration.
func (k Keeper) GetCollectionsWithReport(ctx sdk.Context) ([]types.Collection, []ExportIssue) {
	var (
		cs     []types.Collection
		issues []ExportIssue
	)

	for _, class := range k.nk.GetClasses(ctx) {
		nfts, nftErr := k.GetNFTs(ctx, class.Id)
		if nftErr != nil {
			// Nothing can be exported for this class; keep iterating so one
			// bad class cannot abort the whole export, but record it — going
			// quiet here is what made the data loss invisible.
			issues = append(issues, ExportIssue{
				ClassID: class.Id,
				Kind:    ExportIssueNFTListFailed,
				Detail:  nftErr.Error(),
			})
			continue
		}

		denom, denomErr := k.GetDenomInfo(ctx, class.Id)
		if denomErr != nil {
			// Downgrade instead of dropping: the class and its NFTs stay in
			// the export, only the undecodable metadata fields are lost.
			denom = denomFromClass(class)
		}

		// The metadata is reported through the shared per-class check so the
		// export path and the standalone ExportIssues scan can never disagree
		// about what counts as a degraded class.
		if issue := k.classMetadataIssue(class); issue != nil {
			issues = append(issues, *issue)
		}

		// The supply check rides along with the NFT walk that is already
		// happening: len(nfts) is the independent count, the class' stored
		// counter is the other side of the comparison.
		if issue := k.supplyIssue(ctx, class.Id, uint64(len(nfts))); issue != nil {
			issues = append(issues, *issue)
		}

		cs = append(cs, types.NewCollection(*denom, nfts))
	}
	return cs, issues
}

// classMetadataIssue reports a class whose metadata blob cannot be represented
// faithfully in a genesis file: either it is absent, or it is present but does
// not decode into DenomMetadata. Returns nil for a healthy class.
func (k Keeper) classMetadataIssue(class *nft.Class) *ExportIssue {
	if class.Data == nil {
		return &ExportIssue{
			ClassID: class.Id,
			Kind:    ExportIssueClassMetadataMissing,
			Detail: "class has no metadata blob on chain; the exported genesis will carry " +
				"empty creator/schema/data and unset restrictions for it",
		}
	}

	var denomMetadata types.DenomMetadata
	if err := k.cdc.Unmarshal(class.Data.GetValue(), &denomMetadata); err != nil {
		return &ExportIssue{
			ClassID: class.Id,
			Kind:    ExportIssueClassMetadata,
			Detail:  err.Error(),
		}
	}
	return nil
}

// supplyIssue reports a class whose stored total-supply counter disagrees with
// the number of NFTs it actually holds. held is passed in because every caller
// already holds the NFT list; re-reading it would double the cost of the only
// scan that can observe this.
//
// It is the single predicate behind both the export diagnostic and
// SupplyInvariant, so the two cannot disagree about a broken supply -- the same
// reason classMetadataIssue is shared between the export path and the
// standalone ExportIssues scan.
func (k Keeper) supplyIssue(ctx sdk.Context, classID string, held uint64) *ExportIssue {
	stored := k.GetTotalSupply(ctx, classID)
	if stored == held {
		return nil
	}
	return &ExportIssue{
		ClassID: classID,
		Kind:    ExportIssueSupplyMismatch,
		Detail: fmt.Sprintf(
			"class holds %d nft(s) but its stored total supply counter is %d", held, stored),
	}
}

// ExportIssues scans for the class-level degradations described by
// classMetadataIssue, without building the collection list.
//
// It is the CHEAP scan for callers that must not walk every NFT: it reads only
// the class records. The app-level sidecar no longer uses it (that report needs
// the complete set and calls ExportIssuesWithReport), but it is what the
// round-trip tests assert on.
//
// Two kinds are NOT visible here, because observing them needs the NFT list that
// only the export walk builds -- ExportIssueNFTListFailed and
// ExportIssueSupplyMismatch -- and are reported by GetCollectionsWithReport
// instead. Counting NFTs here would defeat the split and walk every NFT twice on
// a disaster-recovery path.
func (k Keeper) ExportIssues(ctx sdk.Context) []ExportIssue {
	var issues []ExportIssue
	for _, class := range k.nk.GetClasses(ctx) {
		if issue := k.classMetadataIssue(class); issue != nil {
			issues = append(issues, *issue)
		}
	}
	return issues
}

// ExportIssuesWithReport returns the FULL set of degradations an export of the
// collection module can report, including the two kinds ExportIssues cannot see
// (ExportIssueNFTListFailed, ExportIssueSupplyMismatch) because observing them
// needs the per-class NFT list.
//
// It is what the app-level sidecar (<home>/export-issues.json) reads, so the
// durable report and the export walk cannot disagree: both are
// GetCollectionsWithReport, the only place the supply and NFT-list checks run.
// This closes the D-G1 gap, where supply_mismatch -- the one check in the repo
// that can detect a diverged counter -- was logged but never persisted, and so
// vanished with the process.
//
// COST: this re-runs the full per-class NFT walk that ExportGenesis already did.
// Accepted deliberately: it only happens on an operator-initiated export
// (disaster recovery / chain restart), never in a consensus handler.
//
// The alternative -- stash the issue list on the keeper for the app to read
// back -- would avoid the second walk but add hidden mutable state to a keeper
// copied by value into x/erc721, x/cw721 and x/internft (see the convertedNFTs
// field comment in keeper.go), and make the report depend on whether
// ExportGenesis had run. A pure re-read keeps the keeper stateless.
func (k Keeper) ExportIssuesWithReport(ctx sdk.Context) []ExportIssue {
	_, issues := k.GetCollectionsWithReport(ctx)
	return issues
}

// denomFromClass builds a Denom from the base class fields, leaving every
// metadata-derived field at its zero value. Used only by the export path so a
// single corrupt metadata blob cannot delete a class from the genesis file.
func denomFromClass(class *nft.Class) *types.Denom {
	return &types.Denom{
		Id:          class.Id,
		Name:        class.Name,
		Symbol:      class.Symbol,
		Description: class.Description,
		Uri:         class.Uri,
		UriHash:     class.UriHash,
	}
}

// GetTotalSupply returns the number of NFTs by the specified denom ID
func (k Keeper) GetTotalSupply(ctx sdk.Context, denomID string) uint64 {
	return k.nk.GetTotalSupply(ctx, denomID)
}

// GetTotalSupplyOfOwner returns how many NFTs of `id` the owner holds.
func (k Keeper) GetTotalSupplyOfOwner(ctx sdk.Context, id string, owner sdk.AccAddress) (supply uint64) {
	return k.nk.GetBalance(ctx, id, owner)
}
