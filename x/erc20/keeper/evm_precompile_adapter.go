package keeper

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/cosmos/evm/precompiles/common"
	evmerc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/ethereum/go-ethereum/common"
)

// EVMPrecompileERC20Keeper adapts Uptick's custom erc20 keeper to the
// cosmos/evm precompiles/common.ERC20Keeper interface required by the EVM
// precompiles (bank/ics20). Uptick owns its own erc20 module with its own
// token-pair types, so the cosmos/evm erc20 precompile is not functionally
// wired; these methods return safe defaults.
type EVMPrecompileERC20Keeper struct {
	keeper Keeper
}

// NewEVMPrecompileERC20Keeper creates a cosmos/evm-compatible erc20 keeper
// wrapper around Uptick's erc20 keeper.
func NewEVMPrecompileERC20Keeper(k Keeper) *EVMPrecompileERC20Keeper {
	return &EVMPrecompileERC20Keeper{keeper: k}
}

var _ cmn.ERC20Keeper = (*EVMPrecompileERC20Keeper)(nil)

// GetCoinAddress returns the ERC20 contract address mapped to a coin denom.
func (w *EVMPrecompileERC20Keeper) GetCoinAddress(ctx sdk.Context, denom string) (common.Address, error) {
	return common.Address{}, nil
}

// GetERC20Map returns the token pair id stored for the given ERC20 contract.
func (w *EVMPrecompileERC20Keeper) GetERC20Map(ctx sdk.Context, erc20 common.Address) []byte {
	return nil
}

// GetTokenPair returns the token pair with the given id.
func (w *EVMPrecompileERC20Keeper) GetTokenPair(ctx sdk.Context, id []byte) (evmerc20types.TokenPair, bool) {
	return evmerc20types.TokenPair{}, false
}

// IsERC20Enabled returns whether the ERC20 module is enabled.
func (w *EVMPrecompileERC20Keeper) IsERC20Enabled(ctx sdk.Context) bool {
	return false
}

// GetTokenPairID returns the token pair id for the given token (contract or denom).
func (w *EVMPrecompileERC20Keeper) GetTokenPairID(ctx sdk.Context, token string) []byte {
	return nil
}

// ConvertERC20IntoCoinsForNativeToken converts an ERC20 amount into native coins.
func (w *EVMPrecompileERC20Keeper) ConvertERC20IntoCoinsForNativeToken(
	ctx sdk.Context,
	stateDB *statedb.StateDB,
	contract common.Address,
	amount math.Int,
	receiver sdk.AccAddress,
	sender common.Address,
	commit bool,
	callFromPrecompile bool,
) (*evmerc20types.MsgConvertERC20Response, error) {
	return nil, fmt.Errorf("erc20 conversion not supported by Uptick's custom erc20 module")
}
