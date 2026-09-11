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

// ExportIssues scans for the class-level degradations described by
// classMetadataIssue, without building the collection list.
//
// It exists so the app-level export diagnostics report can list what an export
// degraded without walking every NFT a second time: the checks here only read
// the class records themselves. A class whose NFT list fails to read is only
// observable while the NFTs are being iterated, and is reported by
// GetCollectionsWithReport (and the export log) instead.
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
