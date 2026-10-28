package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	"github.com/stretchr/testify/require"

	appmining "github.com/UptickNetwork/uptick/x/appmining/types"
)

// FIELD18 regression: CheckTx uses RejectUnknownFieldsStrict against the
// FileDescriptor. Hand-patched Unmarshal alone is insufficient — regenerating
// types.pb.go must keep field 18 in both Go Unmarshal and the descriptor.
func TestParams_Field18AllowlistRoundTripAndDescriptor(t *testing.T) {
	interfaceRegistry := types.NewInterfaceRegistry()
	appmining.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	src := appmining.DefaultParams()
	src.EntitlementAllowedAssessmentVersions = []string{appmining.DefaultAssessmentVersion}
	require.NotEmpty(t, src.EntitlementAllowedAssessmentVersions)

	bz, err := cdc.Marshal(&src)
	require.NoError(t, err)

	var dst appmining.Params
	require.NoError(t, cdc.Unmarshal(bz, &dst))
	require.Equal(t, src.EntitlementAllowedAssessmentVersions, dst.EntitlementAllowedAssessmentVersions)

	// Strict unknown-field rejector must accept tag 18 (FileDescriptor in sync).
	require.NoError(t, unknownproto.RejectUnknownFieldsStrict(bz, &appmining.Params{}, unknownproto.DefaultAnyResolver{}))

	js, err := cdc.MarshalJSON(&src)
	require.NoError(t, err)
	var viaJSON appmining.Params
	require.NoError(t, cdc.UnmarshalJSON(js, &viaJSON))
	require.Equal(t, []string{appmining.DefaultAssessmentVersion}, viaJSON.EntitlementAllowedAssessmentVersions)

	bz2, err := cdc.Marshal(&viaJSON)
	require.NoError(t, err)
	var again appmining.Params
	require.NoError(t, cdc.Unmarshal(bz2, &again))
	require.Equal(t, viaJSON.EntitlementAllowedAssessmentVersions, again.EntitlementAllowedAssessmentVersions)
}
