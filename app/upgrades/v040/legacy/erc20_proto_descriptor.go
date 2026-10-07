package legacy

import (
	"fmt"
	"sync"

	// gogoproto's own registry: the legacy proposal Go types live here, and it
	// is also the only place that knows about cosmos/bank/v1beta1/bank.proto.
	gogoproto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// This file closes the second half of the legacy-write compatibility work.
//
// erc20_proposals.go registers the four v0.3.3 governance `Content` types with
// **gogoproto** (`proto.RegisterType`) so that `uptickd export`, genesis
// bootstrap and the interface registry can encode and decode them. The
// auto-generated CLI is a different reader: it resolves messages through the
// **protobuf-v2** registries (github.com/cosmos/gogoproto/proto.MergedRegistry,
// which merges protoregistry.GlobalFiles with the gogo registry), then marshals
// the response with aminojson. Its `Any` handling looks the type URL up in that
// merged resolver and fails the whole response when any single nested `Any` is
// unknown:
//
//	Error: ... pagination:{total:11}: can't resolve type URL
//	  /uptick.erc20.v1.RegisterCoinProposal: proto: not found
//
// That made `query gov proposals` (and `query gov proposal <legacy id>`)
// unusable after v0.4.0 removed the module — one unreadable type poisons the
// entire list, so operators could not even see the proposal inventory.
//
// The fix is to publish the *descriptor* of the removed package into
// protoregistry.GlobalFiles. That adds nothing to the wire format and nothing to
// consensus state; it only teaches the client-side resolver a message shape it
// could no longer derive from generated code.
//
// Why the descriptor is hand-written instead of generated:
//
// Generating `uptick/erc20/v1/erc20.pb.go` would call protov2's
// `protoimpl` bootstrap, which registers `uptick.erc20.v1.RegisterCoinProposal`
// in the protov2 registries *and* the file's message set is then also visible to
// gogoproto — colliding with the hand-written types in erc20_proposals.go that
// already own those full names in the gogo registry. The result is a
// `proto: duplicate proto type registered` panic at init. Replacing the
// hand-written types wholesale is not an option either: they are the v0.3.3
// state-export compatibility surface. A bare FileDescriptorProto, therefore,
// gives us exactly the client-side resolution we need and nothing else.
//
// The descriptor below is a field-for-field transcription of
// proto/uptick/erc20/v1/erc20.proto as of v0.3.3. gogoproto's custom options
// (`gogoproto.nullable`, `gogoproto.equal`, ...) are deliberately not carried:
// they affect generated Go code only, never the wire format or the JSON shape,
// and reproducing them would require the gogo extension descriptors.

const (
	// legacyERC20ProtoFilename is the proto path as it existed in v0.3.3. It is
	// kept identical on purpose: `protodesc`/`protoregistry` key files by path,
	// and a future generated descriptor must collide here rather than silently
	// register a second, divergent copy.
	legacyERC20ProtoFilename = "uptick/erc20/v1/erc20.proto"

	legacyERC20ProtoPackage = "uptick.erc20.v1"

	// legacyERC20GoPackage mirrors the original go_package option. Not used to
	// resolve anything, kept so the descriptor matches the source proto.
	legacyERC20GoPackage = "github.com/UptickNetwork/uptick/x/erc20/types"
)

var (
	// legacyERC20ProtoOnce makes registration a one-shot: the result (including
	// the error) is memoised, so a later caller from the application boot path
	// can re-report an init failure instead of silently succeeding.
	legacyERC20ProtoOnce sync.Once
	legacyERC20ProtoErr  error
)

// legacyERC20ProposalNames are the four gov `Content` implementations that can
// appear as the `Any` payload of a historical
// cosmos.gov.v1.MsgExecLegacyContent. Only these must resolve for
// `uptickd query gov ...` to work; the rest of the removed package (Msg/Query/
// GenesisState) never reaches retained state through an `Any`.
var legacyERC20ProposalNames = []protoreflect.FullName{
	"uptick.erc20.v1.RegisterCoinProposal",
	"uptick.erc20.v1.RegisterERC20Proposal",
	"uptick.erc20.v1.ToggleTokenRelayProposal",
	"uptick.erc20.v1.UpdateTokenPairERC20Proposal",
}

func init() {
	legacyERC20ProtoErr = RegisterLegacyERC20ProtoDescriptor()
}

// RegisterLegacyERC20ProtoDescriptor publishes the v0.3.3 `uptick.erc20.v1`
// file descriptor into protoregistry.GlobalFiles, which is half of the resolver
// the auto-generated CLI builds. It is idempotent and safe to call concurrently.
//
// Registration is client-side only: the descriptor is not part of the app's
// interface registry, is never written to state, and is never consulted by the
// node's message routing.
func RegisterLegacyERC20ProtoDescriptor() error {
	legacyERC20ProtoOnce.Do(func() { legacyERC20ProtoErr = registerLegacyERC20ProtoDescriptor() })
	return legacyERC20ProtoErr
}

func registerLegacyERC20ProtoDescriptor() error {
	// Idempotency: GlobalFiles keys by path, and a second RegisterFile for the
	// same path is an error rather than a no-op.
	if _, err := protoregistry.GlobalFiles.FindFileByPath(legacyERC20ProtoFilename); err == nil {
		return nil
	}

	// HybridResolver is required rather than GlobalFiles alone: the file imports
	// cosmos/bank/v1beta1/bank.proto for the embedded Metadata, and that file is
	// only registered with gogoproto (all SDK modules are gogo-generated here).
	fd, err := protodesc.NewFile(legacyERC20FileDescriptorProto(), gogoproto.HybridResolver)
	if err != nil {
		return fmt.Errorf("legacy erc20: build %s descriptor: %w", legacyERC20ProtoFilename, err)
	}

	if err := protoregistry.GlobalFiles.RegisterFile(fd); err != nil {
		// The overwhelmingly likely cause is that a generated uptick/erc20/v1
		// file was reintroduced and now owns these full names too. Say so,
		// because the fix is to delete this hand-written descriptor, not to
		// retry registration.
		return fmt.Errorf(
			"legacy erc20: cannot register %s (%v) — if a generated uptick/erc20/v1 descriptor was reintroduced, "+
				"remove this hand-written one instead of registering both",
			legacyERC20ProtoFilename, err,
		)
	}

	return nil
}

// LegacyERC20ProtoFileDescriptor returns the descriptor as registered in
// protoregistry.GlobalFiles, or nil when registration did not happen. Test-only
// convenience so a guard test can assert against the registered artifact rather
// than a freshly rebuilt one.
func LegacyERC20ProtoFileDescriptor() protoreflect.FileDescriptor {
	fd, err := protoregistry.GlobalFiles.FindFileByPath(legacyERC20ProtoFilename)
	if err != nil {
		return nil
	}
	return fd
}

// legacyERC20FileDescriptorProto builds the FileDescriptorProto for the v0.3.3
// uptick/erc20/v1/erc20.proto.
func legacyERC20FileDescriptorProto() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String(legacyERC20ProtoFilename),
		Package: proto.String(legacyERC20ProtoPackage),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{GoPackage: proto.String(legacyERC20GoPackage)},
		Dependency: []string{
			"cosmos/bank/v1beta1/bank.proto",
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Owner"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("OWNER_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("OWNER_MODULE"), Number: proto.Int32(1)},
					{Name: proto.String("OWNER_EXTERNAL"), Number: proto.Int32(2)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("TokenPair"),
				Field: []*descriptorpb.FieldDescriptorProto{
					proto3String("erc20_address", 1),
					proto3String("denom", 2),
					proto3Bool("enabled", 3),
					proto3Enum("contract_owner", 4, ".uptick.erc20.v1.Owner"),
				},
			},
			{
				Name: proto.String("RegisterCoinProposal"),
				Field: []*descriptorpb.FieldDescriptorProto{
					proto3String("title", 1),
					proto3String("description", 2),
					// gogoproto.nullable=false on the Go side: the field is
					// embedded by value, but that is a codegen concern. On the
					// wire it is still a length-delimited submessage, so the
					// descriptor stays a plain message field.
					proto3Message("metadata", 3, ".cosmos.bank.v1beta1.Metadata"),
				},
			},
			{
				// Field 3 is spelled `erc20address` (no underscore) in the
				// original proto. aminojson keys on the proto field name, so the
				// spelling is user-visible output; do not "tidy" it.
				Name: proto.String("RegisterERC20Proposal"),
				Field: []*descriptorpb.FieldDescriptorProto{
					proto3String("title", 1),
					proto3String("description", 2),
					proto3String("erc20address", 3),
				},
			},
			{
				Name: proto.String("ToggleTokenRelayProposal"),
				Field: []*descriptorpb.FieldDescriptorProto{
					proto3String("title", 1),
					proto3String("description", 2),
					proto3String("token", 3),
				},
			},
			{
				Name: proto.String("UpdateTokenPairERC20Proposal"),
				Field: []*descriptorpb.FieldDescriptorProto{
					proto3String("title", 1),
					proto3String("description", 2),
					proto3String("erc20_address", 3),
					proto3String("new_erc20_address", 4),
				},
			},
		},
	}
}

func proto3String(name string, number int32) *descriptorpb.FieldDescriptorProto {
	return proto3Field(name, number, descriptorpb.FieldDescriptorProto_TYPE_STRING, "")
}

func proto3Bool(name string, number int32) *descriptorpb.FieldDescriptorProto {
	return proto3Field(name, number, descriptorpb.FieldDescriptorProto_TYPE_BOOL, "")
}

func proto3Message(name string, number int32, typeName string) *descriptorpb.FieldDescriptorProto {
	return proto3Field(name, number, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, typeName)
}

func proto3Enum(name string, number int32, typeName string) *descriptorpb.FieldDescriptorProto {
	return proto3Field(name, number, descriptorpb.FieldDescriptorProto_TYPE_ENUM, typeName)
}

func proto3Field(name string, number int32, typ descriptorpb.FieldDescriptorProto_Type, typeName string) *descriptorpb.FieldDescriptorProto {
	f := &descriptorpb.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(number),
		Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:   typ.Enum(),
	}
	if typeName != "" {
		f.TypeName = proto.String(typeName)
	}
	return f
}
