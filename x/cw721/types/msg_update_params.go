package types

import (
	fmt "fmt"
	io "io"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

const TypeMsgUpdateParams = "cw721/MsgUpdateParams"

var (
	_ sdk.Msg              = &MsgUpdateParams{}
	_ sdk.HasValidateBasic = &MsgUpdateParams{}
	_ proto.Message        = &MsgUpdateParams{}
)

// MsgUpdateParams is the governance message to update cw721 params.
type MsgUpdateParams struct {
	Authority string `protobuf:"bytes,1,opt,name=authority,proto3" json:"authority,omitempty"`
	Params    Params `protobuf:"bytes,2,opt,name=params,proto3" json:"params"`
}

func (m *MsgUpdateParams) Reset()         { *m = MsgUpdateParams{} }
func (m *MsgUpdateParams) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParams) ProtoMessage()    {}

func (m *MsgUpdateParams) XXX_Unmarshal(b []byte) error { return m.Unmarshal(b) }
func (m *MsgUpdateParams) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return nil, fmt.Errorf("deterministic marshal not supported")
	}
	return m.Marshal()
}
func (m *MsgUpdateParams) XXX_Merge(src proto.Message) {}
func (m *MsgUpdateParams) XXX_Size() int               { return m.Size() }
func (m *MsgUpdateParams) XXX_DiscardUnknown()         {}

func (m *MsgUpdateParams) Route() string { return RouterKey }
func (m *MsgUpdateParams) Type() string  { return TypeMsgUpdateParams }

func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return sdkerrors.Wrap(err, "invalid authority")
	}
	return m.Params.Validate()
}

func (m *MsgUpdateParams) GetAuthority() string {
	if m != nil {
		return m.Authority
	}
	return ""
}

type MsgUpdateParamsResponse struct{}

func (m *MsgUpdateParamsResponse) Reset()                       { *m = MsgUpdateParamsResponse{} }
func (m *MsgUpdateParamsResponse) String() string               { return proto.CompactTextString(m) }
func (*MsgUpdateParamsResponse) ProtoMessage()                  {}
func (m *MsgUpdateParamsResponse) XXX_Unmarshal(b []byte) error { return nil }
func (m *MsgUpdateParamsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return []byte{}, nil
}
func (m *MsgUpdateParamsResponse) XXX_Merge(src proto.Message) {}
func (m *MsgUpdateParamsResponse) XXX_Size() int               { return 0 }
func (m *MsgUpdateParamsResponse) XXX_DiscardUnknown()         {}
func (m *MsgUpdateParamsResponse) Marshal() ([]byte, error)    { return []byte{}, nil }
func (m *MsgUpdateParamsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	return 0, nil
}
func (m *MsgUpdateParamsResponse) Size() int                   { return 0 }
func (m *MsgUpdateParamsResponse) Unmarshal(dAtA []byte) error { return nil }

func init() {
	proto.RegisterType((*MsgUpdateParams)(nil), "uptick.cw721.v1.MsgUpdateParams")
	proto.RegisterType((*MsgUpdateParamsResponse)(nil), "uptick.cw721.v1.MsgUpdateParamsResponse")
}

func (m *MsgUpdateParams) Marshal() ([]byte, error) {
	size := m.Size()
	dAtA := make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateParams) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Authority) > 0 {
		i -= len(m.Authority)
		copy(dAtA[i:], m.Authority)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Authority)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgUpdateParams) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.Authority)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Params.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgUpdateParams) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return fmt.Errorf("int overflow")
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
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("wrong wireType = %d for field Authority", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return fmt.Errorf("int overflow")
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
			postIndex := iNdEx + intStringLen
			if postIndex < 0 || postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Authority = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return fmt.Errorf("int overflow")
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 || postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			iNdEx += skippy
		}
	}
	return nil
}
