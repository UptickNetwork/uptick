package legacy

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// These tests guard the descriptor in erc20_proto_descriptor.go. That file
// exists because the auto-generated CLI (autocli + aminojson) resolves `Any`
// payloads through github.com/cosmos/gogoproto/proto.MergedRegistry, which the
// gogoproto-only registration in erc20_proposals.go does not reach. The failure
// it fixes was total: one unresolvable legacy Content type made
//
//	uptickd query gov proposals
//
// fail as a whole with
//
//	can't resolve type URL /uptick.erc20.v1.RegisterCoinProposal: proto: not found
//
// so operators could not list proposals at all.

// TestLegacyERC20DescriptorsResolveFromTheCliResolver is the direct regression
// test for P2-0. It asserts through MergedRegistry — the exact resolver
// autocli builds in EnhanceRootCommand — rather than through GlobalFiles, so it
// fails if the descriptor stops reaching the CLI even when the file itself is
// still registered.
func TestLegacyERC20DescriptorsResolveFromTheCliResolver(t *testing.T) {
	require.NoError(t, RegisterLegacyERC20ProtoDescriptor())

	merged, err := gogoproto.MergedRegistry()
	require.NoError(t, err)

	for _, name := range legacyERC20ProposalNames {
		desc, err := merged.FindDescriptorByName(name)
		require.NoErrorf(t, err, "the CLI cannot resolve %s — `query gov proposals` will fail as a whole", name)

		md, ok := desc.(protoreflect.MessageDescriptor)
		require.Truef(t, ok, "%s resolved to %T, want a message descriptor", name, desc)

		// aminojson builds a dynamicpb from this descriptor and unmarshals the
		// Any payload into it, so the shape has to be usable, not merely present.
		msg := dynamicpb.NewMessage(md)
		require.NotPanics(t, func() { proto.Marshal(msg) })
	}
}

// TestLegacyERC20DescriptorMatchesTheGogoStructs pins the descriptor to the
// hand-written Go types. They are two independent transcriptions of the same
// v0.3.3 proto file, so any drift — a renamed field, a renumbered tag — would
// silently mis-decode historical proposals in the CLI while `export` kept
// working. This catches that at test time.
func TestLegacyERC20DescriptorMatchesTheGogoStructs(t *testing.T) {
	require.NoError(t, RegisterLegacyERC20ProtoDescriptor())

	fd := LegacyERC20ProtoFileDescriptor()
	require.NotNil(t, fd)
	require.Equal(t, legacyERC20ProtoFilename, fd.Path())
	require.Equal(t, protoreflect.FullName(legacyERC20ProtoPackage), fd.Package())

	cases := []struct {
		message string
		value   any
	}{
		{"RegisterCoinProposal", &RegisterCoinProposal{}},
		{"RegisterERC20Proposal", &RegisterERC20Proposal{}},
		{"ToggleTokenRelayProposal", &ToggleTokenRelayProposal{}},
		{"UpdateTokenPairERC20Proposal", &UpdateTokenPairERC20Proposal{}},
	}

	for _, tc := range cases {
		t.Run(tc.message, func(t *testing.T) {
			md := fd.Messages().ByName(protoreflect.Name(tc.message))
			require.NotNilf(t, md, "%s is missing from the descriptor", tc.message)

			fromGo := gogoStructFieldShape(t, tc.value)
			fromProto := make(map[int32]string, md.Fields().Len())
			for i := 0; i < md.Fields().Len(); i++ {
				f := md.Fields().Get(i)
				fromProto[int32(f.Number())] = string(f.Name())
			}

			require.Equalf(t, fromGo, fromProto,
				"%s: the descriptor's fields drifted from the hand-written protobuf struct tags", tc.message)
		})
	}

	// The embedded Metadata must resolve to the real bank descriptor, otherwise
	// aminojson would render it as an opaque/empty object.
	coinDesc := fd.Messages().ByName("RegisterCoinProposal")
	require.NotNil(t, coinDesc)
	metaField := coinDesc.Fields().ByName("metadata")
	require.NotNil(t, metaField)
	require.Equal(t, protoreflect.FullName("cosmos.bank.v1beta1.Metadata"), metaField.Message().FullName())
	require.NotNil(t, metaField.Message().Fields().ByName("base"),
		"cosmos.bank.v1beta1.Metadata resolved without its fields — the dependency was not really resolved")
}

