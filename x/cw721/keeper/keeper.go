package keeper

import (
	"fmt"
	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

// Keeper of this module maintains collections of cw721.
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramstore paramtypes.Subspace

	accountKeeper        types.AccountKeeper
	nftKeeper            nftkeeper.Keeper
	cwKeeper             wasmkeeper.Keeper
	cwPermissionedKeeper *wasmkeeper.PermissionedKeeper
	ics4Wrapper          porttypes.ICS4Wrapper
	ibcKeeper            ibcnfttransferkeeper.Keeper
}

// NewKeeper creates new instances of the cw721 Keeper
func NewKeeper(storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ps paramtypes.Subspace,
	ak types.AccountKeeper,
	nk nftkeeper.Keeper,
	ek wasmkeeper.Keeper,
	pk *wasmkeeper.PermissionedKeeper,
	ik ibcnfttransferkeeper.Keeper) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		storeKey:             storeKey,
		cdc:                  cdc,
		paramstore:           ps,
		accountKeeper:        ak,
		nftKeeper:            nk,
		cwKeeper:             ek,
		cwPermissionedKeeper: pk,
		ibcKeeper:            ik,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetICS4Wrapper sets the ICS4 wrapper to the keeper.
// It panics if already set
func (k *Keeper) SetICS4Wrapper(ics4Wrapper porttypes.ICS4Wrapper) {
	if k.ics4Wrapper != nil {
		panic("ICS4 wrapper already set")
	}

	k.ics4Wrapper = ics4Wrapper
}
