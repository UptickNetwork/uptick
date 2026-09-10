package keeper

import (
	"fmt"
	"sort"
	"strings"

	"cosmossdk.io/errors"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// Genesis export/import for the per-token runtime state missing from the
// collection-level ExportGenesis: the bidirectional (tokenUID ↔ nftUID)
// conversion index and the IBC refund receivers. Collection-level TokenPairs
// alone cannot restore this state, so a genesis built from the old export
// silently broke reverse conversions and refunds.

// ExportNFTUIDPairs returns every per-token conversion binding and verifies
// the bidirectional index is internally consistent (forward[x]=y must imply
// reverse[y]=x). A mismatch means the live store is corrupt; failing the
// export loudly is preferable to baking the corruption into a genesis file.
func (k Keeper) ExportNFTUIDPairs(ctx sdk.Context) ([]types.NFTUIDPair, error) {
	pairs, issues := k.ExportNFTUIDPairsWithReport(ctx)
	if len(issues) > 0 {
		return nil, issuesToError(types.ModuleName, issues)
	}
	return pairs, nil
}

// ExportNFTUIDPairsWithReport collects every damaged index entry instead of
// aborting on the first one, so the fail-closed export can report the complete
// repair list in one pass.
func (k Keeper) ExportNFTUIDPairsWithReport(ctx sdk.Context) ([]types.NFTUIDPair, []GenesisExportIssue) {
	forward := make(map[string]string)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	iter := store.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		forward[string(iter.Key())] = string(iter.Value())
	}
	iter.Close()

	pairs := make([]types.NFTUIDPair, 0, len(forward))
	var issues []GenesisExportIssue

	reverseStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByNFTUID)
	reverse := reverseStore.Iterator(nil, nil)
	defer reverse.Close()
	for ; reverse.Valid(); reverse.Next() {
		nftUID := string(reverse.Key())
		tokenUID := string(reverse.Value())

		partner, ok := forward[tokenUID]
		if !ok || partner != nftUID {
			issues = append(issues, GenesisExportIssue{
				Kind: GenesisExportIssueUIDIndexBackward,
				Key:  nftUID,
				Detail: fmt.Sprintf(
					"reverse entry points at token UID %q whose forward entry is %q",
					tokenUID, partner,
				),
			})
			continue
		}
		delete(forward, tokenUID)

		pairs = append(pairs, types.NFTUIDPair{
			TokenUid: tokenUID,
			NftUid:   nftUID,
		})
	}

	for tokenUID, nftUID := range forward {
		issues = append(issues, GenesisExportIssue{
			Kind: GenesisExportIssueUIDIndexForward,
			Key:  tokenUID,
			Detail: fmt.Sprintf(
				"forward entry points at nft UID %q which has no reverse entry",
				nftUID,
			),
		})
	}

	// The forward leftovers are collected from a map, so sort for a stable,
	// reproducible repair list (and reproducible test assertions).
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Key != issues[j].Key {
			return issues[i].Key < issues[j].Key
		}
		return issues[i].Kind < issues[j].Kind
	})

	return pairs, issues
}

// ExportRefundReceivers returns every recorded IBC refund receiver. Refund
// store keys are "<contractAddress><tokenID>" concatenations; the contract
// part is recovered by matching against the registered pair contracts, which
// is lossless for any address format. CW721 contracts are bech32: matching is
// case-insensitive (all-lowercase and all-uppercase encodings are the same
// address), and the exact stored prefix bytes are returned so re-import
// reproduces the original key. Orphaned records fail the export instead of
// being dropped.
func (k Keeper) ExportRefundReceivers(ctx sdk.Context) ([]types.RefundReceiver, error) {
	receivers, issues := k.ExportRefundReceiversWithReport(ctx)
	if len(issues) > 0 {
		return nil, issuesToError(types.ModuleName, issues)
	}
	return receivers, nil
}

// ExportRefundReceiversWithReport exports every refund receiver and reports
// (rather than aborts on) the orphaned keys, so the fail-closed caller can
// print the full repair list at once.
func (k Keeper) ExportRefundReceiversWithReport(ctx sdk.Context) ([]types.RefundReceiver, []GenesisExportIssue) {
	contracts := make([]string, 0)
	for _, pair := range k.GetTokenPairs(ctx) {
		contracts = append(contracts, pair.Cw721Address)
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixCwAddressByContractTokenId)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	receivers := make([]types.RefundReceiver, 0)
	var issues []GenesisExportIssue
	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key())
		contract, tokenID, err := splitContractTokenKey(contracts, key)
		if err != nil {
			issues = append(issues, GenesisExportIssue{
				Kind:   GenesisExportIssueRefundKeyOrphan,
				Key:    fmt.Sprintf("%q", key),
				Detail: err.Error(),
			})
			continue
		}
		receivers = append(receivers, types.RefundReceiver{
			ContractAddress: contract,
			TokenId:         tokenID,
			Owner:           string(iter.Value()),
		})
	}

	return receivers, issues
}

