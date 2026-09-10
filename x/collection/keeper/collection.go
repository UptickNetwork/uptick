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
func (k Keeper) GetCollections(ctx sdk.Context) ([]types.Collection, error) {
	cs, issues := k.GetCollectionsWithReport(ctx)
	for _, issue := range issues {
		k.Logger(ctx).Error("genesis export degraded", "issue", issue.String())
	}
	return cs, nil
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
			issues = append(issues, ExportIssue{
				ClassID: class.Id,
				Kind:    ExportIssueClassMetadata,
				Detail:  denomErr.Error(),
			})
		}

		cs = append(cs, types.NewCollection(*denom, nfts))
	}
	return cs, issues
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
