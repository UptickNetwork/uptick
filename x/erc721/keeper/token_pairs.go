package keeper

import (
	"fmt"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// GetTokenPairs - get all registered token tokenPairs
//
// Corrupt records are skipped here and reported through
// GetTokenPairsWithReport: this accessor is reachable from the gRPC query path,
// where a panic would take the node's query handler down, and it is no longer
// the genesis-export entry point (ExportGenesis fails closed on the report).
func (k Keeper) GetTokenPairs(ctx sdk.Context) []types.TokenPair {
	tokenPairs, _ := k.GetTokenPairsWithReport(ctx)
	return tokenPairs
}

// GetTokenPairsWithReport returns every decodable token pair plus the list of
// records that could not be decoded. Damaged records are never dropped
// silently: the caller decides whether to abort (genesis export does) or to
// continue with the healthy subset.
func (k Keeper) GetTokenPairsWithReport(ctx sdk.Context) ([]types.TokenPair, []GenesisExportIssue) {
	tokenPairs := []types.TokenPair{}
	var issues []GenesisExportIssue

	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.KeyPrefixTokenPair)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var tokenPair types.TokenPair
		if err := k.cdc.Unmarshal(iterator.Value(), &tokenPair); err != nil {
			issues = append(issues, GenesisExportIssue{
				Kind:   GenesisExportIssueTokenPairCorrupt,
				Key:    fmt.Sprintf("%q", iterator.Key()),
				Detail: err.Error(),
			})
			continue
		}

		tokenPairs = append(tokenPairs, tokenPair)
	}

	return tokenPairs, issues
}

// GetTokenPairID returns the pair id from either of the registered tokens.
//
// Deprecated: dispatches by string shape, which is unsafe for hex-shaped
// class ids. Kept only for backwards compatibility with the gRPC TokenPair
// read query. Handlers and write paths MUST use GetPairByClass /
// GetPairByEVM instead.
func (k Keeper) GetTokenPairID(ctx sdk.Context, token string) []byte {

	if common.IsHexAddress(token) {
		addr := common.HexToAddress(token)
		return k.GetERC721Map(ctx, addr)
	}

	return k.GetClassMap(ctx, token)
}

// GetTokenPair - get registered token pair from the identifier
func (k Keeper) GetTokenPair(ctx sdk.Context, id []byte) (types.TokenPair, bool) {
	if id == nil {
		return types.TokenPair{}, false
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	var tokenPair types.TokenPair
	bz := store.Get(id)
	if len(bz) == 0 {
		return types.TokenPair{}, false
	}

	if err := k.cdc.Unmarshal(bz, &tokenPair); err != nil {
		// A single corrupt pair must not crash the node: this lookup runs from
		// query and message-handler paths, so it degrades to "not found" and logs
		// at Error level. (GetTokenPairs, only reachable from genesis export,
		// panics deliberately to keep corruption out of a genesis file.)
		k.Logger(ctx).Error("failed to unmarshal token pair", "id", string(id), "error", err)
		return types.TokenPair{}, false
	}
	return tokenPair, true
}

// SetTokenPair stores a token pair. A protobuf marshal failure is surfaced to
// the caller instead of being swallowed: silently dropping the write would
// leave the token pair unregistered while later lookups assume it exists.
func (k Keeper) SetTokenPair(ctx sdk.Context, tokenPair types.TokenPair) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	key := tokenPair.GetID()
	bz, err := k.cdc.Marshal(&tokenPair)
	if err != nil {
		return sdkerrors.Wrapf(err, "failed to marshal erc721 token pair %s", string(key))
	}
	store.Set(key, bz)
	return nil
}

// DeleteTokenPair removes a token pair.
func (k Keeper) DeleteTokenPair(ctx sdk.Context, tokenPair types.TokenPair) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	key := tokenPair.GetID()
	store.Delete(key)
}

// GetERC721Map returns the token pair id for the given address
func (k Keeper) GetERC721Map(ctx sdk.Context, erc721 common.Address) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC721)
	// Compatible with older versions
	val := store.Get([]byte(strings.ToLower(erc721.String())))
	if len(val) == 0 {
		val = store.Get(erc721.Bytes())
	}
	return val
}

// GetClassMap returns the token pair id for the given class
func (k Keeper) GetClassMap(ctx sdk.Context, classID string) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByClass)
	return store.Get([]byte(classID))
}

// SetERC721Map sets the token pair id for the given address
func (k Keeper) SetERC721Map(ctx sdk.Context, erc721 common.Address, id []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC721)
	store.Set([]byte(strings.ToLower(erc721.String())), id)
}

