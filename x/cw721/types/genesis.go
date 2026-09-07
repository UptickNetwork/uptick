package types

import (
	"fmt"

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
		if seencw721[b.Cw721Address] {
			return fmt.Errorf("token CW721 contract duplicated on genesis '%s'", b.Cw721Address)
		}
		if seenClass[b.ClassId] {
			return fmt.Errorf("nft class duplicated on genesis: '%s'", b.ClassId)
		}
		if err := b.Validate(); err != nil {
			return err
		}
		seencw721[b.Cw721Address] = true
		seenClass[b.ClassId] = true
	}

	// H-02: validate the per-token conversion bindings and IBC refund
	// receivers during genesis validation, so `genesis validate` reports a
	// clean error instead of leaving malformed state to panic in InitGenesis.
	if err := ValidateGenesisPairs(gs.NftUidPairs, gs.RefundReceivers, gs.TokenPairs); err != nil {
		return err
	}

	return gs.Params.Validate()
}

// ValidateGenesisPairs performs the import/validate-side integrity checks that
// the audit required: non-empty UIDs, one-to-one uniqueness across the batch,
// UID membership in a registered token pair, and refund receivers that
// reference registered token pairs. CW721 contracts are bech32: matching is
// exact (case is significant), unlike ERC721 hex addresses.
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
		if !uidBelongsToRegisteredPair(pair.TokenUid, pair.NftUid, tokenPairs) {
			return errors.Wrapf(
				ErrInternalTokenPair,
				"NFT UID pair (token %q, nft %q) does not belong to any registered token pair",
				pair.TokenUid, pair.NftUid,
			)
		}
		seenToken[pair.TokenUid] = struct{}{}
		seenNFT[pair.NftUid] = struct{}{}
	}

	registered := make(map[string]struct{}, len(tokenPairs))
	for _, tp := range tokenPairs {
		registered[tp.Cw721Address] = struct{}{}
	}

	seenRefund := make(map[string]struct{}, len(receivers))
	for _, r := range receivers {
		if r.ContractAddress == "" || r.TokenId == "" || r.Owner == "" {
			return errors.Wrapf(ErrInternalTokenPair, "incomplete refund receiver entry (contract %q, token %q, owner %q)", r.ContractAddress, r.TokenId, r.Owner)
		}
		if _, ok := registered[r.ContractAddress]; !ok {
			return errors.Wrapf(ErrInternalTokenPair, "refund receiver references unregistered CW721 contract %q", r.ContractAddress)
		}
		dedupe := r.ContractAddress + "," + r.TokenId
		if _, dup := seenRefund[dedupe]; dup {
			return errors.Wrapf(ErrInternalTokenPair, "duplicate refund receiver for contract %q token %q", r.ContractAddress, r.TokenId)
		}
		seenRefund[dedupe] = struct{}{}
	}

	return nil
}

// uidBelongsToRegisteredPair reports whether the token UID's contract and the
// NFT UID's class resolve to the same registered TokenPair.
func uidBelongsToRegisteredPair(tokenUID, nftUID string, tokenPairs []TokenPair) bool {
	_, contract := GetNFTFromUID(tokenUID)
	_, classID := GetNFTFromUID(nftUID)
	if contract == "" || classID == "" {
		return false
	}
	for _, tp := range tokenPairs {
		if tp.Cw721Address == contract && tp.ClassId == classID {
			return true
		}
	}
	return false
}
