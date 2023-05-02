package keeper

import (
	"context"
	"fmt"
	"github.com/UptickNetwork/uptick/x/collection/exported"
	"math/big"
	"strings"

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

	fmt.Printf("###xxl ConvertNFT 0 \n")
	ctx := sdk.UnwrapSDKContext(goCtx)

	//classId, nftIDs
	contractAddress, tokenIds, err := k.GetContractAddressAndTokenIds(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ContractAddress = contractAddress
	msg.TokenIds = tokenIds

	fmt.Printf("###xxl ConvertNFT 1 GetContractAddressAndTokenIds %v \n", msg)
	// Error checked during msg validation
	receiver := common.HexToAddress(msg.Receiver)

	fmt.Printf("###xxl ConvertNFT 0.1 %v \n", msg)
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
	fmt.Printf("###xxl ConvertNFT 2 %v \n", msg)
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
	fmt.Printf("###xxl ConvertERC721 0 %v \n", msg)
	//classId, nftId
	classId, nftIds, err := k.GetClassIDAndNFTID(ctx, msg)
	fmt.Printf("###xxl ConvertERC721 1 classId:%v nftId:%v \n", classId, nftIds)
	if err != nil {
		return nil, err
	}
	msg.ClassId = classId
	msg.NftIds = nftIds

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
	_, err = fmt.Sscan(msg.TokenIds[0], bigTokenId)
	if err != nil {
		sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
		return nil, err
	}

	owner, err := k.QueryERC721TokenOwner(ctx, erc721, bigTokenId)
	if err != nil {
		return nil, err
	}
	if owner != sender {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not the owner of erc721 token %s", sender, strings.Join(msg.TokenIds, ","))
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

	var (
		bigTokenIds []*big.Int
		reqInfo     exported.NFT
	)

	fmt.Printf("###xxl 0 convertCosmos2Evm \n")
	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	for i, tokenId := range msg.TokenIds {
		bigTokenId := new(big.Int)
		_, err := fmt.Sscan(tokenId, bigTokenId)
		if err != nil {
			sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
			return nil, err
		}
		bigTokenIds = append(bigTokenIds, bigTokenId)

		reqInfo, err = k.nftKeeper.GetNFT(ctx, msg.ClassId, msg.NftIds[i])
		if err != nil {
			return nil, err
		}

		transferNft := nftTypes.MsgTransferNFT{
			DenomId:   msg.ClassId,
			Id:        msg.NftIds[i],
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

		//	does token id exist
		owner, err := k.QueryERC721TokenOwner(ctx, common.HexToAddress(msg.ContractAddress), bigTokenIds[i])
		fmt.Printf("###xxl 1.1 convertCosmos2Evm owner %v err %v \n", owner, err)
		if err != nil {
			// mint
			// mint enhance
			_, err = k.CallEVM(
				ctx, erc721, types.ModuleAddress, contract, true,
				"mintEnhance", receiver, bigTokenIds[i], reqInfo.GetName(), reqInfo.GetURI(), reqInfo.GetData(), reqInfo.GetURIHash())
			if err != nil {
				fmt.Printf("###xxl 1.2 convertCosmos2Evm err %v \n", err)
				// mint normal
				_, err = k.CallEVM(
					ctx, erc721, receiver, contract, true,
					"mint", receiver, bigTokenIds[i])
				if err != nil {
					return nil, err
				}
			}
		} else if owner == types.ModuleAddress {
			// transfer
			_, err = k.CallEVM(
				ctx, erc721, types.ModuleAddress, contract, true,
				"safeTransferFrom", types.ModuleAddress, receiver, bigTokenIds[i])
			if err != nil {
				return nil, err
			}
		} else {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not the owner of erc721 token %s", types.ModuleAddress, msg.TokenIds)
		}

		fmt.Printf("###xxl 4 convertCosmos2Evm \n")
		// Mint tokens and send to receiver
		if err != nil {
			return nil, err
		}

	}

	fmt.Printf("###xxl 5 convertCosmos2Evm bigTokenIds %v\n", bigTokenIds)
	fmt.Printf("###xxl 5.05 convertCosmos2Evm ContractAddress %v tokeId %v \n", msg.ContractAddress, bigTokenIds[0])

	fmt.Printf("###xxl 4.5 k.SetNFTPairs msg %v \n", msg)
	for i, tokenId := range msg.TokenIds {
		k.SetNFTPairs(ctx, msg.ContractAddress, tokenId, msg.ClassId, msg.NftIds[i])
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertNFT,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, msg.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(msg.NftIds, ",")),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, strings.Join(msg.TokenIds, ",")),
			),
		},
	)

	fmt.Printf("###xxl 5 convertCosmos2Evm \n")
	return &types.MsgConvertNFTResponse{}, nil
}

// convertEvm2Cosmos handles the erc721 conversion for a native erc721 token
// pair:
//  - escrow tokens on module account
//  - mint nft to the receiver: nftId: tokenAddress|tokenID
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

	for i, tokenId := range msg.TokenIds {

		bigTokenId := new(big.Int)
		_, err := fmt.Sscan(tokenId, bigTokenId)
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

		nftId := string(k.GetNFTPairByContractTokenID(ctx, msg.ContractAddress, tokenId))
		if nftId == "" {

			mintNFT := nftTypes.MsgMintNFT{
				DenomId:   msg.ClassId,
				Id:        msg.NftIds[i],
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
				Id:        msg.NftIds[i],
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
	}

	// save nft pair
	fmt.Printf("###xxl 4.5 k.SetNFTPairs msg %v \n", msg)
	for i, tokenId := range msg.TokenIds {
		k.SetNFTPairs(ctx, msg.ContractAddress, tokenId, msg.ClassId, msg.NftIds[i])
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC721,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, pair.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(msg.NftIds, ",")),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, strings.Join(msg.TokenIds, ",")),
			),
		},
	)

	return &types.MsgConvertERC721Response{}, nil
}
