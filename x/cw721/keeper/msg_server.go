package keeper

import (
	"context"
	"strings"

	sdkerrors "cosmossdk.io/errors"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	nftTypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/cw721/types"
)

var _ types.MsgServer = &Keeper{}

const maxCW721BatchSize = 100

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

	resMsg, err := k.ConvertCW721(ctx, &convertMsg)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to ConvertCW721 %v", err)
	}

	ibcMsg := ibcnfttransfertypes.MsgTransfer{
		SourcePort:       msg.SourcePort,
		SourceChannel:    msg.SourceChannel,
		ClassId:          resMsg.ClassId,
		TokenIds:         resMsg.NftIds,
		Sender:           types.AccModuleAddress.String(),
		Receiver:         msg.CosmosReceiver,
		TimeoutHeight:    msg.TimeoutHeight,
		TimeoutTimestamp: msg.TimeoutTimestamp,
		Memo:             msg.Memo + types.TransferCW721Memo,
	}

	_, err = k.ibcTransferKeeper.Transfer(goCtx, &ibcMsg)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to ibc Transfer %v", err)
	}

	for _, cwTokenId := range msg.CwTokenIds {
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
	*types.MsgConvertCW721Response, error,
) {

	ctx := sdk.UnwrapSDKContext(goCtx)
	if !k.GetEnableCw721(ctx) {
		return nil, types.ErrCW721Disabled
	}

	// classId, nftId
	classId, nftIds, err := k.GetClassIDAndNFTID(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ClassId = classId
	msg.NftIds = nftIds
	if len(msg.TokenIds) == 0 || len(msg.TokenIds) != len(msg.NftIds) {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "CW721 token ids and NFT ids length mismatch")
	}
	if len(msg.TokenIds) > maxCW721BatchSize {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "CW721 batch size %d exceeds maximum %d", len(msg.TokenIds), maxCW721BatchSize)
	}

	// Error checked during msg validation
	// sender := common.HexToAddress(msg.Sender)

	id := k.GetCW721Map(ctx, msg.ContractAddress)
	if len(id) == 0 {
		_, err := k.RegisterCW721(ctx, msg)
		if err != nil {
			return nil, err
		}
	}

	pair, err := k.GetPairByCW721(ctx, msg.ContractAddress)
	if err != nil {
		return nil, err
	}
	// Pin the resolved class ID to the pair's canonical class so the caller
	// cannot mint a CW721-derived NFT into an arbitrary third-party denom.
	if msg.ClassId != "" && msg.ClassId != pair.ClassId {
		return nil, sdkerrors.Wrapf(
			types.ErrClassIdNotCorrect,
			"class id is not correct, expect %s got %s",
			pair.ClassId, msg.ClassId,
		)
	}
	msg.ClassId = pair.ClassId

	msgconvertcw721, err := k.convertWasm2Cosmos(ctx, msg) //
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to ConvertCW721 %v", err)
	}
	return &types.MsgConvertCW721Response{
		ContractAddress: msgconvertcw721.ContractAddress,
		TokenIds:        msgconvertcw721.TokenIds,
		Receiver:        msgconvertcw721.Receiver,
		Sender:          msgconvertcw721.Sender,
		ClassId:         msgconvertcw721.ClassId,
		NftIds:          msgconvertcw721.NftIds,
	}, nil
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

		if strings.TrimSpace(allNftInfo.Access.Owner) == "" || allNftInfo.Access.Owner != msg.Sender {
			return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s is not the owner of cw721 token %s", msg.Sender, strings.Join(msg.TokenIds, ","))
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
				return nil, sdkerrors.Wrapf(errortypes.ErrConflict, "fail to get nftInfo classId=%s nftId=%s", msg.ClassId, msg.NftIds[i])
			}
			transferNft := nftTypes.MsgTransferNFT{
				DenomId:   msg.ClassId,
				Id:        msg.NftIds[i],
				Name:      nftInfo.GetName(),
				URI:       nftInfo.GetURI(),
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
		if err := k.SetNFTPairs(ctx, msg.ContractAddress, tokenId, msg.ClassId, msg.NftIds[i]); err != nil {
			return nil, err
		}
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
	if !k.GetEnableCw721(ctx) {
		return nil, types.ErrCW721Disabled
	}

	// classId, nftIDs
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

	pair, err := k.GetPairByClass(ctx, msg.ClassId)
	if err != nil {
		return nil, err
	}
	if msg.ClassId != pair.ClassId {
		return nil, sdkerrors.Wrapf(
			types.ErrClassIdNotCorrect,
			"class id is not correct, expect %s got %s",
			pair.ClassId, msg.ClassId,
		)
	}
	msg.ContractAddress = pair.Cw721Address

	return k.convertCosmos2Wasm(ctx, msg)
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
	if len(msg.TokenIds) == 0 || len(msg.TokenIds) != len(msg.NftIds) {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "CW721 token ids and NFT ids length mismatch")
	}
	if len(msg.TokenIds) > maxCW721BatchSize {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "CW721 batch size %d exceeds maximum %d", len(msg.TokenIds), maxCW721BatchSize)
	}

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

		// Use the module's own NFT pair state to decide mint vs transfer,
		// matching x/erc721. Querying the contract would treat a missing
		// token as mint even when the pair already tracks an escrowed NFT.
		nftPair := k.GetNFTPairByContractTokenID(ctx, msg.ContractAddress, tokenId)
		if len(nftPair) == 0 {
			_, err := k.MintCw721(ctx, msg.ContractAddress, tokenId, msg.Receiver, reqInfo.GetURI())
			if err != nil {
				return nil, err
			}
		} else {
			expectedNFTUID := types.CreateNFTUID(msg.ClassId, msg.NftIds[i])
			if string(nftPair) != expectedNFTUID {
				return nil, sdkerrors.Wrapf(
					types.ErrNFTMappingConflict,
					"cw721 token %s is already bound to nft %s, not %s",
					tokenId, string(nftPair), expectedNFTUID,
				)
			}
			_, err := k.TransferCw721(ctx, msg.ContractAddress, tokenId, msg.Receiver, types.AccModuleAddress.String())
			if err != nil {
				return nil, err
			}
		}

	}

	for i, tokenId := range msg.TokenIds {
		if err := k.SetNFTPairs(ctx, msg.ContractAddress, tokenId, msg.ClassId, msg.NftIds[i]); err != nil {
			return nil, err
		}
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

// refundGroup aggregates the successfully refunded tokens of one
// (contract, receiver) pair, so the refund event stays accurate when a single
// packet carries tokens from several contracts or for several receivers.
type refundGroup struct {
	contract string
	receiver string
	// cw721 token ids refunded to this (contract, receiver) pair
	tokenIds []string
	// native NFT ids burned for those tokens
	nftIds []string
}

// appendToRefundGroups appends to the group matching (contract, receiver),
// creating it when it does not exist yet. Groups keep first-seen order so event
// emission stays deterministic across nodes.
func appendToRefundGroups(groups []refundGroup, contract, receiver, tokenID, nftID string) []refundGroup {
	for i := range groups {
		if groups[i].contract == contract && groups[i].receiver == receiver {
			groups[i].tokenIds = append(groups[i].tokenIds, tokenID)
			groups[i].nftIds = append(groups[i].nftIds, nftID)
			return groups
		}
	}
	return append(groups, refundGroup{
		contract: contract,
		receiver: receiver,
		tokenIds: []string{tokenID},
		nftIds:   []string{nftID},
	})
}

// RefundPacketToken handles the IBC packet timeout/failure for CW721 transfers.
// It reverses the conversion: returns the CW721 to its original owner, cleans up
// token pair mappings, and burns the native NFT returned to the module account.
// This function should be called by the host chain's IBC middleware OnTimeoutPacket handler.
//
// The refund is gated on the module account still owning the CW721 token. A token
// that was already refunded, never escrowed, or moved elsewhere is skipped with an
// event instead of an error: returning an error here aborts the IBC
// OnTimeout/OnAcknowledgement callback, and because the relayer's
// MsgTimeout/MsgAcknowledgement can then never succeed, the packet would be stuck
// forever. This mirrors x/erc721 and keeps the packet finalisable.
func (k Keeper) RefundPacketToken(
	ctx sdk.Context,
	data ibcnfttransfertypes.NonFungibleTokenPacketData,
) error {

	var groups []refundGroup

	for _, tokenId := range data.TokenIds {

		uNftID := types.CreateNFTUID(data.ClassId, tokenId)
		pairUID := k.GetTokenUIDPairByNFTUID(ctx, uNftID)
		if len(pairUID) == 0 {
			return sdkerrors.Wrapf(types.ErrTokenPairNotFound, "missing CW721 pair for class %s token %s", data.ClassId, tokenId)
		}

		cwTokenId, cwContractAddress := types.GetNFTFromUID(string(pairUID))
		if cwTokenId == "" || cwContractAddress == "" {
			return sdkerrors.Wrapf(types.ErrInternalTokenPair, "invalid CW721 uid for class %s token %s", data.ClassId, tokenId)
		}

		owner, err := k.QueryCW721TokenOwner(ctx, cwContractAddress, cwTokenId)
		if err != nil {
			return err
		}
		shouldRefund := moduleOwnsCW721(owner)

		if !shouldRefund {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute(types.AttributeKeyCW721Token, cwContractAddress),
					sdk.NewAttribute(types.AttributeKeyCW721TokenID, cwTokenId),
					sdk.NewAttribute("reason", "owner_is_not_module_account"),
				),
			)
		} else {
			cwReceiver := k.GetCwAddressByContractTokenId(ctx, cwContractAddress, cwTokenId)
			if len(cwReceiver) == 0 {
				return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "missing CW721 refund receiver for contract %s token %s", cwContractAddress, cwTokenId)
			}

			_, err := k.TransferCw721(ctx, cwContractAddress, cwTokenId, string(cwReceiver), types.AccModuleAddress.String())
			if err != nil {
				return err
			}

			groups = appendToRefundGroups(groups, cwContractAddress, string(cwReceiver), cwTokenId, tokenId)
		}

		k.DeleteCwAddressByContractTokenId(ctx, cwContractAddress, cwTokenId)
		k.DeleteNFTPairByTokenID(ctx, cwContractAddress, cwTokenId)
		k.DeleteNFTPairByNFTID(ctx, data.ClassId, tokenId)

		burnMsg := nftTypes.MsgBurnNFT{
			Id:      tokenId,
			DenomId: data.ClassId,
			Sender:  types.AccModuleAddress.String(),
		}
		if _, err := k.nftKeeper.BurnNFT(ctx, &burnMsg); err != nil {
			return err
		}

	}

	if len(groups) > 0 {
		ctx.EventManager().EmitEvents(cw721RefundEvents(data.Sender, data.ClassId, groups))
	}

	return nil
}

// cw721RefundEvents builds one refund event per (contract, receiver) pair from
// the aggregated groups.
func cw721RefundEvents(sender, classID string, groups []refundGroup) sdk.Events {
	events := make(sdk.Events, 0, len(groups))
	for _, group := range groups {
		events = append(events, sdk.NewEvent(
			types.EventTypeRefundPacketToken,
			sdk.NewAttribute(sdk.AttributeKeySender, sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, group.receiver),
			sdk.NewAttribute(types.AttributeKeyNFTClass, classID),
			sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(group.nftIds, ",")),
			sdk.NewAttribute(types.AttributeKeyCW721Token, group.contract),
			sdk.NewAttribute(types.AttributeKeyCW721TokenID, strings.Join(group.tokenIds, ",")),
		))
	}
	return events
}

// moduleOwnsCW721 reports whether the cw721 module account still escrows the
// token. It is the gate for the IBC refund: a token that is no longer escrowed
// must be skipped rather than refunded, otherwise the refund aborts the IBC
// callback and strands the packet (see RefundPacketToken).
func moduleOwnsCW721(owner string) bool {
	return owner == types.AccModuleAddress.String()
}
