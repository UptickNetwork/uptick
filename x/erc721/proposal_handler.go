package erc721

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/UptickNetwork/uptick/x/erc721/keeper"
	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// NewErc721ProposalHandler creates a governance handler to manage new proposal types.
func NewErc721ProposalHandler(k *keeper.Keeper) gov.Handler {
	return func(ctx sdk.Context, content gov.Content) error {
		switch c := content.(type) {
		case *types.RegisterNFTProposal:
			return handleRegisterNFTProposal(ctx, k, c)
		case *types.RegisterERC721Proposal:
			return handleRegisterERC721Proposal(ctx, k, c)
		case *types.ToggleTokenConversionProposal:
			return handleToggleConversionProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized %s proposal content type: %T", types.ModuleName, c)
		}
	}
}

func handleRegisterNFTProposal(ctx sdk.Context, k *keeper.Keeper, p *types.RegisterNFTProposal) error {
	//pair, err := k.RegisterNFT(ctx, p.Class)
	//if err != nil {
	//	return err
	//}
	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypeRegisterNFT,
	//		sdk.NewAttribute(types.AttributeKeyNFTClass, pair.ClassId),
	//		sdk.NewAttribute(types.AttributeKeyERC721Token, pair.Erc721Address),
	//	),
	//)

	return nil
}

func handleRegisterERC721Proposal(ctx sdk.Context, k *keeper.Keeper, p *types.RegisterERC721Proposal) error {

	//pair, err := k.RegisterERC721(ctx, common.HexToAddress(p.Erc721Address))
	//if err != nil {
	//	return err
	//}
	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypeRegisterERC721,
	//		sdk.NewAttribute(types.AttributeKeyNFTClass, pair.ClassId),
	//		sdk.NewAttribute(types.AttributeKeyERC721Token, pair.Erc721Address),
	//	),
	//)

	return nil
}

func handleToggleConversionProposal(ctx sdk.Context, k *keeper.Keeper, p *types.ToggleTokenConversionProposal) error {
	pair, err := k.ToggleConversion(ctx, p.Token)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeToggleTokenConversion,
			sdk.NewAttribute(types.AttributeKeyNFTClass, pair.ClassId),
			sdk.NewAttribute(types.AttributeKeyERC721Token, pair.Erc721Address),
		),
	)

	return nil
}
