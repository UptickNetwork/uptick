package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	"github.com/stretchr/testify/require"

	appmining "github.com/UptickNetwork/uptick/x/appmining/types"
)

func TestParams_TimingEffectiveDefaults(t *testing.T) {
	p := appmining.Params{}
	require.Equal(t, int64(appmining.DefaultChallengeWindowSeconds), p.EffectiveChallengeWindowSeconds())
	require.Equal(t, int64(appmining.DefaultMaxFreezeSeconds), p.EffectiveMaxFreezeSeconds())
	require.Equal(t, int64(appmining.DefaultMaxWireProofValiditySeconds), p.EffectiveMaxWireProofValiditySeconds())
	require.Equal(t, int64(appmining.DefaultNonceRetentionSeconds), p.EffectiveNonceRetentionSeconds())
}

func TestParams_TimingEffectiveClampsHugeUint64(t *testing.T) {
	p := appmining.Params{ChallengeWindowSeconds: ^uint64(0)}
	require.Equal(t, int64(^uint64(0)>>1), p.EffectiveChallengeWindowSeconds()) // MaxInt64
}

func TestParams_TimingValidateBounds(t *testing.T) {
	p := appmining.DefaultParams()
	require.NoError(t, p.Validate())

	p.ChallengeWindowSeconds = 1 // below MinChallengeWindowSeconds
	require.Error(t, p.Validate())

	p = appmining.DefaultParams()
	p.ChallengeWindowSeconds = 600 // origin dry-run window
	require.NoError(t, p.Validate())

	p = appmining.DefaultParams()
	p.MaxFreezeSeconds = 600 // below MinTimingSeconds
	require.Error(t, p.Validate())

	// Wire validity floor is 60s (expiry-path testing), below freeze/nonce's 1h.
	p = appmining.DefaultParams()
	p.MaxWireProofValiditySeconds = appmining.MinWireProofValiditySeconds
	require.NoError(t, p.Validate())
	require.Equal(t, int64(60), p.EffectiveMaxWireProofValiditySeconds())

	p = appmining.DefaultParams()
	p.MaxWireProofValiditySeconds = appmining.MinWireProofValiditySeconds - 1
	require.Error(t, p.Validate())

	p = appmining.DefaultParams()
	p.MaxWireProofValiditySeconds = appmining.DefaultMaxWireProofValiditySeconds
	p.NonceRetentionSeconds = appmining.DefaultMaxWireProofValiditySeconds - 1
	require.Error(t, p.Validate())

	p = appmining.DefaultParams()
	p.ChallengeWindowSeconds = 7200
	p.MaxFreezeSeconds = 7200
	p.MaxWireProofValiditySeconds = 7200
	p.NonceRetentionSeconds = 7200
	require.NoError(t, p.Validate())
	require.Equal(t, int64(7200), p.EffectiveChallengeWindowSeconds())
}

func TestParams_Fields19to22RoundTripAndDescriptor(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	appmining.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	src := appmining.DefaultParams()
	src.ChallengeWindowSeconds = 7200
	src.MaxFreezeSeconds = 7200
	src.MaxWireProofValiditySeconds = 7200
	src.NonceRetentionSeconds = 86400

	bz, err := cdc.Marshal(&src)
	require.NoError(t, err)

	var dst appmining.Params
	require.NoError(t, cdc.Unmarshal(bz, &dst))
	require.Equal(t, src.ChallengeWindowSeconds, dst.ChallengeWindowSeconds)
	require.Equal(t, src.MaxFreezeSeconds, dst.MaxFreezeSeconds)
	require.Equal(t, src.MaxWireProofValiditySeconds, dst.MaxWireProofValiditySeconds)
	require.Equal(t, src.NonceRetentionSeconds, dst.NonceRetentionSeconds)

	require.NoError(t, unknownproto.RejectUnknownFieldsStrict(bz, &appmining.Params{}, unknownproto.DefaultAnyResolver{}))
}
