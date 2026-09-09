package keeper

import (
	"errors"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// RegisterNFT deploys an erc721 contract and creates the token pair for the existing cosmos coin
func (k Keeper) RegisterNFT(ctx sdk.Context, msg *types.MsgConvertNFT) (*types.TokenPair, error) {

	// Check if class is already registered
	if k.IsClassRegistered(ctx, msg.ClassId) {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairAlreadyExists, "class ID already registered: %s", msg.ClassId,
		)
	}
	// Also reject a second class trying to bind to the same EVM contract: a
	// pair's contract identity is fixed (the class↔contract mapping in
	// ConvertNFT relies on it), so two pairs for one contract would silently
	// overwrite each other in the EVM map and orphan the first class's pair.
	contract := common.HexToAddress(msg.EvmContractAddress)
	if k.IsERC721Registered(ctx, contract) {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairAlreadyExists,
			"contract %s already bound to a registered pair", contract.String(),
		)
	}

	pair := types.NewTokenPair(contract, msg.ClassId)
	if err := k.SetTokenPair(ctx, pair); err != nil {
		return nil, err
	}
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, contract, pair.GetID())

	return &pair, nil
}

// RegisterERC721 creates a Cosmos coin and registers the token pair between the nft and the ERC721
func (k Keeper) RegisterERC721(ctx sdk.Context, msg *types.MsgConvertERC721) (*types.TokenPair, error) {

	// Check if ERC721 is already registered
	contract := common.HexToAddress(msg.EvmContractAddress)
	if k.IsERC721Registered(ctx, contract) {
		return nil, sdkerrors.Wrapf(types.ErrTokenPairAlreadyExists,
			"token ERC721 contract already registered: %s", contract.String())
	}

	derivedClassID := types.CreateClassIDFromContractAddress(msg.EvmContractAddress)
	if strings.TrimSpace(msg.ClassId) == "" {
		msg.ClassId = derivedClassID
	} else if msg.ClassId != derivedClassID {
		return nil, sdkerrors.Wrapf(
			types.ErrTokenPairNotFound,
			"class ID %s does not match ERC721 contract derived class ID %s",
			msg.ClassId,
			derivedClassID,
		)
	}

	err := k.CreateNFTClass(ctx, msg)
	if err != nil {

		return nil, sdkerrors.Wrap(err,
			"failed to create wrapped coin denom metadata for ERC721")
	}

	pair := types.NewTokenPair(contract, msg.ClassId)
	if err := k.SetTokenPair(ctx, pair); err != nil {
		return nil, err
	}
	k.SetClassMap(ctx, pair.ClassId, pair.GetID())
	k.SetERC721Map(ctx, common.HexToAddress(pair.Erc721Address), pair.GetID())

	return &pair, nil
}

// CreateNFTClass generates the metadata to represent the ERC721 token .
func (k Keeper) CreateNFTClass(ctx sdk.Context, msg *types.MsgConvertERC721) error {

	contract := common.HexToAddress(msg.EvmContractAddress)
	erc721Data, err := k.QueryERC721(ctx, contract)
	if err != nil {
		return err
	}

	// Pure state checks run BEFORE the enhance-metadata query (round 11, F-2):
	// both are deterministic reads of module state, so "already registered"
	// must return ErrTokenPairAlreadyExists (code 7) and "native denom without
	// a pair" must return ErrInternalTokenPair (code 5) regardless of how
	// healthy the external contract is. Kept AFTER QueryERC721 and BEFORE
	// QueryClassEnhance — the exact same position x/cw721 uses, so an operator
	// sees identical error codes for identical conditions on both modules.

	// A class that is already registered as an ERC721 pair is a plain
	// user-facing conflict: the caller asked for something that exists.
	// ErrTokenPairAlreadyExists (code 7) is the correct signal here;
	// ErrInternalTokenPair is reserved for broken module state.
	//
	// Kept symmetric with x/cw721 CreateNFTClass (round 10, G-1): the two
	// modules expose the same conceptual conditions to operators, so they must
	// surface identical error codes. This also matches this file's own
	// RegisterNFT entry point above.
	if k.IsClassRegistered(ctx, msg.ClassId) {
		return sdkerrors.Wrapf(types.ErrTokenPairAlreadyExists, "nft class already registered: %s", msg.ClassId)
	}

	// A native collection denom that exists WITHOUT an ERC721 pair
	// registration means the two namespaces have drifted apart (e.g. the denom
	// was issued natively, or a pair was deleted without cleaning up the
	// denom). That is an inconsistent-state condition, not a normal "already
	// exists" conflict, so it surfaces as ErrInternalTokenPair (code 5).
	_, err = k.nftKeeper.GetDenomInfo(ctx, msg.ClassId)
	if err == nil {
		return sdkerrors.Wrapf(types.ErrInternalTokenPair, "native NFT class %s already exists but is not registered as an erc721 pair", msg.ClassId)
	}

	classEnhance, err := k.QueryClassEnhance(ctx, contract)
	if err != nil {
		// The contract DOES expose enhance metadata but its restriction flags
		// could not be decoded. Do NOT fall through to the permissive defaults
		// below: `false` means "unrestricted", so guessing would silently turn
		// off the class-level restriction that x/collection enforces on mint.
		// Reject the registration instead — fail closed, not open.
		if errors.Is(err, types.ErrClassEnhanceRestrictions) {
			return err
		}

		// Any other failure means the contract simply does not expose enhance
		// metadata (many ERC721s implement neither getClassEnhanceInfo nor the
		// surrounding calls). That IS the normal case, so fall back to empty
		// metadata rather than blocking every such conversion.
		classEnhance.Uri = ""
		classEnhance.Data = ""
		classEnhance.Schema = ""
		classEnhance.UriHash = ""
		classEnhance.Description = ""
		classEnhance.UpdateRestricted = false
		classEnhance.MintRestricted = false
	}

	err = k.nftKeeper.SaveDenom(ctx, msg.ClassId, erc721Data.Name, classEnhance.Schema,
		erc721Data.Symbol, types.AccModuleAddress, classEnhance.MintRestricted, classEnhance.UpdateRestricted,
		classEnhance.Description, classEnhance.Uri, classEnhance.UriHash, classEnhance.Data)
	if err != nil {
		return err
	}

	return nil
}
