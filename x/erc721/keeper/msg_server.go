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

// pairContractRedeployable reports whether a class whose pair contract lost
// its code can be healed by deploying a fresh module-owned contract. Classes
// derived from a contract address ("uptick-<addr>") are pinned to that
// external contract and cannot be re-deployed.
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

	// Probe the EVM contract map directly rather than routing through
	// GetTokenPairID: that helper dispatches by string shape and a caller that
	// reuses an already-canonical lowercase hex address here would always
	// resolve via the contract map anyway.
	id := k.GetERC721Map(ctx, common.HexToAddress(msg.EvmContractAddress))
	if len(id) == 0 {

		_, err := k.RegisterERC721(ctx, msg)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to RegisterERC721")
		}
	}

	pair, err := k.GetPairByEVM(ctx, msg.EvmContractAddress)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to GetPair")
	}

	erc721 := common.HexToAddress(pair.Erc721Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc721)
	// cosmos/evm stores keccak256(nil) (EmptyCodeHash, 32 bytes) when the
	// account exists but has no code. len==0 only matches a missing hash
	// field; self-destructed contracts still have EmptyCodeHash. Match
	// upstream x/erc20: HasCodeHash is false for nil, empty, and EmptyCodeHash.
	if acc == nil || !acc.HasCodeHash() {
		// ERC721 -> Cosmos conversion requires a live pair contract; a contract
		// without code is terminal for this direction. Do NOT purge the pair
		// here — writes on a failing handler path are rolled back by the SDK.
		// Recovery happens through ConvertNFT (purge + re-deploy).
		k.Logger(ctx).Warn(
			"erc721 pair contract self-destructed; conversion is terminal, state left untouched",
			"class", pair.ClassId,
			"contract", pair.Erc721Address,
		)
		return nil, sdkerrors.Wrapf(types.ErrInternalTokenPair, "erc721 contract %s is self-destructed", pair.Erc721Address)
	}

	// Pin the conversion to the pair's canonical class: a caller-supplied
	// ClassId that differs from the registered pair is rejected so NFTs
	// cannot be minted into arbitrary third-party denoms.
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

	// Pre-validate the batch size before deploying any contract or touching
	// state. A caller who passes an oversized batch would otherwise reach the
	// check inside convertCosmos2Evm only after GetContractAddressAndTokenIds
	// has already deployed an ERC721 contract for an unregistered class: the
	// SDK rolls the handler back, but the transaction has by then burned a
	// whole contract deployment's worth of gas and store writes for a batch
	// that could never succeed. x/cw721 bounds the batch before deploying; this
	// is the same order on the erc721 side.
	//
	// msg.CosmosTokenIds is the caller's list; EvmTokenIds is the handler's
	// output and is still empty here, so checking it would compare against
	// nothing and let an oversized batch through.
	if len(msg.CosmosTokenIds) > maxERC721BatchSize {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "ERC721 batch size %d exceeds maximum %d", len(msg.CosmosTokenIds), maxERC721BatchSize)
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
	// Probe the EVM contract map directly rather than routing through
	// GetTokenPairID: that helper dispatches by string shape and the address
	// here is already canonical lowercase hex.
	id := k.GetERC721Map(ctx, common.HexToAddress(msg.EvmContractAddress))
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

	// Pin the conversion to the pair's canonical class (mirrors ConvertERC721):
	// a mismatch, e.g. a hex-address-shaped denom colliding with a registered
	// contract, must be rejected instead of minting into the wrong namespace.
	if msg.ClassId != pair.ClassId {
		return nil, sdkerrors.Wrapf(
			types.ErrClassIdNotCorrect,
			"class id is not correct, expect %s got %s",
			pair.ClassId, msg.ClassId,
		)
	}

	// Self-heal a pair whose ERC721 contract no longer has code. SDK writes
	// made on a failing handler path are rolled back, so a stale pair can
	// never be purged from inside a failing call: for module-redeployable
	// classes the purge + fresh deploy must happen in THIS successful
	// transaction. Classes pinned to an external contract stay terminal.
	erc721 := common.HexToAddress(pair.Erc721Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc721)

	if acc == nil || !acc.HasCodeHash() {
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

		pair, err = k.GetPairByClass(ctx, msg.ClassId)
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
				// mintEnhance failed — fall back to plain mint. Log the
				// original error so the root cause is not lost.
				k.Logger(ctx).Debug("mintEnhance failed, falling back to mint",
					"token", bigTokenIds[i].String(), "err", err)
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
			// module-escrowed ERC721: a mismatched (classID, nftID) would let an
			// attacker pair their own NFT with a victim's escrowed token and drain it.
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

// validateNoMappingConflict enforces the one-to-one binding before any
// state-mutating call, mirroring convertCosmos2Evm (SetNFTPairs also rejects
// conflicts, but only after the NFT has been minted/transferred).
func (k Keeper) validateNoMappingConflict(ctx sdk.Context, msg *types.MsgConvertERC721) error {
	for i, tokenId := range msg.EvmTokenIds {
		bound := k.GetNFTPairByContractTokenID(ctx, msg.EvmContractAddress, tokenId)
		if len(bound) == 0 {
			continue
		}
		expectedNFTUID := types.CreateNFTUID(msg.ClassId, msg.CosmosTokenIds[i])
		if string(bound) != expectedNFTUID {
			return sdkerrors.Wrapf(
				types.ErrNFTMappingConflict,
				"erc721 token %s is already bound to nft %s, not %s",
				tokenId, string(bound), expectedNFTUID,
			)
		}
		// Reverse check: the Cosmos NFT must not already be bound to a
		// different EVM token (defense-in-depth — SetNFTPairs also
		// rejects, but we want to fail before EVM transfer/NFT mint).
		reverseBound := k.GetTokenUIDPairByNFTUID(ctx, expectedNFTUID)
		if len(reverseBound) > 0 {
			boundEvmTokenId, _ := types.GetNFTFromUID(string(reverseBound))
			if boundEvmTokenId != tokenId {
				return sdkerrors.Wrapf(
					types.ErrNFTMappingConflict,
					"nft %s is already bound to erc721 token %s, not %s",
					expectedNFTUID, boundEvmTokenId, tokenId,
				)
			}
		}
	}
	return nil
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

	if err := k.validateNoMappingConflict(ctx, msg); err != nil {
		return nil, err
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
			return nil, sdkerrors.Wrapf(types.ErrEVMCall, "failed to safeTransferFrom erc721 token %s: %v", tokenId, err)
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
				return nil, sdkerrors.Wrapf(err, "failed to mint nft")
			}

		} else {
			// The native NFT must be escrowed by the module account: it was
			// locked there when it was converted cosmos -> evm. Verify
			// ownership explicitly so a stale mapping (e.g. the NFT has
			// already been returned to a user) fails with a clear error here
			// instead of an opaque unauthorized error inside TransferNFT.
			// validateNoMappingConflict already guarantees that any existing
			// mapping matches (msg.ClassId, msg.CosmosTokenIds[i]).
			if ownerAddr := k.nftKeeper.GetOwner(ctx, msg.ClassId, msg.CosmosTokenIds[i]); !ownerAddr.Equals(types.AccModuleAddress) {
				return nil, sdkerrors.Wrapf(
					errortypes.ErrUnauthorized,
					"nft %s of class %s is not escrowed by the module account (current owner %s)",
					msg.CosmosTokenIds[i], msg.ClassId, ownerAddr,
				)
			}

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
				return nil, sdkerrors.Wrapf(err, "failed to transfer nft")
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

// erc721RefundGroup aggregates the successfully refunded tokens of one
// (contract, receiver) pair, so the refund event stays accurate when a single
// packet carries tokens from several contracts or for several receivers.
type erc721RefundGroup struct {
	contract string
	receiver string
	// EVM ERC721 token ids refunded to this (contract, receiver) pair
	tokenIds []string
	// native NFT ids burned for those tokens
	nftIds []string
}

// appendERC721RefundGroup appends to the group matching (contract, receiver),
// creating it when it does not exist yet. Groups keep first-seen order so event
// emission stays deterministic across nodes.
func appendERC721RefundGroup(groups []erc721RefundGroup, contract, receiver, tokenID, nftID string) []erc721RefundGroup {
	for i := range groups {
		if groups[i].contract == contract && groups[i].receiver == receiver {
			groups[i].tokenIds = append(groups[i].tokenIds, tokenID)
			groups[i].nftIds = append(groups[i].nftIds, nftID)
			return groups
		}
	}
	return append(groups, erc721RefundGroup{
		contract: contract,
		receiver: receiver,
		tokenIds: []string{tokenID},
		nftIds:   []string{nftID},
	})
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
	var groups []erc721RefundGroup

	for _, tokenId := range data.TokenIds {

		uNftID := types.CreateNFTUID(data.ClassId, tokenId)
		pairUID := k.GetTokenUIDPairByNFTUID(ctx, uNftID)
		if len(pairUID) == 0 {
			// No recorded pair means this token never entered the ERC721
			// escrow flow (or its mapping was already cleaned up), so there
			// is nothing to refund and no contract/token id to refund it to.
			// Returning here would abort the whole IBC callback and strand
			// every other token in the packet, so skip with a distinguishable
			// event — consistent with the owner-query and transfer skips below
			// and symmetric with x/cw721 (round 10, N-1).
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute("reason", "erc721_pair_not_found"),
				),
			)
			continue
		}
		evmTokenId, evmContractAddress := types.GetNFTFromUID(string(pairUID))
		if evmTokenId == "" || evmContractAddress == "" {
			// A stored pair that cannot be split into (contract, token) is
			// broken module state. Same rationale — never let one corrupt
			// record abort the refund for the rest of the packet. Emit its
			// own reason so it can be triaged and repaired out of band.
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute("reason", "erc721_pair_uid_invalid"),
				),
			)
			continue
		}

		bigTokenId, err := parseERC721TokenID(evmTokenId)
		if err != nil {
			// A stored EVM token id that is not a valid uint256 is the same
			// class of corrupt-state problem as the invalid UID above: skip
			// it and keep the rest of the packet refundable.
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute(types.AttributeKeyERC721Token, evmContractAddress),
					sdk.NewAttribute(types.AttributeKeyERC721TokenID, evmTokenId),
					sdk.NewAttribute("reason", "erc721_token_id_invalid"),
				),
			)
			continue
		}

		contract := common.HexToAddress(evmContractAddress)

		// Defense-in-depth: if the native NFT is already gone when the refund
		// runs, skip BOTH the native burn and the EVM-side refund — a divergence
		// implies a partial prior refund, and re-running the EVM refund would
		// risk double payment. Checked before the EVM owner query to save gas.
		//
		// NOTE (round 11, F-5): this branch deliberately does NOT clean up the
		// pair mappings, unlike x/cw721 (whose nft_already_gone check sits
		// after its mapping cleanup). Here the check fires before the EVM owner
		// query, so deleting the mapping would leave an escrowed ERC721 token
		// with no on-chain record of where it lives — undiscoverable and
		// unrecoverable. Keeping the mapping makes the residue triageable; the
		// skip is idempotent, so retries just re-emit this event.
		//
		// (round 25) The event carries pairs_retained=true for the same reason
		// the comment is needed: the two modules publish the same event type
		// with the same reason for opposite states, so a reason-keyed alert
		// table cannot tell "residue to triage" from "already converged"
		// without a cross-module lookup. x/cw721 omits the attribute.
		if !k.nftKeeper.HasNFT(ctx, data.ClassId, tokenId) {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute(types.AttributeKeyERC721Token, evmContractAddress),
					sdk.NewAttribute(types.AttributeKeyERC721TokenID, evmTokenId),
					sdk.NewAttribute(types.AttributeKeyPairsRetained, "true"),
					sdk.NewAttribute("reason", "nft_already_gone"),
				),
			)
			continue
		}

		// Check if token has already been refunded
		owner, err := k.QueryERC721TokenOwner(ctx, contract, bigTokenId)
		if err != nil {
			// Mirror of cw721_owner_query_failed: one unreachable or
			// misbehaving contract must not abort the refund of the remaining
			// tokens in the packet. Skipped tokens keep their pair mappings so
			// a later retry can still refund them.
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute(types.AttributeKeyERC721Token, evmContractAddress),
					sdk.NewAttribute(types.AttributeKeyERC721TokenID, evmTokenId),
					sdk.NewAttribute("reason", "erc721_owner_query_failed"),
				),
			)
			continue
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
					sdk.NewAttribute(types.AttributeKeyERC721TokenID, evmTokenId),
					sdk.NewAttribute("reason", "owner_is_not_module_account"),
				),
			)
		} else {
			evmReceiver := k.GetEvmRefundReceiver(ctx, evmContractAddress, tokenId, evmTokenId)
			if len(evmReceiver) == 0 {
				// One token without a recorded receiver must not abort the
				// whole packet. Returning here used to tear down the IBC
				// callback's cache context, leaving every OTHER token in the
				// packet unrefunded and the packet itself retrying forever.
				// Skip with a distinguishable reason — symmetric with every
				// other per-token failure in this function and with x/cw721.
				// The pair mapping and the native NFT are left untouched so a
				// later repair/retry can still refund this token.
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						types.EventTypeRefundPacketTokenSkip,
						sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
						sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
						sdk.NewAttribute(types.AttributeKeyERC721Token, evmContractAddress),
						sdk.NewAttribute(types.AttributeKeyERC721TokenID, evmTokenId),
						sdk.NewAttribute("reason", "erc721_refund_receiver_missing"),
					),
				)
				continue
			}
			receiver = common.HexToAddress(string(evmReceiver))

			_, err = k.CallEVM(
				ctx, erc721, types.ModuleAddress, contract, true,
				"safeTransferFrom", types.ModuleAddress, receiver, bigTokenId)
			if err != nil {
				// Mirror of cw721_transfer_failed. The EVM transfer did NOT
				// happen, so the mapping cleanup and native burn below must
				// not run either (burning the native NFT without the EVM
				// refund completing would destroy the user's asset) — skip
				// the whole token and leave it retryable.
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						types.EventTypeRefundPacketTokenSkip,
						sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
						sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
						sdk.NewAttribute(types.AttributeKeyERC721Token, evmContractAddress),
						sdk.NewAttribute(types.AttributeKeyERC721TokenID, evmTokenId),
						sdk.NewAttribute("reason", "erc721_transfer_failed"),
					),
				)
				continue
			}
		}

		refundContract := strings.ToLower(evmContractAddress)
		k.DeleteEvmAddressByContractTokenId(ctx, refundContract, tokenId)
		if evmTokenId != tokenId {
			k.DeleteEvmAddressByContractTokenId(ctx, refundContract, evmTokenId)
		}
		k.DeleteNFTPairByNFTID(ctx, data.ClassId, tokenId)
		k.DeleteNFTPairByTokenID(ctx, evmContractAddress, evmTokenId)

		// Defense-in-depth: if the native NFT is owned by a non-module address
		// (e.g. already refunded on a parallel path), skip the burn; the ERC721
		// refund proceeds independently below.
		if ownerAddr := k.nftKeeper.GetOwner(ctx, data.ClassId, tokenId); !ownerAddr.Equals(types.AccModuleAddress) {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute(types.AttributeKeyERC721Token, evmContractAddress),
					sdk.NewAttribute(types.AttributeKeyERC721TokenID, evmTokenId),
					sdk.NewAttribute(types.AttributeKeyNFTOwner, ownerAddr.String()),
					sdk.NewAttribute("reason", "nft_owner_not_module"),
				),
			)
			if shouldRefundERC721 {
				groups = appendERC721RefundGroup(
					groups,
					common.HexToAddress(evmContractAddress).Hex(),
					receiver.Hex(),
					evmTokenId,
					tokenId,
				)
			}
			continue
		}

		burnMsg := nftTypes.MsgBurnNFT{
			Id:      tokenId,
			DenomId: data.ClassId,
			Sender:  types.AccModuleAddress.String(),
		}
		if _, err = k.nftKeeper.BurnNFT(ctx, &burnMsg); err != nil {
			// Skip, do not abort. The ERC721 refund for this token has already
			// been recorded above (or the ERC721 was confirmed non-module-owned)
			// and the pair mappings are already gone, so returning the error
			// here would (a) abandon every remaining token in the packet,
			// (b) leave the ack unwritten, which makes the relayer retry this
			// packet forever, and (c) on that retry take a different branch,
			// recording two contradictory outcomes for one token. Same policy
			// as the nft_owner_not_module branch above.
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRefundPacketTokenSkip,
					sdk.NewAttribute(types.AttributeKeyNFTClass, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyNFTID, tokenId),
					sdk.NewAttribute(types.AttributeKeyERC721Token, evmContractAddress),
					sdk.NewAttribute(types.AttributeKeyERC721TokenID, evmTokenId),
					sdk.NewAttribute("reason", "nft_burn_failed"),
					sdk.NewAttribute("error", err.Error()),
				),
			)
			continue
		}

		if shouldRefundERC721 {
			groups = appendERC721RefundGroup(
				groups,
				common.HexToAddress(evmContractAddress).Hex(),
				receiver.Hex(),
				evmTokenId,
				tokenId,
			)
		}
	}

	if len(groups) > 0 {
		ctx.EventManager().EmitEvents(erc721RefundEvents(data.Sender, data.ClassId, groups))
	}

	return nil
}

// erc721RefundEvents builds one refund event per (contract, receiver) pair from
// the aggregated groups.
func erc721RefundEvents(sender, classID string, groups []erc721RefundGroup) sdk.Events {
	events := make(sdk.Events, 0, len(groups))
	for _, group := range groups {
		events = append(events, sdk.NewEvent(
			types.EventTypeRefundPacketToken,
			sdk.NewAttribute(sdk.AttributeKeySender, sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, group.receiver),
			sdk.NewAttribute(types.AttributeKeyNFTClass, classID),
			sdk.NewAttribute(types.AttributeKeyNFTID, strings.Join(group.nftIds, ",")),
			sdk.NewAttribute(types.AttributeKeyERC721Token, group.contract),
			sdk.NewAttribute(types.AttributeKeyERC721TokenID, strings.Join(group.tokenIds, ",")),
		))
	}
	return events
}
