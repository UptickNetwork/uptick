package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// RegisterNFT deploys an erc721 contract and creates the token pair for the existing cosmos coin
func (k Keeper) RegisterNFT(ctx sdk.Context, msg *types.MsgConvertNFT) (*types.TokenPair, error) {

	// Check if class is already registered
	if k.IsClassRegistered(ctx, msg.ClassId) {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairAlreadyExists, "class ID already registered: %s", msg.ClassId,
		)
	}

	pair := types.NewTokenPair(common.HexToAddress(msg.ContractAddress), msg.ClassId)
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, common.HexToAddress(pair.Erc721Address), pair.GetID())

	return &pair, nil
}

// RegisterERC721 creates a Cosmos coin and registers the token pair between the nft and the ERC721
func (k Keeper) RegisterERC721(ctx sdk.Context, msg *types.MsgConvertERC721) (*types.TokenPair, error) {

	// Check if ERC721 is already registered
	contract := common.HexToAddress(msg.ContractAddress)
	if k.IsERC721Registered(ctx, contract) {
		return nil, sdkerrors.Wrapf(types.ErrTokenPairAlreadyExists,
			"token ERC721 contract already registered: %s", contract.String())
	}

	err := k.CreateNFTClass(ctx, msg)
	if err != nil {
		return nil, sdkerrors.Wrap(err,
			"failed to create wrapped coin denom metadata for ERC721")
	}

	pair := types.NewTokenPair(contract, msg.ClassId)
	k.SetTokenPair(ctx, pair)
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, common.HexToAddress(pair.Erc721Address), pair.GetID())

	return &pair, nil
}

// CreateNFTClass generates the metadata to represent the ERC721 token .
func (k Keeper) CreateNFTClass(ctx sdk.Context, msg *types.MsgConvertERC721) error {

	contract := common.HexToAddress(msg.ContractAddress)
	erc721Data, err := k.QueryERC721(ctx, contract)
	if err != nil {
		return err
	}

	classEnhance, err := k.QueryClassEnhance(ctx, contract)
	if err != nil {
		// normal logic
		classEnhance.Uri = ""
		classEnhance.Data = ""
		classEnhance.Schema = ""
		classEnhance.UriHash = ""
		classEnhance.Description = ""
		classEnhance.UpdateRestricted = false
		classEnhance.MintRestricted = false
	}

	if k.IsClassRegistered(ctx, msg.ClassId) {
		return sdkerrors.Wrapf(types.ErrInternalTokenPair, "nft class already registered: %s", msg.ClassId)
	}

	_, err = k.nftKeeper.GetDenomInfo(ctx, msg.ClassId)
	if err == nil {
		return nil
	}

	err = k.nftKeeper.SaveDenom(ctx, msg.ClassId, erc721Data.Name, classEnhance.Schema,
		erc721Data.Symbol, types.AccModuleAddress, classEnhance.MintRestricted, classEnhance.UpdateRestricted,
		classEnhance.Description, classEnhance.Uri, classEnhance.UriHash, classEnhance.Data)
	if err != nil {
		return err
	}

	return nil
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

	k.SetTokenPair(ctx, pair)
	return pair, nil
}
