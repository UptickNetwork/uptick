package keeper

import (
	"sort"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// UIDIndexPruneReport is the outcome of one PruneDuplicateUIDIndexEntries pass:
// what it removed and what it refused to touch. Every list is sorted so the
// caller's log line is reproducible.
type UIDIndexPruneReport struct {
	// Scanned is the number of reverse-index entries inspected.
	Scanned int
	// ForwardBefore is the number of forward keys before the pass.
	ForwardBefore int
	// Deleted holds the forward keys removed because they provably duplicated a
	// binding that survives under another spelling.
	Deleted []string
	// Conflicts holds the forward keys a reverse entry's spelling variant
	// matched but whose value no reverse entry vouches for. Left in place.
	Conflicts []string
	// Orphans holds the forward keys no reverse entry can account for, i.e. the
	// reverse half of the pair is missing. Left in place.
	Orphans []string
	// UnpairedNFTs holds the reverse entries whose own forward key is missing.
	UnpairedNFTs []string
}

// ForwardAfter returns the forward key count once the pass has been applied.
func (r UIDIndexPruneReport) ForwardAfter() int { return r.ForwardBefore - len(r.Deleted) }

// Balanced reports whether the forward and reverse indexes agree in size after the
// pass: the property a healthy store has, so it doubles as the migration's
// postcondition (app/upgrades/v041/migrate.go, uidIndexPruneClean).
func (r UIDIndexPruneReport) Balanced() bool { return r.ForwardAfter() == r.Scanned }

// PruneDuplicateUIDIndexEntries removes the duplicate forward keys the
// pre-v0.4.1 write path could leave behind; it is the one-shot half of the
// A-3/A-4 key-spelling work.
//
// WHY A MIGRATION IS NEEDED AT ALL. The runtime fixes make both components of a
// forward key readable under either spelling and make every write canonical, but
// they only rewrite a pair a write path actually touches; a superseded forward key
// persists with no reverse partner -- exactly the GenesisExportIssueUIDIndexForward
// that degrades every genesis export (B-1) and buries real corruption in fixed noise.
//
// THE REVERSE INDEX IS THE AUTHORITY. A forward key is built from two components, so
// a binding can be stored under a cross product of spellings and no single key is
// self-evidently "the" one. The reverse index is keyed by nftUID -- one entry per
// Cosmos NFT, naming the spelling its binding lives under -- so every decision below
// is anchored on it, which is what makes "is this key redundant?" answerable at all.
//
// WHAT IS DELETED, AND ONLY THIS. A forward key K is removed iff all of:
//
//  1. K is not the tokenUID some reverse entry points at (K is not an
//     authority — the authority is the surviving spelling by construction);
//  2. K denotes the same (uint256 token id, 20-byte address) pair as that
//     reverse entry's tokenUID, i.e. K is one of its spelling variants;
//  3. forward[K] is that reverse entry's very nftUID.
//
// Condition 2 must cover BOTH axes: a duplicate can differ in the token-id
// segment (the v0.3.3 "0x"+hex spelling versus the v0.4.0 sha256 base-10 one) as
// easily as in the address segment. Only the address axis fires today -- the
// legacy token-id keys are all still the authority of their binding -- but the
// axis is covered so the criterion need not be revisited if that changes.
//
// Condition 1 is deliberately "K is not an authority" rather than the blunter "K is
// not disputed": mainnet holds six bindings that TWO reverse entries each point at,
// so a key can be B's spelling sibling while duplicating A -- whose own key survives,
// so dropping the duplicate loses nothing, while refusing on B's account would leave
// the store unconverged.
//
// WHAT IS LEFT ALONE. Anything that cannot self-prove as a duplicate is reported
// instead of guessed at:
//
//   - a forward key no reverse entry accounts for ("orphan"): the reverse half is
//     gone, so deleting it would destroy the only remaining record of a binding;
//   - a spelling variant of a reverse entry's tokenUID whose value no reverse entry
//     vouches for ("conflict"): two NFTs claim one token, and picking a winner is
//     exactly what SetNFTPairs refuses to do;
//   - the authoritative key of a non-canonically spelled binding: the runtime reads
//     it and the next write canonicalises it, so it is a pair not written since, not
//     a duplicate. Rewriting it would be a full normalisation, not a shadow removal.
//
// NO ERROR, NO PANIC, IDEMPOTENT. This runs from an upgrade handler: failing would
// halt the chain at the upgrade height over pre-existing state the handler is not
// required to repair, so a residual orphan or conflict is logged at Error level
// instead -- the genesis export's fail-closed-but-keep-running posture. After the
// pass no deletable key remains, so a replay removes nothing; v0.4.1 needs that, as
// it deliberately carries no UpgradeAlreadyApplied guard.
func (k Keeper) PruneDuplicateUIDIndexEntries(ctx sdk.Context) UIDIndexPruneReport {
	forward := k.readUIDIndex(ctx, types.KeyPrefixNFTUIDPairByTokenUID)
	reverse := k.readUIDIndex(ctx, types.KeyPrefixNFTUIDPairByNFTUID)

	report := UIDIndexPruneReport{
		Scanned:       len(reverse),
		ForwardBefore: len(forward),
	}

	// The spellings some reverse entry actually points at. Note these are read
	// from the reverse index rather than canonicalised: an authority is whatever
	// the authority says it is, even when that is the legacy spelling.
	authorities := make(map[string]struct{}, len(reverse))
	for _, tokenUID := range reverse {
		authorities[tokenUID] = struct{}{}
	}

	// Group the forward keys by the binding they denote — the canonical variant
	// is first in the list by contract — so a candidate can be found by identity
	// instead of by enumerating spellings and hoping they match.
	byBinding := make(map[string][]string, len(forward))
	for tokenUID := range forward {
		canonical := types.TokenUIDKeyVariants(tokenUID)[0]
		byBinding[canonical] = append(byBinding[canonical], tokenUID)
	}

	var (
		accounted = make(map[string]struct{}, len(forward))
		deleted   = make(map[string]struct{})
		conflicts = make(map[string]struct{})
	)

	for nftUID, tokenUID := range reverse {
		accounted[tokenUID] = struct{}{}

		binding := types.TokenUIDKeyVariants(tokenUID)[0]
		for _, variant := range byBinding[binding] {
			// The binding's own key survives; another binding's authority is not
			// this entry's to delete.
			if variant == tokenUID {
				continue
			}
			if _, isAuthority := authorities[variant]; isAuthority {
				continue
			}
			// Present by construction: the group was built by ranging over the
			// forward index, so every candidate in it has a value. The check stays
			// to keep "absent" from silently becoming "conflict" if that ever
			// changes -- unreachable now, and no test can make it fail: a property
			// of the design, not a coverage gap.
			stored, ok := forward[variant]
			if !ok {
				continue
			}

			accounted[variant] = struct{}{}
			if stored == nftUID {
				deleted[variant] = struct{}{}
				continue
			}
			conflicts[variant] = struct{}{}
		}
	}

	// A key proven to duplicate one binding is not a conflict, whichever other
	// binding's spelling variant it also happens to be. Resolved after the loop
	// so the result does not depend on map iteration order.
	for variant := range deleted {
		delete(conflicts, variant)
	}

	orphans := make([]string, 0)
	for tokenUID := range forward {
		if _, ok := accounted[tokenUID]; !ok {
			orphans = append(orphans, tokenUID)
		}
	}

	unpaired := make([]string, 0)
	for _, tokenUID := range reverse {
		if _, ok := forward[tokenUID]; !ok {
			unpaired = append(unpaired, tokenUID)
		}
	}

	for variant := range deleted {
		k.DeleteNFTUIDPairByTokenUID(ctx, variant)
	}

	report.Deleted = sortedKeys(deleted)
	report.Conflicts = sortedKeys(conflicts)
	sort.Strings(orphans)
	report.Orphans = orphans
	sort.Strings(unpaired)
	report.UnpairedNFTs = unpaired
	return report
}

// readUIDIndex materializes one of the two UID indexes as a map. It is a whole-
// map read on purpose: the prune has to compare every forward key against every
// reverse entry, and the indexes are per-token conversion state, so a map is
// cheaper than repeated iterator passes.
func (k Keeper) readUIDIndex(ctx sdk.Context, prefixKey []byte) map[string]string {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), prefixKey)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	entries := make(map[string]string)
	for ; iter.Valid(); iter.Next() {
		entries[string(iter.Key())] = string(iter.Value())
	}
	return entries
}

// sortedKeys returns a set's members in a stable order.
func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
