package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// RegisterInterfaces registers module message implementations.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterApplication{},
		&MsgSubmitAttestation{},
		&MsgChallengeAttestation{},
		&MsgResolveChallenge{},
		&MsgFinalizeAttestation{},
		&MsgGrantGasPayer{},
		&MsgRevokeGasPayer{},
		&MsgSetAttestationSubmitter{},
		&MsgConsumeGasCredit{},
		&MsgSetEpochBudget{},
		&MsgUpdateParams{},
		&MsgSetApplicationStatus{},
		&MsgPause{},
		&MsgFundRewardPool{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec registers amino concrete types.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterApplication{}, "appmining/RegisterApplication", nil)
	cdc.RegisterConcrete(&MsgSubmitAttestation{}, "appmining/SubmitAttestation", nil)
	cdc.RegisterConcrete(&MsgChallengeAttestation{}, "appmining/ChallengeAttestation", nil)
	cdc.RegisterConcrete(&MsgResolveChallenge{}, "appmining/ResolveChallenge", nil)
	cdc.RegisterConcrete(&MsgFinalizeAttestation{}, "appmining/FinalizeAttestation", nil)
	cdc.RegisterConcrete(&MsgGrantGasPayer{}, "appmining/GrantGasPayer", nil)
	cdc.RegisterConcrete(&MsgRevokeGasPayer{}, "appmining/RevokeGasPayer", nil)
	cdc.RegisterConcrete(&MsgSetAttestationSubmitter{}, "appmining/SetAttestationSubmitter", nil)
	cdc.RegisterConcrete(&MsgConsumeGasCredit{}, "appmining/ConsumeGasCredit", nil)
	cdc.RegisterConcrete(&MsgSetEpochBudget{}, "appmining/SetEpochBudget", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "appmining/UpdateParams", nil)
	cdc.RegisterConcrete(&MsgSetApplicationStatus{}, "appmining/SetApplicationStatus", nil)
	cdc.RegisterConcrete(&MsgPause{}, "appmining/Pause", nil)
	cdc.RegisterConcrete(&MsgFundRewardPool{}, "appmining/FundRewardPool", nil)
}
