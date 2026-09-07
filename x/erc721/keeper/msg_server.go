package keeper

import (
	"context"
	"math/big"
	"strings"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"

	"github.com/UptickNetwork/uptick/x/collection/exported"
	"github.com/UptickNetwork/uptick/x/erc721/contracts"

	"github.com/ethereum/go-ethereum/common"

	sdkerrors "cosmossdk.io/errors"
	nftTypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/erc721/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.MsgServer = &Keeper{}

const maxERC721BatchSize = 100

func parseERC721TokenID(tokenID string) (*big.Int, error) {
	n, ok := new(big.Int).SetString(tokenID, 10)
	if !ok || n.Sign() < 0 || n.BitLen() > 256 {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "invalid ERC721 token id %q", tokenID)
	}
	return n, nil
}

// pairContractRedeployable reports whether a class whose pair contract lost its
// code can be healed by deploying a fresh module-owned contract. Plain native
// classes (whose contract was deployed by the module and whose id does not
// encode a contract address) are re-deployable. Classes derived from a
// contract address ("uptick-<addr>") are pinned to that external contract —
// the class id IS the contract identity — and cannot be re-deployed.
func (k Keeper) pairContractRedeployable(classID string) bool {
	return !strings.HasPrefix(classID, types.DefaultPrefix+"-")
}

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
	bech32Address, err := sdk.AccAddressFromBech32(msg.CosmosSender)
	if err != nil {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid cosmos sender: %s", err)
	}
	sender := common.BytesToAddress(bech32Address.Bytes())
	// Record against ConvertERC721 results, not the original msg. CosmosTokenIds
	// on the request is often empty; refund lookup uses the packet cosmos ids
	// and the mapped EVM token id, plus a lowercased contract address.
	k.SetEvmRefundReceiver(ctx, resMsg.EvmContractAddress, resMsg.CosmosTokenIds, resMsg.EvmTokenIds, sender.Hex())

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
	if !k.GetEnableErc721(ctx) {
		return nil, types.ErrERC721Disabled
	}

	// Normalize the EVM contract address to lowercase so token-pair and
	// NFT-pair records are always keyed consistently. GetNFTPairByContractTokenID
	// builds a case-sensitive key (tokenID + "," + address); convertCosmos2Evm
	// lowercases before lookup, so the reverse direction must store lowercase too.
	msg.EvmContractAddress = strings.ToLower(msg.EvmContractAddress)

	// classId, nftId
	classId, nftIds, err := k.GetClassIDAndNFTID(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ClassId = classId
	msg.CosmosTokenIds = nftIds
	if len(msg.EvmTokenIds) == 0 || len(msg.EvmTokenIds) != len(msg.CosmosTokenIds) {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "evm token ids and cosmos token ids length mismatch")
	}
	if len(msg.EvmTokenIds) > maxERC721BatchSize {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "ERC721 batch size %d exceeds maximum %d", len(msg.EvmTokenIds), maxERC721BatchSize)
	}

	bech32Address, err := sdk.AccAddressFromBech32(msg.CosmosSender)
	if err != nil {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid cosmos sender: %s", err)
	}
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

	erc721 := common.HexToAddress(pair.Erc721Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc721)
	if acc == nil || len(acc.CodeHash) == 0 {
		// ERC721 -> Cosmos conversion requires minting/metadata calls against the
		// pair contract, so a contract without code is terminal for this
		// direction: the module cannot re-create an externally-owned contract and
		// the caller's ERC721 tokens are gone with it. Note that this handler
		// must NOT attempt to purge the pair here -- the SDK rolls back every
		// write made on a handler path that returns an error, so the purge would
		// silently never commit and only mislead readers. Recovery for the class
		// happens through ConvertNFT, which purges the stale pair and re-deploys
		// a fresh module contract in a succeeding transaction (see below).
		k.Logger(ctx).Warn(
			"erc721 pair contract self-destructed; conversion is terminal, state left untouched",
			"class", pair.ClassId,
			"contract", pair.Erc721Address,
		)
		return nil, sdkerrors.Wrapf(types.ErrInternalTokenPair, "erc721 contract %s is self-destructed", pair.Erc721Address)
	}

	// Pin the resolved class ID to the pair's canonical class. If the caller
	// supplied a ClassId that differs from the registered pair (e.g. minting an
	// NFT into an arbitrary third-party denom), reject it. This binds the
	// conversion to the registered token pair and blocks minting into
	// non-canonical / attacker-controlled denoms.
	if msg.ClassId != "" && msg.ClassId != pair.ClassId {
		return nil, sdkerrors.Wrapf(
			types.ErrClassIdNotCorrect,
			"class id is not correct, expect %s got %s",
			pair.ClassId, msg.ClassId,
		)
	}
	msg.ClassId = pair.ClassId

	convertedERC721, err := k.convertEvm2Cosmos(ctx, pair, msg, sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to convert EVM to cosmos")
	}

	convertAddress, err := sdk.AccAddressFromBech32(convertedERC721.CosmosSender)
	if err != nil {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid cosmos sender: %s", err)
	}
	evmSender := common.BytesToAddress(convertAddress.Bytes())

	return &types.MsgConvertERC721Response{
		EvmContractAddress: convertedERC721.EvmContractAddress,
		EvmTokenIds:        convertedERC721.EvmTokenIds,
		CosmosReceiver:     convertedERC721.CosmosReceiver,
		EvmSender:          evmSender.Hex(),
		ClassId:            convertedERC721.ClassId,
		CosmosTokenIds:     convertedERC721.CosmosTokenIds,
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
	if !k.GetEnableErc721(ctx) {
		return nil, types.ErrERC721Disabled
	}

	// classId, nftIDs
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

	// Self-heal a pair whose ERC721 contract no longer has code (self-destructed
	// or otherwise lost). SDK state is per-transaction: every write performed on
	// a handler path that later returns an error is rolled back (baseapp only
	// commits the tx cache when the whole tx succeeds), so a stale pair can
	// never be purged from inside a failing call.
	//
	// For classes the module can re-materialize (plain native classes whose
	// contract was deployed by the module), purge the stale pair and deploy a
	// fresh contract in the SAME successful transaction: the purge then commits
	// together with the new pair and the conversion, which is the only way the
	// cleanup can persist. Classes pinned to an external contract (class id
	// derived from the contract address, "uptick-<addr>") cannot be re-deployed
	// — the class↔contract identity is fixed — so those remain a terminal error.
	erc721 := common.HexToAddress(pair.Erc721Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc721)

	if acc == nil || len(acc.CodeHash) == 0 {
		if !k.pairContractRedeployable(msg.ClassId) {
			// External-contract class: the module cannot re-create the contract
			// and the caller's ERC721 tokens are gone with it. Report the
			// terminal condition without touching state (a write here would be
			// rolled back by the SDK when the handler returns this error).
			k.Logger(ctx).Warn(
				"erc721 pair contract self-destructed; class pinned to an external contract and cannot be re-deployed",
				"class", pair.ClassId,
				"contract", pair.Erc721Address,
			)
			return nil, sdkerrors.Wrapf(types.ErrInternalTokenPair, "erc721 contract %s is self-destructed", pair.Erc721Address)
		}

		// Purge the stale pair and every per-token binding / refund record tied
		// to it, then deploy a fresh module-owned contract and continue the
		// conversion. This branch only succeeds when the whole tx commits, so
		// the purge is not rolled back.
		k.PurgeTokenPair(ctx, pair)
		k.Logger(ctx).Info(
			"purged self-destructed erc721 token pair; re-deploying a fresh contract",
			"class", pair.ClassId,
			"old_contract", pair.Erc721Address,
		)

		contractAddress, tokenIds, err = k.GetContractAddressAndTokenIds(ctx, msg)
		if err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to re-deploy erc721 contract for class %s after purging self-destructed pair", msg.ClassId)
		}
		msg.EvmContractAddress = strings.ToLower(contractAddress)
		msg.EvmTokenIds = tokenIds

		if _, err = k.RegisterNFT(ctx, msg); err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to re-register erc721 token pair for class %s", msg.ClassId)
		}

		pair, err = k.GetPair(ctx, msg.ClassId)
		if err != nil {
			return nil, err
		}
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
	if len(msg.EvmTokenIds) == 0 || len(msg.EvmTokenIds) != len(msg.CosmosTokenIds) {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "evm token ids and cosmos token ids length mismatch")
	}
	if len(msg.EvmTokenIds) > maxERC721BatchSize {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "ERC721 batch size %d exceeds maximum %d", len(msg.EvmTokenIds), maxERC721BatchSize)
	}

	var (
		bigTokenIds []*big.Int
		reqInfo     exported.NFT
	)

	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()
	msg.EvmContractAddress = strings.ToLower(contract.String())

	for i, tokenId := range msg.EvmTokenIds {
		bigTokenId, err := parseERC721TokenID(tokenId)
		if err != nil {
			return nil, err
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

		//	does token id exist. Use the module's own tracked NFT pair state to
		//	decide between mint and transfer instead of trusting the (potentially
		//	attacker-controlled) ERC721 contract's ownerOf, which could be
		//	spoofed to forge ownership.
		nftPair := k.GetNFTPairByContractTokenID(ctx, msg.EvmContractAddress, tokenId)
		if len(nftPair) == 0 {
			// token not previously converted -> mint a new ERC721 token
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
		} else {
			// Enforce a strict one-to-one NFT mapping before releasing the
			// module-escrowed ERC721. The user-supplied (classID, nftID) MUST
			// equal the persisted binding for this (contract, tokenID). If it
			// does not, an attacker could pair their own Cosmos NFT with a
			// victim's already-escrowed ERC721 token id and drain it.
			expectedNFTUID := types.CreateNFTUID(msg.ClassId, msg.CosmosTokenIds[i])
			if string(nftPair) != expectedNFTUID {
				return nil, sdkerrors.Wrapf(
					types.ErrNFTMappingConflict,
					"erc721 token %s is already bound to nft %s, not %s",
					tokenId, string(nftPair), expectedNFTUID,
				)
			}
			// token previously converted and escrowed by the module -> transfer
			owner, err := k.QueryERC721TokenOwner(ctx, common.HexToAddress(msg.EvmContractAddress), bigTokenIds[i])
			if err != nil {
				return nil, sdkerrors.Wrap(err, "failed to query erc721 token owner")
			}
			if owner != types.ModuleAddress {
				return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s is not the owner of erc721 token %s", types.ModuleAddress, tokenId)
			}
			_, err = k.CallEVM(
				ctx, erc721, types.ModuleAddress, contract, true,
				"safeTransferFrom", types.ModuleAddress, receiver, bigTokenIds[i])
			if err != nil {
				return nil, err
			}
		}

	}

	for i, tokenId := range msg.EvmTokenIds {
		if err := k.SetNFTPairs(ctx, msg.EvmContractAddress, tokenId, msg.ClassId, msg.CosmosTokenIds[i]); err != nil {
			return nil, err
		}
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
	if len(msg.EvmTokenIds) == 0 || len(msg.EvmTokenIds) != len(msg.CosmosTokenIds) {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "evm token ids and cosmos token ids length mismatch")
	}
	if len(msg.EvmTokenIds) > maxERC721BatchSize {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "ERC721 batch size %d exceeds maximum %d", len(msg.EvmTokenIds), maxERC721BatchSize)
	}

	erc721 := contracts.ERC721UpticksContract.ABI
	contract := pair.GetERC721Contract()

	for i, tokenId := range msg.EvmTokenIds {

		bigTokenId, err := parseERC721TokenID(tokenId)
		if err != nil {
			return nil, err
		}

		reqInfo, err := k.QueryNFTEnhance(ctx, contract, bigTokenId)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to query NFT enhance")
		}

		owner, err := k.QueryERC721TokenOwner(ctx, contract, bigTokenId)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to query ERC721 token owner")
		}
		// UNIQUE AUTHORITATIVE OWNERSHIP CHECK. This loop-level check is the
		// only place ConvertERC721 verifies the caller owns every token in the
		// batch. Do not remove or replace it with a single-element sample -- a
		// batch may contain tokens belonging to different owners.
		if owner != sender {
			return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s is not the owner of erc721 token %s", sender, tokenId)
		}

		_, err = k.CallEVM(
			ctx, erc721, sender, contract, true,
			"safeTransferFrom", sender, types.ModuleAddress, bigTokenId,
		)
		if err != nil {
			return nil, sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s error safeTransferFrom ", err)
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
		if err := k.SetNFTPairs(ctx, msg.EvmContractAddress, tokenId, msg.ClassId, msg.CosmosTokenIds[i]); err != nil {
			return nil, err
		}
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
		pairUID := k.GetTokenUIDPairByNFTUID(ctx, uNftID)
		if len(pairUID) == 0 {
			return sdkerrors.Wrapf(types.ErrTokenPairNotFound, "missing ERC721 pair for class %s token %s", data.ClassId, tokenId)
		}
		emvTokenId, evmContractAddress := types.GetNFTFromUID(string(pairUID))
		if emvTokenId == "" || evmContractAddress == "" {
			return sdkerrors.Wrapf(types.ErrInternalTokenPair, "invalid ERC721 uid for class %s token %s", data.ClassId, tokenId)
		}

		bigTokenId, err := parseERC721TokenID(emvTokenId)
		if err != nil {
			return err
		}

		contract := common.HexToAddress(evmContractAddress)

		// Check if token has already been refunded
		owner, err := k.QueryERC721TokenOwner(ctx, contract, bigTokenId)
		if err != nil {
			return err
		}
		shouldRefundERC721 := owner == types.ModuleAddress
		var receiver common.Address
		if !shouldRefundERC721 {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute(types.AttributeKeyERC721Token, evmContractAddress),
					sdk.NewAttribute(types.AttributeKeyERC721TokenID, emvTokenId),
					sdk.NewAttribute("reason", "owner_is_not_module_account"),
				),
			)
		} else {
			evmReceiver := k.GetEvmRefundReceiver(ctx, evmContractAddress, tokenId, emvTokenId)
			if len(evmReceiver) == 0 {
				return sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "missing ERC721 refund receiver for contract %s token %s", evmContractAddress, tokenId)
			}
			receiver = common.HexToAddress(string(evmReceiver))

			_, err = k.CallEVM(
				ctx, erc721, types.ModuleAddress, contract, true,
				"safeTransferFrom", types.ModuleAddress, receiver, bigTokenId)
			if err != nil {
				return err
			}
		}

		refundContract := strings.ToLower(evmContractAddress)
		k.DeleteEvmAddressByContractTokenId(ctx, refundContract, tokenId)
		if emvTokenId != tokenId {
			k.DeleteEvmAddressByContractTokenId(ctx, refundContract, emvTokenId)
		}
		k.DeleteNFTPairByNFTID(ctx, data.ClassId, tokenId)
		k.DeleteNFTPairByTokenID(ctx, evmContractAddress, emvTokenId)

		burnMsg := nftTypes.MsgBurnNFT{
			Id:      tokenId,
			DenomId: data.ClassId,
			Sender:  types.AccModuleAddress.String(),
		}
		if _, err = k.nftKeeper.BurnNFT(ctx, &burnMsg); err != nil {
			return err
		}

		if shouldRefundERC721 {
			refundedTokenIds = append(refundedTokenIds, tokenId)
			refundedContract = evmContractAddress
			refundedReceiver = receiver
		}
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
