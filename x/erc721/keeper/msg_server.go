package keeper

import (
	"context"
	"fmt"
	"github.com/UptickNetwork/uptick/x/erc721/contracts"
	"math/big"
	"strings"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"

	"github.com/UptickNetwork/uptick/x/collection/exported"

	"github.com/ethereum/go-ethereum/common"

	sdkerrors "cosmossdk.io/errors"
	nftTypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/erc721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.MsgServer = &Keeper{}

// TransferERC721 converts ERC721 tokens into native Cosmos nft for both
// Cosmos-native and ERC721 TokenPair Owners and transfer through IBC
func (k Keeper) TransferERC721(
	goCtx context.Context,
	msg *types.MsgTransferERC721,
) (
	*types.MsgTransferERC721Response, error,
) {

	ctx := sdk.UnwrapSDKContext(goCtx)
	convertMsg := types.MsgConvertERC721{
		EvmContractAddress: msg.EvmContractAddress,
		EvmTokenIds:        msg.EvmTokenIds,
		CosmosReceiver:     types.AccModuleAddress.String(),
		CosmosSender:       msg.CosmosSender,
		ClassId:            msg.ClassId,
		CosmosTokenIds:     msg.CosmosTokenIds,
	}
	resMsg, err := k.ConvertERC721(ctx, &convertMsg)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to ConvertERC721")
	}

	ibcMsg := ibcnfttransfertypes.MsgTransfer{
		SourcePort:       msg.SourcePort,
		SourceChannel:    msg.SourceChannel,
		ClassId:          resMsg.ClassId,
		TokenIds:         resMsg.CosmosTokenIds,
		Sender:           types.AccModuleAddress.String(),
		Receiver:         msg.CosmosReceiver,
		TimeoutHeight:    msg.TimeoutHeight,
		TimeoutTimestamp: msg.TimeoutTimestamp,
		Memo:             msg.Memo + types.TransferERC721Memo,
	}

	_, err = k.ibcKeeper.Transfer(goCtx, &ibcMsg)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to ibc Transfer")
	}
	bech32Address, _ := sdk.AccAddressFromBech32(msg.CosmosSender)
	sender := common.BytesToAddress(bech32Address.Bytes())
	for _, evmTokenId := range msg.CosmosTokenIds {
		k.SetEvmAddressByContractTokenId(ctx, msg.EvmContractAddress, evmTokenId, sender.Hex())
	}

	return &types.MsgTransferERC721Response{}, nil

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
	//classId, nftId
	classId, nftIds, err := k.GetClassIDAndNFTID(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ClassId = classId
	msg.CosmosTokenIds = nftIds

	// Error checked during msg validation
	//sender := common.HexToAddress(msg.EvmSender)
	bech32Address, _ := sdk.AccAddressFromBech32(msg.CosmosSender)
	sender := common.BytesToAddress(bech32Address.Bytes())

	id := k.GetTokenPairID(ctx, msg.EvmContractAddress)
	if len(id) == 0 {

		_, err := k.RegisterERC721(ctx, msg)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to RegisterERC721")
		}
	}

	pair, err := k.GetPair(ctx, msg.EvmContractAddress)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to GetPair")
	}

	// Remove token pair if contract is suicided
	erc721 := common.HexToAddress(pair.Erc721Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc721)

	bigTokenId := new(big.Int)
	_, err = fmt.Sscan(msg.EvmTokenIds[0], bigTokenId)
	if err != nil {
		return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error scanning value", err)
	}

	owner, err := k.QueryERC721TokenOwner(ctx, erc721, bigTokenId)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to QueryERC721TokenOwner")
	}
	if owner != sender {
		return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s is not the owner of erc721 token %s", sender, strings.Join(msg.EvmTokenIds, ","))
	}

	if acc == nil || len(acc.CodeHash) == 0 {

		k.DeleteTokenPair(ctx, pair)
		k.Logger(ctx).Debug(
			"deleting self destructed token pair from state",
			"contract", pair.Erc721Address,
		)
		// NOTE: return nil error to persist the changes from the deletion
		return nil, nil
	}

	msgconverterc721, err := k.convertEvm2Cosmos(ctx, pair, msg, sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to convert EVM to cosmos")
	}

	convertAddress, _ := sdk.AccAddressFromBech32(msgconverterc721.CosmosSender)
	evmSender := common.BytesToAddress(convertAddress.Bytes())

	return &types.MsgConvertERC721Response{
		EvmContractAddress: msgconverterc721.EvmContractAddress,
		EvmTokenIds:        msgconverterc721.EvmTokenIds,
		CosmosReceiver:     msgconverterc721.CosmosReceiver,
		EvmSender:          evmSender.Hex(),
		ClassId:            msgconverterc721.ClassId,
		CosmosTokenIds:     msgconverterc721.CosmosTokenIds,
	}, nil

}

