package types

import (
	context "context"
	"github.com/cosmos/cosmos-sdk/x/nft"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/evmos/ethermint/x/evm/statedb"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// AccountKeeper defines the expected interface needed to retrieve account info.
type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
	GetSequence(sdk.Context, sdk.AccAddress) (uint64, error)
}

// NFTKeeper defines the expected interface needed to retrieve account balances.
type NFTKeeper interface {
	SaveClass(ctx sdk.Context, class nft.Class) error

	HasClass(ctx sdk.Context, classID string) bool
	GetClass(ctx sdk.Context, classID string) (nft.Class, bool)

	Mint(ctx sdk.Context, token nft.NFT, receiver sdk.AccAddress) error
	Burn(ctx sdk.Context, classID string, nftID string) error
	Transfer(ctx sdk.Context, classID string, nftID string, receiver sdk.AccAddress) error

	GetNFT(ctx sdk.Context, denomID string, tokenID string) (nft.NFT, error)

	HasNFT(ctx sdk.Context, classID, id string) bool
	GetOwner(ctx sdk.Context, classID string, nftID string) sdk.AccAddress
}

// EVMKeeper defines the expected EVM keeper interface used on erc721
type EVMKeeper interface {
	GetParams(ctx sdk.Context) evmtypes.Params
	GetAccountWithoutBalance(ctx sdk.Context, addr common.Address) *statedb.Account
	EstimateGas(c context.Context, req *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error)
	ApplyMessage(ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool) (*evmtypes.MsgEthereumTxResponse, error)
}
