package legacy

import (
	"bytes"
	"fmt"
	"io"

	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/crypto"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// compile-time assertions
var (
	_ proto.Message     = (*EthSecp256k1PubKey)(nil)
	_ proto.Marshaler   = (*EthSecp256k1PubKey)(nil)
	_ proto.Unmarshaler = (*EthSecp256k1PubKey)(nil)
	_ cryptotypes.PubKey = (*EthSecp256k1PubKey)(nil)
)

func init() {
	// register the legacy ethermint pubkey type so the v0.3.3 -> v0.4.0
	// account migration can decode BaseAccount.PubKey Any values, and so
	// signature verification keeps working for migrated accounts.
	proto.RegisterType((*EthSecp256k1PubKey)(nil), "ethermint.crypto.v1.ethsecp256k1.PubKey")
}

// EthSecp256k1PubKey is the legacy ethermint eth_secp256k1 public key,
// registered under /ethermint.crypto.v1.ethsecp256k1.PubKey solely for the
// v0.3.3 -> v0.4.0 account migration (decoding + verification of migrated keys).
type EthSecp256k1PubKey struct {
	// Key is the public key in compressed secp256k1 byte form (33 bytes).
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (m *EthSecp256k1PubKey) Reset()         { *m = EthSecp256k1PubKey{} }
func (m *EthSecp256k1PubKey) String() string { return fmt.Sprintf("EthPubKeySecp256k1{%X}", m.Key) }
func (*EthSecp256k1PubKey) ProtoMessage()    {}

// GetKey returns the raw compressed public key bytes.
func (m *EthSecp256k1PubKey) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

// Marshal implements proto.Marshaler.
func (m *EthSecp256k1PubKey) Marshal() ([]byte, error) {
	size := m.Size()
	dAtA := make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

// MarshalTo implements proto.Marshaler.
func (m *EthSecp256k1PubKey) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

// MarshalToSizedBuffer writes the key as proto3 bytes field 1.
func (m *EthSecp256k1PubKey) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	if len(m.Key) > 0 {
		i -= len(m.Key)
		copy(dAtA[i:], m.Key)
		i = encodeVarintLegacyKeys(dAtA, i, uint64(len(m.Key)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

// Size returns the serialized size.
func (m *EthSecp256k1PubKey) Size() int {
	if m == nil {
		return 0
	}
	var size int
	if len(m.Key) > 0 {
		l := len(m.Key)
		size += 1 + l + sovLegacyKeys(uint64(l))
	}
	return size
}

// Unmarshal implements proto.Unmarshaler.
func (m *EthSecp256k1PubKey) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLegacyKeys
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
			return fmt.Errorf("proto: EthSecp256k1PubKey: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EthSecp256k1PubKey: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Key", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLegacyKeys
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
			if byteLen < 0 {
				return ErrInvalidLengthLegacyKeys
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthLegacyKeys
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Key = append(m.Key[:0], dAtA[iNdEx:postIndex]...)
			if m.Key == nil {
				m.Key = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipLegacyKeys(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthLegacyKeys
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthLegacyKeys
			}
			iNdEx += skippy
		}
	}
	return nil
}

// XXX_* methods for gogoproto compatibility.
func (m *EthSecp256k1PubKey) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}

func (m *EthSecp256k1PubKey) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return m.Marshal()
}

func (m *EthSecp256k1PubKey) XXX_Merge(src proto.Message) {
	if other, ok := src.(*EthSecp256k1PubKey); ok {
		m.Key = append(m.Key[:0], other.Key...)
	}
}

func (m *EthSecp256k1PubKey) XXX_Size() int { return m.Size() }

func (m *EthSecp256k1PubKey) XXX_DiscardUnknown() {}

// Address returns the EVM address derived from the compressed pubkey.
func (m *EthSecp256k1PubKey) Address() cryptotypes.Address {
	pubk, err := crypto.DecompressPubkey(m.Key)
	if err != nil {
		return nil
	}
	return cryptotypes.Address(crypto.PubkeyToAddress(*pubk).Bytes())
}

// Bytes returns the raw compressed public key bytes.
func (m *EthSecp256k1PubKey) Bytes() []byte {
	bz := make([]byte, len(m.Key))
	copy(bz, m.Key)
	return bz
}

// Type returns the key algorithm name.
func (m *EthSecp256k1PubKey) Type() string { return "eth_secp256k1" }

// Equals reports whether two pubkeys are deeply equal.
func (m *EthSecp256k1PubKey) Equals(other cryptotypes.PubKey) bool {
	return m.Type() == other.Type() && bytes.Equal(m.Bytes(), other.Bytes())
}

// VerifySignature verifies an ECDSA signature ([R||S||V] or [R||S]) over the
// keccak256 hash of msg, matching the legacy ethermint signing scheme.
func (m *EthSecp256k1PubKey) VerifySignature(msg, sig []byte) bool {
	if len(sig) == crypto.SignatureLength {
		// drop recovery ID (V)
		sig = sig[:len(sig)-1]
	}
	return crypto.VerifySignature(m.Key, crypto.Keccak256Hash(msg).Bytes(), sig)
}

// ---- tiny protobuf helpers (ported from gogoproto generated code) ----

// ErrIntOverflowLegacyKeys is returned on varint overflow.
var ErrIntOverflowLegacyKeys = fmt.Errorf("proto: integer overflow")

// ErrInvalidLengthLegacyKeys is returned on invalid length.
var ErrInvalidLengthLegacyKeys = fmt.Errorf("proto: negative length found during unmarshaling")

func encodeVarintLegacyKeys(dAtA []byte, offset int, v uint64) int {
	offset -= sovLegacyKeys(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func sovLegacyKeys(x uint64) (n int) {
	return (int(bitsLen64(x)) + 6) / 7
}

func bitsLen64(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}

func skipLegacyKeys(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowLegacyKeys
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
					return 0, ErrIntOverflowLegacyKeys
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
					return 0, ErrIntOverflowLegacyKeys
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
				return 0, ErrInvalidLengthLegacyKeys
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, fmt.Errorf("proto: unexpected end group")
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthLegacyKeys
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}
