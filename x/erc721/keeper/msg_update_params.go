package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// UpdateParams updates erc721 params. Authority must be the gov module account.
func (k Keeper) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != authtypes.NewModuleAddress(govtypes.ModuleName).String() {
		return nil, sdkerrors.Wrapf(govtypes.ErrInvalidSigner, "expected %s, got %s", authtypes.NewModuleAddress(govtypes.ModuleName).String(), msg.Authority)
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)
	return &types.MsgUpdateParamsResponse{}, nil
}
