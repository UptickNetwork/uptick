package v2

import (
	"fmt"
	"strings"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

// Problem describes a single legacy collection store record that the 1→2
// migration would fail on.
type Problem struct {
	// Kind is "denom" or "nft".
	Kind string
	// StoreKey is the raw legacy store key the record was found under.
	StoreKey []byte
	// DenomID / TokenID identify the record when they could be parsed.
	DenomID string
	TokenID string
	// Reason explains why the migration would reject this record.
	Reason string
}

func (p Problem) String() string {
	id := p.DenomID
	if p.TokenID != "" {
		id += "/" + p.TokenID
	}
	return fmt.Sprintf("%s %q: %s", p.Kind, id, p.Reason)
}

// PrecheckLegacyStore scans the legacy (irismod/nft-shaped) collection store
// with the exact iteration and parsing logic the 1→2 migration uses, but
// performs no writes and does not stop on the first bad record: every legacy
// denom / NFT that would make Migrate fail is collected and reported, so an
// upgrade operator gets a complete, readable report before any state change
// instead of an opaque hard stop mid-migration (L-4).
//
// It must run on the pre-migration store, i.e. before the module's
// RunMigrations call in the upgrade handler.
func PrecheckLegacyStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.Codec) []Problem {
	store := ctx.KVStore(storeKey)
	var problems []Problem

	// Denoms: KeyDenom prefix (0x04/), value = types.Denom.
	denomIter := storetypes.KVStorePrefixIterator(store, KeyDenom(""))
	defer denomIter.Close()
	for ; denomIter.Valid(); denomIter.Next() {
		var denom types.Denom
		if err := cdc.Unmarshal(denomIter.Value(), &denom); err != nil {
			problems = append(problems, Problem{
				Kind:     "denom",
				StoreKey: append([]byte(nil), denomIter.Key()...),
				DenomID:  legacyDenomIDFromKey(denomIter.Key()),
				Reason:   fmt.Sprintf("unmarshal Denom: %v", err),
			})
			continue
		}
		if _, err := sdk.AccAddressFromBech32(denom.Creator); err != nil {
			problems = append(problems, Problem{
				Kind:     "denom",
				StoreKey: append([]byte(nil), denomIter.Key()...),
				DenomID:  denom.Id,
				Reason:   fmt.Sprintf("creator bech32 %q: %v", denom.Creator, err),
			})
		}
	}
	denomIter.Close()

	// NFTs are stored per-denom under the KeyNFT prefix; walk every denom
	// subtree the same way migrateToken does.
	nftIter := storetypes.KVStorePrefixIterator(store, KeyNFT("", ""))
	defer nftIter.Close()
	for ; nftIter.Valid(); nftIter.Next() {
		denomID, tokenID := legacyNFTKeyParts(nftIter.Key())
		var baseNFT types.BaseNFT
		if err := cdc.Unmarshal(nftIter.Value(), &baseNFT); err != nil {
			problems = append(problems, Problem{
				Kind:     "nft",
				StoreKey: append([]byte(nil), nftIter.Key()...),
				DenomID:  denomID,
				TokenID:  tokenID,
				Reason:   fmt.Sprintf("unmarshal BaseNFT: %v", err),
			})
			continue
		}
		if _, err := sdk.AccAddressFromBech32(baseNFT.Owner); err != nil {
			problems = append(problems, Problem{
				Kind:     "nft",
				StoreKey: append([]byte(nil), nftIter.Key()...),
				DenomID:  denomID,
				TokenID:  tokenID,
				Reason:   fmt.Sprintf("owner bech32 %q: %v", baseNFT.Owner, err),
			})
		}
	}
	return problems
}

// legacyDenomIDFromKey extracts the denom id from a KeyDenom(denomID) key.
func legacyDenomIDFromKey(key []byte) string {
	// key = PrefixDenom(0x04) + "/" + denomID
	if len(key) <= len(KeyDenom("")) {
		return ""
	}
	return string(key[len(KeyDenom("")):])
}

// legacyNFTKeyParts splits a KeyNFT(denomID, tokenID) key into its two
// components. Both ids may technically contain the legacy "/" delimiter is not
// possible (ids are validated before issue), so splitting on the single
// delimiter after the prefix is safe.
func legacyNFTKeyParts(key []byte) (denomID, tokenID string) {
	prefixLen := len(KeyNFT("", ""))
	rest := key[prefixLen:]
	sep := indexByte(rest, '/')
	if sep < 0 {
		return string(rest), ""
	}
	return string(rest[:sep]), string(rest[sep+1:])
}

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// FormatProblems renders a precheck report for upgrade-handler error output.
func FormatProblems(problems []Problem) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("collection v1→v2 migration precheck found %d record(s) that would abort the upgrade:\n", len(problems)))
	for i, p := range problems {
		sb.WriteString(fmt.Sprintf("  %d) %s (store key %x)\n", i+1, p.String(), p.StoreKey))
	}
	return sb.String()
}
