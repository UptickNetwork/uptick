package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// Hooks wrapper struct for erc20 keeper
type Hooks struct {
	k Keeper
}

var _ evmtypes.EvmHooks = Hooks{}

// Return the wrapper struct
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// TODO: Make sure that if ConvertERC20 is called, that the Hook doesn't trigger
// if it does, delete minting from ConvertErc20

// PostTxProcessing implements EvmHooks.PostTxProcessing
func (h Hooks) PostTxProcessing(
	ctx sdk.Context,
	msg core.Message,
	receipt *ethtypes.Receipt,
) error {
	return nil
}
