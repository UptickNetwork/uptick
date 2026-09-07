package erc721

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/UptickNetwork/uptick/x/erc721/keeper"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	data types.GenesisState,
) {
	if err := k.SetParams(ctx, data.Params); err != nil {
		panic(err)
	}

	// ensure erc721 module account is set on genesis
	if acc := accountKeeper.GetModuleAccount(ctx, types.ModuleName); acc == nil {
		panic("the erc721 module account has not been set")
	}

	for _, pair := range data.TokenPairs {
		id := pair.GetID()
		k.SetTokenPair(ctx, pair)
		k.SetClassMap(ctx, pair.ClassId, id)
		k.SetERC721Map(ctx, pair.GetERC721Contract(), id)
	}

	// H-02: restore the per-token conversion bindings and IBC refund
	// receivers that collection-level TokenPairs cannot represent. Integrity
	// violations (duplicates, orphans) must abort genesis import, not pass
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
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	// H-02: export the per-token runtime state alongside the collection-level
	// pairs. A failure here means the live store violates the invariants the
	// genesis file must preserve; aborting the export beats writing a
	// genesis that silently loses conversions or refunds.
	nftUIDPairs, err := k.ExportNFTUIDPairs(ctx)
	if err != nil {
		panic(err)
	}
	refundReceivers, err := k.ExportRefundReceivers(ctx)
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Params:          k.GetParams(ctx),
		TokenPairs:      k.GetTokenPairs(ctx),
		NftUidPairs:     nftUIDPairs,
		RefundReceivers: refundReceivers,
	}
}
