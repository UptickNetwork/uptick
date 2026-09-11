package keeper

import (
	"fmt"
	"sort"
	"strings"

	"cosmossdk.io/errors"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
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
//
// A binding is exported only if the import side would accept it:
// types.ValidateGenesisPairs requires the token UID and the NFT UID to resolve
// to the same registered token pair, and InitGenesis panics on the same
// condition. A consistent binding can still outlive its pair -- a pair record
// that no longer decodes is dropped from the exported TokenPairs, and a
// removed pair's index entries are swept best-effort -- so emitting them
// verbatim produced a genesis that failed its own validation: an unusable
// backup exactly when it is needed. Such bindings are dropped AND reported:
// dropping is the only importable option (nothing this validation rejects can
// survive an InitGenesis), and the dropped record stays visible in
// <home>/export-issues.json so it can be repaired out of band. The live chain
// is untouched either way -- this is purely about producing a genesis that can
// be read back.
func (k Keeper) ExportNFTUIDPairsWithReport(ctx sdk.Context) ([]types.NFTUIDPair, []GenesisExportIssue) {
	forward := make(map[string]string)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	iter := store.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		forward[string(iter.Key())] = string(iter.Value())
	}
	iter.Close()

	// GetTokenPairs returns exactly the healthy subset ExportGenesis writes
	// into TokenPairs (records that do not decode are left out), so the
	// membership predicate is evaluated against what the genesis will contain.
	registered := k.GetTokenPairs(ctx)

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

		// Consume the forward entry on every path below, so an unusable
		// binding is reported once (as unregistered) instead of twice (un-
		// registered here and forward-without-reverse at the end).
		delete(forward, tokenUID)

		if !types.UIDBelongsToRegisteredPair(tokenUID, nftUID, registered) {
			issues = append(issues, GenesisExportIssue{
				Kind: GenesisExportIssueUIDIndexUnregisteredPair,
				Key:  tokenUID,
				Detail: fmt.Sprintf(
					"binding (%q -> %q) does not belong to any registered token pair",
					tokenUID, nftUID,
				),
			})
			continue
		}

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
// is lossless for any address format and fails loudly on orphaned records
// instead of silently dropping refund state.
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
		contracts = append(contracts, pair.Erc721Address)
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEvmAddressByContractTokenId)
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
			EvmContractAddress: contract,
			TokenId:            tokenID,
			EvmAddress:         string(iter.Value()),
		})
	}

	return receivers, issues
}

// ExportIssues aggregates every degradation the genesis export can report,
// without requiring the caller to build the genesis state first. The app-level
// diagnostics report uses it to describe a degraded export without a second
// full export pass.
func (k Keeper) ExportIssues(ctx sdk.Context) []GenesisExportIssue {
	_, pairIssues := k.GetTokenPairsWithReport(ctx)
	_, uidIssues := k.ExportNFTUIDPairsWithReport(ctx)
	_, refundIssues := k.ExportRefundReceiversWithReport(ctx)
	return MergeExportIssues(pairIssues, uidIssues, refundIssues)
}

// splitContractTokenKey recovers the (contract, tokenID) parts of a refund
// store key. The longest registered-contract prefix with a non-empty
// remainder wins; zero or ambiguous matches are rejected.
func splitContractTokenKey(contracts []string, key string) (string, string, error) {
	match := func(haystack string, candidates []string) string {
		best := ""
		for _, c := range candidates {
			if len(c) == 0 || len(c) <= len(best) {
				continue
			}
			if strings.HasPrefix(haystack, c) && len(haystack) > len(c) {
				best = c
			}
		}
		return best
	}

	if best := match(key, contracts); best != "" {
		return best, key[len(best):], nil
	}

	// Case-insensitive retry: refund keys are stored lowercased, but a
	// historical TokenPair.Erc721Address may still be checksum-cased.
	lowerKey := strings.ToLower(key)
	lowerContracts := make([]string, 0, len(contracts))
	for _, c := range contracts {
		lowerContracts = append(lowerContracts, strings.ToLower(c))
	}
	if best := match(lowerKey, lowerContracts); best != "" {
		return best, key[len(best):], nil
	}

	return "", "", errors.Wrapf(
		types.ErrInternalTokenPair,
		"refund record key %q does not belong to any registered ERC721 contract (orphaned refund state)",
		key,
	)
}

// SetGenesisNFTUIDPair restores one bidirectional conversion binding during
// InitGenesis, writing both directions of the runtime index.
func (k Keeper) SetGenesisNFTUIDPair(ctx sdk.Context, pair types.NFTUIDPair) {
	k.SetNFTUIDPairByTokenUID(ctx, pair.TokenUid, pair.NftUid)
	k.SetNFTUIDPairByNFTUID(ctx, pair.NftUid, pair.TokenUid)
}

// SetGenesisRefundReceiver restores one IBC refund receiver during
// InitGenesis. The contract address is lowercased to match runtime refund keys.
func (k Keeper) SetGenesisRefundReceiver(ctx sdk.Context, receiver types.RefundReceiver) {
	k.SetEvmAddressByContractTokenId(ctx, receiver.EvmContractAddress, receiver.TokenId, receiver.EvmAddress)
}

// DeletePairPerTokenState removes the bidirectional NFT UID index entries and
// IBC refund receivers that belong to pair, so a self-destructed (or otherwise
// deleted) TokenPair cannot leave orphans behind and degrade every subsequent
// genesis export.
func (k Keeper) DeletePairPerTokenState(ctx sdk.Context, pair types.TokenPair) {
	tokenStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	nftStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByNFTUID)

	var tokenUIDs []string
	iter := tokenStore.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		tokenUID := string(iter.Key())
		_, contract := types.GetNFTFromUID(tokenUID)
		if strings.EqualFold(contract, pair.Erc721Address) {
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

	contract := strings.ToLower(pair.Erc721Address)
	refundStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEvmAddressByContractTokenId)
	refundIter := refundStore.Iterator(nil, nil)
	var refundKeys [][]byte
	for ; refundIter.Valid(); refundIter.Next() {
		if strings.HasPrefix(strings.ToLower(string(refundIter.Key())), contract) {
			keyCopy := make([]byte, len(refundIter.Key()))
			copy(keyCopy, refundIter.Key())
			refundKeys = append(refundKeys, keyCopy)
		}
	}
	refundIter.Close()
	for _, key := range refundKeys {
		refundStore.Delete(key)
	}
}

// PurgeTokenPair removes a token pair and every piece of state that refers to
// it: the per-token bidirectional index entries, the IBC refund receivers, and
// both lookup maps (contract -> pair id, class -> pair id). It is idempotent.
//
// IMPORTANT — commit semantics: Cosmos SDK state is transactional per
// message-execution. Any write performed on a handler path that eventually
// returns an error is rolled back by baseapp, so PurgeTokenPair must only be
// invoked from a path that ends in success (or from tests / a future
// governance message), otherwise the purge silently never persists. See
// Keeper.ConvertNFT for the one production caller that uses it this way.
func (k Keeper) PurgeTokenPair(ctx sdk.Context, pair types.TokenPair) {
	k.DeletePairPerTokenState(ctx, pair)
	k.DeleteTokenPair(ctx, pair)
	k.DeleteERC721Map(ctx, pair.GetERC721Contract())
	k.DeleteClassMap(ctx, pair.ClassId)
}
