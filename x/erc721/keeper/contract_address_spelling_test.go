package keeper

import (
	"testing"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// ---------------------------------------------------------------------------
// Contract address spelling compatibility (A-4)
//
// The fixtures are real mainnet state. Note the difference from the token-id
// axis: an address has only ONE value, so the fix here is a normalisation
// (converge on the lowercase key) rather than a spelling-only unification. The
// two axes are independent, and the pair below is non-canonical on BOTH.
//
//	singular pair — 452 of them on mainnet, only one key present:
//	  tokenUID = "0x746f6b656e31303030,0xa92ca1f63C993F2DD1Ec18209579E3b263fdB09c"
//	  nftUID   = "token1000,uptick-1000"
//
//	duplicated pair — the 32 mainnet reserves reported as
//	GenesisExportIssueUIDIndexForward, reachable under two case-variant keys:
//	  tokenUID = "1703751205993357472,0x3bc44CB88233f75B858d0748a45d196bE8375159"
//	          or "1703751205993357472,0x3bc44cb88233f75b858d0748a45d196be8375159"
//	  nftUID   = "uptick1703751205993357472,uptick-3bc44cb88233f75b858d0748a45d196be8375159"
//	  (the reverse index points at the LOWERCASE variant)
// ---------------------------------------------------------------------------

const (
	checksumContract = "0xa92ca1f63C993F2DD1Ec18209579E3b263fdB09c"
	lowerContract    = "0xa92ca1f63c993f2dd1ec18209579e3b263fdb09c"

	legacyToken1000    = "0x746f6b656e31303030"
	canonicalToken1000 = "2147850934834972078128"

	checksumBatchContract = "0x3bc44CB88233f75B858d0748a45d196bE8375159"
	lowerBatchContract    = "0x3bc44cb88233f75b858d0748a45d196be8375159"

	addrClassID      = "uptick-1000"
	addrNFTID        = "token1000"
	addrBatchNFTID   = "uptick1703751205993357472"
	addrBatchClassID = "uptick-3bc44cb88233f75b858d0748a45d196be8375159"
)

func singularTokenUID() string {
	return types.CreateTokenUID(checksumContract, legacyToken1000)
}

func singularNFTUID() string {
	return types.CreateNFTUID(addrClassID, addrNFTID)
}

func singularCanonicalTokenUID() string {
	return types.CreateTokenUID(lowerContract, canonicalToken1000)
}

// seedSingularPair writes the binding exactly as mainnet holds it: both key
// components in their pre-v0.4.0 spelling, and only that one key present.
func seedSingularPair(t *testing.T, k Keeper, ctx sdk.Context) {
	t.Helper()
	k.SetNFTUIDPairByTokenUID(ctx, singularTokenUID(), singularNFTUID())
	k.SetNFTUIDPairByNFTUID(ctx, singularNFTUID(), singularTokenUID())
}

// The load-bearing case. Before A-4 was fixed this binding was unreachable:
// the handler lowercases the address before building the key, and the stored
// key is checksummed, so every handler path missed it.
func TestResolveNFTUIDPair_ReadsChecksummedContractUnderEverySpelling(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedSingularPair(t, k, ctx)

	for _, tc := range []struct {
		name     string
		contract string
		tokenID  string
	}{
		{"canonical token + lowercase contract (normalised handler input)", lowerContract, canonicalToken1000},
		{"legacy token + lowercase contract", lowerContract, legacyToken1000},
		{"canonical token + checksummed contract", checksumContract, canonicalToken1000},
		{"the literal stored key", checksumContract, legacyToken1000},
	} {
		nftUID, matched := k.ResolveNFTUIDPair(ctx, tc.contract, tc.tokenID)
		require.Equal(t, singularNFTUID(), string(nftUID), tc.name)
		require.Equal(t, legacyToken1000, matched,
			"%s: the binding is stored under the legacy token spelling", tc.name)
	}

	// GetNFTPairByContractTokenID inherits the behaviour, so existing callers
	// become address-spelling-agnostic without extra plumbing.
	require.Equal(t, singularNFTUID(),
		string(k.GetNFTPairByContractTokenID(ctx, lowerContract, canonicalToken1000)))
}

func TestResolveNFTUIDPair_IsReadOnlyOnTheAddressAxis(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedSingularPair(t, k, ctx)

	_, _ = k.ResolveNFTUIDPair(ctx, lowerContract, canonicalToken1000)

	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, singularTokenUID()),
		"the checksummed key must survive a read")
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularCanonicalTokenUID()),
		"a read must not create the canonical key")
}

