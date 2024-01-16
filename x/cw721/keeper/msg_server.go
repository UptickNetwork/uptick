package keeper

import (
	"context"
	"fmt"
	nftTypes "github.com/UptickNetwork/uptick/x/collection/types"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strings"

	"github.com/UptickNetwork/uptick/x/cw721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.MsgServer = &Keeper{}

// TransferCW721 converts CW721 tokens into native Cosmos nft for both
// Cosmos-native and CW721 TokenPair Owners and transfer through IBC
func (k Keeper) TransferCW721(
	goCtx context.Context,
	msg *types.MsgTransferCW721,
) (
	*types.MsgTransferCW721Response, error,
) {

	ctx := sdk.UnwrapSDKContext(goCtx)
	convertMsg := types.MsgConvertCW721{
		ContractAddress: msg.CwContractAddress,
		TokenIds:        msg.CwTokenIds,
		Receiver:        types.AccModuleAddress.String(),
		Sender:          msg.CwSender,
		ClassId:         msg.ClassId,
		NftIds:          msg.CosmosTokenIds,
	}

	fmt.Printf("xxl convertMsg %v - %v -%v\n", convertMsg.ClassId, convertMsg.NftIds, msg)
	resMsg, err := k.ConvertCW721(ctx, &convertMsg)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to ConvertCW721 %v", err)
	}

	ibcMsg := ibcnfttransfertypes.MsgTransfer{
		SourcePort:       msg.SourcePort,
		SourceChannel:    msg.SourceChannel,
		ClassId:          resMsg.ClassId,
		TokenIds:         resMsg.TokenIds,
		Sender:           types.AccModuleAddress.String(),
		Receiver:         msg.CosmosReceiver,
		TimeoutHeight:    msg.TimeoutHeight,
		TimeoutTimestamp: msg.TimeoutTimestamp,
		Memo:             msg.Memo + types.TransferCW721Memo,
	}

	_, err = k.ibcKeeper.Transfer(goCtx, &ibcMsg)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to ibc Transfer %v", err)
	}

	fmt.Printf("xxl 0000 TransferCW721 msg:%v- CosmosTokenIds:%v \n", msg, msg.CosmosTokenIds)
	for _, cwTokenId := range msg.CwTokenIds {
		fmt.Printf("xxl 0001 TransferCW721 CwSender: %v-cwTokenId :%v \n", msg.CwSender, cwTokenId)
		k.SetCwAddressByContractTokenId(ctx, msg.CwContractAddress, cwTokenId, msg.CwSender)
	}

	return &types.MsgTransferCW721Response{}, nil

}

// ConvertCW721 converts CW721 tokens into native Cosmos nft for both
// Cosmos-native and CW721 TokenPair Owners
func (k Keeper) ConvertCW721(
	goCtx context.Context,
	msg *types.MsgConvertCW721,
) (
	*types.MsgConvertCW721, error,
) {

	ctx := sdk.UnwrapSDKContext(goCtx)

	//classId, nftId
	fmt.Printf("xxl msg %v\n", msg)
	classId, nftIds, err := k.GetClassIDAndNFTID(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ClassId = classId
	msg.NftIds = nftIds

	// Error checked during msg validation
	// sender := common.HexToAddress(msg.Sender)

	id := k.GetCW721Map(ctx, msg.ContractAddress)
	if len(id) == 0 {
		_, err := k.RegisterCW721(ctx, msg)
		if err != nil {
			return nil, err
		}
	}

	return k.convertWasm2Cosmos(ctx, msg) //

}

// convertWasm2Cosmos handles the cw721 conversion for a native cw721 token
// pair:
//   - escrow tokens on module account
//   - mint nft to the receiver: nftId: tokenAddress|tokenID
func (k Keeper) convertWasm2Cosmos(
	ctx sdk.Context,
	msg *types.MsgConvertCW721,
) (
	*types.MsgConvertCW721, error,
) {

	for i, tokenId := range msg.TokenIds {

		allNftInfo, err := k.QueryCW721AllNftInfo(ctx, msg.ContractAddress, tokenId)
		if err != nil {
			return nil, err
		}

		if allNftInfo.Access.Owner != msg.Sender {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not the owner of cw721 token %s", msg.Sender, strings.Join(msg.TokenIds, ","))
		}

		// transfer to module address
		_, err = k.TransferCw721(ctx, msg.ContractAddress, tokenId, types.AccModuleAddress.String(), msg.Sender)
		if err != nil {
			return nil, err
		}

		// query cw721 token
		nftId := string(k.GetNFTPairByContractTokenID(ctx, msg.ContractAddress, tokenId))
		if nftId == "" {

			mintNFT := nftTypes.MsgMintNFT{
				DenomId:   msg.ClassId,
				Id:        msg.NftIds[i],
				Name:      "",
				URI:       allNftInfo.Info.TokenUri,
				Data:      "",
				UriHash:   "",
				Sender:    types.AccModuleAddress.String(),
				Recipient: msg.Receiver,
			}

			// mint nft
			if _, err = k.nftKeeper.MintNFT(ctx, &mintNFT); err != nil {
				return nil, err
			}
		} else {
			nftInfo, err := k.nftKeeper.GetNFT(ctx, msg.ClassId, msg.NftIds[i])
			if err != nil {
				return nil, sdkerrors.Wrapf(sdkerrors.ErrConflict, "fail to get nftInfo ", "classId", msg.ClassId, "nftId", msg.NftIds[i])
			}
			transferNft := nftTypes.MsgTransferNFT{
				DenomId:   msg.ClassId,
				Id:        msg.NftIds[i],
				Name:      nftInfo.GetName(),
				URI:       allNftInfo.Info.TokenUri,
				Data:      nftInfo.GetData(),
				UriHash:   nftInfo.GetURIHash(),
				Sender:    types.AccModuleAddress.String(),
				Recipient: msg.Receiver,
			}
			if _, err = k.nftKeeper.TransferNFT(ctx, &transferNft); err != nil {
				return nil, err
			}
		}
	}

	// save nft pair
	for i, tokenId := range msg.TokenIds {
		k.SetNFTPairs(ctx, msg.ContractAddress, tokenId, msg.ClassId, msg.NftIds[i])
	}

	return msg, nil
}

// ConvertNFT ConvertCoin converts native Cosmos nft into CW721 tokens for both
// Cosmos-native and CW721 TokenPair Owners
func (k Keeper) ConvertNFT(
	goCtx context.Context,
	msg *types.MsgConvertNFT,
) (
	*types.MsgConvertNFTResponse, error,
) {

	ctx := sdk.UnwrapSDKContext(goCtx)
	//classId, nftIDs
	contractAddress, tokenIds, err := k.GetContractAddressAndTokenIds(ctx, msg)
	if err != nil {
		return nil, err
	}

	msg.ContractAddress = contractAddress
	msg.TokenIds = tokenIds

	id := k.GetClassMap(ctx, msg.ClassId)
	k.Logger(ctx).Info("ConvertNFT ", "id", id, "msg", msg)
	if len(id) == 0 {
		_, err := k.RegisterNFT(ctx, msg)
		if err != nil {
			return nil, err
		}
	}

	_, err = k.GetPair(ctx, msg.ClassId)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertNFT,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, msg.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(msg.NftIds, ",")),
				sdk.NewAttribute(types.AttributeKeyCW721Token, msg.ContractAddress),
				sdk.NewAttribute(types.AttributeKeyCW721TokenID, strings.Join(msg.TokenIds, ",")),
			),
		},
	)

	return k.convertCosmos2Wasm(ctx, msg) //
}

