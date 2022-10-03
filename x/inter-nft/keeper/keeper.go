package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/x/nft"
	nftkeeper "github.com/cosmos/cosmos-sdk/x/nft/keeper"
)

// Keeper maintains the link to data storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	storeKey storetypes.StoreKey // Unexposed key to access store from sdk.Context
	cdc      codec.Codec
	nftkeeper.Keeper
}

// NewKeeper creates a new instance of the NFT Keeper
func NewKeeper(cdc codec.Codec,
	storeKey storetypes.StoreKey,
	ak nft.AccountKeeper,
	bk nft.BankKeeper,
) Keeper {
	return Keeper{
		storeKey: storeKey,
		cdc:      cdc,
		Keeper:   nftkeeper.NewKeeper(storeKey, cdc, ak, bk),
	}
}