// DeleteERC721Map deletes the token pair id for the given address
func (k Keeper) DeleteERC721Map(ctx sdk.Context, erc721 common.Address) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC721)
	store.Delete([]byte(strings.ToLower(erc721.String())))
}

// SetClassMap sets the token pair id for the classID
func (k Keeper) SetClassMap(ctx sdk.Context, classID string, id []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByClass)
	store.Set([]byte(classID), id)
}

// DeleteClassMap deletes the token pair id for the given class.
func (k Keeper) DeleteClassMap(ctx sdk.Context, classID string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByClass)
	store.Delete([]byte(classID))
}

// IsTokenPairRegistered - check if registered token tokenPair is registered
func (k Keeper) IsTokenPairRegistered(ctx sdk.Context, id []byte) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	return store.Has(id)
}

// IsERC721Registered check if registered ERC721 token is registered
func (k Keeper) IsERC721Registered(ctx sdk.Context, erc721 common.Address) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC721)
	// Compatible with older versions
	val := store.Has([]byte(strings.ToLower(erc721.String())))
	if !val {
		val = store.Has(erc721.Bytes())
	}
	return val
}

// IsClassRegistered check if registered nft class is registered
func (k Keeper) IsClassRegistered(ctx sdk.Context, classID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByClass)
	return store.Has([]byte(classID))
}

// SetNFTPairs atomically binds an EVM (contract, tokenID) to a Cosmos NFT
// (classID, nftID) in both directions, enforcing the one-to-one invariant
// forward[x]=y ⟺ reverse[y]=x. A conflicting binding would let an attacker
// release a module-escrowed victim token (registry poisoning) and is rejected
// with ErrNFTMappingConflict so the conversion rolls back.
//
// Key spelling: BOTH components of the forward key are normalised. The token
// id is written as the canonical base-10 value (types.CanonicalEVMTokenID) and
// the contract address as its lowercase form (types.CanonicalContractAddress),
// and any pre-existing key for the same binding under another spelling of
// either component is removed in the same write. That is what lets a pair
// created before v0.4.0 stay readable (see ResolveNFTUIDPair) while the store
// converges on one key per binding instead of accumulating duplicates.
//
// Creation is still strictly gated — but only on the token id, because only
// there do the two spellings mean different things. A legacy "0x"+hex token id
// is accepted only when it is upgrading a binding that already exists; letting
// a fresh binding use the old spelling would key it by a value that differs
// from what the caller's string names. The two address spellings are the same
// 20 bytes, so an address is simply normalised and needs no gate.
func (k Keeper) SetNFTPairs(ctx sdk.Context, contractAddress string, tokenID string, classID string, nftID string) error {
	// GetContractAddressAndTokenIds replays the identifiers it read out of an
	// existing pair, so an upgrade legitimately arrives in the legacy/checksum
	// spellings. Normalise both components here once; everything below works
	// on `canonicalTokenID` and `canonicalContract`.
	canonicalTokenID, isLegacyTokenID, err := types.CanonicalEVMTokenID(tokenID)
	if err != nil {
		return err
	}
	canonicalContract := types.CanonicalContractAddress(contractAddress)

	tokenUID := types.CreateTokenUID(canonicalContract, canonicalTokenID)
	nftUID := types.CreateNFTUID(classID, nftID)

	// Probe every spelling this binding may already be stored under: the cross
	// product of both components' spellings. Without this an upgrade would be
	// mistaken for a new binding, which would both reject the upgrade and
	// leave the old key behind as a duplicate.
	var (
		boundNFT   []byte
		staleKeys  []string
		foundForms int
	)
	for _, tokenVariant := range types.EVMTokenIDKeyVariants(canonicalTokenID) {
		for _, contractVariant := range types.ContractAddressKeyVariants(contractAddress) {
			variantUID := types.CreateTokenUID(contractVariant, tokenVariant)
			stored := k.GetNFTUIDPairByTokenUID(ctx, variantUID)
			if len(stored) == 0 {
				continue
			}
			foundForms++

			if boundNFT == nil {
				boundNFT = stored
			} else if string(stored) != string(boundNFT) {
				// Two spellings of one value bound to two different NFTs is
				// the duplicate-key residue this normalisation exists to
				// prevent. Refuse to guess which is authoritative.
				return sdkerrors.Wrapf(
					types.ErrNFTMappingConflict,
					"erc721 token %s on %s is stored under %d spellings bound to conflicting nfts (%s vs %s)",
					canonicalTokenID, contractAddress, foundForms, string(boundNFT), string(stored),
				)
			}
			if variantUID != tokenUID {
				staleKeys = append(staleKeys, variantUID)
			}
		}
	}

	if len(boundNFT) != 0 && string(boundNFT) != nftUID {
		return sdkerrors.Wrapf(
			types.ErrNFTMappingConflict,
			"erc721 token %s on %s is already bound to nft %s (attempted %s)",
			canonicalTokenID, canonicalContract, string(boundNFT), nftUID,
		)
	}

	// Creation-time gate, token id only (see the doc comment). Msg*.ValidateBasic
	// rejects it too; this closes the internal callers.
	if foundForms == 0 && isLegacyTokenID {
		return sdkerrors.Wrapf(
			errortypes.ErrInvalidRequest,
			"legacy hex ERC721 token id %q cannot create a new binding; use the base-10 token id",
			tokenID,
		)
	}

	reverse := k.GetTokenUIDPairByNFTUID(ctx, nftUID)
	if len(reverse) != 0 && string(reverse) != tokenUID {
		// The reverse index may still hold the other spelling of either
		// component (a pair created before v0.4.0 was written entirely in the
		// legacy forms), so compare the values rather than the strings.
		boundTokenID, boundContract := types.GetNFTFromUID(string(reverse))
		if !types.EqualEVMTokenID(boundTokenID, canonicalTokenID) ||
			!types.EqualContractAddress(boundContract, canonicalContract) {
			return sdkerrors.Wrapf(
				types.ErrNFTMappingConflict,
				"nft %s of class %s is already bound to token %s (attempted %s)",
				nftID, classID, string(reverse), tokenUID,
			)
		}
	}

	// Unconditional upsert: either the binding is new, or it exists and is
	// being rewritten under the canonical spelling. Writing the canonical key
	// also has to happen when the binding was just read under another
	// spelling, otherwise the pair would keep only the non-canonical key.
	k.SetNFTPairByContractTokenID(ctx, canonicalContract, canonicalTokenID, classID, nftID)
	k.SetNFTPairByClassNFTID(ctx, classID, nftID, canonicalContract, canonicalTokenID)
	for _, stale := range staleKeys {
		k.DeleteNFTUIDPairByTokenUID(ctx, stale)
	}
	return nil
}

