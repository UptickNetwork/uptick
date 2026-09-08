package keeper

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// GetTokenPairs - get all registered token tokenPairs
func (k Keeper) GetTokenPairs(ctx sdk.Context) []types.TokenPair {
	tokenPairs := []types.TokenPair{}

	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.KeyPrefixTokenPair)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var tokenPair types.TokenPair
		if err := k.cdc.Unmarshal(iterator.Value(), &tokenPair); err != nil {
			// Fail loud: skipping a corrupt pair would drop it from ExportGenesis
			// and orphan UID/refund records, which then panics on the next export.
			panic(sdkerrors.Wrap(err, "failed to unmarshal erc721 token pair"))
		}

		tokenPairs = append(tokenPairs, tokenPair)
	}

	return tokenPairs
}

// GetTokenPairID returns the pair id from either of the registered tokens.
//
// DEPRECATED: dispatches by string shape, which is unsafe for hex-shaped
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
func (k Keeper) SetNFTPairs(ctx sdk.Context, contractAddress string, tokenID string, classID string, nftID string) error {
	tokenUID := types.CreateTokenUID(contractAddress, tokenID)
	nftUID := types.CreateNFTUID(classID, nftID)

	forward := k.GetNFTUIDPairByTokenUID(ctx, tokenUID)
	reverse := k.GetTokenUIDPairByNFTUID(ctx, nftUID)

	if len(forward) != 0 && string(forward) != nftUID {
		return sdkerrors.Wrapf(
			types.ErrNFTMappingConflict,
			"erc721 token %s on %s is already bound to nft %s (attempted %s)",
			tokenID, contractAddress, string(forward), nftUID,
		)
	}
	if len(reverse) != 0 && string(reverse) != tokenUID {
		return sdkerrors.Wrapf(
			types.ErrNFTMappingConflict,
			"nft %s of class %s is already bound to token %s (attempted %s)",
			nftID, classID, string(reverse), tokenUID,
		)
	}

	if len(forward) == 0 {
		k.SetNFTPairByContractTokenID(ctx, contractAddress, tokenID, classID, nftID)
	}
	if len(reverse) == 0 {
		k.SetNFTPairByClassNFTID(ctx, classID, nftID, contractAddress, tokenID)
	}
	return nil
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
	tokenUID := types.CreateTokenUID(contractAddress, tokenID)
	return k.GetNFTUIDPairByTokenUID(ctx, tokenUID)
}

func (k Keeper) GetNFTUIDPairByTokenUID(ctx sdk.Context, tokenUID string) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixNFTUIDPairByTokenUID)
	return store.Get([]byte(tokenUID))
}

func (k Keeper) DeleteNFTPairByTokenID(ctx sdk.Context, contractAddress string, tokenID string) {
	tokenUID := types.CreateTokenUID(contractAddress, tokenID)
	k.DeleteNFTUIDPairByTokenUID(ctx, tokenUID)
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
