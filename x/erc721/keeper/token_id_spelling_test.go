package keeper

import (
	"testing"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// Real mainnet state: the pair keyed by
// "0x6e66746d70313233343536373839,0x087254935ad3d71deacbd2789c3a954b7f9e0822"
// binds cosmos nft "nftmp123456789" of class "mp123456". The token id is the
// legacy spelling (hex of the nft id bytes); its base-10 value is what the
// canonical spelling denotes, so the two are interchangeable for lookups.
const (
	legacyContract    = "0x087254935ad3d71deacbd2789c3a954b7f9e0822"
	legacyTokenID     = "0x6e66746d70313233343536373839"
	canonicalTokenID  = "2239182361542004876924632130074681"
	legacyClassID     = "mp123456"
	legacyNFTID       = "nftmp123456789"
	otherContractAddr = "0x1111111111111111111111111111111111111111"
)

func legacyTokenUID() string {
	return types.CreateTokenUID(legacyContract, legacyTokenID)
}

func canonicalTokenUID() string {
	return types.CreateTokenUID(legacyContract, canonicalTokenID)
}

func legacyNFTUID() string {
	return types.CreateNFTUID(legacyClassID, legacyNFTID)
}

// seedLegacyPair writes the binding exactly as it exists on mainnet: both
// directions keyed by the legacy spelling, with no normalisation applied.
func seedLegacyPair(t *testing.T, k Keeper, ctx sdk.Context) {
	t.Helper()
	k.SetNFTUIDPairByTokenUID(ctx, legacyTokenUID(), legacyNFTUID())
	k.SetNFTUIDPairByNFTUID(ctx, legacyNFTUID(), legacyTokenUID())
}

// Reading must reach a legacy binding from either spelling, and must not
// rewrite anything while doing so.
func TestResolveNFTUIDPair_ReadsLegacyBindingFromEitherSpelling(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedLegacyPair(t, k, ctx)

	byCanonical, matched := k.ResolveNFTUIDPair(ctx, legacyContract, canonicalTokenID)
	require.Equal(t, legacyNFTUID(), string(byCanonical))
	require.Equal(t, legacyTokenID, matched, "the binding is still stored under its legacy spelling")

	byLegacy, matchedLegacy := k.ResolveNFTUIDPair(ctx, legacyContract, legacyTokenID)
	require.Equal(t, string(byCanonical), string(byLegacy))
	require.Equal(t, legacyTokenID, matchedLegacy)

	// GetNFTPairByContractTokenID inherits the same behaviour, so every
	// existing caller becomes spelling-agnostic without extra plumbing.
	require.Equal(t, legacyNFTUID(),
		string(k.GetNFTPairByContractTokenID(ctx, legacyContract, canonicalTokenID)))
}

func TestResolveNFTUIDPair_IsReadOnly(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedLegacyPair(t, k, ctx)

	_, _ = k.ResolveNFTUIDPair(ctx, legacyContract, canonicalTokenID)

	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, legacyTokenUID()),
		"the legacy key must survive a read")
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID()),
		"a read must not create the canonical key")
}

func TestResolveNFTUIDPair_IsContractScoped(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedLegacyPair(t, k, ctx)

	nftUID, _ := k.ResolveNFTUIDPair(ctx, otherContractAddr, canonicalTokenID)
	require.Empty(t, nftUID, "the same value on another contract is a different binding")
}

// The upgrade path: GetContractAddressAndTokenIds replays the stored spelling,
// so this is exactly what a conversion of a pre-v0.4.0 pair looks like.
func TestSetNFTPairs_UpgradesLegacyKeyInPlace(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedLegacyPair(t, k, ctx)

	require.NoError(t, k.SetNFTPairs(ctx, legacyContract, legacyTokenID, legacyClassID, legacyNFTID))

	require.Equal(t, legacyNFTID, func() string {
		nft, _ := types.GetNFTFromUID(string(k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID())))
		return nft
	}(), "the canonical key now holds the binding")

	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, legacyTokenUID()),
		"the legacy key must be dropped, not left as a duplicate")

	require.Equal(t, canonicalTokenUID(), string(k.GetTokenUIDPairByNFTUID(ctx, legacyNFTUID())),
		"the reverse index must store the canonical spelling too")
}

func TestSetNFTPairs_UpgradeIsIdempotent(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedLegacyPair(t, k, ctx)

	require.NoError(t, k.SetNFTPairs(ctx, legacyContract, legacyTokenID, legacyClassID, legacyNFTID))
	// Replaying either spelling after the upgrade must be a no-op, not a
	// conflict with the binding it just wrote.
	require.NoError(t, k.SetNFTPairs(ctx, legacyContract, legacyTokenID, legacyClassID, legacyNFTID))
	require.NoError(t, k.SetNFTPairs(ctx, legacyContract, canonicalTokenID, legacyClassID, legacyNFTID))

	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID()))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, legacyTokenUID()))
}

