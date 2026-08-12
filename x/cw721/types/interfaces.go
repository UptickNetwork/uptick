package types

import (
	context "context"

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

// Compile-time assertion that ibcnfttransferkeeper.Keeper implements IBCNFTTransferKeeper
var _ IBCNFTTransferKeeper = ibcnfttransferkeeper.Keeper{}
