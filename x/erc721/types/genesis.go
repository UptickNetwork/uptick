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
	seenErc721 := make(map[string]bool)
	seenClass := make(map[string]bool)

	for _, b := range gs.TokenPairs {
		key := strings.ToLower(b.Erc721Address)
		if seenErc721[key] {
			return fmt.Errorf("token ERC721 contract duplicated on genesis '%s'", b.Erc721Address)
		}
		if seenClass[b.ClassId] {
			return fmt.Errorf("nft class duplicated on genesis: '%s'", b.ClassId)
		}
		if err := b.Validate(); err != nil {
			return err
		}
		seenErc721[key] = true
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
// non-empty UIDs, one-to-one uniqueness across the whole batch, UID membership
// in a registered token pair, and refund receivers that reference registered
// token pairs.
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
		// Refund keys store the lowercased contract address, so membership is
		// checked case-insensitively (EVM hex addresses only differ by case).
		registered[strings.ToLower(tp.Erc721Address)] = struct{}{}
	}

	seenRefund := make(map[string]struct{}, len(receivers))
	for _, r := range receivers {
		if r.EvmContractAddress == "" || r.TokenId == "" || r.EvmAddress == "" {
			return errors.Wrapf(ErrInternalTokenPair, "incomplete refund receiver entry (contract %q, token %q, address %q)", r.EvmContractAddress, r.TokenId, r.EvmAddress)
		}
		if _, ok := registered[strings.ToLower(r.EvmContractAddress)]; !ok {
			return errors.Wrapf(ErrInternalTokenPair, "refund receiver references unregistered ERC721 contract %q", r.EvmContractAddress)
		}
		dedupe := strings.ToLower(r.EvmContractAddress) + r.TokenId
		if _, dup := seenRefund[dedupe]; dup {
			return errors.Wrapf(ErrInternalTokenPair, "duplicate refund receiver for contract %q token %q", r.EvmContractAddress, r.TokenId)
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
		if strings.EqualFold(tp.Erc721Address, contract) && tp.ClassId == classID {
			return true
		}
	}
	return false
}
