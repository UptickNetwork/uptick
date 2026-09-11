package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// ---------------------------------------------------------------------------
// Forward-index prune (③-a): delete only keys that self-prove redundant
//
// The fixtures below are the shapes the live chains actually hold. The one
// load-bearing number: mainnet has 1110 forward keys against 1078 reverse
// entries, testnet 132 against 102 — the difference is exactly the duplicate
// keys that make every genesis export report degraded, and the prune must
// remove those and nothing else.
//
//	duplicated pair (the 32 mainnet / 30 testnet shadows)
//	  reverse  nftUID -> "1703751205993357472,0x3bc44...5159"   (lowercase)
//	  forward  "1703751205993357472,0x3bc44...5159"  and
//	           "1703751205993357472,0x3bc44CB8...5159"          (checksummed)
//	           both carrying the same nftUID
//
//	singular pair (the 452 mainnet / 36 testnet uniques)
//	  only ONE key present, and it is the non-canonical one; it must survive
// ---------------------------------------------------------------------------

// pruneSeedForward writes one forward binding without a reverse entry.
func pruneSeedForward(t *testing.T, k Keeper, ctx sdk.Context, tokenUID, nftUID string) {
	t.Helper()
	k.SetNFTUIDPairByTokenUID(ctx, tokenUID, nftUID)
}

// pruneSeedReverse writes the reverse entry, which is the authority the prune
// anchors every decision on.
func pruneSeedReverse(t *testing.T, k Keeper, ctx sdk.Context, nftUID, tokenUID string) {
	t.Helper()
	k.SetNFTUIDPairByNFTUID(ctx, nftUID, tokenUID)
}

// The tie-breaker in the criterion: a spelling variant of a reverse entry's
// tokenUID whose value that entry does NOT vouch for is not a duplicate. Deleting
// it would be a guess about which of two nfts owns the token, which is exactly
// what SetNFTPairs refuses to do — so it is reported and left alone, and this is
// the only test that can tell the value comparison apart from "any variant will
// do".
func TestPruneDuplicateUIDIndexEntries_KeepsVariantsBoundToAnotherNFT(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	authority := types.CreateTokenUID(lowerContract, canonicalToken1000)
	stray := types.CreateTokenUID(lowerContract, legacyToken1000)
	nftUID := types.CreateNFTUID(addrClassID, "nft-owner")
	otherNFTUID := types.CreateNFTUID(addrClassID, "nft-trespasser")

	pruneSeedReverse(t, k, ctx, nftUID, authority)
	pruneSeedForward(t, k, ctx, authority, nftUID)
	pruneSeedForward(t, k, ctx, stray, otherNFTUID)

	report := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Empty(t, report.Deleted,
		"the stray key spells out the same binding but carries a different nft, so it is not provably redundant")
	require.Equal(t, []string{stray}, report.Conflicts)
	require.False(t, report.Balanced())
	require.Equal(t, otherNFTUID, string(k.GetNFTUIDPairByTokenUID(ctx, stray)),
		"the prune must not resolve a two-nft dispute by deleting one of them")
}

func warnBatchTokenUID(contract string) string {
	return types.CreateTokenUID(contract, "1703751205993357472")
}

func warnBatchNFTUID() string {
	return types.CreateNFTUID(addrBatchClassID, addrBatchNFTID)
}

// The mainnet shape: the reverse index points at the lowercase key and the
// checksummed spelling is a leftover from the pre-v0.4.0 module.
func TestPruneDuplicateUIDIndexEntries_RemovesAddressSpellingShadow(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	authority := warnBatchTokenUID(lowerBatchContract)
	shadow := warnBatchTokenUID(checksumBatchContract)
	nftUID := warnBatchNFTUID()
	require.NotEqual(t, authority, shadow, "the fixture must actually differ in spelling")

	pruneSeedReverse(t, k, ctx, nftUID, authority)
	pruneSeedForward(t, k, ctx, authority, nftUID)
	pruneSeedForward(t, k, ctx, shadow, nftUID)

	report := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Equal(t, []string{shadow}, report.Deleted,
		"the checksummed key is the duplicate; the authority is what the reverse index points at")
	require.Empty(t, report.Conflicts)
	require.Empty(t, report.Orphans)
	require.Empty(t, report.UnpairedNFTs)
	require.True(t, report.Balanced(), "one forward key per reverse entry afterwards")
	require.Equal(t, 1, report.ForwardAfter())

	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, authority), "the authority must survive")
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, shadow), "the shadow must be gone")
	require.Equal(t, authority, string(k.GetTokenUIDPairByNFTUID(ctx, nftUID)),
		"the reverse index is not rewritten by the prune")
}

