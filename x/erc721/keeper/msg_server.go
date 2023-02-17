package keeper

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/contracts"
	"github.com/UptickNetwork/uptick/x/erc721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	nftTypes "github.com/UptickNetwork/uptick/x/collection/types"
)

var _ types.MsgServer = &Keeper{}

// ConvertNFT ConvertCoin converts native Cosmos nft into ERC721 tokens for both
// Cosmos-native and ERC721 TokenPair Owners
func (k Keeper) ConvertNFT(
	goCtx context.Context,
	msg *types.MsgConvertNFT,
) (
	*types.MsgConvertNFTResponse, error,
) {

	ctx := sdk.UnwrapSDKContext(goCtx)

	//classID, nftID
	contractAddress, tokenID, err := k.GetContractAddressAndTokenID(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ContractAddress = contractAddress
	msg.TokenId = tokenID

	// Error checked during msg validation
	receiver := common.HexToAddress(msg.Receiver)

	id := k.GetTokenPairID(ctx, msg.ContractAddress)
	if len(id) == 0 {
		_, err := k.RegisterNFT(ctx, msg)
		if err != nil {
			return nil, err
		}
	}

	pair, err := k.GetPair(ctx, msg.ClassId)
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

	return k.convertCosmos2Evm(ctx, pair, msg, receiver) // case 2.2

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

	//classID, nftID
	classID, nftID, err := k.GetClassIDAndNFTID(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ClassId = classID
	msg.NftId = nftID

	// Error checked during msg validation
	sender := common.HexToAddress(msg.Sender)

	id := k.GetTokenPairID(ctx, msg.ContractAddress)
	if len(id) == 0 {
		_, err := k.RegisterERC721(ctx, msg)
		if err != nil {
			return nil, err
		}
	}

	pair, err := k.GetPair(ctx, msg.ContractAddress)
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

	return k.convertEvm2Cosmos(ctx, pair, msg, sender) //

}

// convertCosmos2Evm handles the nft conversion for a native ERC721 token
// pair:
//  - escrow nft on module account
//  - unescrow nft that have been previously escrowed with ConvertERC721 and send to receiver
//  - burn escrowed nft
func (k Keeper) convertCosmos2Evm(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertNFT,
	receiver common.Address,
) (
	*types.MsgConvertNFTResponse, error,
) {

	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	bigTokenId := new(big.Int)
	_, err := fmt.Sscan(msg.TokenId, bigTokenId)
	if err != nil {
		sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
		return nil, err
	}

	reqInfo, err := k.nftKeeper.GetNFT(ctx, msg.ClassId, msg.NftId)
	if err != nil {
		return nil, err
	}

	transferNft := nftTypes.MsgTransferNFT{
		DenomId:   msg.ClassId,
		Id:        msg.NftId,
		Name:      reqInfo.GetName(),
		URI:       reqInfo.GetURI(),
		Data:      reqInfo.GetData(),
		UriHash:   reqInfo.GetURIHash(),
		Sender:    msg.Sender,
		Recipient: types.AccModuleAddress.String(),
	}
	if _, err = k.nftKeeper.TransferNFT(ctx, &transferNft); err != nil {
		return nil, err
	}

	// query tokenID by given nftID
	tokenID := string(k.GetNFTPairByNFTID(ctx, msg.ClassId, msg.NftId))
	// sender := common.Address{msg.Sender}

	//	does token id exist
	owner, err := k.QueryERC721TokenOwner(ctx, common.HexToAddress(msg.ContractAddress), bigTokenId)
	if err != nil {
		// mint
		// mint enhance
		_, err = k.CallEVM(
			ctx, erc721, receiver, contract, true,
			"mintEnhance", receiver, bigTokenId, reqInfo.GetName(), reqInfo.GetURI(), reqInfo.GetData(), reqInfo.GetURIHash())
		if err != nil {
			// mint normal
			_, err = k.CallEVM(
				ctx, erc721, receiver, contract, true,
				"mint", receiver, bigTokenId)
			if err != nil {
				return nil, err
			}
		}
	} else if owner == types.ModuleAddress {
		// transfer
		_, err = k.CallEVM(
			ctx, erc721, types.ModuleAddress, contract, true,
			"safeTransferFrom", types.ModuleAddress, receiver, bigTokenId)
		if err != nil {
			return nil, err
		}

	} else {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not the owner of erc721 token %s", types.ModuleAddress, msg.TokenId)
	}

	// Mint tokens and send to receiver
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	// delete nft pair
	k.SetNFTPairs(ctx, msg.ContractAddress, msg.TokenId, msg.ClassId, msg.NftId)

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

// convertEvm2Cosmos handles the erc721 conversion for a native erc721 token
// pair:
//  - escrow tokens on module account
//  - mint nft to the receiver: nftID: tokenAddress|tokenID
func (k Keeper) convertEvm2Cosmos(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertERC721,
	sender common.Address,
) (
	*types.MsgConvertERC721Response, error,
) {

	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	bigTokenId := new(big.Int)
	_, err := fmt.Sscan(msg.TokenId, bigTokenId)
	if err != nil {
		sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
		return nil, err
	}

	reqInfo, err := k.QueryNFTEnhance(ctx, contract, bigTokenId)
	_, err = k.CallEVM(
		ctx, erc721, sender, contract, true,
		"safeTransferFrom", sender, types.ModuleAddress, bigTokenId,
	)
	if err != nil {
		return nil, err
	}

	// query erc721 token
	_, err = k.QueryERC721Token(ctx, contract)
	if err != nil {
		return nil, err
	}

	nftID := string(k.GetNFTPairByTokenID(ctx, msg.ContractAddress, msg.TokenId))
	if nftID == "" {

		mintNFT := nftTypes.MsgMintNFT{
			DenomId:   msg.ClassId,
			Id:        msg.NftId,
			Name:      reqInfo.Name,
			URI:       reqInfo.Uri,
			Data:      reqInfo.Data,
			UriHash:   reqInfo.UriHash,
			Sender:    types.AccModuleAddress.String(),
			Recipient: msg.Receiver,
		}

		// mint nft
		if _, err = k.nftKeeper.MintNFT(ctx, &mintNFT); err != nil {
			return nil, err
		}
	} else {
		transferNft := nftTypes.MsgTransferNFT{
			DenomId:   msg.ClassId,
			Id:        msg.NftId,
			Name:      reqInfo.Name,
			URI:       reqInfo.Uri,
			Data:      reqInfo.Data,
			UriHash:   reqInfo.UriHash,
			Sender:    types.AccModuleAddress.String(),
			Recipient: msg.Receiver,
		}
		if _, err = k.nftKeeper.TransferNFT(ctx, &transferNft); err != nil {
			return nil, err
		}
	}

	// save nft pair
	k.SetNFTPairs(ctx, msg.ContractAddress, msg.TokenId, msg.ClassId, msg.NftId)

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC721,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, pair.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, msg.NftId),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, msg.TokenId),
			),
		},
	)

	return &types.MsgConvertERC721Response{}, nil
}
