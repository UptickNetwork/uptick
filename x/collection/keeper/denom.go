package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/x/nft"

	"github.com/UptickNetwork/uptick/x/collection/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

// SaveDenom issues a denom according to the given params
func (k Keeper) SaveDenom(ctx sdk.Context, id,
	name,
	schema,
	symbol string,
	creator sdk.AccAddress,
	mintRestricted,
	updateRestricted bool,
	description,
	uri,
	uriHash,
	data string,
) error {
	// Baseline denom-id validation. SaveDenom is the shared entry point for
	// genesis import (ValidateGenesis already runs ValidateDenomID, so this is
	// a redundant backstop there), user issuance (MsgIssueDenom, which runs the
	// stricter ValidateIssueDenomID upstream) and module-derived classes
	// (erc721/cw721 Convert, whose derived id is "uptick-<hex>" — exactly the
	// reserved shape ValidateIssueDenomID rejects but ValidateDenomID allows).
	// ValidateDenomID enforces charset, length [3,128], NUL, comma, and the
	// uptick-/ibc- prefix rules; it does NOT apply the issuance-only
	// bech32/hex-address collision policy, which stays at ValidateIssueDenomID.
	if err := types.ValidateDenomID(id); err != nil {
		return err
	}
	// Oversized schema/data are bounded separately: the schema is embedded in
	// the first ERC721 deployment calldata, so an unbounded schema can
	// permanently DoS conversion; data is arbitrary metadata.
	if len(schema) > types.MaxDenomSchemaLen {
		return sdkerrors.Wrapf(types.ErrInvalidDenom, "schema too long: %d > %d", len(schema), types.MaxDenomSchemaLen)
	}
	if len(data) > types.MaxDenomDataLen {
		return sdkerrors.Wrapf(types.ErrInvalidDenom, "data too long: %d > %d", len(data), types.MaxDenomDataLen)
	}

	denomMetadata := &types.DenomMetadata{
		Creator:          creator.String(),
		Schema:           schema,
		MintRestricted:   mintRestricted,
		UpdateRestricted: updateRestricted,
		Data:             data,
	}
	metadata, err := codectypes.NewAnyWithValue(denomMetadata)
	if err != nil {
		return err
	}
	return k.nk.SaveClass(ctx, nft.Class{
		Id:          id,
		Name:        name,
		Symbol:      symbol,
		Description: description,
		Uri:         uri,
		UriHash:     uriHash,
		Data:        metadata,
	})
}

// TransferDenomOwner transfers the ownership of the given denom to the new owner
func (k Keeper) TransferDenomOwner(
	ctx sdk.Context,
	denomID string,
	srcOwner,
	dstOwner sdk.AccAddress,
) error {
	denom, err := k.GetDenomInfo(ctx, denomID)
	if err != nil {
		return err
	}

	// authorize. Compare on bytes, not on the bech32 string representation:
	// two addresses that decode to the same bytes but use different bech32
	// display variants (e.g. uppercase/lowercase, or future HRP changes) would
	// pass a string comparison inconsistently. Authorize/NFT ownership checks
	// elsewhere in this module use byte equality.
	creatorAcc, err := sdk.AccAddressFromBech32(denom.Creator)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid creator address")
	}
	if !srcOwner.Equals(creatorAcc) {
		return sdkerrors.Wrapf(errortypes.ErrUnauthorized, "%s is not allowed to transfer denom %s", srcOwner.String(), denomID)
	}

	denomMetadata := &types.DenomMetadata{
		Creator:          dstOwner.String(),
		Schema:           denom.Schema,
		MintRestricted:   denom.MintRestricted,
		UpdateRestricted: denom.UpdateRestricted,
		Data:             denom.Data,
	}
	data, err := codectypes.NewAnyWithValue(denomMetadata)
	if err != nil {
		return err
	}
	class := nft.Class{
		Id:     denom.Id,
		Name:   denom.Name,
		Symbol: denom.Symbol,
		Data:   data,

		Description: denom.Description,
		Uri:         denom.Uri,
		UriHash:     denom.UriHash,
	}

	return k.nk.UpdateClass(ctx, class)
}

// GetDenomInfo return the denom information
//
// A legacy / migrated class may carry nil Data; return zero-value metadata
// instead of an error so callers like ExportGenesis and the query endpoints
// never abort on a single corrupt record.
func (k Keeper) GetDenomInfo(ctx sdk.Context, denomID string) (*types.Denom, error) {
	class, has := k.nk.GetClass(ctx, denomID)
	if !has {
		return nil, sdkerrors.Wrapf(types.ErrInvalidDenom, "denom ID %s not exists", denomID)
	}

	// A nil Data means the on-chain class was created before the metadata
	// wrapper existed (pre-migration snapshot, ICS-721 voucher, etc.). Fall
	// back to zero-value DenomMetadata rather than failing the whole call --
	// this matches GetNFT/GetNFTs behavior.
	var denomMetadata types.DenomMetadata
	if class.Data != nil {
		if err := k.cdc.Unmarshal(class.Data.GetValue(), &denomMetadata); err != nil {
			return nil, err
		}
	}
	return &types.Denom{
		Id:               class.Id,
		Name:             class.Name,
		Schema:           denomMetadata.Schema,
		Creator:          denomMetadata.Creator,
		Symbol:           class.Symbol,
		MintRestricted:   denomMetadata.MintRestricted,
		UpdateRestricted: denomMetadata.UpdateRestricted,
		Description:      class.Description,
		Uri:              class.Uri,
		UriHash:          class.UriHash,
		Data:             denomMetadata.Data,
	}, nil
}

// HasDenom determine whether denom exists
func (k Keeper) HasDenom(ctx sdk.Context, denomID string) bool {
	return k.nk.HasClass(ctx, denomID)
}