// The token-id axis. It fires on neither live chain today — the legacy token-id
// keys are all still the authority of their own binding, so nothing about them
// is duplicated — but it is the same defect class as the address axis and the
// criterion has to cover it, or a future v0.4.x write that re-keys a pair would
// leave the old key behind with this prune unable to see it.
func TestPruneDuplicateUIDIndexEntries_RemovesTokenIDSpellingShadow(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	authority := canonicalTokenUID()
	shadow := legacyTokenUID()
	nftUID := legacyNFTUID()

	pruneSeedReverse(t, k, ctx, nftUID, authority)
	pruneSeedForward(t, k, ctx, authority, nftUID)
	pruneSeedForward(t, k, ctx, shadow, nftUID)

	report := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Equal(t, []string{shadow}, report.Deleted,
		"the legacy \"0x\"+hex spelling is the duplicate of the base-10 authority")
	require.True(t, report.Balanced())
	require.Empty(t, k.GetNFTUIDPairByTokenUID(ctx, shadow))
	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, authority))
}

// The case ③-a must NOT touch: a binding whose only key is spelled the old way.
// Deleting it would destroy the only record of the binding, and it is not a
// duplicate of anything — it is the authority. Rewriting it to the canonical
// spelling would be a full store normalisation, a different and much larger
// change than removing shadows.
func TestPruneDuplicateUIDIndexEntries_KeepsTheOnlyNonCanonicalKey(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)
	seedSingularPair(t, k, ctx) // checksummed address + legacy token id, single key

	report := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Empty(t, report.Deleted, "a binding with no sibling has nothing to prune")
	require.Empty(t, report.Conflicts)
	require.Empty(t, report.Orphans)
	require.True(t, report.Balanced())
	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, singularTokenUID()),
		"the only key of a live binding must survive the prune")
}

// Orphans are the one thing the prune must never "clean up". A forward key with
// no reverse entry is unpaired, not duplicated: it is the last record of a
// binding whose other half is gone, and the genesis export reports it rather
// than dropping it for the same reason.
func TestPruneDuplicateUIDIndexEntries_KeepsOrphansAndReportsThem(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	authority := warnBatchTokenUID(lowerBatchContract)
	shadow := warnBatchTokenUID(checksumBatchContract)
	nftUID := warnBatchNFTUID()
	pruneSeedReverse(t, k, ctx, nftUID, authority)
	pruneSeedForward(t, k, ctx, authority, nftUID)
	pruneSeedForward(t, k, ctx, shadow, nftUID)

	orphan := types.CreateTokenUID(lowerContract, canonicalToken1000)
	pruneSeedForward(t, k, ctx, orphan, "orphaned-nft,uptick-1000")

	report := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Equal(t, []string{shadow}, report.Deleted,
		"the healthy binding must still be pruned when an orphan is present")
	require.Equal(t, []string{orphan}, report.Orphans)
	require.Empty(t, report.Conflicts)
	require.False(t, report.Balanced(), "an orphan leaves the two counts unequal by design")
	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, orphan),
		"an unpaired key is not a duplicate and must not be deleted")
}

// The shape mainnet actually holds six of: TWO reverse entries pointing at
// canonically equal token UIDs, each bound to a different nft. The key that
// duplicates one of them must still go, and neither authority may be touched —
// which is why the rule keeps any key a reverse entry points at, instead of
// refusing to delete anything a second binding also lays claim to.
func TestPruneDuplicateUIDIndexEntries_CleansShadowWithoutTouchingAnotherBindingsAuthority(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	authorityA := types.CreateTokenUID(lowerContract, canonicalToken1000)
	authorityB := types.CreateTokenUID(lowerContract, legacyToken1000)
	shadow := types.CreateTokenUID(checksumContract, canonicalToken1000)
	nftA := types.CreateNFTUID(addrClassID, "nft-a")
	nftB := types.CreateNFTUID(addrClassID, "nft-b")

	// B's token id is the legacy spelling of A's value, so A and B denote the
	// same binding while carrying different nfts: exactly the mainnet collision.
	require.NotEqual(t, nftA, nftB)

	pruneSeedReverse(t, k, ctx, nftA, authorityA)
	pruneSeedReverse(t, k, ctx, nftB, authorityB)
	pruneSeedForward(t, k, ctx, authorityA, nftA)
	pruneSeedForward(t, k, ctx, authorityB, nftB)
	pruneSeedForward(t, k, ctx, shadow, nftA)

	report := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Equal(t, []string{shadow}, report.Deleted,
		"a key proven to duplicate binding A is prunable even though it also spells out B's binding")
	require.Empty(t, report.Conflicts,
		"it is not a conflict: it was proven redundant for one binding and is not any binding's authority")
	require.Empty(t, report.Orphans)
	require.True(t, report.Balanced())

	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, authorityA),
		"binding A's authority must survive")
	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, authorityB),
		"binding B's authority must survive: the prune must not pick a winner between two nfts")
}

