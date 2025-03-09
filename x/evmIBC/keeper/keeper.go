package keeper

import (
	"cosmossdk.io/log"
	"fmt"

	"github.com/UptickNetwork/evm-nft-convert/types"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cw721keep "github.com/UptickNetwork/wasm-nft-convert/keeper"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"

	erc721keeper "github.com/UptickNetwork/evm-nft-convert/keeper"
)

// Keeper of this module maintains collections of erc721.
type Keeper struct {
	ibcKeeper    ibcnfttransferkeeper.Keeper
	cw721Keeper  cw721keep.Keeper
	erc721keeper erc721keeper.Keeper
}

//// NewKeeper creates new instances of the erc721 Keeper
//func NewKeeper(
//	ek erc721keeper.Keeper,
//	ik ibcnfttransferkeeper.Keeper,
//) Keeper {
//
//	return Keeper{
//		erc721keeper: ek,
//		ibcKeeper:    ik,
//	}
//}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetCw721Keeper sets the ICS4 wrapper to the keeper.
// It panics if already set
func (k *Keeper) SetCw721Keeper(cw721keeper cw721keep.Keeper) {

	k.cw721Keeper = cw721keeper
}

// SetErc721Keeper sets the ICS4 wrapper to the keeper.
// It panics if already set
func (k *Keeper) SetErc721Keeper(crc721keeper erc721keeper.Keeper) {

	k.erc721keeper = crc721keeper
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
