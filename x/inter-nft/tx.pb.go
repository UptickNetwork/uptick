// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: internft/tx.proto

package inter_nft

import (
	context "context"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	grpc1 "github.com/gogo/protobuf/grpc"
	proto "github.com/gogo/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// MsgIssueClass defines the payload for Msg/IssueClass
type MsgIssueClass struct {
	// id defines the unique identifier of the NFT classification, similar to the
	// contract address of ERC721
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// name defines the human-readable name of the NFT classification. Optional
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// symbol is an abbreviated name for nft classification. Optional
	Symbol string `protobuf:"bytes,3,opt,name=symbol,proto3" json:"symbol,omitempty"`
	// description is a brief description of nft classification. Optional
	Description string `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	// uri for the class metadata stored off chain. It can define schema for Class
	// and NFT `Data` attributes. Optional
	Uri string `protobuf:"bytes,5,opt,name=uri,proto3" json:"uri,omitempty"`
	// uri_hash is a hash of the document pointed by uri. Optional
	UriHash string `protobuf:"bytes,6,opt,name=uri_hash,json=uriHash,proto3" json:"uri_hash,omitempty"`
	Issuer  string `protobuf:"bytes,7,opt,name=issuer,proto3" json:"issuer,omitempty"`
}

func (m *MsgIssueClass) Reset()         { *m = MsgIssueClass{} }
func (m *MsgIssueClass) String() string { return proto.CompactTextString(m) }
func (*MsgIssueClass) ProtoMessage()    {}
func (*MsgIssueClass) Descriptor() ([]byte, []int) {
	return fileDescriptor_6ff49e2dccae052a, []int{0}
}
func (m *MsgIssueClass) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgIssueClass) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgIssueClass.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgIssueClass) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgIssueClass.Merge(m, src)
}
func (m *MsgIssueClass) XXX_Size() int {
	return m.Size()
}
func (m *MsgIssueClass) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgIssueClass.DiscardUnknown(m)
}

var xxx_messageInfo_MsgIssueClass proto.InternalMessageInfo

// MsgIssueClassResponse defines the response for Msg/MsgIssueClass
type MsgIssueClassResponse struct {
}

func (m *MsgIssueClassResponse) Reset()         { *m = MsgIssueClassResponse{} }
func (m *MsgIssueClassResponse) String() string { return proto.CompactTextString(m) }
func (*MsgIssueClassResponse) ProtoMessage()    {}
func (*MsgIssueClassResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_6ff49e2dccae052a, []int{1}
}
func (m *MsgIssueClassResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgIssueClassResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgIssueClassResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgIssueClassResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgIssueClassResponse.Merge(m, src)
}
func (m *MsgIssueClassResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgIssueClassResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgIssueClassResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgIssueClassResponse proto.InternalMessageInfo

// MsgMintNFT defines the payload for Msg/MintNFT
type MsgMintNFT struct {
	// class_id associated with the NFT, similar to the contract address of ERC721
	ClassId string `protobuf:"bytes,1,opt,name=class_id,json=classId,proto3" json:"class_id,omitempty"`
	// id is a unique identifier of the NFT
	Id string `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	// uri for the NFT metadata stored off chain
	Uri string `protobuf:"bytes,3,opt,name=uri,proto3" json:"uri,omitempty"`
	// uri_hash is a hash of the document pointed by uri
	UriHash string `protobuf:"bytes,4,opt,name=uri_hash,json=uriHash,proto3" json:"uri_hash,omitempty"`
	// minter is a minter of the NFT
	Minter string `protobuf:"bytes,5,opt,name=minter,proto3" json:"minter,omitempty"`
	// receiver is a receiver of the NFT, optional
	Receiver string `protobuf:"bytes,6,opt,name=receiver,proto3" json:"receiver,omitempty"`
}

func (m *MsgMintNFT) Reset()         { *m = MsgMintNFT{} }
func (m *MsgMintNFT) String() string { return proto.CompactTextString(m) }
func (*MsgMintNFT) ProtoMessage()    {}
func (*MsgMintNFT) Descriptor() ([]byte, []int) {
	return fileDescriptor_6ff49e2dccae052a, []int{2}
}
func (m *MsgMintNFT) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgMintNFT) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgMintNFT.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgMintNFT) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgMintNFT.Merge(m, src)
}
func (m *MsgMintNFT) XXX_Size() int {
	return m.Size()
}
func (m *MsgMintNFT) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgMintNFT.DiscardUnknown(m)
}

