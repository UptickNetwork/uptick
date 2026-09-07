package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	collectionkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	"github.com/UptickNetwork/uptick/x/cw721/types"
)

var _ types.MsgServer = &Keeper{}

// Keeper is the x/cw721 keeper
type Keeper struct {
	storeKey          storetypes.StoreKey
	cdc               codec.BinaryCodec
	accountKeeper     types.AccountKeeper
	nftKeeper         collectionkeeper.Keeper
	wasmKeeper        types.WasmKeeper
	ibcTransferKeeper types.IBCNFTTransferKeeper
}

// NewKeeper creates a new Keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	accountKeeper types.AccountKeeper,
	nftKeeper collectionkeeper.Keeper,
	wasmKeeper types.WasmKeeper,
	ibcTransferKeeper types.IBCNFTTransferKeeper,
) Keeper {
	return Keeper{
		storeKey:          storeKey,
		cdc:               cdc,
		accountKeeper:     accountKeeper,
		nftKeeper:         nftKeeper,
		wasmKeeper:        wasmKeeper,
		ibcTransferKeeper: ibcTransferKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAccountKeeper returns the account keeper
func (k Keeper) GetAccountKeeper() types.AccountKeeper {
	return k.accountKeeper
}

// GetNFTKeeper returns the nft keeper
func (k Keeper) GetNFTKeeper() collectionkeeper.Keeper {
	return k.nftKeeper
}

// GetWasmKeeper returns the CosmWasm surface used by this module.
func (k Keeper) GetWasmKeeper() types.WasmKeeper {
	return k.wasmKeeper
}
