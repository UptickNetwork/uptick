package keeper

import (
	"strings"

	"cosmossdk.io/errors"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
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
// is lossless for any address format (the CW721 contract address format is
// not a fixed-width hex address). Matching is exact: bech32 case is
// significant. Orphaned records fail the export instead of being dropped.
func (k Keeper) ExportRefundReceivers(ctx sdk.Context) ([]types.RefundReceiver, error) {
	contracts := make([]string, 0)
	for _, pair := range k.GetTokenPairs(ctx) {
		contracts = append(contracts, pair.Cw721Address)
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixCwAddressByContractTokenId)
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
			ContractAddress: contract,
			TokenId:         tokenID,
			Owner:           string(iter.Value()),
		})
	}

	return receivers, nil
}

// splitContractTokenKey recovers the (contract, tokenID) parts of a refund
// store key. The longest registered-contract prefix with a non-empty
// remainder wins; zero or ambiguous matches are rejected. Matching is exact:
// CW721 contracts are bech32, so case is significant.
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
			"refund record key %q does not belong to any registered CW721 contract (orphaned refund state)",
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
	k.SetCwAddressByContractTokenId(ctx, receiver.ContractAddress, receiver.TokenId, receiver.Owner)
}
