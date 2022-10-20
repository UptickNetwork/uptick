package keeper

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/nft"

	"github.com/UptickNetwork/uptick/contracts"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

var _ types.MsgServer = &Keeper{}

// ConvertCoin converts native Cosmos nft into ERC721 tokens for both
// Cosmos-native and ERC721 TokenPair Owners
func (k Keeper) ConvertNFT(
	goCtx context.Context,
	msg *types.MsgConvertNFT,
) (
	*types.MsgConvertNFTResponse, error,
) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Error checked during msg validation
	receiver := common.HexToAddress(msg.Receiver)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	pair, err := k.MintingEnabled(ctx, sender, receiver.Bytes(), msg.ClassId)
	if err != nil {
		return nil, err
	}

	// Remove token pair if contract is suicided
	erc721 := common.HexToAddress(pair.Erc721Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc721)

	if acc == nil || !acc.IsContract() {
		k.DeleteTokenPair(ctx, pair)
		k.Logger(ctx).Debug(
			"deleting selfdestructed token pair from state",
			"contract", pair.Erc721Address,
		)
		// NOTE: return nil error to persist the changes from the deletion
		return nil, nil
	}

	if !k.nftKeeper.HasNFT(ctx, msg.ClassId, msg.NftId) {
		return nil, sdkerrors.Wrapf(types.ErrNFTNotExist, "nft not exist: %s", msg.NftId)
	}

	if owner := k.nftKeeper.GetOwner(ctx, msg.ClassId, msg.NftId); !owner.Equals(sender) {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not the owner of nft %s", sender, msg.NftId)
	}

	// Check ownership and execute conversion
	switch {
	case pair.IsNativeNFT():
		return k.convertNFTNativeNFT(ctx, pair, msg, receiver, sender) // case 1.1
	case pair.IsNativeERC721():
		return k.convertNFTNativeERC721(ctx, pair, msg, receiver, sender) // case 2.2
	default:
		return nil, types.ErrUndefinedOwner
	}
}

// ConvertERC721 converts ERC721 tokens into native Cosmos nft for both
// Cosmos-native and ERC721 TokenPair Owners
func (k Keeper) ConvertERC721(
	goCtx context.Context,
	msg *types.MsgConvertERC721,
) (
	*types.MsgConvertERC721Response, error,
) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Error checked during msg validation
	receiver := sdk.MustAccAddressFromBech32(msg.Receiver)
	sender := common.HexToAddress(msg.Sender)

	pair, err := k.MintingEnabled(ctx, sender.Bytes(), receiver, msg.ContractAddress)
	if err != nil {
		return nil, err
	}

	// Remove token pair if contract is suicided
	erc721 := common.HexToAddress(pair.Erc721Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc721)

	bigTokenId := new(big.Int)
	_, err = fmt.Sscan(msg.TokenId, bigTokenId)
	if err != nil {
		sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
		return nil, err
	}

	owner, err := k.QueryERC721TokenOwner(ctx, erc721, bigTokenId)
	if err != nil {
		return nil, err
	}
	if owner != sender {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not the owner of erc721 token %s", sender, msg.TokenId)
	}

	if acc == nil || !acc.IsContract() {
		k.DeleteTokenPair(ctx, pair)
		k.Logger(ctx).Debug(
			"deleting selfdestructed token pair from state",
			"contract", pair.Erc721Address,
		)
		// NOTE: return nil error to persist the changes from the deletion
		return nil, nil
	}

	// Check ownership and execute conversion
	switch {
	case pair.IsNativeNFT():
		return k.convertERC721NativeNFT(ctx, pair, msg, receiver, sender) // case 1.2
	case pair.IsNativeERC721():
		return k.convertERC721NativeERC721(ctx, pair, msg, receiver, sender) // case 2.1
	default:
		return nil, types.ErrUndefinedOwner
	}
}

// convertNFTNativeNFT handles the nft conversion for a native Cosmos nft
// token pair:
//  - escrow nft on module account
//  - mint nft and send to receiver
func (k Keeper) convertNFTNativeNFT(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertNFT,
	receiver common.Address,
	sender sdk.AccAddress,
) (
	*types.MsgConvertNFTResponse, error,
) {
	erc721 := contracts.ERC721PresetMinterPauserAutoIdsContract.ABI
	contract := pair.GetERC721Contract()

	// Escrow nft on module account
	if err := k.nftKeeper.Transfer(ctx, msg.ClassId, msg.NftId, types.ModuleAddress.Bytes()); err != nil {
		return nil, sdkerrors.Wrap(err, "failed to escrow nft")
	}

	// Get next token id
	tokenID, err := k.QueryERC721NextTokenID(ctx, contract)
	if err != nil {
		return nil, err
	}

	// Mint tokens and send to receiver
	if _, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, true, "mint", receiver); err != nil {
		return nil, err
	}

	// set nft pair
	k.SetNFTPairByNFTID(ctx, msg.NftId, tokenID.String())
	k.SetNFTPairByTokenID(ctx, tokenID.String(), msg.NftId)

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertNFT,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, msg.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, msg.NftId),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, tokenID.String()),
			),
		},
	)

	return &types.MsgConvertNFTResponse{}, nil
}

