package params

import (
	"fmt"

	"cosmossdk.io/x/tx/signing"
	v041 "github.com/UptickNetwork/uptick/app/upgrades/v041"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	enccodec "github.com/cosmos/evm/encoding/codec"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MakeEncodingConfig creates an EncodingConfig for an amino based test configuration.
//
// Boot-time fail-fast wrapper around MakeEncodingConfigChecked (M-7): a
// Keplr-compat registration failure is a build/SDK-compatibility defect that
// must stop the process with a readable message instead of silently serving
// EIP-712 txs that cannot be verified.
func MakeEncodingConfig() EncodingConfig {
	enc, err := MakeEncodingConfigChecked()
	if err != nil {
		panic(err)
	}
	return enc
}

// MakeEncodingConfigChecked is MakeEncodingConfig with an explicit error path.
func MakeEncodingConfigChecked() (EncodingConfig, error) {
	amino := codec.NewLegacyAmino()
	signingOptions := signing.Options{
		AddressCodec: address.Bech32Codec{
			Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
		},
		ValidatorAddressCodec: address.Bech32Codec{
			Bech32Prefix: sdk.GetConfig().GetBech32ValidatorAddrPrefix(),
		},
		CustomGetSigners: map[protoreflect.FullName]signing.GetSignersFunc{
			evmtypes.MsgEthereumTxCustomGetSigner.MsgType:     evmtypes.MsgEthereumTxCustomGetSigner.Fn,
			erc20types.MsgConvertERC20CustomGetSigner.MsgType: erc20types.MsgConvertERC20CustomGetSigner.Fn,
		},
	}
	interfaceRegistry, _ := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles:     proto.HybridResolver,
		SigningOptions: signingOptions,
	})
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	// Register the evm types
	enccodec.RegisterLegacyAminoCodec(amino)
	enccodec.RegisterInterfaces(interfaceRegistry)

	// Register v0.4.1 Keplr compatibility: legacy ethermint pubkey and EIP-712
	// extension option type URLs map onto the complete cosmos/evm types, and
	// the v0.3.3-era account/proposal records remain decodable for genesis
	// bootstrap and state export.
	if err := v041.RegisterCompatInterfaces(interfaceRegistry); err != nil {
		return EncodingConfig{}, fmt.Errorf("register v0.4.1 Keplr compat interfaces: %w", err)
	}

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             marshaler,
		TxConfig:          txCfg,
		LegacyAmino:       amino,
	}, nil
}