// ConvertNFT ConvertCoin converts native Cosmos nft into ERC721 tokens for both
// Cosmos-native and ERC721 TokenPair Owners
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
	msg.EvmContractAddress = strings.ToLower(contractAddress)
	msg.EvmTokenIds = tokenIds

	// Error checked during msg validation
	receiver := common.HexToAddress(msg.EvmReceiver)
	id := k.GetTokenPairID(ctx, msg.EvmContractAddress)
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

	if acc == nil || len(acc.CodeHash) == 0 {
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

// convertCosmos2Evm handles the nft conversion for a native ERC721 token
// pair:
//   - escrow nft on module account
//   - unescrow nft that have been previously escrowed with ConvertERC721 and send to receiver
//   - burn escrowed nft
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

	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()
	msg.EvmContractAddress = strings.ToLower(contract.String())

	for i, tokenId := range msg.EvmTokenIds {
		bigTokenId := new(big.Int)
		_, err := fmt.Sscan(tokenId, bigTokenId)
		if err != nil {
			return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error scanning value", err)
		}
		bigTokenIds = append(bigTokenIds, bigTokenId)

		reqInfo, err = k.nftKeeper.GetNFT(ctx, msg.ClassId, msg.CosmosTokenIds[i])
		if err != nil {
			return nil, err
		}

		transferNft := nftTypes.MsgTransferNFT{
			DenomId:   msg.ClassId,
			Id:        msg.CosmosTokenIds[i],
			Name:      reqInfo.GetName(),
			URI:       reqInfo.GetURI(),
			Data:      reqInfo.GetData(),
			UriHash:   reqInfo.GetURIHash(),
			Sender:    msg.CosmosSender,
			Recipient: types.AccModuleAddress.String(),
		}

		if _, err = k.nftKeeper.TransferNFT(ctx, &transferNft); err != nil {
			return nil, err
		}

		//	does token id exist
		owner, err := k.QueryERC721TokenOwner(ctx, common.HexToAddress(msg.EvmContractAddress), bigTokenIds[i])
		if err != nil {

			_, err = k.CallEVM(
				ctx, erc721, types.ModuleAddress, contract, true,
				"mintEnhance", receiver, bigTokenIds[i], reqInfo.GetName(), reqInfo.GetURI(), reqInfo.GetData(), reqInfo.GetURIHash())
			if err != nil {
				// mint normal
				_, err = k.CallEVM(
					ctx, erc721, types.ModuleAddress, contract, true,
					"mint", receiver, bigTokenIds[i], reqInfo.GetURI())
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
			return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s is not the owner of erc721 token %s", types.ModuleAddress, msg.EvmTokenIds)
		}

		// Mint tokens and send to receiver
		if err != nil {
			return nil, err
		}

	}

	for i, tokenId := range msg.EvmTokenIds {

		k.SetNFTPairs(ctx, msg.EvmContractAddress, tokenId, msg.ClassId, msg.CosmosTokenIds[i])
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertNFT,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.CosmosSender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.EvmReceiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, msg.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(msg.CosmosTokenIds, ",")),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, strings.Join(msg.EvmTokenIds, ",")),
			),
		},
	)

	return &types.MsgConvertNFTResponse{}, nil
}

