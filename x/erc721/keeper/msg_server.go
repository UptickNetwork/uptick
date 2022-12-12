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

	nftTypes "github.com/irisnet/irismod/modules/nft/types"
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
	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	fmt.Printf("xxl 02 ConvertNFT 002 receive:%v - sender:%v \n", receiver, sender)

	id := k.GetTokenPairID(ctx, msg.ContractAddress)
	fmt.Printf("xxl 02 ConvertERC721 003 RegisterERC721 %v \n", id)
	if len(id) == 0 {
		fmt.Printf("xxl 02 ConvertERC721 004 RegisterERC721 start \n")
		_, err := k.RegisterNFT(ctx, msg)
		if err != nil {
			return nil, err
		}
		fmt.Printf("xxl 02 ConvertERC721 005 RegisterERC721 end \n")
	}

	fmt.Printf("xxl 02 ConvertERC721 006 GetPair start \n")
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
	fmt.Printf("#### xxl 01 ConvertERC721 001 start msg %v \n", msg)
	ctx := sdk.UnwrapSDKContext(goCtx)

	//classID, nftID
	classID, nftID, err := k.GetClassIDAndNFTID(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ClassId = classID
	msg.NftId = nftID

	// Error checked during msg validation
	receiver := sdk.MustAccAddressFromBech32(msg.Receiver)
	sender := common.HexToAddress(msg.Sender)

	fmt.Printf("xxl 01 ConvertERC721 002 receive:%v - sender:%v \n", receiver, sender)

	id := k.GetTokenPairID(ctx, msg.ContractAddress)
	fmt.Printf("xxl 01 ConvertERC721 003 RegisterERC721 %v \n", id)
	if len(id) == 0 {
		fmt.Printf("xxl 01 ConvertERC721 004 RegisterERC721 start \n")
		_, err := k.RegisterERC721(ctx, msg)
		if err != nil {
			return nil, err
		}
		fmt.Printf("xxl 01 ConvertERC721 005 RegisterERC721 end \n")
	}

	fmt.Printf("xxl 01 ConvertERC721 006 GetPair start \n")
	pair, err := k.GetPair(ctx, msg.ContractAddress)
	if err != nil {
		return nil, err
	}
	fmt.Printf("xxl 01 ConvertERC721 003 GetPair %v \n", pair)

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
	fmt.Printf("xxl 02 convertNFTNativeERC721 001 start \n")
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
	fmt.Printf("xxl 02 convertEvm2Cosmos 003 TransferNFT \n")
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
	fmt.Printf("xxl 02 convertEvm2Cosmos 005 TransferNFT %v \n", transferNft)
	if _, err = k.nftKeeper.TransferNFT(ctx, &transferNft); err != nil {
		return nil, err
	}

	// query tokenID by given nftID
	tokenID := string(k.GetNFTPairByNFTID(ctx, msg.ClassId, msg.NftId))
	// sender := common.Address{msg.Sender}
	fmt.Printf("xxl 02 convertNFTNativeERC721 002 %v \n", tokenID)

	fmt.Printf("xxl 02 k.CallEVM 003 ModuleAddress:%v,contract:%v,receiver:%v, TokenId:%v \n",
		types.ModuleAddress, contract, receiver, msg.TokenId,
	)

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
	fmt.Printf("xxl 02 after CallEVM 004 \n")

	fmt.Printf("xxl 02 k.CallEVM 004 CallEVM end \n")
	if err != nil {
		fmt.Printf("xxl 02 k.CallEVM 005 CallEVM err \n")
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

	fmt.Printf("xxl 01 convertEvm2Cosmos 001 start \n")
	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	bigTokenId := new(big.Int)
	_, err := fmt.Sscan(msg.TokenId, bigTokenId)
	if err != nil {
		sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
		return nil, err
	}
	fmt.Printf("xxl 01 convertEvm2Cosmos 002 k.CallEVM \n")

	reqInfo, err := k.QueryNFTEnhance(ctx, contract, bigTokenId)
	fmt.Printf("xxl 01 getEnhanceInfo 026 k.CallEVM res %v \n", reqInfo)

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

		fmt.Printf("xxl 01 convertEvm2Cosmos 003 MintNFT \n")
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

		fmt.Printf("xxl 01 convertEvm2Cosmos 005 MsgMintNFT %v \n", mintNFT.Sender)
		// mint nft
		if _, err = k.nftKeeper.MintNFT(ctx, &mintNFT); err != nil {
			return nil, err
		}
	} else {
		fmt.Printf("xxl 01 convertEvm2Cosmos 003 TransferNFT \n")
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

	fmt.Printf("xxl 01 convertEvm2Cosmos 004 end \n")
	return &types.MsgConvertERC721Response{}, nil
}