// SetNFTPairs normalises BOTH key components, so the single mainnet key is
// rewritten as the canonical key and the old one is dropped in the same write.
func TestSetNFTPairs_UpgradesBothKeyComponentsInPlace(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedSingularPair(t, k, ctx)

	// The handler normalises the caller's address before this point, and the
	// token id is replayed out of the stored pair, so it arrives legacy.
	require.NoError(t, k.SetNFTPairs(ctx, lowerContract, legacyToken1000, addrClassID, addrNFTID))

	require.Equal(t, singularNFTUID(), string(k.GetNFTUIDPairByTokenUID(ctx, singularCanonicalTokenUID())),
		"the canonical key must hold the binding")
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularTokenUID()),
		"the checksummed-contract key must be gone, not left as a duplicate")
	require.Equal(t, singularCanonicalTokenUID(), string(k.GetTokenUIDPairByNFTUID(ctx, singularNFTUID())),
		"the reverse index must be canonical on both components too")
}

func TestSetNFTPairs_AddressUpgradeIsIdempotent(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedSingularPair(t, k, ctx)

	require.NoError(t, k.SetNFTPairs(ctx, lowerContract, legacyToken1000, addrClassID, addrNFTID))
	// Replaying any spelling of either component must be a no-op, not a
	// conflict with the binding the first call just wrote.
	require.NoError(t, k.SetNFTPairs(ctx, checksumContract, legacyToken1000, addrClassID, addrNFTID))
	require.NoError(t, k.SetNFTPairs(ctx, lowerContract, canonicalToken1000, addrClassID, addrNFTID))
	require.NoError(t, k.SetNFTPairs(ctx, checksumContract, canonicalToken1000, addrClassID, addrNFTID))

	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, singularCanonicalTokenUID()))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularTokenUID()))
}

// A brand-new binding may arrive with EITHER address spelling. Unlike the
// token-id axis the two spellings are the same 20 bytes, so there is nothing
// legitimate to refuse — the point is that both land on the same key.
func TestSetNFTPairs_AcceptsEitherAddressSpellingForANewBinding(t *testing.T) {
	t.Parallel()

	for _, contract := range []string{lowerContract, checksumContract} {
		k, ctx := setupKeeperContext(t)

		require.NoError(t, k.SetNFTPairs(ctx, contract, canonicalToken1000, addrClassID, addrNFTID), contract)

		require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, singularCanonicalTokenUID()), contract)
		require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularTokenUID()),
			"%s: the checksummed spelling must never be written as a key", contract)
	}
}

// Normalising the address must NOT loosen the token-id gate: a legacy token id
// still cannot create a binding, whatever the address spelling is.
func TestSetNFTPairs_LegacyTokenIDIsStillGatedOnAChecksummedContract(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	err := k.SetNFTPairs(ctx, checksumContract, legacyToken1000, addrClassID, addrNFTID)
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, errortypes.ErrInvalidRequest), "got %v", err)

	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularTokenUID()))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularCanonicalTokenUID()))
	require.Empty(t, k.GetTokenUIDPairByNFTUID(ctx, singularNFTUID()))
}

// A binding already taken by another NFT is still a conflict, no matter which
// spelling of the address the caller holds.
func TestSetNFTPairs_ConflictIsStillDetectedAcrossAddressSpellings(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedSingularPair(t, k, ctx)

	for _, contract := range []string{lowerContract, checksumContract} {
		err := k.SetNFTPairs(ctx, contract, canonicalToken1000, addrClassID, "token9999")
		require.Error(t, err, contract)
		require.True(t, errorsmod.IsOf(err, types.ErrNFTMappingConflict), "got %v", err)
	}

	// Nothing moved.
	require.Equal(t, singularNFTUID(), string(k.GetNFTUIDPairByTokenUID(ctx, singularTokenUID())))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularCanonicalTokenUID()))
}

// The mainnet duplicate residue: one nftUID reachable under two forward keys
// that differ only in the case of the contract. A write must collapse them to
// one canonical key instead of preserving both.
func TestSetNFTPairs_CollapsesMainnetDuplicateAddressKeys(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	nftUID := types.CreateNFTUID(addrBatchClassID, addrBatchNFTID)
	tokenID := "1703751205993357472"
	checksumKey := types.CreateTokenUID(checksumBatchContract, tokenID)
	lowerKey := types.CreateTokenUID(lowerBatchContract, tokenID)

	// Mainnet shape: both keys present, reverse index pointing at the lowercase one.
	k.SetNFTUIDPairByTokenUID(ctx, checksumKey, nftUID)
	k.SetNFTUIDPairByTokenUID(ctx, lowerKey, nftUID)
	k.SetNFTUIDPairByNFTUID(ctx, nftUID, lowerKey)

	require.NoError(t, k.SetNFTPairs(ctx, lowerBatchContract, tokenID, addrBatchClassID, addrBatchNFTID))

	require.Equal(t, nftUID, string(k.GetNFTUIDPairByTokenUID(ctx, lowerKey)))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, checksumKey),
		"the checksummed twin must be collapsed, which is what clears the export issue")
	require.Equal(t, lowerKey, string(k.GetTokenUIDPairByNFTUID(ctx, nftUID)))
}