// ResolveNFTUIDPair resolves the forward (contract, tokenID) → NFT binding
// for a key written under ANY spelling of either component, returning the
// stored NFT UID and the exact token-id spelling it was found under. Both
// spellings of each component denote the same value (a uint256, and a 20-byte
// address), so callers that hold any of them reach the same binding.
//
// Read-only by construction: it never rewrites the key it found. The upgrade
// to the canonical spelling happens in SetNFTPairs, i.e. only when a write
// path is already touching the pair.
func (k Keeper) ResolveNFTUIDPair(ctx sdk.Context, contractAddress string, tokenID string) (nftUID []byte, matchedTokenID string) {
	for _, tokenVariant := range types.EVMTokenIDKeyVariants(tokenID) {
		for _, contractVariant := range types.ContractAddressKeyVariants(contractAddress) {
			tokenUID := types.CreateTokenUID(contractVariant, tokenVariant)
			if stored := k.GetNFTUIDPairByTokenUID(ctx, tokenUID); len(stored) > 0 {
				return stored, tokenVariant
			}
		}
	}
	return nil, ""
}

func (k Keeper) SetNFTPairByContractTokenID(ctx sdk.Context, contractAddress string, tokenID string, classID string, nftID string) {

	tokenUID := types.CreateTokenUID(contractAddress, tokenID)
	nftUID := types.CreateNFTUID(classID, nftID)

	k.SetNFTUIDPairByTokenUID(ctx, tokenUID, nftUID)
}

func (k Keeper) SetNFTUIDPairByTokenUID(ctx sdk.Context, tokenUID string, nftUID string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	store.Set([]byte(tokenUID), []byte(nftUID))
}

func (k Keeper) GetNFTPairByContractTokenID(ctx sdk.Context, contractAddress string, tokenID string) []byte {
	nftUID, _ := k.ResolveNFTUIDPair(ctx, contractAddress, tokenID)
	return nftUID
}

func (k Keeper) GetNFTUIDPairByTokenUID(ctx sdk.Context, tokenUID string) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	return store.Get([]byte(tokenUID))
}

