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

	// Error checked during msg validation
	receiver := common.HexToAddress(msg.Receiver)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	fmt.Printf("xxl 02 ConvertNFT 002 receive:%v - sender:%v \n", receiver, sender)

	id := k.GetTokenPairID(ctx, msg.ContractAddress)
	fmt.Printf("xxl 01 ConvertERC721 003 RegisterERC721 %v \n", id)
	if len(id) == 0 {
		fmt.Printf("xxl 01 ConvertERC721 004 RegisterERC721 start \n")
		_, err := k.RegisterNFT(ctx, msg)
		if err != nil {
			return nil, err
		}
		fmt.Printf("xxl 01 ConvertERC721 005 RegisterERC721 end \n")
	}

	fmt.Printf("xxl 01 ConvertERC721 006 MintingEnabled start \n")
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

	// Check ownership and execute conversion
	switch {
	case pair.IsNativeNFT():
		fmt.Printf("xxl 02 ConvertNFT 071 IsNativeNFT k.convertNFTNativeNFT \n")
		return k.convertNFTNativeNFT(ctx, pair, msg, receiver, sender) // case 1.1
	case pair.IsNativeERC721():
		fmt.Printf("xxl 02 ConvertERC721 072 IsNativeERC721 k.convertNFTNativeERC721 \n")
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
	fmt.Printf("#### xxl 01 ConvertERC721 001 start msg %v \n", msg)
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Error checked during msg validation
	fmt.Printf("..xxl 01 before MustAccAddressFromBech32 Receiver %v \n", msg.Receiver)
	receiver := sdk.MustAccAddressFromBech32(msg.Receiver)
	fmt.Printf("..xxl 01 after MustAccAddressFromBech32 Receiver %v \n", receiver)

	fmt.Printf("..xxl 01 before HexToAddress Sender %v \n", msg.Sender)
	sender := common.HexToAddress(msg.Sender)
	fmt.Printf("..xxl 01 after HexToAddress Sender %v \n", sender)

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

	fmt.Printf("xxl 01 ConvertERC721 006 MintingEnabled start \n")
	pair, err := k.MintingEnabled(ctx, sender.Bytes(), receiver, msg.ContractAddress)
	if err != nil {
		return nil, err
	}
	fmt.Printf("xxl 01 ConvertERC721 003 MintingEnabled %v \n", pair)

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
		fmt.Printf("xxl 01 ConvertERC721 071 IsNativeNFT k.convertERC721NativeNFT \n")
		return k.convertERC721NativeNFT(ctx, pair, msg, receiver, sender) // case 1.2
	case pair.IsNativeERC721():
		fmt.Printf("xxl 01 ConvertERC721 072 IsNativeERC721 k.convertERC721NativeERC721 \n")
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

	fmt.Printf("xxl 03 convertNFTNativeNFT 001 start \n")
	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	// Escrow nft on module account
	//if err := k.nftKeeper.Transfer(ctx, msg.ClassId, msg.NftId, types.ModuleAddress.Bytes()); err != nil {
	//	return nil, sdkerrors.Wrap(err, "failed to escrow nft")
	//}

	// Get next token id
	tokenID, err := k.QueryERC721NextTokenID(ctx, contract)
	if err != nil {
		return nil, err
	}
	fmt.Printf("xxl 03 convertNFTNativeNFT 002 %v \n", tokenID)

	fmt.Printf("xxl 03 before CallEVM 003 convertNFTNativeNFT 002 erc721:%v ModuleAddress:%v,contract:%v,receiver:%v\n",
		erc721,
		types.ModuleAddress,
		contract,
		receiver,
	)
	// Mint tokens and send to receiver
	if _, err := k.CallEVM(ctx, erc721, types.ModuleAddress, contract, true, "mint", receiver); err != nil {
		return nil, err
	}
	fmt.Printf("xxl 03 after CallEVM 004 \n")

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
	fmt.Printf("xxl 02 convertNFTNativeERC721 001 start \n")

	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	// burn nft
	nft := nftTypes.MsgBurnNFT{
		DenomId: msg.ClassId,
		Id:      msg.NftId,
		Sender:  msg.Sender,
	}

	if _, err := k.nftKeeper.BurnNFT(ctx, &nft); err != nil {
		return nil, err
	}

	// query tokenID by given nftID
	tokenID := string(k.GetNFTPairByNFTID(ctx, msg.NftId))
	// sender := common.Address{msg.Sender}
	fmt.Printf("xxl 02 convertNFTNativeERC721 002 %v \n", tokenID)

	fmt.Printf("xxl 02 k.CallEVM 003 ModuleAddress:%v,contract:%v,receiver:%v, TokenId:%v \n",
		types.ModuleAddress, contract, receiver, msg.TokenId,
	)
	bigTokenId := new(big.Int)
	_, err := fmt.Sscan(msg.TokenId, bigTokenId)
	if err != nil {
		sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
		return nil, err
	}
	reqInfo, _ := k.QueryERC721Enhance(ctx, contract, bigTokenId)
	fmt.Printf("xxl 02 reqInfo 004 %v \n", reqInfo)

	//res, err := k.CallEVM(ctx, erc721, receiver, contract, true, "mintEnhance",
	//	receiver,
	//	bigTokenId,
	//	reqInfo.Name,
	//	reqInfo.Uri,
	//	reqInfo.Data,
	//	reqInfo.UriHash,
	//)
	//xxl 04
	res, err := k.CallEVM(
		ctx, erc721, types.ModuleAddress, contract, true,
		"safeTransferFrom", types.ModuleAddress, receiver, bigTokenId)
	if err != nil {
		return nil, err
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
	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	// Burn escrowed tokens
	if _, err := k.CallEVM(ctx, erc721, common.HexToAddress(msg.Sender), contract, true, "burn", big.NewInt(0)); err != nil { // TODO
		return nil, err
	}

	// query nftID by given tokenID
	nftID := string(k.GetNFTPairByTokenID(ctx, msg.TokenId))

	// unlock nft
	//if err := k.nftKeeper.Transfer(ctx, pair.ClassId, nftID, receiver); err != nil {
	//	return nil, err
	//}

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

	fmt.Printf("xxl 01 convertERC721NativeERC721 001 start \n")
	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	bigTokenId := new(big.Int)
	_, err := fmt.Sscan(msg.TokenId, bigTokenId)
	if err != nil {
		sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s error scanning value", err)
		return nil, err
	}
	fmt.Printf("xxl 01 convertERC721NativeERC721 002 k.CallEVM \n")

	reqInfo, err := k.QueryERC721Enhance(ctx, contract, bigTokenId)
	fmt.Printf("xxl 01 getEnhanceInfo 026 k.CallEVM res %v \n", reqInfo)

	// Burn escrowed tokens
	//res, err := k.CallEVM(ctx, erc721, sender, contract, true, "burn", bigTokenId)
	//fmt.Printf("xxl 01 convertERC721NativeERC721 003 k.CallEVM %v \n", res)
	//if err != nil {
	//	//xxl TODO
	//	fmt.Printf("xxl 01 convertERC721NativeERC721 004 k.CallEVM %v \n", err)
	//	return nil, err
	//}
	// xxl 04
	// Escrow tokens on module account
	res, err := k.CallEVM(
		ctx, erc721, sender, contract, true,
		"safeTransferFrom", sender, types.ModuleAddress, bigTokenId,
	)
	if err != nil {
		return nil, err
	}

	fmt.Printf("xxl 01 convertERC721NativeERC721 004 burn is OK \n")
	tokenID, success := big.NewInt(0).SetString(msg.TokenId, 10)
	if !success {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid tokenID")
	}

	// query erc721 token
	_, err = k.QueryERC721Token(ctx, contract, tokenID)
	if err != nil {
		return nil, err
	}

	fmt.Printf("xxl 01 convertERC721NativeERC721 003 tokenID \n")
	nft := nftTypes.MsgMintNFT{
		DenomId:   msg.ClassId,
		Id:        msg.NftId,
		Name:      reqInfo.Name,
		URI:       reqInfo.Uri,
		Data:      reqInfo.Data,
		UriHash:   reqInfo.UriHash,
		Sender:    msg.Receiver,
		Recipient: msg.Receiver,
	}

	fmt.Printf("xxl 01 convertERC721NativeERC721 005 MsgMintNFT %v \n", nft)
	// mint nft
	if _, err = k.nftKeeper.MintNFT(ctx, &nft); err != nil {
		return nil, err
	}

	// save nft pair
	k.SetNFTPairByNFTID(ctx, msg.NftId, msg.TokenId)
	k.SetNFTPairByTokenID(ctx, msg.TokenId, msg.NftId)

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
				sdk.NewAttribute(types.AttributeKeyNFTID, msg.NftId),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, msg.TokenId),
			),
		},
	)

	fmt.Printf("xxl 01 convertERC721NativeERC721 004 end \n")
	return &types.MsgConvertERC721Response{}, nil
}
