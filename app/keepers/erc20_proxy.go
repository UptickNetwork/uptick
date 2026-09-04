package keepers

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmoserc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// erc20KeeperProxy satisfies evmtypes.Erc20Keeper so the EVM keeper can be
// constructed before the ERC20 keeper exists. The inner pointer is set after
// NewKeeper returns.
type erc20KeeperProxy struct {
	keeper *cosmoserc20keeper.Keeper
}

var _ evmtypes.Erc20Keeper = (*erc20KeeperProxy)(nil)

// errErc20ProxyNotWired is returned when the proxy is consulted before the
// ERC20 keeper has been wired in. Audit P3-2: this used to degrade silently to
// (nil, false, nil), which reads as "no precompile at this address" and hides
// wiring bugs — precompile calls would silently miss ERC20 overrides.
var errErc20ProxyNotWired = errors.New(
	"erc20 keeper proxy not wired: GetERC20PrecompileInstance called before the ERC20 keeper was set")

func (p *erc20KeeperProxy) GetERC20PrecompileInstance(
	ctx sdk.Context,
	address common.Address,
) (vm.PrecompiledContract, bool, error) {
	if p == nil || p.keeper == nil {
		return nil, false, errErc20ProxyNotWired
	}
	return p.keeper.GetERC20PrecompileInstance(ctx, address)
}
