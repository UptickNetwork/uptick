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
// Fail-closed, and diagnosable: every damaged record in the live store is
// collected first, and the export is aborted with the complete repair list
// instead of either panicking on the first bad key (the old bare panic gave an
// operator nothing to act on) or dropping records from a "successful" genesis.
// A genesis that silently loses conversions or refunds is worse than a failed
// export.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	tokenPairs, pairIssues := k.GetTokenPairsWithReport(ctx)
	nftUIDPairs, uidIssues := k.ExportNFTUIDPairsWithReport(ctx)
	refundReceivers, refundIssues := k.ExportRefundReceiversWithReport(ctx)

	issues := make([]keeper.GenesisExportIssue, 0, len(pairIssues)+len(uidIssues)+len(refundIssues))
	issues = append(issues, pairIssues...)
	issues = append(issues, uidIssues...)
	issues = append(issues, refundIssues...)

	if len(issues) > 0 {
		panic(&keeper.GenesisExportError{Module: types.ModuleName, Issues: issues})
	}

	return &types.GenesisState{
		Params:          k.GetParams(ctx),
		TokenPairs:      tokenPairs,
		NftUidPairs:     nftUIDPairs,
		RefundReceivers: refundReceivers,
	}
}
