package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/x/nft"
	nftkeeper "cosmossdk.io/x/nft/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// Keeper maintains the link to data storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	storeService store.KVStoreService // Unexposed key to access store from sdk.Context
	cdc          codec.Codec
	nk           nftkeeper.Keeper
	// convertedNFTs answers "does this native NFT have a contract-side
	// counterpart?". Wiring it is what lets RemoveNFT refuse to burn a token
	// whose contract half is escrowed -- see converted_nft.go.
	//
	// It is a *pointer* to a slot rather than a plain interface field, and that
	// is load-bearing. This keeper is copied by value into x/erc721, x/cw721
	// and x/internft, all of which are constructed *before* the checker can
	// exist (it is built from x/erc721 and x/cw721 themselves). A plain field
	// would be frozen as nil inside those copies, so the late wiring in
	// app/keepers would only reach the one instance the app holds -- and the
	// guard that x/erc721's refund path depends on (it calls
	// nftKeeper.BurnNFT -> RemoveNFT through its own copy) would silently never
	// fire. Sharing one slot keeps the wiring visible through every copy.
	convertedNFTs *convertedNFTCheckerSlot
}

// NewKeeper creates a new instance of the NFT Keeper
func NewKeeper(cdc codec.Codec,
	storeService store.KVStoreService,
	ak nft.AccountKeeper,
	bk nft.BankKeeper,
) Keeper {
	return Keeper{
		storeService:  storeService,
		cdc:           cdc,
		nk:            nftkeeper.NewKeeper(storeService, cdc, ak, bk),
		convertedNFTs: &convertedNFTCheckerSlot{},
	}
}

// NFTkeeper returns a cosmos-sdk nftkeeper.Keeper.
func (k Keeper) NFTkeeper() nftkeeper.Keeper {
	return k.nk
}

// GetOwner returns the owner of the given NFT, delegating to the underlying
// x/nft keeper.
//
// Exposed so dependent modules (x/erc721) can query ownership through the
// collection keeper instead of reaching into NFTkeeper() and coupling to the
// underlying x/nft keeper type.
func (k Keeper) GetOwner(ctx sdk.Context, classID, nftID string) sdk.AccAddress {
	return k.nk.GetOwner(ctx, classID, nftID)
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("uptick/%s", types.ModuleName))
}