// convertCosmos2Wasm handles the nft conversion for a native CW721 token
// pair:
//   - escrow nft on module account
//   - unescrow nft that have been previously escrowed with ConvertCW721 and send to receiver
//   - burn escrowed nft
func (k Keeper) convertCosmos2Wasm(
	ctx sdk.Context,
	msg *types.MsgConvertNFT,
) (
	*types.MsgConvertNFTResponse, error,
) {

	for i, tokenId := range msg.TokenIds {

		reqInfo, err := k.nftKeeper.GetNFT(ctx, msg.ClassId, msg.NftIds[i])
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
		// owner, err := k.QueryCW721TokenOwner(ctx, common.HexToAddress(msg.ContractAddress), bigTokenIds[i])
		nftInfo, err := k.QueryCW721AllNftInfo(ctx, msg.ContractAddress, tokenId)

		if err != nil {

			// mint
			_, err := k.MintCw721(ctx, msg.ContractAddress, tokenId, msg.Receiver, reqInfo.GetURI())
			if err != nil {
				return nil, err
			}

		} else if nftInfo.Access.Owner == types.AccModuleAddress.String() {
			// transfer
			_, err := k.TransferCw721(ctx, msg.ContractAddress, tokenId, msg.Receiver, types.AccModuleAddress.String())
			if err != nil {
				return nil, err
			}
		} else {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not the owner of cw721 token %s", types.ModuleAddress, msg.TokenIds)
		}

	}

	for i, tokenId := range msg.TokenIds {
		k.SetNFTPairs(ctx, msg.ContractAddress, tokenId, msg.ClassId, msg.NftIds[i])
	}
	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertCW721,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, msg.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(msg.NftIds, ",")),
				sdk.NewAttribute(types.AttributeKeyCW721Token, msg.ContractAddress),
				sdk.NewAttribute(types.AttributeKeyCW721TokenID, strings.Join(msg.TokenIds, ",")),
			),
		},
	)
	return &types.MsgConvertNFTResponse{}, nil
}

// RefundPacketToken handles the erc721 conversion for a native erc721 token
// pair:
//   - escrow tokens on module account
//   - mint nft to the receiver: nftId: tokenAddress|tokenID
func (k Keeper) RefundPacketToken(
	ctx sdk.Context,
	data ibcnfttransfertypes.NonFungibleTokenPacketData,
) error {

	fmt.Printf("xxl RefundPacketToken data %v\n", data)
	for _, tokenId := range data.TokenIds {

		uNftID := types.CreateNFTUID(data.ClassId, tokenId)
		fmt.Printf("xxl RefundPacketToken uNftID %v\n", uNftID)

		cwTokenId, cwContractAddress := types.GetNFTFromUID(string(k.GetTokenUIDPairByNFTUID(ctx, uNftID)))
		cwReceiver := k.GetCwAddressByContractTokenId(ctx, cwContractAddress, tokenId)

		fmt.Printf("xxl RefundPacketToken %s - %s - %s - %s\n", cwContractAddress, cwTokenId, string(cwReceiver), data.Sender)
		_, err := k.TransferCw721(ctx, cwContractAddress, cwTokenId, string(cwReceiver), data.Sender)
		if err != nil {
			fmt.Printf("xxl RefundPacketToken %v \n", err)
			return err
		}

	}

	return nil
}
