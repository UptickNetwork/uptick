package keeper

import (
	"fmt"

	"cosmossdk.io/x/nft"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// SaveCollection saves all NFTs and returns an error if there already exists
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
	// all, so the collection-level fields it is supposed to hold (creator,
	// schema, data, mint/update restrictions) are not on chain. This is the
	// shape of an ICS-721 voucher class -- nft-transfer calls the underlying
	// nft keeper directly -- and of classes created before the metadata wrapper
	// existed. Nothing that the chain ever held is dropped, but a round-tripped
	// genesis will contain those classes with empty metadata, so the export
	// says so instead of passing for a faithful copy.
	ExportIssueClassMetadataMissing ExportIssueKind = "class_metadata_missing"
	// ExportIssueSupplyMismatch means the class' stored total-supply counter
	// disagrees with the number of NFTs actually stored under that class.
	//
	// The counter is maintained by the upstream nft keeper's mint/burn, and
	// every write path in this repository goes through them, so a mismatch
	// means the counter was diverged by a historical bypass -- or by
	// decrTotalSupply wrapping a zero counter to 2^64-1. Either way the supply
	// every nft query reports for that class is wrong, and it is the one
	// degradation that a round-tripped genesis preserves (the counter is not
	// part of GenesisState, so it is recomputed on import and the mismatch
	// quietly disappears). Reported here so it cannot disappear unobserved.
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

// GetCollections returns all the collections.
//
// A class whose metadata cannot be decoded is exported with its base fields
// only — never dropped. Skipping it used to remove the class AND all of its
// NFTs from the exported genesis while the export still reported success.
// Each degradation is logged at Error level; callers that need the machine
// readable list should use GetCollectionsWithReport.
//
// There is deliberately no error return. The iteration cannot fail as a whole
// — every degradation is per class and already recorded as an ExportIssue —
// and the error channel this used to have was never non-nil, which left the
// failure branch in SupplyInvariant as unreachable code.
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
// already has the class' NFT list in hand; reading it again here would double
// the cost of the only scan that can observe this.
//
// This is the single predicate behind both the export diagnostic and
// SupplyInvariant, so the two can never disagree about what counts as a broken
// supply -- the same reason classMetadataIssue is shared between the export
// path and the standalone ExportIssues scan.
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
// It exists so the app-level export diagnostics report can list what an export
// degraded without walking every NFT a second time: the checks here only read
// the class records themselves.
//
// Two kinds are therefore NOT visible to this scan, because observing them
// needs the NFT list that only the export walk builds:
//
//   - ExportIssueNFTListFailed, and
//   - ExportIssueSupplyMismatch,
//
// which are reported by GetCollectionsWithReport (and, on the export path, by
// the export log). Adding a count of NFTs here would defeat the point of the
// split and walk every NFT twice on a disaster-recovery path.
func (k Keeper) ExportIssues(ctx sdk.Context) []ExportIssue {
	var issues []ExportIssue
	for _, class := range k.nk.GetClasses(ctx) {
		if issue := k.classMetadataIssue(class); issue != nil {
			issues = append(issues, *issue)
		}
	}
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

// GetTotalSupplyOfOwner returns the amount of NFTs by the specified conditions
func (k Keeper) GetTotalSupplyOfOwner(ctx sdk.Context, id string, owner sdk.AccAddress) (supply uint64) {
	return k.nk.GetBalance(ctx, id, owner)
}

//// GetBalance returns the amount of NFTs by the specified conditions
// func (k Keeper) GetBalance(ctx sdk.Context, id string, owner sdk.AccAddress) (supply uint64) {
//	return k.nk.GetBalance(ctx, id, owner)
//}