// convertNFTNativeERC721 handles the nft conversion for a native ERC721 token
// pair:
//  - escrow nft on module account
//  - unescrow nft that have been previously escrowed with ConvertERC721 and send to receiver
//  - burn escrowed nft
func (k Keeper) convertNFTNativeERC721(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertNFT,
	receiver common.Address,
	sender sdk.AccAddress,
) (
	*types.MsgConvertNFTResponse, error,
) {
	erc721 := contracts.ERC721PresetMinterPauserAutoIdsContract.ABI
	contract := pair.GetERC721Contract()

	// burn nft
	if err := k.nftKeeper.Burn(ctx, msg.ClassId, msg.NftId); err != nil {
		return nil, err
	}

	// query tokenID by given nftID
	tokenID := string(k.GetNFTPairByNFTID(ctx, msg.NftId))
	// sender := common.Address{msg.Sender}

	res, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, true, "safeTransferFrom",receiver,receiver, msg.NftId)
	if err != nil {
		return nil, err
	}

	// delete nft pair
	k.DeleteNFTPairByNFTID(ctx, msg.NftId)
	k.DeleteNFTPairByTokenID(ctx, tokenID)

	// Check for unexpected `Approval` event in logs
	if err := k.monitorApprovalEvent(res); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertNFT,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, msg.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, msg.NftId),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, tokenID),
			),
		},
	)

	return &types.MsgConvertNFTResponse{}, nil
}

// convertERC721NativeNFT handles the erc721 conversion for a native Cosmos nft
// token pair:
//  - burn escrowed nft
//  - unescrow nft that have been previously escrowed
func (k Keeper) convertERC721NativeNFT(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertERC721,
	receiver sdk.AccAddress,
	sender common.Address,
) (
	*types.MsgConvertERC721Response, error,
) {
	erc721 := contracts.ERC721PresetMinterPauserAutoIdsContract.ABI
	contract := pair.GetERC721Contract()

	// Burn escrowed tokens
	if _, err := k.CallEVM(ctx, erc721, common.HexToAddress(msg.Sender), contract, true, "burn", big.NewInt(0)); err != nil { // TODO
		return nil, err
	}

	// query nftID by given tokenID
	nftID := string(k.GetNFTPairByTokenID(ctx, msg.TokenId))

	// unlock nft
	if err := k.nftKeeper.Transfer(ctx, pair.ClassId, nftID, receiver); err != nil {
		return nil, err
	}

	// delete nft pair
	k.DeleteNFTPairByNFTID(ctx, nftID)
	k.DeleteNFTPairByTokenID(ctx, msg.TokenId)

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC721,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, pair.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, nftID),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, msg.TokenId),
			),
		},
	)

	return &types.MsgConvertERC721Response{}, nil
}

// convertERC721NativeERC721 handles the erc721 conversion for a native erc721 token
// pair:
//  - escrow tokens on module account
//  - mint nft to the receiver: nftID: tokenAddress|tokenID
func (k Keeper) convertERC721NativeERC721(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertERC721,
	receiver sdk.AccAddress,
	sender common.Address,
) (
	*types.MsgConvertERC721Response, error,
) {
	erc721 := contracts.ERC721PresetMinterPauserAutoIdsContract.ABI
	contract := pair.GetERC721Contract()

	bigTokenId := new(big.Int)
	_, err := fmt.Sscan(msg.TokenId, bigTokenId)
	if err != nil {
		sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
		return nil, err
	}

	// Escrow tokens on module account
	res, err := k.CallEVM(ctx, erc721, sender, contract, true, "safeTransferFrom", sender,types.ModuleAddress, bigTokenId)
	if err != nil {
		return nil, err
	}

	tokenID, success := big.NewInt(0).SetString(msg.TokenId, 10)
	if !success {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid tokenID")
	}

	// query erc721 token
	token, err := k.QueryERC721Token(ctx, contract, tokenID)
	if err != nil {
		return nil, err
	}

	// generate nftID
	// nftID := contract.String() + "|" + msg.TokenId
	// xxl TODO
	nftID := "Cat-" + msg.TokenId
	nft := nft.NFT{
		ClassId: types.CreateClassID(contract.String()),
		Id:      nftID,
		Uri:     token.URI,
	}

	// mint nft
	if err := k.nftKeeper.Mint(ctx, nft, receiver); err != nil {
		return nil, err
	}

	// save nft pair
	k.SetNFTPairByNFTID(ctx, nftID, msg.TokenId)
	k.SetNFTPairByTokenID(ctx, msg.TokenId, nftID)

	// Check for unexpected `Approval` event in logs
	if err := k.monitorApprovalEvent(res); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC721,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, pair.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, nftID),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, msg.TokenId),
			),
		},
	)

	return &types.MsgConvertERC721Response{}, nil
}
