package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/x/nft"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/UptickNetwork/uptick/x/collection/exported"
	"github.com/UptickNetwork/uptick/x/collection/types"
)

// SaveNFT mints an NFT and manages the NFT's existence within Collections and Owners.
//
// Empty inputs are rejected up front: they would write a corrupt state entry
// (empty key / zero address) and break downstream iteration.
func (k Keeper) SaveNFT(ctx sdk.Context, denomID,
	tokenID,
	tokenNm,
	tokenURI,
	tokenUriHash,
	tokenData string,
	receiver sdk.AccAddress,
) error {
	if denomID == "" {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "denom ID cannot be empty")
	}
	if tokenID == "" {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "token ID cannot be empty")
	}
	if receiver == nil || receiver.Empty() {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "receiver cannot be empty")
	}
	nftMetadata := &types.NFTMetadata{
		Name: tokenNm,
		Data: tokenData,
	}
	data, err := codectypes.NewAnyWithValue(nftMetadata)
	if err != nil {
		return err
	}
	return k.nk.Mint(ctx, nft.NFT{
		ClassId: denomID,
		Id:      tokenID,
		Uri:     tokenURI,
		UriHash: tokenUriHash,
		Data:    data,
	}, receiver)
}

// UpdateNFT updates an already existing NFT
func (k Keeper) UpdateNFT(ctx sdk.Context, denomID,
	tokenID,
	tokenNm,
	tokenURI,
	tokenURIHash,
	tokenData string,
	owner sdk.AccAddress,
) error {
	denom, err := k.GetDenomInfo(ctx, denomID)
	if err != nil {
		return err
	}

	if denom.UpdateRestricted {
		// if true , nobody can update the NFT under this denom
		return sdkerrors.Wrapf(errortypes.ErrUnauthorized, "nobody can update the NFT under this denom %s", denomID)
	}

	// Existence must be checked before Authorize: otherwise a missing NFT
	// would surface as ErrUnauthorized (because GetOwner returns empty for a
	// non-existent token) instead of ErrUnknownNFT. Matches the order used
	// by TransferOwnership.
	token, exist := k.nk.GetNFT(ctx, denomID, tokenID)
	if !exist {
		return sdkerrors.Wrapf(types.ErrUnknownNFT, "nft not exist: %s-%s", denomID, tokenID)
	}

	// just the owner of NFT can edit
	if err := k.Authorize(ctx, denomID, tokenID, owner); err != nil {
		return err
	}

	if !types.Modified(tokenURI) &&
		!types.Modified(tokenURIHash) &&
		!types.Modified(tokenNm) &&
		!types.Modified(tokenData) {
		return nil
	}

	// token was fetched above; reuse it instead of reading again.

	token.Uri = types.Modify(token.Uri, tokenURI)
	token.UriHash = types.Modify(token.UriHash, tokenURIHash)
	if types.Modified(tokenNm) || types.Modified(tokenData) {
		// A nil-Data record is metadata-less, not invalid: build from an empty
		// value so owners can attach metadata to such NFTs (same policy as
		// TransferOwnership).
		var nftMetadata types.NFTMetadata
		if token.Data != nil {
			nftMetadata, err = types.UnmarshalNFTMetadata(k.cdc, token.Data.GetValue())
			if err != nil {
				return err
			}
		}

		nftMetadata.Name = types.Modify(nftMetadata.Name, tokenNm)
		nftMetadata.Data = types.Modify(nftMetadata.Data, tokenData)
		data, err := codectypes.NewAnyWithValue(&nftMetadata)
		if err != nil {
			return err
		}
		token.Data = data
	}
	return k.nk.Update(ctx, token)
}

