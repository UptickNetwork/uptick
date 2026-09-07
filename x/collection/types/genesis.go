package types

import (
	sdkerrors "cosmossdk.io/errors"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewGenesisState creates a new genesis state.
func NewGenesisState(collections []Collection) *GenesisState {
	return &GenesisState{
		Collections: collections,
	}
}

// DefaultGenesisState returns a default genesis state
func DefaultGenesisState() *GenesisState {
	return NewGenesisState([]Collection{})
}

// ValidateGenesis performs basic validation of nfts genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data GenesisState) error {
	for _, c := range data.Collections {
		if c.Denom.Id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "collection denom id is empty")
		}
		if err := ValidateDenomID(c.Denom.Id); err != nil {
			return err
		}

		seenTokenIDs := make(map[string]struct{}, len(c.NFTs))
		for _, nft := range c.NFTs {
			if nft.GetOwner().Empty() {
				return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "missing owner")
			}

			if err := ValidateTokenIDForDenom(c.Denom.Id, nft.GetID()); err != nil {
				return err
			}
			if _, ok := seenTokenIDs[nft.GetID()]; ok {
				return sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "duplicate token id %s in denom %s", nft.GetID(), c.Denom.Id)
			}
			seenTokenIDs[nft.GetID()] = struct{}{}

			if err := ValidateTokenURI(nft.GetURI()); err != nil {
				return err
			}
		}
	}
	return nil
}
