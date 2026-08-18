package types

import (
	context "context"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	ibcnfttransferkeeper "github.com/bianjieai/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the account keeper interface for the cw721 module.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// IBCNFTTransferKeeper defines the interface for IBC NFT transfer operations.
type IBCNFTTransferKeeper interface {
	Transfer(goCtx context.Context, msg *ibcnfttransfertypes.MsgTransfer) (*ibcnfttransfertypes.MsgTransferResponse, error)
}

// WasmKeeper is the CosmWasm surface used by cw721 convert/refund.
type WasmKeeper interface {
	StoreCode(ctx context.Context, msg *wasmtypes.MsgStoreCode) (*wasmtypes.MsgStoreCodeResponse, error)
	InstantiateContract(ctx context.Context, msg *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error)
	ExecuteContract(ctx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error)
	SmartContractState(ctx context.Context, req *wasmtypes.QuerySmartContractStateRequest) (*wasmtypes.QuerySmartContractStateResponse, error)
}

// Compile-time assertion that ibcnfttransferkeeper.Keeper implements IBCNFTTransferKeeper
var _ IBCNFTTransferKeeper = ibcnfttransferkeeper.Keeper{}
