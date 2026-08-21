// Package precompiles provides Uptick's EVM static precompile configuration.
//
// Now uses cosmos/evm v0.6.1's ERC20 keeper directly (concrete type) instead of the
// cmn.ERC20Keeper interface, since Uptick has moved to cosmos/evm's x/erc20 module.
package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	bankprecompile "github.com/cosmos/evm/precompiles/bank"
	"github.com/cosmos/evm/precompiles/bech32"
	cmn "github.com/cosmos/evm/precompiles/common"
	distprecompile "github.com/cosmos/evm/precompiles/distribution"
	govprecompile "github.com/cosmos/evm/precompiles/gov"
	ics20precompile "github.com/cosmos/evm/precompiles/ics20"
	"github.com/cosmos/evm/precompiles/p256"
	slashingprecompile "github.com/cosmos/evm/precompiles/slashing"
	stakingprecompile "github.com/cosmos/evm/precompiles/staking"
	erc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	transferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"

	"cosmossdk.io/core/address"

	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	evmaddress "github.com/cosmos/evm/encoding/address"
)

const bech32PrecompileBaseGas = 6_000

// Optionals define some optional params that can be applied to _some_ precompiles.
type Optionals struct {
	AddressCodec       address.Codec // used by gov/staking
	ValidatorAddrCodec address.Codec // used by slashing
	ConsensusAddrCodec address.Codec // used by slashing
}

