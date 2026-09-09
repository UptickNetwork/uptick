package keeper

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	// Also reject a second class trying to bind to the same EVM contract: a
	// pair's contract identity is fixed (the class↔contract mapping in
	// ConvertNFT relies on it), so two pairs for one contract would silently
	// overwrite each other in the EVM map and orphan the first class's pair.
	contract := common.HexToAddress(msg.EvmContractAddress)
	if k.IsERC721Registered(ctx, contract) {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairAlreadyExists,
			"contract %s already bound to a registered pair", contract.String(),
		)
	}

	pair := types.NewTokenPair(contract, msg.ClassId)
	if err := k.SetTokenPair(ctx, pair); err != nil {
		return nil, err
	}
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, contract, pair.GetID())

	return &pair, nil
}

// RegisterERC721 creates a Cosmos coin and registers the token pair between the nft and the ERC721
func (k Keeper) RegisterERC721(ctx sdk.Context, msg *types.MsgConvertERC721) (*types.TokenPair, error) {

	// Check if ERC721 is already registered
	contract := common.HexToAddress(msg.EvmContractAddress)
	if k.IsERC721Registered(ctx, contract) {
		return nil, sdkerrors.Wrapf(types.ErrTokenPairAlreadyExists,
			"token ERC721 contract already registered: %s", contract.String())
	}

	derivedClassID := types.CreateClassIDFromContractAddress(msg.EvmContractAddress)
	if strings.TrimSpace(msg.ClassId) == "" {
		msg.ClassId = derivedClassID
	} else if msg.ClassId != derivedClassID {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound,
			"class ID %s does not match ERC721 contract derived class ID %s",
			msg.ClassId,
			derivedClassID,
		)
	}

	err := k.CreateNFTClass(ctx, msg)
	if err != nil {

		return nil, sdkerrors.Wrap(err,
			"failed to create wrapped coin denom metadata for ERC721")
	}

	pair := types.NewTokenPair(contract, msg.ClassId)
	if err := k.SetTokenPair(ctx, pair); err != nil {
		return nil, err
	}
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, common.HexToAddress(pair.Erc721Address), pair.GetID())

	return &pair, nil
}

// CreateNFTClass generates the metadata to represent the ERC721 token .
func (k Keeper) CreateNFTClass(ctx sdk.Context, msg *types.MsgConvertERC721) error {

	contract := common.HexToAddress(msg.EvmContractAddress)
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
		return sdkerrors.Wrapf(types.ErrTokenPairAlreadyExists, "native NFT class already exists: %s", msg.ClassId)
	}

	err = k.nftKeeper.SaveDenom(ctx, msg.ClassId, erc721Data.Name, classEnhance.Schema,
		erc721Data.Symbol, types.AccModuleAddress, classEnhance.MintRestricted, classEnhance.UpdateRestricted,
		classEnhance.Description, classEnhance.Uri, classEnhance.UriHash, classEnhance.Data)
	if err != nil {
		return err
	}

	return nil
}
