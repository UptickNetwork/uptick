package keeper

import (
	"math/big"
	"strings"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	nftTypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/UptickNetwork/uptick/x/erc721/contracts"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// voucherBinding is one per-token binding (reverse index entry) of a voucher
// pair: a native NFT (classID, nftID) bound to an ERC721 (contract, tokenID).
type voucherBinding struct {
	classID  string
	nftID    string
	tokenID  string
	contract string
}

// settleOutcome classifies what a single binding's settlement did, so the
// per-pair purge decision can tell "everything cleared" from "something kept".
type settleOutcome int

const (
	settledOutcome      settleOutcome = iota // wrapped/orphan: ERC burned, binding gone
	staleClearedOutcome                      // stale: binding deleted, nothing burned
	keptOutcome                              // ambiguous: left untouched for A3 self-heal
)

// SettleIBCVoucherPairs clears legacy ibc/ voucher pairs left behind by the
// pre-R1-C convert-memo / manual-convert flow, which bound an ICS-721 voucher
// to an ERC721 contract. Such a pair's presence made the ICS-721 burn guard
// (x/internft IsConvertedNFT) refuse to release the voucher on its way back to
// the origin chain (R1). This is the v0.5.0 migration companion to the
// registration gate (A1) and the terminal un-wrap (A3): the gate stops new
// pairs, this function settles the existing ones, and A3 lets any pair it keeps
// (ambiguous state) self-heal through a normal ConvertERC721.
//
// Per binding it classifies the (ERC owner × native owner) state:
//
//   - user × module (wrapped): burn the ERC721 half and hand the native voucher
//     to the same 20-byte account, then drop the binding — settled.
//   - module × none (orphan, e.g. a pre-v0.4.0 redeem residue): burn the
//     module-held ERC721 half and drop the binding — settled.
//   - user × none / none × none (stale): only drop the binding — stale_cleared.
//   - anything else (native in escrow, ERC held by a contract, ambiguous owner,
//     EVM query failure): leave untouched and log — kept (never destroy an asset
//     we cannot positively attribute; A3 remains the self-heal path).
//
// Each binding is settled in its own cache context so a mid-token failure rolls
// back to a clean state instead of leaving a burned ERC721 with a surviving
// native NFT. It is idempotent: a second run sees no bindings and reports all
// zeros.
func (k Keeper) SettleIBCVoucherPairs(ctx sdk.Context) (settled, staleCleared, kept, pairsDeleted int) {
	// Group every reverse-index binding by class id in one pass, so the pair
	// loop below does not re-scan the store per pair. Corrupt entries (an
	// unparseable UID) are skipped here and surfaced as "kept" pairs.
	bindingsByClass := map[string][]voucherBinding{}
	reverse := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByNFTUID)
	iter := reverse.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		nftID, classID := types.GetNFTFromUID(string(iter.Key()))
		tokenID, contract := types.GetNFTFromUID(string(iter.Value()))
		if nftID == "" || classID == "" || tokenID == "" || contract == "" {
			continue
		}
		bindingsByClass[classID] = append(bindingsByClass[classID], voucherBinding{
			classID:  classID,
			nftID:    nftID,
			tokenID:  tokenID,
			contract: contract,
		})
	}
	iter.Close()

	for _, pair := range k.GetTokenPairs(ctx) {
		if !strings.HasPrefix(pair.ClassId, "ibc/") {
			continue
		}

		bindings := bindingsByClass[pair.ClassId]
		// A voucher pair with no per-token bindings is an empty shell: purge it.
		if len(bindings) == 0 {
			k.PurgeTokenPair(ctx, pair)
			pairsDeleted++
			k.Logger(ctx).Info("ibc voucher pair settlement: purged empty pair", "class", pair.ClassId)
			continue
		}

		cleared := 0
		for _, b := range bindings {
			switch k.settleVoucherBinding(ctx, pair, b) {
			case settledOutcome:
				settled++
				cleared++
			case staleClearedOutcome:
				staleCleared++
				cleared++
			case keptOutcome:
				kept++
			}
		}

		// Purge the pair only when every binding was cleared; a kept binding
		// must keep its pair record so A3 / a future run can still find it.
		if cleared == len(bindings) {
			k.PurgeTokenPair(ctx, pair)
			pairsDeleted++
		}
	}

	k.Logger(ctx).Info(
		"ibc voucher pair settlement",
		"settled", settled,
		"stale_cleared", staleCleared,
		"kept", kept,
		"pairs_deleted", pairsDeleted,
	)
	return settled, staleCleared, kept, pairsDeleted
}