// TestLegacyERC20DescriptorMarshalIsByteIdentical proves the descriptor is wire
// compatible with the hand-written marshalers. If it were not, the CLI would
// decode historical proposals into a different shape than `export` writes.
//
// Deterministic marshaling is required for the byte comparison, and that is not
// a convenience: dynamicpb.Message.Range iterates a Go map
// (types/dynamicpb/dynamic.go, `for num, v := range m.known`), so the field
// order of a plain Marshal of a dynamic message is deliberately unspecified and
// changes between runs. Deterministic:true sorts by field number, which is the
// order the hand-written marshaler appends in. Comparing non-deterministic
// output would be flaky, and asserting on a specific unordered permutation would
// be asserting on nothing meaningful — protobuf field order is not significant.
func TestLegacyERC20DescriptorMarshalIsByteIdentical(t *testing.T) {
	require.NoError(t, RegisterLegacyERC20ProtoDescriptor())

	fd := LegacyERC20ProtoFileDescriptor()
	require.NotNil(t, fd)

	metadata := banktypes.Metadata{
		Description: "IBC voucher of auoc",
		Base:        "ibc/9BD618B22904F348824AB73154F134E59FCA43EC002D64428247675EBBE74C64",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "ibc/9BD618B22904F348824AB73154F134E59FCA43EC002D64428247675EBBE74C64", Exponent: 0, Aliases: []string{"auoc"}},
			{Denom: "uoc", Exponent: 18},
		},
		Display: "uoc",
		Name:    "Uptick",
		Symbol:  "UOC",
	}

	cases := []struct {
		message string
		value   gogoproto.Message
	}{
		{"RegisterCoinProposal", &RegisterCoinProposal{
			Title: "Register IBC ERC20", Description: "vet the pair", Metadata: metadata,
		}},
		{"RegisterERC20Proposal", &RegisterERC20Proposal{
			Title: "Register ERC20", Description: "external contract", Erc20Address: "0x1122334455667788990011223344556677889900",
		}},
		{"ToggleTokenRelayProposal", &ToggleTokenRelayProposal{
			Title: "Toggle relay", Description: "disable relaying", Token: "0xaabbccddeeff00112233445566778899aabbccdd",
		}},
		{"UpdateTokenPairERC20Proposal", &UpdateTokenPairERC20Proposal{
			Title: "Update pair", Description: "rotate contract",
			Erc20Address:    "0x1122334455667788990011223344556677889900",
			NewErc20Address: "0x9988776655443322110099887766554433221100",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.message, func(t *testing.T) {
			md := fd.Messages().ByName(protoreflect.Name(tc.message))
			require.NotNil(t, md)

			// ugly, but every case needs the typed marshaler, which only exists
			// on the concrete Go type.
			typed, ok := tc.value.(interface {
				gogoproto.Message
				Marshal() ([]byte, error)
				Unmarshal([]byte) error
			})
			require.Truef(t, ok, "%T does not implement the hand-written marshaler", tc.value)

			goBytes, err := typed.Marshal()
			require.NoError(t, err)
			require.NotEmpty(t, goBytes)

			dyn := dynamicpb.NewMessage(md)
			require.NoError(t, proto.Unmarshal(goBytes, dyn))

			protoBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(dyn)
			require.NoError(t, err)
			require.Equalf(t, goBytes, protoBytes,
				"%s: deterministic dynamic marshal differs from the hand-written marshaler — the two definitions have diverged", tc.message)

			// Field order is not semantically load-bearing, so also prove the
			// default (map-ordered) encoding carries the same message.
			unordered, err := proto.Marshal(dyn)
			require.NoError(t, err)
			require.Truef(t, proto.Equal(dyn, decodeDynamic(t, md, unordered)),
				"%s: the map-ordered encoding does not decode back to the same message", tc.message)

			// And the reverse: what the CLI's descriptor produces must still be
			// readable by the state-export code path.
			fresh := reflect.New(reflect.TypeOf(tc.value).Elem()).Interface().(interface {
				Unmarshal([]byte) error
			})
			require.NoError(t, fresh.Unmarshal(protoBytes))
			require.Equalf(t, tc.value, fresh, "%s: round-trip through the descriptor changed the value", tc.message)
		})
	}
}

func decodeDynamic(t *testing.T, md protoreflect.MessageDescriptor, bz []byte) *dynamicpb.Message {
	t.Helper()
	msg := dynamicpb.NewMessage(md)
	require.NoError(t, proto.Unmarshal(bz, msg))
	return msg
}

// gogoStructFieldShape extracts `number -> proto field name` from a
// gogoproto-generated struct tag such as
// `protobuf:"bytes,3,opt,name=new_erc20_address,json=newErc20Address,proto3"`.
//
// The `name=` component is the proto field name, not the JSON name: that is what
// aminojson keys its output on, so it is what the descriptor has to match.
func gogoStructFieldShape(t *testing.T, value any) map[int32]string {
	t.Helper()

	out := map[int32]string{}
	typ := reflect.TypeOf(value).Elem()
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("protobuf")
		require.NotEmptyf(t, tag, "%s.%s has no protobuf tag", typ.Name(), typ.Field(i).Name)

		parts := strings.Split(tag, ",")
		require.GreaterOrEqualf(t, len(parts), 4, "%s.%s has an unparseable protobuf tag %q", typ.Name(), typ.Field(i).Name, tag)

		number, err := strconv.ParseInt(parts[1], 10, 32)
		require.NoErrorf(t, err, "%s.%s field number %q is not an integer", typ.Name(), typ.Field(i).Name, parts[1])

		var name string
		for _, p := range parts[2:] {
			if strings.HasPrefix(p, "name=") {
				name = strings.TrimPrefix(p, "name=")
				break
			}
		}
		require.NotEmptyf(t, name, "%s.%s has no name= component in tag %q", typ.Name(), typ.Field(i).Name, tag)

		out[int32(number)] = name
	}
	return out
}
