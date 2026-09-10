package internft

import (
	"cosmossdk.io/log"

	"cosmossdk.io/x/nft"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	nfttransfer "github.com/bianjieai/nft-transfer/types"

	nftkeeper "github.com/UptickNetwork/uptick/x/collection/keeper"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// NewInterNftKeeper creates a new ics721 Keeper instance
func NewInterNftKeeper(cdc codec.Codec,
	k nftkeeper.Keeper,
	ak AccountKeeper,
) InterNftKeeper {
	return InterNftKeeper{
		nk:  k.NFTkeeper(),
		cdc: cdc,
		ak:  ak,
		cb:  types.NewClassBuilder(cdc, ak.GetModuleAddress),
		tb:  types.NewTokenBuilder(cdc),
	}
}

// CreateOrUpdateClass implement the method of ICS721Keeper.CreateOrUpdateClass
func (ik InterNftKeeper) CreateOrUpdateClass(ctx sdk.Context,
	classID,
	classURI,
	classData string,
) error {
	var (
		class nft.Class
		err   error
	)
	if len(classData) != 0 {
		class, err = ik.cb.Build(classID, classURI, classData)
		if err != nil {
			return err
		}
	} else {
		var denomMetadata = &types.DenomMetadata{
			Creator:          ik.ak.GetModuleAddress(types.ModuleName).String(),
			MintRestricted:   true,
			UpdateRestricted: true,
		}

		metadata, err := codectypes.NewAnyWithValue(denomMetadata)
		if err != nil {
			return err
		}
		class = nft.Class{
			Id:   classID,
			Uri:  classURI,
			Data: metadata,
		}
	}

	if ik.nk.HasClass(ctx, classID) {
		return ik.nk.UpdateClass(ctx, class)
	}
	return ik.nk.SaveClass(ctx, class)
}

// Mint implement the method of ICS721Keeper.Mint
func (ik InterNftKeeper) Mint(ctx sdk.Context,
	classID,
	tokenID,
	tokenURI,
	tokenData string,
	receiver sdk.AccAddress,
) error {
	token, err := ik.tb.Build(classID, tokenID, tokenURI, tokenData)
	if err != nil {
		return err
	}
	return ik.nk.Mint(ctx, token, receiver)
}

// Transfer implement the method of ICS721Keeper.Transfer
func (ik InterNftKeeper) Transfer(
	ctx sdk.Context,
	classID,
	tokenID,
	tokenData string,
	receiver sdk.AccAddress,
) error {
	if err := ik.nk.Transfer(ctx, classID, tokenID, receiver); err != nil {
		return err
	}
	if len(tokenData) == 0 {
		return nil
	}
	nft, found := ik.nk.GetNFT(ctx, classID, tokenID)
	if !found {
		return nil
	}
	token, err := ik.tb.Build(classID, tokenID, nft.Uri, tokenData)
	if err != nil {
		return err
	}

	return ik.nk.Update(ctx, token)
}

// GetClass implement the method of ICS721Keeper.GetClass
func (ik InterNftKeeper) GetClass(ctx sdk.Context, classID string) (nfttransfer.Class, bool) {
	class, exist := ik.nk.GetClass(ctx, classID)
	if !exist {
		return nil, false
	}

	metadata, err := ik.cb.BuildMetadata(class)
	if err != nil {
		ik.Logger(ctx).Error("encode class data failed", "error", err.Error(), "classId", classID)
		return nil, false
	}
	return InterClass{
		ID:   classID,
		URI:  class.Uri,
		Data: metadata,
	}, true
}

// GetNFT implement the method of ICS721Keeper.GetNFT
func (ik InterNftKeeper) GetNFT(ctx sdk.Context, classID, tokenID string) (nfttransfer.NFT, bool) {
	nft, has := ik.nk.GetNFT(ctx, classID, tokenID)
	if !has {
		return nil, false
	}
	metadata, err := ik.tb.BuildMetadata(nft)
	if err != nil {
		ik.Logger(ctx).Error("encode nft data failed", "error", err.Error(), "classId", classID, "tokenId", tokenID)
		return nil, false
	}
	return InterToken{
		ClassID: classID,
		ID:      tokenID,
		URI:     nft.Uri,
		Data:    metadata,
	}, true
}

// Burn implement the method of ICS721Keeper.Burn
//
// Fault-tolerant on missing NFTs (matches x/erc721 and x/cw721
// RefundPacketToken): if the NFT is already gone, skipping keeps the IBC
// refund path alive instead of rolling back the whole cache context.
//
// The absence check is a plain HasNFT read on purpose. An earlier version
// wrapped it in a blanket recover(), which swallowed every panic — including
// the SDK gas meter's OutOfGas and store/runtime panics — and then reported
// the burn as successful with reason "nft_already_gone". That silently turned
// a failed transaction into a successful ICS-721 burn, so the recover() was
// removed: only a genuine HasNFT == false may skip, everything else must
// propagate.
func (ik InterNftKeeper) Burn(ctx sdk.Context, classID string, tokenID string) error {
	if !ik.nk.HasNFT(ctx, classID, tokenID) {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"inter_nft_burn_skip",
				sdk.NewAttribute("class_id", classID),
				sdk.NewAttribute("token_id", tokenID),
				sdk.NewAttribute("reason", "nft_already_gone"),
			),
		)
		ik.Logger(ctx).Debug(
			"interNft.Burn: NFT not present in store, skipping burn",
			"class_id", classID,
			"token_id", tokenID,
			"reason", "nft_already_gone",
		)
		return nil
	}
	return ik.nk.Burn(ctx, classID, tokenID)
}

// GetOwner implement the method of ICS721Keeper.GetOwner
func (ik InterNftKeeper) GetOwner(ctx sdk.Context, classID string, tokenID string) sdk.AccAddress {
	return ik.nk.GetOwner(ctx, classID, tokenID)
}

// HasClass implement the method of ICS721Keeper.HasClass
func (ik InterNftKeeper) HasClass(ctx sdk.Context, classID string) bool {
	return ik.nk.HasClass(ctx, classID)
}

// Logger returns a module-specific logger.
func (ik InterNftKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "ics721/NFTKeeper")
}
