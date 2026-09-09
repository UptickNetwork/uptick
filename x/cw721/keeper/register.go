package keeper

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// RegisterNFT deploys an cw721 contract and creates the token pair for the existing cosmos coin
func (k Keeper) RegisterNFT(ctx sdk.Context, msg *types.MsgConvertNFT) (*types.TokenPair, error) {

	// Canonicalize the contract address before any key is derived from it so
	// case aliases of the same bech32 address cannot create duplicate pairs.
	cw721Addr, err := NormalizeCW721Address(msg.ContractAddress)
	if err != nil {
		return nil, err
	}
	msg.ContractAddress = cw721Addr

	// Check if class is already registered
	if k.IsClassRegistered(ctx, msg.ClassId) {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairAlreadyExists, "class ID already registered: %s", msg.ClassId,
		)
	}
	if k.IsCW721Registered(ctx, msg.ContractAddress) {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairAlreadyExists, "token CW721 contract already registered: %s", msg.ContractAddress,
		)
	}

	// Validate the CW721 contract address. CW721 contracts are CosmWasm
	// contracts, so the address must be a valid bech32 account address (not a
	// 0x hex address); it is passed directly to the wasm querier/executor below.
	// Validation already happened in NormalizeCW721Address above.

	if len(strings.TrimSpace(msg.ClassId)) == 0 {
		return nil, sdkerrors.Wrapf(
			types.ErrInternalTokenPair, "class ID must not be empty: %s", msg.ClassId,
		)
	}

	pair := types.NewTokenPair(msg.ContractAddress, msg.ClassId)
	k.Logger(ctx).Info("RegisterNFT ", "ClassId", pair.ClassId, "Cw721Address", pair.Cw721Address)
	if err := k.SetTokenPair(ctx, pair); err != nil {
		return nil, err
	}
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetCW721Map(ctx, pair.Cw721Address, pair.GetID())

	return &pair, nil
}

// RegisterCW721 creates a Cosmos coin and registers the token pair between the nft and the CW721
func (k Keeper) RegisterCW721(ctx sdk.Context, msg *types.MsgConvertCW721) (*types.TokenPair, error) {

	// Canonicalize the contract address before the duplicate check, class ID
	// derivation and pair creation so case aliases of the same bech32 address
	// cannot create duplicate pairs.
	cw721Addr, err := NormalizeCW721Address(msg.ContractAddress)
	if err != nil {
		return nil, err
	}
	msg.ContractAddress = cw721Addr

	// Check if CW721 is already registered
	if k.IsCW721Registered(ctx, msg.ContractAddress) {
		return nil, sdkerrors.Wrapf(types.ErrTokenPairAlreadyExists,
			"token CW721 contract already registered: %s", msg.ContractAddress)
	}

	derivedClassID := types.CreateClassIDFromContractAddress(msg.ContractAddress)
	trimmedClassID := strings.TrimSpace(msg.ClassId)
	if trimmedClassID == "" {
		msg.ClassId = derivedClassID
	} else if trimmedClassID != derivedClassID {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound,
			"class ID %s does not match CW721 contract derived class ID %s",
			trimmedClassID,
			derivedClassID,
		)
	}

	err = k.CreateNFTClass(ctx, msg)
	if err != nil {
		return nil, sdkerrors.Wrap(err,
			"failed to create wrapped coin denom metadata for CW721")
	}

	pair := types.NewTokenPair(msg.ContractAddress, msg.ClassId)
	if err := k.SetTokenPair(ctx, pair); err != nil {
		return nil, err
	}
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetCW721Map(ctx, pair.Cw721Address, pair.GetID())

	return &pair, nil
}

// CreateNFTClass generates the metadata to represent the CW721 token .
func (k Keeper) CreateNFTClass(ctx sdk.Context, msg *types.MsgConvertCW721) error {

	cw721Data, err := k.QueryCW721(ctx, msg.ContractAddress)
	if err != nil {
		return err
	}

	// QueryClassEnhance is not implemented for CosmWasm contracts yet;
	// keep empty enhance fields rather than treating success as a wipe.
	classEnhance := types.ClassEnhance{}

	// A class that is already registered as a CW721 pair is a plain
	// user-facing conflict: the caller asked for something that exists.
	// ErrTokenPairAlreadyExists (code 7) is the correct signal here;
	// ErrInternalTokenPair is reserved for broken module state.
	if k.IsClassRegistered(ctx, msg.ClassId) {
		return sdkerrors.Wrapf(types.ErrTokenPairAlreadyExists, "nft class already registered: %s", msg.ClassId)
	}

	// A native collection denom that exists WITHOUT a CW721 pair registration
	// means the two namespaces have drifted apart (e.g. the denom was issued
	// natively, or a pair was deleted without cleaning up the denom). That is
	// an inconsistent-state condition, not a normal "already exists" conflict,
	// so it surfaces as ErrInternalTokenPair (code 5).
	_, err = k.nftKeeper.GetDenomInfo(ctx, msg.ClassId)
	if err == nil {
		return sdkerrors.Wrapf(types.ErrInternalTokenPair, "native NFT class %s already exists but is not registered as a cw721 pair", msg.ClassId)
	}

	err = k.nftKeeper.SaveDenom(ctx, msg.ClassId, cw721Data.Name, classEnhance.Schema,
		cw721Data.Symbol, types.AccModuleAddress, classEnhance.MintRestricted, classEnhance.UpdateRestricted,
		classEnhance.Description, classEnhance.Uri, classEnhance.UriHash, classEnhance.Data)
	if err != nil {
		return err
	}

	return nil
}