// convertEvm2Cosmos handles the erc721 conversion for a native erc721 token
// pair:
//   - escrow tokens on module account
//   - mint nft to the receiver: nftId: tokenAddress|tokenID
func (k Keeper) convertEvm2Cosmos(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertERC721,
	sender common.Address,
) (
	*types.MsgConvertERC721, error,
) {

	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	for i, tokenId := range msg.EvmTokenIds {

		bigTokenId := new(big.Int)
		_, err := fmt.Sscan(tokenId, bigTokenId)
		if err != nil {
			return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error scanning", err)
		}

		reqInfo, err := k.QueryNFTEnhance(ctx, contract, bigTokenId)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to query NFT enhance")
		}

		_, err = k.CallEVM(
			ctx, erc721, sender, contract, true,
			"safeTransferFrom", sender, types.ModuleAddress, bigTokenId,
		)
		if err != nil {
			return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error safeTransferFrom ", err)
		}

		// query erc721 token
		_, err = k.QueryERC721Token(ctx, contract)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "error QueryERC721Token")
		}

		nftId := string(k.GetNFTPairByContractTokenID(ctx, msg.EvmContractAddress, tokenId))
		if nftId == "" {

			//
			mintNFT := nftTypes.MsgMintNFT{
				DenomId:   msg.ClassId,
				Id:        msg.CosmosTokenIds[i],
				Name:      reqInfo.Name,
				URI:       reqInfo.Uri,
				Data:      reqInfo.Data,
				UriHash:   reqInfo.UriHash,
				Sender:    types.AccModuleAddress.String(),
				Recipient: msg.CosmosReceiver,
			}

			// mint nft
			if _, err = k.nftKeeper.MintNFT(ctx, &mintNFT); err != nil {
				return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error MsgMintNFT ", err)
			}

		} else {
			transferNft := nftTypes.MsgTransferNFT{
				DenomId:   msg.ClassId,
				Id:        msg.CosmosTokenIds[i],
				Name:      reqInfo.Name,
				URI:       reqInfo.Uri,
				Data:      reqInfo.Data,
				UriHash:   reqInfo.UriHash,
				Sender:    types.AccModuleAddress.String(),
				Recipient: msg.CosmosReceiver,
			}
			if _, err = k.nftKeeper.TransferNFT(ctx, &transferNft); err != nil {
				return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error MsgTransferNFT ", err)
			}
		}
	}

	// save nft pair
	for i, tokenId := range msg.EvmTokenIds {
		k.SetNFTPairs(ctx, msg.EvmContractAddress, tokenId, msg.ClassId, msg.CosmosTokenIds[i])
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC721,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.CosmosSender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.CosmosReceiver),
				sdk.NewAttribute(types.AttributeKeyNFTClass, pair.ClassId),
				sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(msg.CosmosTokenIds, ",")),
				sdk.NewAttribute(types.AttributeKeyERC721Token, contract.String()),
				sdk.NewAttribute(types.AttributeKeyERC721TokenID, strings.Join(msg.EvmTokenIds, ",")),
			),
		},
	)

	return msg, nil
}

// RefundPacketToken handles the erc721 conversion for a native erc721 token
// pair:
//   - escrow tokens on module account
//   - mint nft to the receiver: nftId: tokenAddress|tokenID
func (k Keeper) RefundPacketToken(
	ctx sdk.Context,
	data ibcnfttransfertypes.NonFungibleTokenPacketData,
) error {

	erc721 := contracts.ERC721UpticksContract.ABI
	var refundedTokenIds []string
	var refundedContract string
	var refundedReceiver common.Address

	for _, tokenId := range data.TokenIds {

		uNftID := types.CreateNFTUID(data.ClassId, tokenId)
		emvTokenId, evmContractAddress := types.GetNFTFromUID(string(k.GetTokenUIDPairByNFTUID(ctx, uNftID)))

		bigTokenId := new(big.Int)
		_, err := fmt.Sscan(emvTokenId, bigTokenId)
		if err != nil {
			return sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error scanning value", err)
		}

		contract := common.HexToAddress(evmContractAddress)

		// Check if token has already been refunded
		owner, err := k.QueryERC721TokenOwner(ctx, contract, bigTokenId)
		if err != nil {
			return err
		}
		if owner != types.ModuleAddress {
			continue
		}

		evmReceiver := k.GetEvmAddressByContractTokenId(ctx, evmContractAddress, tokenId)
		receiver := common.HexToAddress(string(evmReceiver))

		_, err = k.CallEVM(
			ctx, erc721, types.ModuleAddress, contract, true,
			"safeTransferFrom", types.ModuleAddress, receiver, bigTokenId)
		if err != nil {
			return err
		}

		k.DeleteEvmAddressByContractTokenId(ctx, evmContractAddress, tokenId)
		k.DeleteNFTPairByNFTID(ctx, data.ClassId, tokenId)
		k.DeleteNFTPairByTokenID(ctx, evmContractAddress, emvTokenId)

		refundedTokenIds = append(refundedTokenIds, tokenId)
		refundedContract = evmContractAddress
		refundedReceiver = receiver
	}

	if len(refundedTokenIds) > 0 {
		ctx.EventManager().EmitEvents(
			sdk.Events{
				sdk.NewEvent(
					types.EventTypeRefundPacketToken,
					sdk.NewAttribute(sdk.AttributeKeySender, data.Sender),
					sdk.NewAttribute(types.AttributeKeyReceiver, refundedReceiver.Hex()),
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(refundedTokenIds, ",")),
					sdk.NewAttribute(types.AttributeKeyERC721Token, refundedContract),
					sdk.NewAttribute(types.AttributeKeyERC721TokenID, strings.Join(refundedTokenIds, ",")),
				),
			},
		)
	}

	return nil
}