// Creation is the gated direction: a legacy spelling is only ever accepted
// when it upgrades a binding that already exists.
func TestSetNFTPairs_RejectsLegacySpellingForNewBinding(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	err := k.SetNFTPairs(ctx, legacyContract, legacyTokenID, legacyClassID, legacyNFTID)
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, errortypes.ErrInvalidRequest), "got %v", err)

	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, legacyTokenUID()))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID()))
	require.Empty(t, k.GetTokenUIDPairByNFTUID(ctx, legacyNFTUID()))

	// The canonical spelling still creates the same binding.
	require.NoError(t, k.SetNFTPairs(ctx, legacyContract, canonicalTokenID, legacyClassID, legacyNFTID))
	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID()))
}

func TestSetNFTPairs_UpgradeStillRejectsConflictingBinding(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedLegacyPair(t, k, ctx)

	// The value is bound to nftmp123456789; upgrading it as a different NFT
	// must be refused even though the spelling is now canonical.
	err := k.SetNFTPairs(ctx, legacyContract, canonicalTokenID, legacyClassID, "nft-other")
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, types.ErrNFTMappingConflict), "got %v", err)

	// Nothing moved: the original binding keeps its spelling.
	require.Equal(t, legacyNFTUID(), string(k.GetNFTUIDPairByTokenUID(ctx, legacyTokenUID())))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID()))
}

// The reverse index may hold the legacy spelling of the same value; that is
// equivalence, not a conflict.
func TestSetNFTPairs_ReverseIndexLegacySpellingIsEquivalence(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	// Forward deliberately absent, so this also covers the "reverse exists
	// but forward does not" case: the value comparison must still match.
	k.SetNFTUIDPairByNFTUID(ctx, legacyNFTUID(), legacyTokenUID())

	require.NoError(t, k.SetNFTPairs(ctx, legacyContract, canonicalTokenID, legacyClassID, legacyNFTID))

	require.Equal(t, canonicalTokenUID(), string(k.GetTokenUIDPairByNFTUID(ctx, legacyNFTUID())))
	require.Equal(t, legacyNFTUID(), string(k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID())))
}

func TestSetNFTPairs_ReverseIndexDifferentTokenIsConflict(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	k.SetNFTUIDPairByNFTUID(ctx, legacyNFTUID(), types.CreateTokenUID(legacyContract, "999"))

	err := k.SetNFTPairs(ctx, legacyContract, canonicalTokenID, legacyClassID, legacyNFTID)
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, types.ErrNFTMappingConflict), "got %v", err)

	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID()))
}

// The refund cleanup replays the value read out of the reverse index, so a
// delete driven by the canonical spelling must still clear a legacy key.
func TestDeleteNFTPairByTokenID_RemovesEverySpelling(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedLegacyPair(t, k, ctx)

	k.DeleteNFTPairByTokenID(ctx, legacyContract, canonicalTokenID)

	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, legacyTokenUID()))
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, canonicalTokenUID()))
}

// End-to-end read compatibility on the resolution used by MsgConvertERC721:
// a holder of the canonical value must be resolved to the NFT the legacy pair
// binds, instead of deriving a fresh nft id from the token id.
func TestGetClassIDAndNFTID_ResolvesLegacyPairFromCanonicalInput(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedLegacyPair(t, k, ctx)

	classID, nftIDs, err := k.GetClassIDAndNFTID(ctx, &types.MsgConvertERC721{
		EvmContractAddress: legacyContract,
		EvmTokenIds:        []string{canonicalTokenID},
	})
	require.NoError(t, err)
	require.Equal(t, legacyClassID, classID)
	require.Equal(t, []string{legacyNFTID}, nftIDs)
}

// Without the lookup the keeper would derive CreateNFTIDFromTokenID(id) and
// mint into a class that does not match what the pair recorded.
func TestGetClassIDAndNFTID_DoesNotDeriveFromUnboundToken(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	classID, nftIDs, err := k.GetClassIDAndNFTID(ctx, &types.MsgConvertERC721{
		EvmContractAddress: legacyContract,
		EvmTokenIds:        []string{canonicalTokenID},
	})
	require.NoError(t, err)
	require.Equal(t, types.CreateClassIDFromContractAddress(legacyContract), classID)
	require.Equal(t, []string{types.CreateNFTIDFromTokenID(canonicalTokenID)}, nftIDs)
}