func defaultOptionals() Optionals {
	return Optionals{
		AddressCodec:       evmaddress.NewEvmCodec(sdktypes.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddrCodec: evmaddress.NewEvmCodec(sdktypes.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddrCodec: evmaddress.NewEvmCodec(sdktypes.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// Option customizes the precompiles' optional params.
type Option func(opts *Optionals)

// WithAddressCodec sets the address codec.
func WithAddressCodec(codec address.Codec) Option {
	return func(opts *Optionals) { opts.AddressCodec = codec }
}

// WithValidatorAddrCodec sets the validator address codec.
func WithValidatorAddrCodec(codec address.Codec) Option {
	return func(opts *Optionals) { opts.ValidatorAddrCodec = codec }
}

// WithConsensusAddrCodec sets the consensus address codec.
func WithConsensusAddrCodec(codec address.Codec) Option {
	return func(opts *Optionals) { opts.ConsensusAddrCodec = codec }
}

// StaticPrecompiles is a map of precompile addresses to contracts.
type StaticPrecompiles map[common.Address]vm.PrecompiledContract

// NewStaticPrecompiles returns an empty static precompiles map.
func NewStaticPrecompiles() StaticPrecompiles {
	return make(StaticPrecompiles)
}

// WithPraguePrecompiles adds the Prague precompiles.
func (s StaticPrecompiles) WithPraguePrecompiles() StaticPrecompiles {
	s[common.HexToAddress("0x1")] = &p256.Precompile{}
	return s
}

// WithP256Precompile adds the P256 precompile.
func (s StaticPrecompiles) WithP256Precompile() StaticPrecompiles {
	p256Precompile := &p256.Precompile{}
	s[p256Precompile.Address()] = p256Precompile
	return s
}

// WithBech32Precompile adds the bech32 precompile.
func (s StaticPrecompiles) WithBech32Precompile() StaticPrecompiles {
	bech32Precompile, err := bech32.NewPrecompile(bech32PrecompileBaseGas)
	if err != nil {
		panic(err)
	}
	s[bech32Precompile.Address()] = bech32Precompile
	return s
}

// WithStakingPrecompile adds the staking precompile.
func (s StaticPrecompiles) WithStakingPrecompile(
	stakingKeeper stakingkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	opts ...Option,
) StaticPrecompiles {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	stakingPrecompile := stakingprecompile.NewPrecompile(
		stakingKeeper,
		stakingkeeper.NewMsgServerImpl(&stakingKeeper),
		stakingkeeper.NewQuerier(&stakingKeeper),
		bankKeeper,
		options.AddressCodec,
	)

	s[stakingPrecompile.Address()] = stakingPrecompile
	return s
}

// WithDistributionPrecompile adds the distribution precompile.
func (s StaticPrecompiles) WithDistributionPrecompile(
	distributionKeeper distributionkeeper.Keeper,
	stakingKeeper stakingkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	opts ...Option,
) StaticPrecompiles {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	distributionPrecompile := distprecompile.NewPrecompile(
		distributionKeeper,
		distributionkeeper.NewMsgServerImpl(distributionKeeper),
		distributionkeeper.NewQuerier(distributionKeeper),
		stakingKeeper,
		bankKeeper,
		options.AddressCodec,
	)

	s[distributionPrecompile.Address()] = distributionPrecompile
	return s
}

// WithICS20Precompile adds the ICS20 (transfer) precompile with cosmos/evm v0.6.1
// ERC20 keeper. The concrete *erc20keeper.Keeper satisfies cmn.ERC20Keeper.
func (s StaticPrecompiles) WithICS20Precompile(
	bankKeeper cmn.BankKeeper,
	stakingKeeper stakingkeeper.Keeper,
	transferKeeper *transferkeeper.Keeper,
	channelKeeper *channelkeeper.Keeper,
	erc20Keeper *erc20keeper.Keeper,
) StaticPrecompiles {
	ibcTransferPrecompile := ics20precompile.NewPrecompile(
		bankKeeper,
		stakingKeeper,
		transferKeeper,
		channelKeeper,
		erc20Keeper,
	)

	s[ibcTransferPrecompile.Address()] = ibcTransferPrecompile
	return s
}

// WithBankPrecompile adds the bank precompile with cosmos/evm v0.6.1 ERC20 keeper.
func (s StaticPrecompiles) WithBankPrecompile(
	bankKeeper cmn.BankKeeper,
	erc20Keeper *erc20keeper.Keeper,
) StaticPrecompiles {
	bankPrecompile := bankprecompile.NewPrecompile(bankKeeper, erc20Keeper)
	s[bankPrecompile.Address()] = bankPrecompile
	return s
}

// WithGovPrecompile adds the gov precompile.
func (s StaticPrecompiles) WithGovPrecompile(
	govKeeper govkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	codec codec.Codec,
	opts ...Option,
) StaticPrecompiles {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	govPrecompile := govprecompile.NewPrecompile(
		govkeeper.NewMsgServerImpl(&govKeeper),
		govkeeper.NewQueryServer(&govKeeper),
		bankKeeper,
		codec,
		options.AddressCodec,
	)

	s[govPrecompile.Address()] = govPrecompile
	return s
}

// WithSlashingPrecompile adds the slashing precompile.
func (s StaticPrecompiles) WithSlashingPrecompile(
	slashingKeeper slashingkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	opts ...Option,
) StaticPrecompiles {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	slashingPrecompile := slashingprecompile.NewPrecompile(
		slashingKeeper,
		slashingkeeper.NewMsgServerImpl(slashingKeeper),
		bankKeeper,
		options.ValidatorAddrCodec,
		options.ConsensusAddrCodec,
	)

	s[slashingPrecompile.Address()] = slashingPrecompile
	return s
}

// DefaultStaticPrecompiles returns Uptick's list of static precompiled contracts.
// Uses cosmos/evm v0.6.1's ERC20 keeper (concrete type) directly.
func DefaultStaticPrecompiles(
	stakingKeeper stakingkeeper.Keeper,
	distributionKeeper distributionkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	erc20Keeper *erc20keeper.Keeper,
	transferKeeper *transferkeeper.Keeper,
	channelKeeper *channelkeeper.Keeper,
	govKeeper govkeeper.Keeper,
	slashingKeeper slashingkeeper.Keeper,
	codec codec.Codec,
	opts ...Option,
) map[common.Address]vm.PrecompiledContract {
	precompiles := NewStaticPrecompiles().
		WithPraguePrecompiles().
		WithP256Precompile().
		WithBech32Precompile().
		WithStakingPrecompile(stakingKeeper, bankKeeper, opts...).
		WithDistributionPrecompile(distributionKeeper, stakingKeeper, bankKeeper, opts...).
		WithICS20Precompile(bankKeeper, stakingKeeper, transferKeeper, channelKeeper, erc20Keeper).
		WithBankPrecompile(bankKeeper, erc20Keeper).
		WithGovPrecompile(govKeeper, bankKeeper, codec, opts...).
		WithSlashingPrecompile(slashingKeeper, bankKeeper, opts...)

	return map[common.Address]vm.PrecompiledContract(precompiles)
}
