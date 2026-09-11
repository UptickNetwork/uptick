package cw721

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/UptickNetwork/uptick/x/cw721/keeper"
	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	data types.GenesisState,
) {
	if err := k.SetParams(ctx, data.Params); err != nil {
		panic(err) // genesis InitGenesis must not silently skip params
	}

	// ensure cw721 module account is set on genesis
	if acc := accountKeeper.GetModuleAccount(ctx, types.ModuleName); acc == nil {
		panic("the cw721 module account has not been set")
	}

	for _, pair := range data.TokenPairs {
		id := pair.GetID()
		if err := k.SetTokenPair(ctx, pair); err != nil {
			// genesis import must not silently drop a token pair
			panic(err)
		}
		k.SetClassMap(ctx, pair.ClassId, id)
		k.SetCW721Map(ctx, pair.Cw721Address, id)
	}

	// Restore the per-token conversion bindings and IBC refund receivers that
	// collection-level TokenPairs cannot represent. Integrity violations
	// (duplicates, orphans) must abort genesis import, not pass
	// silently.
	if err := importPerTokenState(ctx, k, data); err != nil {
		panic(err)
	}
}

// importPerTokenState validates and restores the per-token runtime state
// (bidirectional NFT UID pairs and IBC refund receivers). It is a separate
// function so the integrity checks are unit-testable without constructing a
// full auth AccountKeeper.
func importPerTokenState(ctx sdk.Context, k keeper.Keeper, data types.GenesisState) error {
	if err := types.ValidateGenesisPairs(data.NftUidPairs, data.RefundReceivers, data.TokenPairs); err != nil {
		return err
	}

	for _, pair := range data.NftUidPairs {
		k.SetGenesisNFTUIDPair(ctx, pair)
	}

	for _, receiver := range data.RefundReceivers {
		k.SetGenesisRefundReceiver(ctx, receiver)
	}
	return nil
}

// ExportGenesis export module status
//
// Degrade and report, never abort: the records that can be represented are
// exported, and every damaged record is listed at Error level (and, on the
// node's export path, in <home>/export-issues.json -- see
// app/export_diagnostics.go). This matches x/collection.
//
// The previous behaviour was fail-closed (panic with the list of damaged
// keys). That made the diagnosis excellent but the export unusable: a single
// corrupt key locked the whole chain out of its own backup, which is the worst
// possible failure mode on a disaster-recovery path. Reporting the damage and
// still producing a genesis keeps both properties -- the operator can see
// exactly what was dropped, and the chain can still be restored.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	tokenPairs, pairIssues := k.GetTokenPairsWithReport(ctx)
	nftUIDPairs, uidIssues := k.ExportNFTUIDPairsWithReport(ctx)
	refundReceivers, refundIssues := k.ExportRefundReceiversWithReport(ctx)

	logExportIssues(ctx, keeper.MergeExportIssues(pairIssues, uidIssues, refundIssues))

	return &types.GenesisState{
		Params:          k.GetParams(ctx),
		TokenPairs:      tokenPairs,
		NftUidPairs:     nftUIDPairs,
		RefundReceivers: refundReceivers,
	}
}

// logExportIssues records the degradations of a genesis export at Error level
// so they are visible in the node log even when nobody reads the diagnostics
// file.
func logExportIssues(ctx sdk.Context, issues []keeper.GenesisExportIssue) {
	if len(issues) == 0 {
		return
	}

	logger := ctx.Logger()
	for _, issue := range issues {
		logger.Error("ExportGenesis: export degraded", "module", types.ModuleName, "issue", issue.String())
	}
	logger.Error("ExportGenesis: exported genesis is partially degraded; the records listed above are missing from it",
		"module", types.ModuleName, "dropped_records", len(issues))
}
