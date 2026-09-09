package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
)

// Keeper of this module maintains collections of erc721.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	accountKeeper types.AccountKeeper
	nftKeeper     nftkeeper.Keeper
	evmKeeper     types.EVMKeeper
	ibcKeeper     ibcnfttransferkeeper.Keeper
}

// NewKeeper creates new instances of the erc721 Keeper
func NewKeeper(storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ak types.AccountKeeper,
	nk nftkeeper.Keeper,
	ek types.EVMKeeper,
	ik ibcnfttransferkeeper.Keeper,
) Keeper {
	return Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		accountKeeper: ak,
		nftKeeper:     nk,
		evmKeeper:     ek,
		ibcKeeper:     ik,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetVoucherClassID returns the canonical IBC voucher class ID for a given
// (port, channel, originalClassID) tuple.
//
// KEEP-IN-SYNC: duplicated in x/evmibc/keeper/keeper.go (the keepers cannot
// import each other). Bodies MUST stay byte-identical.
func (k *Keeper) GetVoucherClassID(port string, channel string, classId string) string {
	// since SendPacket did not prefix the classID, we must prefix classID here
	classPrefix := ibcnfttransfertypes.GetClassPrefix(port, channel)
	// NOTE: sourcePrefix contains the trailing "/"
	prefixedClassID := classPrefix + classId

	// construct the class trace from the full raw classID
	classTrace := ibcnfttransfertypes.ParseClassTrace(prefixedClassID)
	voucherClassID := classTrace.IBCClassID()

	return voucherClassID
}