// settleVoucherBinding settles a single binding, isolated in a cache context so
// a failure rolls back cleanly. It returns the outcome classification.
func (k Keeper) settleVoucherBinding(ctx sdk.Context, pair types.TokenPair, b voucherBinding) settleOutcome {
	logger := k.Logger(ctx)

	// Native owner: empty means the NFT does not exist.
	nativeOwner := k.nftKeeper.GetOwner(ctx, b.classID, b.nftID)
	nativeIsModule := len(nativeOwner) > 0 && nativeOwner.Equals(types.AccModuleAddress)
	nativeIsNone := len(nativeOwner) == 0

	bigTokenID, err := parseERC721TokenID(b.tokenID)
	if err != nil {
		logger.Warn("ibc voucher pair settlement: unparseable token id, kept", "class", b.classID, "nft", b.nftID, "token", b.tokenID, "err", err)
		return keptOutcome
	}

	contract := common.HexToAddress(b.contract)
	ercOwner, err := k.QueryERC721TokenOwner(ctx, contract, bigTokenID)
	ercIsModule := false
	ercIsNone := false
	ercIsUser := false
	if err != nil {
		// ownerOf reverted — the ERC721 half no longer exists. This is the
		// normal "already burned" case, not an ambiguous one.
		ercIsNone = true
	} else if ercOwner == (common.Address{}) {
		ercIsNone = true
	} else if ercOwner == types.ModuleAddress {
		ercIsModule = true
	} else if acc := k.evmKeeper.GetAccountWithoutBalance(ctx, ercOwner); acc != nil && acc.HasCodeHash() {
		// The ERC721 is held by a contract, not a user — burning it would
		// destroy a third party's asset. Leave it for manual handling.
		logger.Warn("ibc voucher pair settlement: erc721 held by a contract, kept", "class", b.classID, "nft", b.nftID, "holder", ercOwner)
		return keptOutcome
	} else {
		ercIsUser = true
	}

	switch {
	case ercIsUser && nativeIsModule:
		// wrapped: burn the user's ERC721, hand the escrowed native voucher to
		// the same account, drop the binding.
		return k.settleWrapped(ctx, pair, b, contract, bigTokenID, ercOwner)

	case ercIsModule && nativeIsNone:
		// orphan: burn the module-held ERC721 and drop the binding.
		return k.settleOrphan(ctx, pair, b, contract, bigTokenID)

	case nativeIsNone && (ercIsUser || ercIsNone):
		// stale: the native NFT is already gone; just drop the binding.
		k.deleteBinding(ctx, b)
		return staleClearedOutcome

	default:
		// native in escrow / a user holds the native / any other combination.
		logger.Warn("ibc voucher pair settlement: ambiguous state, kept",
			"class", b.classID, "nft", b.nftID,
			"native_owner", nativeOwner.String(),
			"erc_owner", ercOwner.Hex(),
		)
		return keptOutcome
	}
}

// settleWrapped burns the ERC721 half (held by the user) and transfers the
// native voucher from the module account to the same 20-byte account, then
// drops the binding. All writes run in a cache context so a transfer failure
// rolls the burn back.
func (k Keeper) settleWrapped(ctx sdk.Context, pair types.TokenPair, b voucherBinding, contract common.Address, bigTokenID *big.Int, ercOwner common.Address) settleOutcome {
	cctx, commit := ctx.CacheContext()
	abi := contracts.ERC721UpticksContract.ABI

	if _, err := k.CallEVM(cctx, abi, ercOwner, contract, true, "burn", bigTokenID); err != nil {
		k.Logger(ctx).Warn("ibc voucher pair settlement: burn failed, kept",
			"class", b.classID, "nft", b.nftID, "err", err)
		return keptOutcome
	}

	nft, err := k.nftKeeper.GetNFT(cctx, b.classID, b.nftID)
	if err != nil {
		k.Logger(ctx).Warn("ibc voucher pair settlement: native nft lookup failed, kept",
			"class", b.classID, "nft", b.nftID, "err", err)
		return keptOutcome
	}
	recipient := sdk.AccAddress(ercOwner.Bytes())
	if _, err := k.nftKeeper.TransferNFT(cctx, &nftTypes.MsgTransferNFT{
		DenomId:   b.classID,
		Id:        b.nftID,
		Name:      nft.GetName(),
		URI:       nft.GetURI(),
		Data:      nft.GetData(),
		UriHash:   nft.GetURIHash(),
		Sender:    types.AccModuleAddress.String(),
		Recipient: recipient.String(),
	}); err != nil {
		k.Logger(ctx).Warn("ibc voucher pair settlement: native transfer failed, kept",
			"class", b.classID, "nft", b.nftID, "err", err)
		return keptOutcome
	}

	k.deleteBinding(cctx, b)
	commit()
	k.Logger(ctx).Info("ibc voucher pair settlement: settled wrapped voucher",
		"class", b.classID, "nft", b.nftID, "recipient", recipient.String())
	return settledOutcome
}

// settleOrphan burns the module-held ERC721 half of a voucher whose native side
// is already gone, then drops the binding.
func (k Keeper) settleOrphan(ctx sdk.Context, pair types.TokenPair, b voucherBinding, contract common.Address, bigTokenID *big.Int) settleOutcome {
	cctx, commit := ctx.CacheContext()
	abi := contracts.ERC721UpticksContract.ABI

	if _, err := k.CallEVM(cctx, abi, types.ModuleAddress, contract, true, "burn", bigTokenID); err != nil {
		k.Logger(ctx).Warn("ibc voucher pair settlement: orphan burn failed, kept",
			"class", b.classID, "nft", b.nftID, "err", err)
		return keptOutcome
	}

	k.deleteBinding(cctx, b)
	commit()
	k.Logger(ctx).Info("ibc voucher pair settlement: settled orphan erc721",
		"class", b.classID, "nft", b.nftID)
	return settledOutcome
}

// deleteBinding removes the bidirectional per-token binding and the IBC refund
// receiver record (if any). It mirrors the terminal-un-wrap deletion in
// convertEvm2Cosmos (R1-C A3).
func (k Keeper) deleteBinding(ctx sdk.Context, b voucherBinding) {
	k.DeleteNFTPairByTokenID(ctx, b.contract, b.tokenID)
	k.DeleteNFTPairByNFTID(ctx, b.classID, b.nftID)
	k.DeleteEvmAddressByContractTokenId(ctx, b.contract, b.tokenID)
	k.DeleteEvmAddressByContractTokenId(ctx, b.contract, b.nftID)
}
