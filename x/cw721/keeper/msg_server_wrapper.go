package keeper

import (
	"context"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) ConvertNFT(goCtx context.Context, msg *types.MsgConvertNFT) (*types.MsgConvertNFTResponse, error) {
	return k.Keeper.ConvertNFT(goCtx, msg)
}

func (k msgServer) ConvertCW721(goCtx context.Context, msg *types.MsgConvertCW721) (*types.MsgConvertCW721Response, error) {
	return k.Keeper.ConvertCW721(goCtx, msg)
}

func (k msgServer) TransferCW721(goCtx context.Context, msg *types.MsgTransferCW721) (*types.MsgTransferCW721Response, error) {
	return k.Keeper.TransferCW721(goCtx, msg)
}
