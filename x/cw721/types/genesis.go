package types

import (
	"fmt"
	"strings"

	"cosmossdk.io/errors"
)

// NewGenesisState creates a new genesis state.
func NewGenesisState(params Params, pairs []TokenPair) GenesisState {
	return GenesisState{
		Params:     params,
		TokenPairs: pairs,
	}
}

// DefaultGenesisState sets default evm genesis state with empty accounts and
// default params and chain config values.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	seencw721 := make(map[string]bool)
	seenClass := make(map[string]bool)

	for _, b := range gs.TokenPairs {
		// CW721 contracts are bech32: character case is not significant, so the
		// duplicate check is case-insensitive.
		key := strings.ToLower(b.Cw721Address)
		if seencw721[key] {
			return fmt.Errorf("token CW721 contract duplicated on genesis '%s'", b.Cw721Address)
		}
		if seenClass[b.ClassId] {
			return fmt.Errorf("nft class duplicated on genesis: '%s'", b.ClassId)
		}
		if err := b.Validate(); err != nil {
			return err
		}
		seencw721[key] = true
		seenClass[b.ClassId] = true
	}

	// Validate the per-token conversion bindings and IBC refund receivers
	// during genesis validation, so `genesis validate` reports a
	// clean error instead of leaving malformed state to panic in InitGenesis.
	if err := ValidateGenesisPairs(gs.NftUidPairs, gs.RefundReceivers, gs.TokenPairs); err != nil {
		return err
	}

	return gs.Params.Validate()
}

// ValidateGenesisPairs performs the import/validate-side integrity checks:
// non-empty UIDs, one-to-one uniqueness across the batch,
// UID membership in a registered token pair, and refund receivers that
// reference registered token pairs. CW721 contracts are bech32: membership
// matching is case-insensitive (character case is not significant), mirroring
// the EqualFold comparisons the runtime conversion code uses.
func ValidateGenesisPairs(pairs []NFTUIDPair, receivers []RefundReceiver, tokenPairs []TokenPair) error {
	seenToken := make(map[string]struct{}, len(pairs))
	seenNFT := make(map[string]struct{}, len(pairs))
	for _, pair := range pairs {
		if pair.TokenUid == "" || pair.NftUid == "" {
			return errors.Wrapf(ErrInternalTokenPair, "empty NFT UID pair entry (token %q, nft %q)", pair.TokenUid, pair.NftUid)
		}
		if _, dup := seenToken[pair.TokenUid]; dup {
			return errors.Wrapf(ErrInternalTokenPair, "duplicate token UID %q in genesis NFT UID pairs", pair.TokenUid)
		}
		if _, dup := seenNFT[pair.NftUid]; dup {
			return errors.Wrapf(ErrInternalTokenPair, "duplicate NFT UID %q in genesis NFT UID pairs", pair.NftUid)
		}
		if !UIDBelongsToRegisteredPair(pair.TokenUid, pair.NftUid, tokenPairs) {
			return errors.Wrapf(
				ErrInternalTokenPair,
				"NFT UID pair (token %q, nft %q) does not belong to any registered token pair "+
					"(a genesis export drops such records and lists them in <home>/export-issues.json; "+
					"check that file on the node that produced this genesis)",
				pair.TokenUid, pair.NftUid,
			)
		}
		seenToken[pair.TokenUid] = struct{}{}
		seenNFT[pair.NftUid] = struct{}{}
	}

	registered := make(map[string]struct{}, len(tokenPairs))
	for _, tp := range tokenPairs {
		registered[strings.ToLower(tp.Cw721Address)] = struct{}{}
	}

	seenRefund := make(map[string]struct{}, len(receivers))
	for _, r := range receivers {
		if r.ContractAddress == "" || r.TokenId == "" || r.Owner == "" {
			return errors.Wrapf(ErrInternalTokenPair, "incomplete refund receiver entry (contract %q, token %q, owner %q)", r.ContractAddress, r.TokenId, r.Owner)
		}
		if _, ok := registered[strings.ToLower(r.ContractAddress)]; !ok {
			return errors.Wrapf(ErrInternalTokenPair, "refund receiver references unregistered CW721 contract %q", r.ContractAddress)
		}
		dedupe := strings.ToLower(r.ContractAddress) + "," + r.TokenId
		if _, dup := seenRefund[dedupe]; dup {
			return errors.Wrapf(ErrInternalTokenPair, "duplicate refund receiver for contract %q token %q", r.ContractAddress, r.TokenId)
		}
		seenRefund[dedupe] = struct{}{}
	}

	return nil
}

// UIDBelongsToRegisteredPair reports whether the token UID's contract and the
// NFT UID's class resolve to the same registered TokenPair. Contract matching
// is case-insensitive (bech32).
//
// It is exported because it is not only the import-side check: it is also the
// predicate ExportGenesis uses to decide which bindings it may write into a
// genesis file, so that an export can never emit a record this validation (and
// therefore InitGenesis) rejects.
func UIDBelongsToRegisteredPair(tokenUID, nftUID string, tokenPairs []TokenPair) bool {
	_, contract := GetNFTFromUID(tokenUID)
	_, classID := GetNFTFromUID(nftUID)
	if contract == "" || classID == "" {
		return false
	}
	for _, tp := range tokenPairs {
		if strings.EqualFold(tp.Cw721Address, contract) && tp.ClassId == classID {
			return true
		}
	}
	return false
}