// splitContractTokenKey recovers the (contract, tokenID) parts of a refund
// store key. The longest registered-contract prefix with a non-empty remainder
// wins; zero or ambiguous matches are rejected. Matching is case-insensitive
// because CW721 contracts are bech32 (character case is not significant), and
// the matched prefix is returned with its ORIGINAL stored casing so that
// SetGenesisRefundReceiver reconstructs the exact key on import.
func splitContractTokenKey(contracts []string, key string) (string, string, error) {
	best := 0
	for _, c := range contracts {
		if len(c) <= best || len(c) >= len(key) {
			continue
		}
		if strings.EqualFold(key[:len(c)], c) {
			best = len(c)
		}
	}
	if best == 0 {
		return "", "", errors.Wrapf(
			types.ErrInternalTokenPair,
			"refund record key %q does not belong to any registered CW721 contract (orphaned refund state)",
			key,
		)
	}
	return key[:best], key[best:], nil
}

// SetGenesisNFTUIDPair restores one bidirectional conversion binding during
// InitGenesis, writing both directions of the runtime index.
func (k Keeper) SetGenesisNFTUIDPair(ctx sdk.Context, pair types.NFTUIDPair) {
	k.SetNFTUIDPairByTokenUID(ctx, pair.TokenUid, pair.NftUid)
	k.SetNFTUIDPairByNFTUID(ctx, pair.NftUid, pair.TokenUid)
}

// SetGenesisRefundReceiver restores one IBC refund receiver during
// InitGenesis under the exact store key it was exported from.
func (k Keeper) SetGenesisRefundReceiver(ctx sdk.Context, receiver types.RefundReceiver) {
	k.SetCwAddressByContractTokenId(ctx, receiver.ContractAddress, receiver.TokenId, receiver.Owner)
}

// DeletePairPerTokenState removes the bidirectional NFT UID index entries and
// IBC refund receivers that belong to pair, so a removed token pair cannot
// leave orphans that make ExportGenesis fail (mirrors the erc721 keeper).
//
// Symmetry note: unlike erc721 there is currently no production path that
// removes a cw721 token pair (CW721 contracts cannot self-destruct and no
// message deletes a pair), so today this helper has no live caller. It exists
// so any future pair-removal path (or governance purge message) cleans the
// per-token state in exactly the same way as erc721 instead of leaving
// orphaned UID/refund records behind.
//
// Commit semantics: Cosmos SDK state is per-transaction — a write made on a
// handler path that returns an error is rolled back by baseapp. Callers must
// therefore only invoke this on paths that end in success.
func (k Keeper) DeletePairPerTokenState(ctx sdk.Context, pair types.TokenPair) {
	tokenStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	nftStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByNFTUID)

	var tokenUIDs []string
	iter := tokenStore.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		tokenUID := string(iter.Key())
		_, contract := types.GetNFTFromUID(tokenUID)
		if strings.EqualFold(contract, pair.Cw721Address) {
			tokenUIDs = append(tokenUIDs, tokenUID)
		}
	}
	iter.Close()

	for _, tokenUID := range tokenUIDs {
		nftUID := string(tokenStore.Get([]byte(tokenUID)))
		tokenStore.Delete([]byte(tokenUID))
		if nftUID != "" {
			nftStore.Delete([]byte(nftUID))
		}
	}

	var nftUIDs []string
	reverse := nftStore.Iterator(nil, nil)
	for ; reverse.Valid(); reverse.Next() {
		nftUID := string(reverse.Key())
		_, classID := types.GetNFTFromUID(nftUID)
		if classID == pair.ClassId {
			nftUIDs = append(nftUIDs, nftUID)
		}
	}
	reverse.Close()

	for _, nftUID := range nftUIDs {
		tokenUID := string(nftStore.Get([]byte(nftUID)))
		nftStore.Delete([]byte(nftUID))
		if tokenUID != "" {
			tokenStore.Delete([]byte(tokenUID))
		}
	}

	contract := strings.ToLower(pair.Cw721Address)
	refundStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixCwAddressByContractTokenId)
	refundIter := refundStore.Iterator(nil, nil)
	var refundKeys [][]byte
	for ; refundIter.Valid(); refundIter.Next() {
		key := refundIter.Key()
		// bech32 is case-insensitive at the character level, so compare
		// lowercased. The contract part is the full registered address; slicing
		// on its length keeps the token-id remainder intact.
		if len(key) > len(contract) && strings.EqualFold(string(key[:len(contract)]), pair.Cw721Address) {
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			refundKeys = append(refundKeys, keyCopy)
		}
	}
	refundIter.Close()
	for _, key := range refundKeys {
		refundStore.Delete(key)
	}
}
