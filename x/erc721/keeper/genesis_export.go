package keeper

import (
	"strings"

	"cosmossdk.io/errors"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// Genesis export/import for the per-token runtime state that H-02 identified
// as missing from ExportGenesis: the bidirectional (tokenUID ↔ nftUID)
// conversion index and the IBC refund receivers. Collection-level TokenPairs
// alone cannot restore this state, so a genesis built from the old export
// silently broke reverse conversions and refunds.

// ExportNFTUIDPairs returns every per-token conversion binding and verifies
// the bidirectional index is internally consistent (forward[x]=y must imply
// reverse[y]=x). A mismatch means the live store is corrupt; failing the
// export loudly is preferable to baking the corruption into a genesis file.
func (k Keeper) ExportNFTUIDPairs(ctx sdk.Context) ([]types.NFTUIDPair, error) {
	forward := make(map[string]string)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	iter := store.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		forward[string(iter.Key())] = string(iter.Value())
	}
	iter.Close()

	pairs := make([]types.NFTUIDPair, 0, len(forward))

	reverseStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByNFTUID)
	reverse := reverseStore.Iterator(nil, nil)
	defer reverse.Close()
	for ; reverse.Valid(); reverse.Next() {
		nftUID := string(reverse.Key())
		tokenUID := string(reverse.Value())

		partner, ok := forward[tokenUID]
		if !ok || partner != nftUID {
			return nil, errors.Wrapf(
				types.ErrInternalTokenPair,
				"inconsistent bidirectional NFT UID index: token %q <-> nft %q has no matching forward entry (found %q)",
				tokenUID, nftUID, partner,
			)
		}
		delete(forward, tokenUID)

		pairs = append(pairs, types.NFTUIDPair{
			TokenUid: tokenUID,
			NftUid:   nftUID,
		})
	}

	for tokenUID, nftUID := range forward {
		return nil, errors.Wrapf(
			types.ErrInternalTokenPair,
			"inconsistent bidirectional NFT UID index: token %q -> nft %q has no reverse entry",
			tokenUID, nftUID,
		)
	}

	return pairs, nil
}

// ExportRefundReceivers returns every recorded IBC refund receiver. Refund
// store keys are "<contractAddress><tokenID>" concatenations; the contract
// part is recovered by matching against the registered pair contracts, which
// is lossless for any address format and fails loudly on orphaned records
// instead of silently dropping refund state.
func (k Keeper) ExportRefundReceivers(ctx sdk.Context) ([]types.RefundReceiver, error) {
	contracts := make([]string, 0)
	for _, pair := range k.GetTokenPairs(ctx) {
		contracts = append(contracts, pair.Erc721Address)
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEvmAddressByContractTokenId)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	receivers := make([]types.RefundReceiver, 0)
	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key())
		contract, tokenID, err := splitContractTokenKey(contracts, key)
		if err != nil {
			return nil, err
		}
		receivers = append(receivers, types.RefundReceiver{
			EvmContractAddress: contract,
			TokenId:            tokenID,
			EvmAddress:         string(iter.Value()),
		})
	}

	return receivers, nil
}

// splitContractTokenKey recovers the (contract, tokenID) parts of a refund
// store key. The longest registered-contract prefix with a non-empty
// remainder wins; zero or ambiguous matches are rejected.
func splitContractTokenKey(contracts []string, key string) (string, string, error) {
	best := ""
	for _, c := range contracts {
		if len(c) == 0 || len(c) <= len(best) {
			continue
		}
		if strings.HasPrefix(key, c) && len(key) > len(c) {
			best = c
		}
	}
	if best == "" {
		return "", "", errors.Wrapf(
			types.ErrInternalTokenPair,
			"refund record key %q does not belong to any registered ERC721 contract (orphaned refund state)",
			key,
		)
	}
	return best, key[len(best):], nil
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
	k.SetEvmAddressByContractTokenId(ctx, receiver.EvmContractAddress, receiver.TokenId, receiver.EvmAddress)
}

// ValidateGenesisPairs performs the import-side integrity checks that the
// audit required: non-empty UIDs and one-to-one uniqueness across the whole
// batch. It also returns the set of refund contracts for cross-checking
// against the registered token pairs.
func ValidateGenesisPairs(pairs []types.NFTUIDPair, receivers []types.RefundReceiver, tokenPairs []types.TokenPair) error {
	seenToken := make(map[string]struct{}, len(pairs))
	seenNFT := make(map[string]struct{}, len(pairs))
	for _, pair := range pairs {
		if pair.TokenUid == "" || pair.NftUid == "" {
			return errors.Wrapf(types.ErrInternalTokenPair, "empty NFT UID pair entry (token %q, nft %q)", pair.TokenUid, pair.NftUid)
		}
		if _, dup := seenToken[pair.TokenUid]; dup {
			return errors.Wrapf(types.ErrInternalTokenPair, "duplicate token UID %q in genesis NFT UID pairs", pair.TokenUid)
		}
		if _, dup := seenNFT[pair.NftUid]; dup {
			return errors.Wrapf(types.ErrInternalTokenPair, "duplicate NFT UID %q in genesis NFT UID pairs", pair.NftUid)
		}
		seenToken[pair.TokenUid] = struct{}{}
		seenNFT[pair.NftUid] = struct{}{}
	}

	registered := make(map[string]struct{}, len(tokenPairs))
	for _, tp := range tokenPairs {
		// Refund keys store the lowercased contract address, so membership is
		// checked case-insensitively (EVM hex addresses only differ by case).
		registered[strings.ToLower(tp.Erc721Address)] = struct{}{}
	}

	seenRefund := make(map[string]struct{}, len(receivers))
	for _, r := range receivers {
		if r.EvmContractAddress == "" || r.TokenId == "" || r.EvmAddress == "" {
			return errors.Wrapf(types.ErrInternalTokenPair, "incomplete refund receiver entry (contract %q, token %q, address %q)", r.EvmContractAddress, r.TokenId, r.EvmAddress)
		}
		if _, ok := registered[strings.ToLower(r.EvmContractAddress)]; !ok {
			return errors.Wrapf(types.ErrInternalTokenPair, "refund receiver references unregistered ERC721 contract %q", r.EvmContractAddress)
		}
		dedupe := strings.ToLower(r.EvmContractAddress) + r.TokenId
		if _, dup := seenRefund[dedupe]; dup {
			return errors.Wrapf(types.ErrInternalTokenPair, "duplicate refund receiver for contract %q token %q", r.EvmContractAddress, r.TokenId)
		}
		seenRefund[dedupe] = struct{}{}
	}

	return nil
}
