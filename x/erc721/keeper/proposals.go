package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/cosmos-sdk/x/nft"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// RegisterNFT deploys an erc721 contract and creates the token pair for the existing cosmos coin
func (k Keeper) RegisterNFT(ctx sdk.Context, class nft.Class) (*types.TokenPair, error) {
	// Check if the conversion is globally enabled
	params := k.GetParams(ctx)
	if !params.EnableErc721 {
		return nil, sdkerrors.Wrap(
			types.ErrERC721Disabled, "registration is currently disabled by governance",
		)
	}

	if !k.nftKeeper.HasClass(ctx, class.Id) {
		return nil, sdkerrors.Wrapf(
			types.ErrClassNotExist, "nft class not exist: %s", class.Id,
		)
	}

	// Check if class is already registered
	if k.IsClassRegistered(ctx, class.Id) {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairAlreadyExists, "class ID already registered: %s", class.Id,
		)
	}

	addr, err := k.DeployERC721Contract(ctx, class)
	if err != nil {
		return nil, sdkerrors.Wrap(
			err, "failed to create wrapped coin denom metadata for ERC721",
		)
	}

	pair := types.NewTokenPair(addr, class.Id, true, types.OWNER_MODULE)
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, common.HexToAddress(pair.Erc721Address), pair.GetID())

	return &pair, nil
}

// RegisterERC721 creates a Cosmos coin and registers the token pair between the nft and the ERC721
func (k Keeper) RegisterERC721(ctx sdk.Context, contract common.Address) (*types.TokenPair, error) {
	// Check if the conversion is globally enabled
	params := k.GetParams(ctx)
	if !params.EnableErc721 {
		return nil, sdkerrors.Wrap(types.ErrERC721Disabled, "registration is currently disabled by governance")
	}

	// Check if ERC721 is already registered
	if k.IsERC721Registered(ctx, contract) {
		return nil, sdkerrors.Wrapf(types.ErrTokenPairAlreadyExists, "token ERC721 contract already registered: %s", contract.String())
	}

	class, err := k.CreateNFTClass(ctx, contract)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to create wrapped coin denom metadata for ERC721")
	}

	pair := types.NewTokenPair(contract, class.Id, true, types.OWNER_EXTERNAL)
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, common.HexToAddress(pair.Erc721Address), pair.GetID())
	return &pair, nil
}

// CreateCoinMetadata generates the metadata to represent the ERC721 token on evmos.
func (k Keeper) CreateNFTClass(ctx sdk.Context, contract common.Address) (*nft.Class, error) {
	strContract := contract.String()

	erc721Data, err := k.QueryERC721(ctx, contract)
	if err != nil {
		return nil, err
	}

	classID := types.CreateClassID(strContract)

 	fmt.Printf("################### classID is %v+ \n",classID)
	fmt.Printf("################### ctx is %v+ \n",ctx)
	// Check if class already exists
	if found := k.nftKeeper.HasClass(ctx, classID); found {
		return nil, sdkerrors.Wrap(types.ErrInternalTokenPair, "class already exist")
	}

	if k.IsClassRegistered(ctx, classID) {
		return nil, sdkerrors.Wrapf(types.ErrInternalTokenPair, "nft class already registered: %s", classID)
	}

	class := nft.Class{
		Id:          classID,
		Name:        erc721Data.Name,
		Symbol:      erc721Data.Symbol,
		Description: "internal nft from erc721",
		Uri:         "",
		UriHash:     "",
		Data:        nil,
	}

	_ = k.nftKeeper.SaveClass(ctx, class)

	return &class, nil
}

// ToggleConversion toggles conversion for a given token pair
func (k Keeper) ToggleConversion(ctx sdk.Context, token string) (types.TokenPair, error) {
	id := k.GetTokenPairID(ctx, token)
	if len(id) == 0 {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered by id", token,
		)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return types.TokenPair{}, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered", token,
		)
	}

	pair.Enabled = !pair.Enabled

	k.SetTokenPair(ctx, pair)
	return pair, nil
}
