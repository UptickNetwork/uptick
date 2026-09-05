package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

// Keeper of this module maintains collections of erc721.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	accountKeeper types.AccountKeeper
	nftKeeper     nftkeeper.Keeper
	evmKeeper     types.EVMKeeper
	ics4Wrapper   porttypes.ICS4Wrapper
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

// SetICS4Wrapper sets the ICS4 wrapper to the keeper.
// It returns an error if the wrapper has already been set.
func (k *Keeper) SetICS4Wrapper(ics4Wrapper porttypes.ICS4Wrapper) error {
	if k.ics4Wrapper != nil {
		return errors.Wrap(errortypes.ErrInvalidRequest, "ICS4 wrapper already set")
	}

	k.ics4Wrapper = ics4Wrapper
	return nil
}

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
