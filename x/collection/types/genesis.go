package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
//
// The rules here must accept exactly what InitGenesis can import: a genesis
// that passes validation but makes InitGenesis panic is worse than one that is
// rejected up front, because `uptickd validate-genesis` would wave it through
// and the node would then fail to start from its own backup.
func ValidateGenesis(data GenesisState) error {
	seenDenomIDs := make(map[string]struct{}, len(data.Collections))
	for _, c := range data.Collections {
		if c.Denom.Id == "" {
			return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "collection denom id is empty")
		}
		if err := ValidateDenomID(c.Denom.Id); err != nil {
			return err
		}
		// Duplicate denom ids would make InitGenesis write the same class
		// twice, with the second write silently winning. The NFT side already
		// rejects duplicate token ids within a class; the class side must not
		// be laxer.
		if _, ok := seenDenomIDs[c.Denom.Id]; ok {
			return sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "duplicate denom id %s", c.Denom.Id)
		}
		seenDenomIDs[c.Denom.Id] = struct{}{}

		// An EMPTY creator is legitimate and must stay valid: ICS-721 voucher
		// classes (created by nft-transfer directly in the underlying nft
		// store) and pre-metadata classes have no collection-level issuer, and
		// InitGenesis imports them with an empty creator. A NON-empty creator
		// that does not decode is genuine corruption and must be rejected --
		// InitGenesis panics on it.
		if c.Denom.Creator != "" {
			if _, err := sdk.AccAddressFromBech32(c.Denom.Creator); err != nil {
				return sdkerrors.Wrapf(errortypes.ErrInvalidAddress,
					"invalid collection creator %q: %s", c.Denom.Creator, err)
			}
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
