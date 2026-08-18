package keeper

import (
	"context"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

type wasmKeeperAdapter struct {
	k *wasmkeeper.Keeper
}

// WrapWasmKeeper adapts wasmd's keeper to the cw721 WasmKeeper interface.
func WrapWasmKeeper(k *wasmkeeper.Keeper) types.WasmKeeper {
	return wasmKeeperAdapter{k: k}
}

func (a wasmKeeperAdapter) StoreCode(ctx context.Context, msg *wasmtypes.MsgStoreCode) (*wasmtypes.MsgStoreCodeResponse, error) {
	return wasmkeeper.NewMsgServerImpl(a.k).StoreCode(ctx, msg)
}

func (a wasmKeeperAdapter) InstantiateContract(ctx context.Context, msg *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error) {
	return wasmkeeper.NewMsgServerImpl(a.k).InstantiateContract(ctx, msg)
}

func (a wasmKeeperAdapter) ExecuteContract(ctx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
	return wasmkeeper.NewMsgServerImpl(a.k).ExecuteContract(ctx, msg)
}

func (a wasmKeeperAdapter) SmartContractState(ctx context.Context, req *wasmtypes.QuerySmartContractStateRequest) (*wasmtypes.QuerySmartContractStateResponse, error) {
	return wasmkeeper.Querier(a.k).SmartContractState(ctx, req)
}

var _ types.WasmKeeper = wasmKeeperAdapter{}