// TransferOwnership transfers the ownership of the given NFT to the new owner
func (k Keeper) TransferOwnership(ctx sdk.Context, denomID,
	tokenID,
	tokenNm,
	tokenURI,
	tokenURIHash,
	tokenData string,
	srcOwner,
	dstOwner sdk.AccAddress,
) error {
	token, exist := k.nk.GetNFT(ctx, denomID, tokenID)
	if !exist {
		return sdkerrors.Wrapf(types.ErrInvalidTokenID, "nft ID %s not exists", tokenID)
	}

	if err := k.Authorize(ctx, denomID, tokenID, srcOwner); err != nil {
		return err
	}

	denom, err := k.GetDenomInfo(ctx, denomID)
	if err != nil {
		return err
	}

	// A field only counts as a real change if the incoming value actually
	// differs from the stored one. Copying the current URI/URIHash/metadata on a
	// pure transfer must not be treated as an update, otherwise an
	// UpdateRestricted denom would block legitimate ownership transfers.
	tokenChanged := (types.Modified(tokenURI) && tokenURI != token.Uri) ||
		(types.Modified(tokenURIHash) && tokenURIHash != token.UriHash)

	tokenMetadataChanged := false
	if types.Modified(tokenNm) || types.Modified(tokenData) {
		if token.Data == nil {
			tokenMetadataChanged = true
		} else {
			nftMetadata, err := types.UnmarshalNFTMetadata(k.cdc, token.Data.GetValue())
			if err != nil {
				return err
			}
			tokenMetadataChanged = (types.Modified(tokenNm) && tokenNm != nftMetadata.Name) ||
				(types.Modified(tokenData) && tokenData != nftMetadata.Data)
		}
	}

	if denom.UpdateRestricted && (tokenChanged || tokenMetadataChanged) {
		return sdkerrors.Wrapf(errortypes.ErrUnauthorized, "it is restricted to update NFT under this denom %s", denom.Id)
	}

	if !tokenChanged && !tokenMetadataChanged {
		return k.nk.Transfer(ctx, denomID, tokenID, dstOwner)
	}

	token.Uri = types.Modify(token.Uri, tokenURI)
	token.UriHash = types.Modify(token.UriHash, tokenURIHash)
	if tokenMetadataChanged {
		// A nil Data record is metadata-less, not invalid: build the metadata
		// from an empty value so users can attach metadata to such NFTs (and
		// transfers can repair dirty records) instead of failing with the
		// misleading "has no metadata" error.
		var nftMetadata types.NFTMetadata
		if token.Data != nil {
			nftMetadata, err = types.UnmarshalNFTMetadata(k.cdc, token.Data.GetValue())
			if err != nil {
				return err
			}
		}

		nftMetadata.Name = types.Modify(nftMetadata.Name, tokenNm)
		nftMetadata.Data = types.Modify(nftMetadata.Data, tokenData)
		data, err := codectypes.NewAnyWithValue(&nftMetadata)
		if err != nil {
			return err
		}
		token.Data = data
	}

	if err := k.nk.Update(ctx, token); err != nil {
		return err
	}
	return k.nk.Transfer(ctx, denomID, tokenID, dstOwner)
}

// RemoveNFT deletes a specified NFT.
//
// A token paired with an ERC721/CW721 contract token is refused: the conversion
// escrowed the contract-side half in a module account, and only this module
// knows the binding, so burning the native half would delete the last on-chain
// record of the escrowed asset. The state is what x/erc721 itself describes as
// "undiscoverable and unrecoverable". The user must un-wrap first.
func (k Keeper) RemoveNFT(ctx sdk.Context, denomID, tokenID string, owner sdk.AccAddress) error {
	// Existence is checked before Authorize on purpose: GetOwner reports an
	// empty address for a missing token, so burning a non-existent NFT used to
	// surface as ErrUnauthorized against a blank address instead of
	// ErrUnknownNFT. UpdateNFT and TransferOwnership order the two checks the
	// same way.
	if _, exist := k.nk.GetNFT(ctx, denomID, tokenID); !exist {
		return sdkerrors.Wrapf(types.ErrUnknownNFT, "not found NFT %s from collection %s", tokenID, denomID)
	}

	if err := k.Authorize(ctx, denomID, tokenID, owner); err != nil {
		return err
	}

	if k.IsConvertedNFT(ctx, denomID, tokenID) {
		return sdkerrors.Wrapf(types.ErrNFTBoundToContract,
			"nft %s/%s is paired with a contract token; convert it back (un-wrap) before burning it",
			denomID, tokenID)
	}

	return k.nk.Burn(ctx, denomID, tokenID)
}