// DeleteNFTPairByTokenID removes the forward binding for a token id under
// every spelling it may have been stored in — both components of the key can
// be spelled more than one way — so a purge driven by a value read out of the
// reverse index cannot leave a key orphaned behind.
func (k Keeper) DeleteNFTPairByTokenID(ctx sdk.Context, contractAddress string, tokenID string) {
	for _, tokenVariant := range types.EVMTokenIDKeyVariants(tokenID) {
		for _, contractVariant := range types.ContractAddressKeyVariants(contractAddress) {
			k.DeleteNFTUIDPairByTokenUID(ctx, types.CreateTokenUID(contractVariant, tokenVariant))
		}
	}
}

func (k Keeper) DeleteNFTUIDPairByTokenUID(ctx sdk.Context, tokenUID string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	store.Delete([]byte(tokenUID))
}

func (k Keeper) SetNFTPairByClassNFTID(ctx sdk.Context, classID string, nftID string, contractAddress string, tokenID string) {

	nftUID := types.CreateNFTUID(classID, nftID)
	tokenUID := types.CreateTokenUID(contractAddress, tokenID)

	k.SetNFTUIDPairByNFTUID(ctx, nftUID, tokenUID)
}

func (k Keeper) SetNFTUIDPairByNFTUID(ctx sdk.Context, nftUID string, tokenUID string) {

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByNFTUID)
	store.Set([]byte(nftUID), []byte(tokenUID))
}

func (k Keeper) GetNFTPairByClassNFTID(ctx sdk.Context, classID string, nftID string) []byte {
	nftUID := types.CreateNFTUID(classID, nftID)
	return k.GetTokenUIDPairByNFTUID(ctx, nftUID)
}

func (k Keeper) GetTokenUIDPairByNFTUID(ctx sdk.Context, nftUID string) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByNFTUID)
	return store.Get([]byte(nftUID))
}

func (k Keeper) DeleteNFTPairByNFTID(ctx sdk.Context, classID string, nftID string) {
	nftUID := types.CreateNFTUID(classID, nftID)
	k.DeleteNFTUIDPairByNFTUID(ctx, nftUID)
}

func (k Keeper) DeleteNFTUIDPairByNFTUID(ctx sdk.Context, nftUID string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByNFTUID)
	store.Delete([]byte(nftUID))
}

// SetEvmAddressByContractTokenId
func (k Keeper) SetEvmAddressByContractTokenId(ctx sdk.Context, evmContractAddress string, evmTokenId string, evmAddress string) {
	contractAndTokenId := strings.ToLower(evmContractAddress) + evmTokenId
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEvmAddressByContractTokenId)
	store.Set([]byte(contractAndTokenId), []byte(evmAddress))
}

func (k Keeper) GetEvmAddressByContractTokenId(ctx sdk.Context, evmContractAddress string, evmTokenId string) []byte {
	contractAndTokenId := strings.ToLower(evmContractAddress) + evmTokenId
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEvmAddressByContractTokenId)
	return store.Get([]byte(contractAndTokenId))
}

func (k Keeper) DeleteEvmAddressByContractTokenId(ctx sdk.Context, evmContractAddress string, evmTokenId string) {
	contractAndTokenId := strings.ToLower(evmContractAddress) + evmTokenId
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEvmAddressByContractTokenId)
	store.Delete([]byte(contractAndTokenId))
}

// SetEvmRefundReceiver records the original ERC721 owner for IBC timeout/error
// refunds. Keys use a lowercased contract address plus both cosmos and EVM
// token ids so lookup matches packet TokenIds and the NFT-UID mapping.
func (k Keeper) SetEvmRefundReceiver(ctx sdk.Context, evmContractAddress string, cosmosTokenIds, evmTokenIds []string, evmAddress string) {
	contract := strings.ToLower(evmContractAddress)
	seen := make(map[string]struct{}, len(cosmosTokenIds)+len(evmTokenIds))
	for _, ids := range [][]string{cosmosTokenIds, evmTokenIds} {
		for _, id := range ids {
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			k.SetEvmAddressByContractTokenId(ctx, contract, id, evmAddress)
		}
	}
}

// GetEvmRefundReceiver looks up the original ERC721 owner recorded at IBC send.
func (k Keeper) GetEvmRefundReceiver(ctx sdk.Context, evmContractAddress, cosmosTokenId, evmTokenId string) []byte {
	contract := strings.ToLower(evmContractAddress)
	if addr := k.GetEvmAddressByContractTokenId(ctx, contract, cosmosTokenId); len(addr) > 0 {
		return addr
	}
	if evmTokenId == "" || evmTokenId == cosmosTokenId {
		return nil
	}
	return k.GetEvmAddressByContractTokenId(ctx, contract, evmTokenId)
}
