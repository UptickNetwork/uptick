package types

import (
	"fmt"
	"io"
	"math/bits"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func init() {
	proto.RegisterType((*MsgRegisterAssessmentSigner)(nil), "uptick.appmining.v1.MsgRegisterAssessmentSigner")
	proto.RegisterType((*MsgRegisterAssessmentSignerResponse)(nil), "uptick.appmining.v1.MsgRegisterAssessmentSignerResponse")
	proto.RegisterType((*MsgSetAssessmentSignerStatus)(nil), "uptick.appmining.v1.MsgSetAssessmentSignerStatus")
	proto.RegisterType((*MsgSetAssessmentSignerStatusResponse)(nil), "uptick.appmining.v1.MsgSetAssessmentSignerStatusResponse")
}

// MsgRegisterAssessmentSigner registers or rotates an assessment Ed25519 pubkey (authority).
type MsgRegisterAssessmentSigner struct {
	Authority string `protobuf:"bytes,1,opt,name=authority,proto3" json:"authority,omitempty"`
	KeyId     string `protobuf:"bytes,2,opt,name=key_id,json=keyId,proto3" json:"key_id,omitempty"`
	Pubkey    []byte `protobuf:"bytes,3,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	Version   uint64 `protobuf:"varint,4,opt,name=version,proto3" json:"version,omitempty"`
}

func (m *MsgRegisterAssessmentSigner) Reset()         { *m = MsgRegisterAssessmentSigner{} }
func (m *MsgRegisterAssessmentSigner) String() string { return fmt.Sprintf("%v", *m) }
func (*MsgRegisterAssessmentSigner) ProtoMessage()    {}
func (m *MsgRegisterAssessmentSigner) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgRegisterAssessmentSigner) ValidateBasic() error {
	if err := ValidateAddress(m.Authority); err != nil {
		return err
	}
	if err := ValidateAssessmentSignerKeyID(m.KeyId); err != nil {
		return err
	}
	if err := ValidateEd25519PubKey(m.Pubkey); err != nil {
		return err
	}
	return nil
}

type MsgRegisterAssessmentSignerResponse struct{}

func (m *MsgRegisterAssessmentSignerResponse) Reset()         { *m = MsgRegisterAssessmentSignerResponse{} }
func (m *MsgRegisterAssessmentSignerResponse) String() string { return "" }
func (*MsgRegisterAssessmentSignerResponse) ProtoMessage()    {}

// MsgSetAssessmentSignerStatus pauses or revokes an assessment signer (authority).
type MsgSetAssessmentSignerStatus struct {
	Authority string        `protobuf:"bytes,1,opt,name=authority,proto3" json:"authority,omitempty"`
	KeyId     string        `protobuf:"bytes,2,opt,name=key_id,json=keyId,proto3" json:"key_id,omitempty"`
	Status    RoleKeyStatus `protobuf:"varint,3,opt,name=status,proto3,enum=uptick.appmining.v1.RoleKeyStatus" json:"status,omitempty"`
}

func (m *MsgSetAssessmentSignerStatus) Reset()         { *m = MsgSetAssessmentSignerStatus{} }
func (m *MsgSetAssessmentSignerStatus) String() string { return fmt.Sprintf("%v", *m) }
func (*MsgSetAssessmentSignerStatus) ProtoMessage()    {}
func (m *MsgSetAssessmentSignerStatus) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgSetAssessmentSignerStatus) ValidateBasic() error {
	if err := ValidateAddress(m.Authority); err != nil {
		return err
	}
	if err := ValidateAssessmentSignerKeyID(m.KeyId); err != nil {
		return err
	}
	if m.Status == RoleKeyStatusUnspecified {
		return ErrInvalidRoleKeyStatus
	}
	return nil
}

type MsgSetAssessmentSignerStatusResponse struct{}

func (m *MsgSetAssessmentSignerStatusResponse) Reset() {
	*m = MsgSetAssessmentSignerStatusResponse{}
}
func (m *MsgSetAssessmentSignerStatusResponse) String() string { return "" }
func (*MsgSetAssessmentSignerStatusResponse) ProtoMessage()    {}

// --- protobuf codecs (hand-maintained; Docker proto-gen unavailable) ---

func (m *MsgRegisterAssessmentSigner) Marshal() ([]byte, error) {
	size := m.Size()
	dAtA := make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}
func (m *MsgRegisterAssessmentSigner) MarshalTo(dAtA []byte) (int, error) {
	return m.MarshalToSizedBuffer(dAtA[:m.Size()])
}
func (m *MsgRegisterAssessmentSigner) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.Version != 0 {
		i = encodeVarintAssessmentMsg(dAtA, i, m.Version)
		i--
		dAtA[i] = 0x20
	}
	if len(m.Pubkey) > 0 {
		i -= len(m.Pubkey)
		copy(dAtA[i:], m.Pubkey)
		i = encodeVarintAssessmentMsg(dAtA, i, uint64(len(m.Pubkey)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.KeyId) > 0 {
		i -= len(m.KeyId)
		copy(dAtA[i:], m.KeyId)
		i = encodeVarintAssessmentMsg(dAtA, i, uint64(len(m.KeyId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Authority) > 0 {
		i -= len(m.Authority)
		copy(dAtA[i:], m.Authority)
		i = encodeVarintAssessmentMsg(dAtA, i, uint64(len(m.Authority)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}
func (m *MsgRegisterAssessmentSigner) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.Authority)
	if l > 0 {
		n += 1 + l + sovAssessmentMsg(uint64(l))
	}
	l = len(m.KeyId)
	if l > 0 {
		n += 1 + l + sovAssessmentMsg(uint64(l))
	}
	l = len(m.Pubkey)
	if l > 0 {
		n += 1 + l + sovAssessmentMsg(uint64(l))
	}
	if m.Version != 0 {
		n += 1 + sovAssessmentMsg(m.Version)
	}
	return n
}
func (m *MsgRegisterAssessmentSigner) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAssessment
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
			return fmt.Errorf("proto: MsgRegisterAssessmentSigner: wiretype end group")
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType for Authority")
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAssessment
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
			postIndex := iNdEx + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Authority = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType for KeyId")
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAssessment
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
			postIndex := iNdEx + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.KeyId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType for Pubkey")
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAssessment
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Pubkey = append(m.Pubkey[:0], dAtA[iNdEx:postIndex]...)
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType for Version")
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAssessment
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Version |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipAssessment(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			iNdEx += skippy
		}
	}
	return nil
}

func (m *MsgRegisterAssessmentSignerResponse) Marshal() ([]byte, error) { return []byte{}, nil }
func (m *MsgRegisterAssessmentSignerResponse) MarshalTo(dAtA []byte) (int, error) {
	return 0, nil
}
func (m *MsgRegisterAssessmentSignerResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	return 0, nil
}
func (m *MsgRegisterAssessmentSignerResponse) Size() int { return 0 }
func (m *MsgRegisterAssessmentSignerResponse) Unmarshal(dAtA []byte) error {
	return nil
}

func (m *MsgSetAssessmentSignerStatus) Marshal() ([]byte, error) {
	size := m.Size()
	dAtA := make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}
func (m *MsgSetAssessmentSignerStatus) MarshalTo(dAtA []byte) (int, error) {
	return m.MarshalToSizedBuffer(dAtA[:m.Size()])
}
func (m *MsgSetAssessmentSignerStatus) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.Status != 0 {
		i = encodeVarintAssessmentMsg(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x18
	}
	if len(m.KeyId) > 0 {
		i -= len(m.KeyId)
		copy(dAtA[i:], m.KeyId)
		i = encodeVarintAssessmentMsg(dAtA, i, uint64(len(m.KeyId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Authority) > 0 {
		i -= len(m.Authority)
		copy(dAtA[i:], m.Authority)
		i = encodeVarintAssessmentMsg(dAtA, i, uint64(len(m.Authority)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}
func (m *MsgSetAssessmentSignerStatus) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.Authority)
	if l > 0 {
		n += 1 + l + sovAssessmentMsg(uint64(l))
	}
	l = len(m.KeyId)
	if l > 0 {
		n += 1 + l + sovAssessmentMsg(uint64(l))
	}
	if m.Status != 0 {
		n += 1 + sovAssessmentMsg(uint64(m.Status))
	}
	return n
}
func (m *MsgSetAssessmentSignerStatus) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAssessment
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
				return fmt.Errorf("proto: wrong wireType for Authority")
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAssessment
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
			postIndex := iNdEx + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Authority = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType for KeyId")
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAssessment
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
			postIndex := iNdEx + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.KeyId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType for Status")
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAssessment
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= RoleKeyStatus(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipAssessment(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			iNdEx += skippy
		}
	}
	return nil
}

func (m *MsgSetAssessmentSignerStatusResponse) Marshal() ([]byte, error) { return []byte{}, nil }
func (m *MsgSetAssessmentSignerStatusResponse) MarshalTo(dAtA []byte) (int, error) {
	return 0, nil
}
func (m *MsgSetAssessmentSignerStatusResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	return 0, nil
}
func (m *MsgSetAssessmentSignerStatusResponse) Size() int { return 0 }
func (m *MsgSetAssessmentSignerStatusResponse) Unmarshal(dAtA []byte) error {
	return nil
}

func encodeVarintAssessmentMsg(dAtA []byte, offset int, v uint64) int {
	offset -= sovAssessmentMsg(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		offset++
		v >>= 7
	}
	dAtA[offset] = uint8(v)
	return base
}

func sovAssessmentMsg(x uint64) int {
	return (bits.Len64(x|1) + 6) / 7
}

var (
	_ sdk.Msg = &MsgRegisterAssessmentSigner{}
	_ sdk.Msg = &MsgSetAssessmentSignerStatus{}
)