// GetNFT gets the specified NFT
func (k Keeper) GetNFT(ctx sdk.Context, denomID, tokenID string) (nft exported.NFT, err error) {
	token, exist := k.nk.GetNFT(ctx, denomID, tokenID)
	if !exist {
		return nil, sdkerrors.Wrapf(types.ErrUnknownNFT, "not found NFT %s from collection %s", tokenID, denomID)
	}

	// A legacy / migrated NFT may carry nil Data; degrade to empty metadata so
	// the single-NFT query matches GetNFTs / ExportGenesis behavior instead of
	// erroring on the same record.
	//
	// Consumers include STATE-MUTATING paths, not just queries: x/cw721 and
	// x/erc721 read this during conversion and feed Name/Data straight into
	// TransferNFT, which rewrites the record. So an unreadable Data field is
	// silently replaced with empty metadata on that write — the log level below
	// therefore has to be observable in production, not Debug.
	var nftMetadata types.NFTMetadata
	if token.Data != nil {
		if err := k.cdc.Unmarshal(token.Data.GetValue(), &nftMetadata); err != nil {
			k.Logger(ctx).Warn("unreadable NFT data, degrading to empty metadata",
				"denom", denomID, "token", tokenID, "err", err)
			nftMetadata = types.NFTMetadata{}
		}
	}

	owner := k.nk.GetOwner(ctx, denomID, tokenID)
	return types.BaseNFT{
		Id:      token.GetId(),
		Name:    nftMetadata.Name,
		URI:     token.GetUri(),
		Data:    nftMetadata.Data,
		Owner:   owner.String(),
		UriHash: token.UriHash,
	}, nil
}

// GetNFTs returns all NFTs by the specified denom ID
//
// An NFT with corrupt Data is logged and downgraded to empty metadata instead
// of aborting the whole query (matches GetCollections / GetDenomInfo).
func (k Keeper) GetNFTs(ctx sdk.Context, denom string) (nfts []exported.NFT, err error) {
	tokens := k.nk.GetNFTsOfClass(ctx, denom)
	for _, token := range tokens {
		var nftMetadata types.NFTMetadata
		if token.Data != nil {
			if uerr := k.cdc.Unmarshal(token.Data.GetValue(), &nftMetadata); uerr != nil {
				ctx.Logger().Debug("GetNFTs: skipping NFT with undecodable metadata",
					"denom", denom, "token_id", token.GetId(), "err", uerr.Error())
				nftMetadata = types.NFTMetadata{}
			}
		}
		nfts = append(nfts, types.BaseNFT{
			Id:      token.GetId(),
			Name:    nftMetadata.Name,
			URI:     token.GetUri(),
			UriHash: token.GetUriHash(),
			Data:    nftMetadata.Data,
			Owner:   k.nk.GetOwner(ctx, denom, token.GetId()).String(),
		})
	}
	return nfts, nil
}

// Authorize checks if the sender is the owner of the given NFT
// Return the NFT if true, an error otherwise
func (k Keeper) Authorize(ctx sdk.Context, denomID, tokenID string, owner sdk.AccAddress) error {
	if !owner.Equals(k.nk.GetOwner(ctx, denomID, tokenID)) {
		return sdkerrors.Wrap(types.ErrUnauthorized, owner.String())
	}
	return nil
}

// HasNFT checks if the specified NFT exists
func (k Keeper) HasNFT(ctx sdk.Context, denomID, tokenID string) bool {
	return k.nk.HasNFT(ctx, denomID, tokenID)
}
