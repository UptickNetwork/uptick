package keeper

import (
	"fmt"

	"cosmossdk.io/log"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/evmibc/types"
)

// Keeper of this module maintains collections of erc721.
type Keeper struct {
	ibcKeeper    ICS721Keeper
	cw721Keeper  CW721Converter
	erc721keeper ERC721Converter
}

// NewKeeper creates new instances of the erc721 Keeper
func NewKeeper(
	ik ICS721Keeper,
) Keeper {

	return Keeper{
		ibcKeeper: ik,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetCw721Keeper sets the CW721 keeper reference.
func (k *Keeper) SetCw721Keeper(cw721keeper CW721Converter) {
	k.cw721Keeper = cw721keeper
}

// SetErc721Keeper sets the ERC721 keeper reference.
func (k *Keeper) SetErc721Keeper(crc721keeper ERC721Converter) {
	k.erc721keeper = crc721keeper
}

// GetVoucherClassID returns the canonical IBC voucher class ID for a given
// (port, channel, originalClassID) tuple.
//
// KEEP-IN-SYNC: duplicated in x/erc721/keeper/keeper.go (the keepers cannot
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
