package v2

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	"time"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// Migrate is used to migrate nft data from irismod/nft to x/nft
func Migrate(ctx sdk.Context,
	storeService store.KVStoreService,
	cdc codec.Codec,
	logger log.Logger,
	saveDenom SaveDenom,
) error {
	logger.Info("migrate store data from version 1 to 2")
	startTime := time.Now()
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, KeyDenom(""))
	defer iterator.Close()

	k := keeper{
		storeService: storeService,
		cdc:          cdc,
	}

	var (
		denomNum        int64
		tokenNum        int64
		skippedDenomNum int64
		skippedTokenNum int64
	)
	for ; iterator.Valid(); iterator.Next() {
		var denom types.Denom
		if err := cdc.Unmarshal(iterator.Value(), &denom); err != nil {
			logger.Error("failed to unmarshal denom during v2 migration", "error", err.Error())
			skippedDenomNum++
			continue
		}

		creator, err := sdk.AccAddressFromBech32(denom.Creator)
		if err != nil {
			logger.Error("skipping denom with invalid creator during v2 migration", "denom", denom.Id, "error", err.Error())
			skippedDenomNum++
			continue
		}

		if err := saveDenom(ctx, denom.Id,
			denom.Name,
			denom.Schema,
			denom.Symbol,
			creator,
			denom.MintRestricted,
			denom.UpdateRestricted,
			denom.Description,
			denom.Uri,
			denom.UriHash,
			denom.Data,
		); err != nil {
			return err
		}

		// delete old keys only after new data is successfully saved
		store.Delete(KeyDenom(denom.Id))
		store.Delete(KeyDenomName(denom.Name))
		store.Delete(KeyCollection(denom.Id))

		tokenInDenom, skippedInDenom, err := migrateToken(ctx, k, logger, denom.Id)
		if err != nil {
			return err
		}
		denomNum++
		tokenNum += tokenInDenom
		skippedTokenNum += skippedInDenom

	}
	logger.Info("migrate store data success",
		"denomTotalNum", denomNum,
		"tokenTotalNum", tokenNum,
		"skippedDenomNum", skippedDenomNum,
		"skippedTokenNum", skippedTokenNum,
		"consume", time.Since(startTime).String(),
	)
	return nil
}
func migrateToken(
	ctx sdk.Context,
	k keeper,
	logger log.Logger,
	denomID string,
) (migrated int64, skipped int64, err error) {
	var iterator storetypes.Iterator
	defer func() {
		if iterator != nil {
			_ = iterator.Close()
		}
	}()

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator = storetypes.KVStorePrefixIterator(store, KeyNFT(denomID, ""))
	for ; iterator.Valid(); iterator.Next() {
		var baseNFT types.BaseNFT
		if err := k.cdc.Unmarshal(iterator.Value(), &baseNFT); err != nil {
			logger.Error("failed to unmarshal NFT during v2 migration", "error", err.Error())
			skipped++
			continue
		}

		owner, err := sdk.AccAddressFromBech32(baseNFT.Owner)
		if err != nil {
			logger.Error("skipping NFT with invalid owner during v2 migration", "denomID", denomID, "token", baseNFT.Id, "error", err.Error())
			skipped++
			continue
		}

		if err := k.saveNFT(ctx, denomID,
			baseNFT.Id,
			baseNFT.Name,
			baseNFT.URI,
			baseNFT.UriHash,
			baseNFT.Data,
			owner,
		); err != nil {
			return 0, skipped, err
		}

		// delete old keys only after new data is successfully saved
		store.Delete(KeyNFT(denomID, baseNFT.Id))
		store.Delete(KeyOwner(owner, denomID, baseNFT.Id))
		migrated++
	}
	logger.Info("migrate nft success", "denomID", denomID, "nftNum", migrated, "skippedNftNum", skipped)
	return migrated, skipped, nil
}