var xxx_messageInfo_MsgMintNFT proto.InternalMessageInfo

// MintNFTResponse defines the response for Msg/MintNFT
type MintNFTResponse struct {
}

func (m *MintNFTResponse) Reset()         { *m = MintNFTResponse{} }
func (m *MintNFTResponse) String() string { return proto.CompactTextString(m) }
func (*MintNFTResponse) ProtoMessage()    {}
func (*MintNFTResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_6ff49e2dccae052a, []int{3}
}
func (m *MintNFTResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MintNFTResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MintNFTResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MintNFTResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MintNFTResponse.Merge(m, src)
}
func (m *MintNFTResponse) XXX_Size() int {
	return m.Size()
}
func (m *MintNFTResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MintNFTResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MintNFTResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgIssueClass)(nil), "internft.MsgIssueClass")
	proto.RegisterType((*MsgIssueClassResponse)(nil), "internft.MsgIssueClassResponse")
	proto.RegisterType((*MsgMintNFT)(nil), "internft.MsgMintNFT")
	proto.RegisterType((*MintNFTResponse)(nil), "internft.MintNFTResponse")
}

func init() { proto.RegisterFile("internft/tx.proto", fileDescriptor_6ff49e2dccae052a) }

var fileDescriptor_6ff49e2dccae052a = []byte{
	// 406 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x92, 0xbd, 0x6e, 0xd4, 0x40,
	0x14, 0x85, 0x3d, 0xeb, 0x65, 0x6d, 0x2e, 0xe2, 0x27, 0xa3, 0x40, 0x1c, 0x17, 0xde, 0x68, 0x2b,
	0x0a, 0x62, 0x8b, 0x50, 0x20, 0xa5, 0x04, 0x69, 0x45, 0x0a, 0x53, 0x44, 0x54, 0x34, 0x91, 0x7f,
	0x26, 0xf6, 0x45, 0xf1, 0xcc, 0x6a, 0x66, 0x8c, 0xc2, 0x1b, 0x40, 0xc7, 0x23, 0x44, 0x3c, 0x07,
	0x0f, 0x40, 0x99, 0x92, 0x12, 0xed, 0x36, 0x3c, 0x06, 0xf2, 0xec, 0x78, 0x65, 0x0b, 0xe8, 0xee,
	0x39, 0xc7, 0x9a, 0xfb, 0x9d, 0x19, 0xc3, 0x1e, 0x72, 0xcd, 0x24, 0xbf, 0xd4, 0x89, 0xbe, 0x8e,
	0x57, 0x52, 0x68, 0x41, 0xfd, 0xde, 0x0a, 0xf7, 0x2b, 0x51, 0x09, 0x63, 0x26, 0xdd, 0xb4, 0xcd,
	0x17, 0xdf, 0x09, 0xdc, 0x4f, 0x55, 0x75, 0xa6, 0x54, 0xcb, 0x5e, 0x5f, 0x65, 0x4a, 0xd1, 0x07,
	0x30, 0xc1, 0x32, 0x20, 0x47, 0xe4, 0xe9, 0xdd, 0xf3, 0x09, 0x96, 0x94, 0xc2, 0x94, 0x67, 0x0d,
	0x0b, 0x26, 0xc6, 0x31, 0x33, 0x7d, 0x02, 0x33, 0xf5, 0xa9, 0xc9, 0xc5, 0x55, 0xe0, 0x1a, 0xd7,
	0x2a, 0x7a, 0x04, 0xf7, 0x4a, 0xa6, 0x0a, 0x89, 0x2b, 0x8d, 0x82, 0x07, 0x53, 0x13, 0x0e, 0x2d,
	0xfa, 0x08, 0xdc, 0x56, 0x62, 0x70, 0xc7, 0x24, 0xdd, 0x48, 0x0f, 0xc1, 0x6f, 0x25, 0x5e, 0xd4,
	0x99, 0xaa, 0x83, 0x99, 0xb1, 0xbd, 0x56, 0xe2, 0x9b, 0x4c, 0xd5, 0xdd, 0x1a, 0xec, 0xc0, 0x64,
	0xe0, 0x6d, 0xd7, 0x6c, 0xd5, 0xa9, 0xff, 0xf9, 0x66, 0xee, 0xfc, 0xbe, 0x99, 0x3b, 0x8b, 0x03,
	0x78, 0x3c, 0xa2, 0x3f, 0x67, 0x6a, 0x25, 0xb8, 0x62, 0x8b, 0x6f, 0x04, 0x20, 0x55, 0x55, 0x8a,
	0x5c, 0xbf, 0x5d, 0xbe, 0xeb, 0x96, 0x14, 0x5d, 0x7e, 0xb1, 0xab, 0xe6, 0x19, 0x7d, 0x56, 0xda,
	0xbe, 0x93, 0x5d, 0x5f, 0x4b, 0xe8, 0xfe, 0x9b, 0x70, 0xfa, 0x17, 0x61, 0x63, 0x6e, 0xd8, 0x36,
	0xb2, 0x8a, 0x86, 0xe0, 0x4b, 0x56, 0x30, 0xfc, 0xc8, 0xa4, 0x2d, 0xb5, 0xd3, 0x03, 0xfa, 0x3d,
	0x78, 0x68, 0x01, 0x7b, 0xee, 0x93, 0x2f, 0x04, 0xdc, 0x54, 0x55, 0x74, 0x09, 0x30, 0x78, 0x93,
	0x83, 0xb8, 0x7f, 0xc6, 0x78, 0x54, 0x37, 0x9c, 0xff, 0x27, 0xe8, 0xcf, 0xa3, 0xa7, 0xe0, 0xf5,
	0x77, 0xb0, 0x3f, 0xfa, 0xd6, 0xba, 0xe1, 0xe1, 0xc0, 0x1d, 0xb3, 0xbc, 0x5a, 0xfe, 0x58, 0x47,
	0xe4, 0x76, 0x1d, 0x91, 0x5f, 0xeb, 0x88, 0x7c, 0xdd, 0x44, 0xce, 0xed, 0x26, 0x72, 0x7e, 0x6e,
	0x22, 0xe7, 0xfd, 0xb3, 0x0a, 0x75, 0xdd, 0xe6, 0x71, 0x21, 0x9a, 0x24, 0xc7, 0x8c, 0x7f, 0x40,
	0x96, 0x61, 0x82, 0x85, 0x7a, 0x79, 0xf2, 0xfc, 0xb8, 0x64, 0x8d, 0x48, 0xae, 0x13, 0x73, 0xec,
	0x31, 0xbf, 0xd4, 0xf9, 0xcc, 0xfc, 0x6a, 0x2f, 0xfe, 0x04, 0x00, 0x00, 0xff, 0xff, 0x65, 0xf4,
	0xd8, 0xbd, 0x9f, 0x02, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	// Register defines a rpc handler for MsgIssueClass
	IssueClass(ctx context.Context, in *MsgIssueClass, opts ...grpc.CallOption) (*MsgIssueClassResponse, error)
	// MintNFT defines a rpc handler for MsgMintNFT
	MintNFT(ctx context.Context, in *MsgMintNFT, opts ...grpc.CallOption) (*MintNFTResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) IssueClass(ctx context.Context, in *MsgIssueClass, opts ...grpc.CallOption) (*MsgIssueClassResponse, error) {
	out := new(MsgIssueClassResponse)
	err := c.cc.Invoke(ctx, "/internft.Msg/IssueClass", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) MintNFT(ctx context.Context, in *MsgMintNFT, opts ...grpc.CallOption) (*MintNFTResponse, error) {
	out := new(MintNFTResponse)
	err := c.cc.Invoke(ctx, "/internft.Msg/MintNFT", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// Register defines a rpc handler for MsgIssueClass
	IssueClass(context.Context, *MsgIssueClass) (*MsgIssueClassResponse, error)
	// MintNFT defines a rpc handler for MsgMintNFT
	MintNFT(context.Context, *MsgMintNFT) (*MintNFTResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) IssueClass(ctx context.Context, req *MsgIssueClass) (*MsgIssueClassResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IssueClass not implemented")
}
func (*UnimplementedMsgServer) MintNFT(ctx context.Context, req *MsgMintNFT) (*MintNFTResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MintNFT not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_IssueClass_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgIssueClass)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).IssueClass(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/internft.Msg/IssueClass",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).IssueClass(ctx, req.(*MsgIssueClass))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_MintNFT_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgMintNFT)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).MintNFT(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/internft.Msg/MintNFT",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).MintNFT(ctx, req.(*MsgMintNFT))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "internft.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "IssueClass",
			Handler:    _Msg_IssueClass_Handler,
		},
		{
			MethodName: "MintNFT",
			Handler:    _Msg_MintNFT_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "internft/tx.proto",
}

func (m *MsgIssueClass) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgIssueClass) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgIssueClass) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Issuer) > 0 {
		i -= len(m.Issuer)
		copy(dAtA[i:], m.Issuer)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Issuer)))
		i--
		dAtA[i] = 0x3a
	}
	if len(m.UriHash) > 0 {
		i -= len(m.UriHash)
		copy(dAtA[i:], m.UriHash)
		i = encodeVarintTx(dAtA, i, uint64(len(m.UriHash)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.Uri) > 0 {
		i -= len(m.Uri)
		copy(dAtA[i:], m.Uri)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Uri)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Symbol) > 0 {
		i -= len(m.Symbol)
		copy(dAtA[i:], m.Symbol)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Symbol)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgIssueClassResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgIssueClassResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgIssueClassResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgMintNFT) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgMintNFT) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgMintNFT) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Receiver) > 0 {
		i -= len(m.Receiver)
		copy(dAtA[i:], m.Receiver)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Receiver)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.Minter) > 0 {
		i -= len(m.Minter)
		copy(dAtA[i:], m.Minter)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Minter)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.UriHash) > 0 {
		i -= len(m.UriHash)
		copy(dAtA[i:], m.UriHash)
		i = encodeVarintTx(dAtA, i, uint64(len(m.UriHash)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Uri) > 0 {
		i -= len(m.Uri)
		copy(dAtA[i:], m.Uri)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Uri)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ClassId) > 0 {
		i -= len(m.ClassId)
		copy(dAtA[i:], m.ClassId)
		i = encodeVarintTx(dAtA, i, uint64(len(m.ClassId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MintNFTResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MintNFTResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MintNFTResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgIssueClass) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Symbol)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Uri)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.UriHash)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Issuer)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgIssueClassResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgMintNFT) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ClassId)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Uri)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.UriHash)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Minter)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Receiver)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MintNFTResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgIssueClass) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgIssueClass: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgIssueClass: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Symbol", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Symbol = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Uri", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Uri = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field UriHash", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.UriHash = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Issuer", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Issuer = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgIssueClassResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgIssueClassResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgIssueClassResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgMintNFT) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgMintNFT: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgMintNFT: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClassId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClassId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Uri", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Uri = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field UriHash", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.UriHash = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Minter", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Minter = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Receiver", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Receiver = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MintNFTResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MintNFTResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MintNFTResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)