// The prune is the only thing that removes these keys, and v0.4.1 deliberately
// carries no UpgradeAlreadyApplied guard, so a replayed plan runs it again. A
// second pass must be a no-op rather than, say, deleting a key it just learned
// to treat as canonical.
func TestPruneDuplicateUIDIndexEntries_IsIdempotent(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	authority := warnBatchTokenUID(lowerBatchContract)
	shadow := warnBatchTokenUID(checksumBatchContract)
	nftUID := warnBatchNFTUID()
	pruneSeedReverse(t, k, ctx, nftUID, authority)
	pruneSeedForward(t, k, ctx, authority, nftUID)
	pruneSeedForward(t, k, ctx, shadow, nftUID)

	first := k.PruneDuplicateUIDIndexEntries(ctx)
	require.Equal(t, []string{shadow}, first.Deleted)

	second := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Empty(t, second.Deleted, "a replayed plan must delete nothing")
	require.Equal(t, first.Scanned, second.Scanned)
	require.Equal(t, first.ForwardAfter(), second.ForwardBefore)
	require.Equal(t, first.ForwardAfter(), second.ForwardAfter())
	require.True(t, second.Balanced())
	require.NotEmpty(t, k.GetNFTUIDPairByTokenUID(ctx, authority))
}

// The IBC refund keys live in the same module store under their own prefix and
// are deliberately NOT normalised (their key carries the token id as the module
// received it, and rewriting it would orphan the records already on chain). The
// prune walks the two UID indexes and must leave that prefix completely alone.
func TestPruneDuplicateUIDIndexEntries_DoesNotTouchRefundReceivers(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	k.SetEvmAddressByContractTokenId(ctx, lowerContract, canonicalToken1000, "uptick1receiver")

	authority := warnBatchTokenUID(lowerBatchContract)
	shadow := warnBatchTokenUID(checksumBatchContract)
	nftUID := warnBatchNFTUID()
	pruneSeedReverse(t, k, ctx, nftUID, authority)
	pruneSeedForward(t, k, ctx, authority, nftUID)
	pruneSeedForward(t, k, ctx, shadow, nftUID)

	report := k.PruneDuplicateUIDIndexEntries(ctx)
	require.Equal(t, []string{shadow}, report.Deleted)

	require.Equal(t, "uptick1receiver",
		string(k.GetEvmAddressByContractTokenId(ctx, lowerContract, canonicalToken1000)),
		"refund state is out of scope for the index prune")
}

// An empty store is the state of a chain that never converted anything. The
// prune runs from PreBlocker regardless, so it has to be a no-op without
// reporting damage it cannot see.
func TestPruneDuplicateUIDIndexEntries_EmptyStoreIsCleanAndReportsNothing(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	report := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Zero(t, report.Scanned)
	require.Zero(t, report.ForwardBefore)
	require.Empty(t, report.Deleted)
	require.Empty(t, report.Conflicts)
	require.Empty(t, report.Orphans)
	require.Empty(t, report.UnpairedNFTs)
	require.True(t, report.Balanced())
}

// A reverse entry whose own forward key is missing is the mirror image of an
// orphan and must be reported, not repaired by inventing a key.
func TestPruneDuplicateUIDIndexEntries_ReportsUnpairedReverseEntries(t *testing.T) {
	t.Parallel()

	k, ctx := setupKeeperContext(t)

	missing := types.CreateTokenUID(lowerContract, canonicalToken1000)
	pruneSeedReverse(t, k, ctx, "lost-nft,uptick-1000", missing)

	report := k.PruneDuplicateUIDIndexEntries(ctx)

	require.Equal(t, []string{missing}, report.UnpairedNFTs)
	require.Empty(t, report.Deleted)
	require.False(t, report.Balanced(), "the reverse index has an entry the forward one does not")
}
