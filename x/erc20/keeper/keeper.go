package keeper

import (
	"fmt"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/UptickNetwork/uptick/x/erc20/types"
	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"

	evmerc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// Keeper of this module maintains collections of erc20.
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramstore paramtypes.Subspace

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	evmKeeper     types.EVMKeeper
	ics4Wrapper   porttypes.ICS4Wrapper
	ibcKeeper     ibctransferkeeper.Keeper
	ibcKeeperSet  bool
}

// NewKeeper creates new instances of the erc20 Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ek types.EVMKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		paramstore:    ps,
		accountKeeper: ak,
		bankKeeper:    bk,
		evmKeeper:     ek,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetICS4Wrapper sets the ICS4 wrapper to the keeper.
// It panics if already set
func (k *Keeper) SetICS4Wrapper(ics4Wrapper porttypes.ICS4Wrapper) {
	if k.ics4Wrapper != nil {
		panic("ICS4 wrapper already set")
	}

	k.ics4Wrapper = ics4Wrapper
}

// SetIBCKeeper sets the ICS4 wrapper to the keeper.
// It panics if already set
func (k *Keeper) SetIBCKeeper(ibcKeeper ibctransferkeeper.Keeper) {
	// prevent accidental overwrite
	if k.ibcKeeperSet {
		panic("IBC keeper already set")
	}
	k.ibcKeeperSet = true
	k.ibcKeeper = ibcKeeper
}

// SetEVMKeeper sets the EVM keeper to the erc20 keeper.
// Used to break the circular dependency between the EVM and erc20 keepers.
func (k *Keeper) SetEVMKeeper(ek types.EVMKeeper) {
	k.evmKeeper = ek
}

// GetERC20PrecompileInstance implements the cosmos/evm Erc20Keeper interface.
// It returns the ERC20 precompile contract for the given token pair address.
// Since Uptick uses its own erc20 implementation (not cosmos/evm's), this is a stub
// that returns false (not found) — precompile integration is handled separately.
func (k Keeper) GetERC20PrecompileInstance(ctx sdk.Context, address common.Address) (vm.PrecompiledContract, bool, error) {
	// TODO: implement proper precompile instance lookup
	// For now, return not found — this allows the EVM keeper to compile
	return nil, false, nil
}

// The following methods implement the cosmos/evm precompiles/common.ERC20Keeper
// interface so that Uptick's erc20 keeper can be wired into the EVM precompiles
// (bank/ics20). Uptick uses its own erc20 module, so these return safe defaults.

// GetCoinAddress returns the ERC20 contract address mapped to a coin denom.
func (k Keeper) GetCoinAddress(ctx sdk.Context, denom string) (common.Address, error) {
	return common.Address{}, nil
}

// IsERC20Enabled returns whether the ERC20 module is enabled.
func (k Keeper) IsERC20Enabled(ctx sdk.Context) bool {
	return false
}

// ConvertERC20IntoCoinsForNativeToken converts an ERC20 amount into native coins.
func (k Keeper) ConvertERC20IntoCoinsForNativeToken(
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