// A purge that replays a stored value must clear the checksummed key too,
// otherwise the orphan keeps degrading every genesis export.
func TestDeleteNFTPairByTokenID_ClearsAddressSpellings(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedSingularPair(t, k, ctx)

	k.DeleteNFTPairByTokenID(ctx, lowerContract, canonicalToken1000)

	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularTokenUID()))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, singularCanonicalTokenUID()))
}

// ---------------------------------------------------------------------------
// Handler-level symptoms. These call GetContractAddressAndTokenIds directly so
// the fix is pinned at the layer that performs the comparison, independently of
// the entry-point normalisation in ConvertNFT.
// ---------------------------------------------------------------------------

// installPair rewrites the "kitty" pair so its stored Erc721Address is exactly
// the given spelling. types.NewTokenPair lowercases, so the record is built
// literally — which is how 14 of the 84 mainnet pairs are stored.
func installPair(t *testing.T, k Keeper, ctx sdk.Context, address string) {
	t.Helper()

	pair := types.TokenPair{Erc721Address: address, ClassId: "kitty"}
	require.NoError(t, k.SetTokenPair(ctx, pair))
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, common.HexToAddress(address), pair.GetID())
}

// seedReverseBinding writes only the reverse (nftUID -> tokenUID) index, the
// way the pre-v0.4.0 module wrote it.
func seedReverseBinding(t *testing.T, k Keeper, ctx sdk.Context, contract, tokenID, nftID string) string {
	t.Helper()

	tokenUID := types.CreateTokenUID(contract, tokenID)
	nftUID := types.CreateNFTUID("kitty", nftID)
	k.SetNFTUIDPairByTokenUID(ctx, tokenUID, nftUID)
	k.SetNFTUIDPairByNFTUID(ctx, nftUID, tokenUID)
	return tokenUID
}

// Symptom 2: a pair whose stored address is the checksummed spelling used to
// fail against its OWN address, because the comparison lowercased one side and
// not the other — the "expect 0x…59 got 0x…59" error.
func TestGetContractAddressAndTokenIds_ChecksummedPairIsNotRejectedAgainstItself(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	installPair(t, k, ctx, checksumBatchContract)
	seedReverseBinding(t, k, ctx, checksumBatchContract, "1000", "nft1")

	contract, tokenIDs, err := k.GetContractAddressAndTokenIds(ctx, &types.MsgConvertNFT{
		ClassId:        "kitty",
		CosmosTokenIds: []string{"nft1"},
	})
	require.NoError(t, err, "the pair's own address must not be rejected against itself")
	require.Equal(t, checksumBatchContract, contract)
	require.Equal(t, []string{"1000"}, tokenIDs)
}

// Symptom 3: a healthy lowercase pair used to fail when the caller spelled the
// very same address with EIP-55 casing, which is independent of any stored
// state — the caller's spelling alone decided the outcome.
func TestGetContractAddressAndTokenIds_AcceptsCallerAddressSpelling(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	installPair(t, k, ctx, lowerBatchContract)
	seedReverseBinding(t, k, ctx, lowerBatchContract, "1000", "nft1")

	contract, tokenIDs, err := k.GetContractAddressAndTokenIds(ctx, &types.MsgConvertNFT{
		ClassId:            "kitty",
		CosmosTokenIds:     []string{"nft1"},
		EvmContractAddress: checksumBatchContract, // same 20 bytes, other spelling
	})
	require.NoError(t, err, "the same address in another spelling is not a contradiction")
	require.Equal(t, lowerBatchContract, contract, "the pair's own spelling is pinned")
	require.Equal(t, []string{"1000"}, tokenIDs)
}

// The comparison must stay strict about genuinely different addresses.
func TestGetContractAddressAndTokenIds_StillRejectsADifferentContract(t *testing.T) {
	k, ctx, _ := setupConvertKeeper(t)
	installPair(t, k, ctx, lowerBatchContract)

	_, _, err := k.GetContractAddressAndTokenIds(ctx, &types.MsgConvertNFT{
		ClassId:            "kitty",
		CosmosTokenIds:     []string{"nft1"},
		EvmContractAddress: "0x9999999999999999999999999999999999999999",
	})
	require.ErrorIs(t, err, types.ErrContractAddressNotCorrect)
}
