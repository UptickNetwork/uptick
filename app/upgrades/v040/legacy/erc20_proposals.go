package legacy

import (
	"fmt"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/gogoproto/proto"
)

// Legacy ERC20 governance proposal types carried over from v0.3.3.
//
// v0.4.0 removed the self-developed x/erc20 module, but proposals created before
// the upgrade remain in x/gov state. `uptickd export` therefore fails with
// "no concrete type registered for type URL /uptick.erc20.v1.RegisterERC20Proposal".
// These minimal types keep the old binary-encoded proposals readable after the
// upgrade without resurrecting the module logic.

const (
	legacyERC20Router                 = "erc20"
	legacyProposalTypeRegisterCoin    = "RegisterCoin"
	legacyProposalTypeRegisterERC20   = "RegisterERC20"
	legacyProposalTypeToggleRelay     = "ToggleTokenRelay"
	legacyProposalTypeUpdateTokenPair = "UpdateTokenPairERC20"
)

var (
	_ proto.Message      = (*RegisterCoinProposal)(nil)
	_ proto.Message      = (*RegisterERC20Proposal)(nil)
	_ proto.Message      = (*ToggleTokenRelayProposal)(nil)
	_ proto.Message      = (*UpdateTokenPairERC20Proposal)(nil)
	_ govv1beta1.Content = (*RegisterCoinProposal)(nil)
	_ govv1beta1.Content = (*RegisterERC20Proposal)(nil)
	_ govv1beta1.Content = (*ToggleTokenRelayProposal)(nil)
	_ govv1beta1.Content = (*UpdateTokenPairERC20Proposal)(nil)
)

func init() {
	proto.RegisterType((*RegisterCoinProposal)(nil), "uptick.erc20.v1.RegisterCoinProposal")
	proto.RegisterType((*RegisterERC20Proposal)(nil), "uptick.erc20.v1.RegisterERC20Proposal")
	proto.RegisterType((*ToggleTokenRelayProposal)(nil), "uptick.erc20.v1.ToggleTokenRelayProposal")
	proto.RegisterType((*UpdateTokenPairERC20Proposal)(nil), "uptick.erc20.v1.UpdateTokenPairERC20Proposal")
}

// RegisterCoinProposal is the legacy proposal used to register a Cosmos coin
// with the old uptick x/erc20 module.
type RegisterCoinProposal struct {
	Title       string             `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Description string             `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	Metadata    banktypes.Metadata `protobuf:"bytes,3,opt,name=metadata,proto3" json:"metadata"`
}

func (m *RegisterCoinProposal) Reset() { *m = RegisterCoinProposal{} }
func (m *RegisterCoinProposal) String() string {
	return fmt.Sprintf("RegisterCoinProposal{Title:%s Description:%s}", m.Title, m.Description)
}
func (*RegisterCoinProposal) ProtoMessage() {}

func (m *RegisterCoinProposal) GetTitle() string       { return m.Title }
func (m *RegisterCoinProposal) GetDescription() string { return m.Description }
func (*RegisterCoinProposal) ProposalRoute() string    { return legacyERC20Router }
func (*RegisterCoinProposal) ProposalType() string     { return legacyProposalTypeRegisterCoin }
func (m *RegisterCoinProposal) ValidateBasic() error   { return govv1beta1.ValidateAbstract(m) }

// RegisterERC20Proposal is the legacy proposal used to register an external
// ERC20 contract with the old uptick x/erc20 module.
type RegisterERC20Proposal struct {
	Title        string `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Description  string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	Erc20Address string `protobuf:"bytes,3,opt,name=erc20address,proto3" json:"erc20address,omitempty"`
}

func (m *RegisterERC20Proposal) Reset() { *m = RegisterERC20Proposal{} }
func (m *RegisterERC20Proposal) String() string {
	return fmt.Sprintf("RegisterERC20Proposal{Title:%s Description:%s Erc20Address:%s}", m.Title, m.Description, m.Erc20Address)
}
func (*RegisterERC20Proposal) ProtoMessage() {}

func (m *RegisterERC20Proposal) GetTitle() string       { return m.Title }
func (m *RegisterERC20Proposal) GetDescription() string { return m.Description }
func (*RegisterERC20Proposal) ProposalRoute() string    { return legacyERC20Router }
func (*RegisterERC20Proposal) ProposalType() string     { return legacyProposalTypeRegisterERC20 }
func (m *RegisterERC20Proposal) ValidateBasic() error   { return govv1beta1.ValidateAbstract(m) }

// ToggleTokenRelayProposal is the legacy proposal used to toggle relaying of a
// token pair in the old uptick x/erc20 module.
type ToggleTokenRelayProposal struct {
	Title       string `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	Token       string `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`
}

func (m *ToggleTokenRelayProposal) Reset() { *m = ToggleTokenRelayProposal{} }
func (m *ToggleTokenRelayProposal) String() string {
	return fmt.Sprintf("ToggleTokenRelayProposal{Title:%s Description:%s Token:%s}", m.Title, m.Description, m.Token)
}
func (*ToggleTokenRelayProposal) ProtoMessage() {}

func (m *ToggleTokenRelayProposal) GetTitle() string       { return m.Title }
func (m *ToggleTokenRelayProposal) GetDescription() string { return m.Description }
func (*ToggleTokenRelayProposal) ProposalRoute() string    { return legacyERC20Router }
func (*ToggleTokenRelayProposal) ProposalType() string     { return legacyProposalTypeToggleRelay }
func (m *ToggleTokenRelayProposal) ValidateBasic() error   { return govv1beta1.ValidateAbstract(m) }

// UpdateTokenPairERC20Proposal is the legacy proposal used to update the ERC20
// contract address of a token pair in the old uptick x/erc20 module.
type UpdateTokenPairERC20Proposal struct {
	Title           string `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Description     string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	Erc20Address    string `protobuf:"bytes,3,opt,name=erc20_address,json=erc20Address,proto3" json:"erc20_address,omitempty"`
	NewErc20Address string `protobuf:"bytes,4,opt,name=new_erc20_address,json=newErc20Address,proto3" json:"new_erc20_address,omitempty"`
}

func (m *UpdateTokenPairERC20Proposal) Reset() { *m = UpdateTokenPairERC20Proposal{} }
func (m *UpdateTokenPairERC20Proposal) String() string {
	return fmt.Sprintf("UpdateTokenPairERC20Proposal{Title:%s Description:%s Erc20Address:%s NewErc20Address:%s}", m.Title, m.Description, m.Erc20Address, m.NewErc20Address)
}
func (*UpdateTokenPairERC20Proposal) ProtoMessage() {}

func (m *UpdateTokenPairERC20Proposal) GetTitle() string       { return m.Title }
func (m *UpdateTokenPairERC20Proposal) GetDescription() string { return m.Description }
func (*UpdateTokenPairERC20Proposal) ProposalRoute() string    { return legacyERC20Router }
func (*UpdateTokenPairERC20Proposal) ProposalType() string     { return legacyProposalTypeUpdateTokenPair }
func (m *UpdateTokenPairERC20Proposal) ValidateBasic() error   { return govv1beta1.ValidateAbstract(m) }

// ---- proto wire helpers ----

func appendLegacyProposalVarint(dAtA []byte, v uint64) []byte {
	for v >= 1<<7 {
		dAtA = append(dAtA, byte(v&0x7f|0x80))
		v >>= 7
	}
	return append(dAtA, byte(v))
}

func appendLegacyProposalTag(dAtA []byte, field, wire int) []byte {
	return appendLegacyProposalVarint(dAtA, uint64(field<<3|wire))
}

func appendLegacyProposalString(dAtA []byte, field int, s string) []byte {
	if len(s) == 0 {
		return dAtA
	}
	dAtA = appendLegacyProposalTag(dAtA, field, 2)
	dAtA = appendLegacyProposalVarint(dAtA, uint64(len(s)))
	return append(dAtA, s...)
}

func appendLegacyProposalBytes(dAtA []byte, field int, bz []byte) []byte {
	if len(bz) == 0 {
		return dAtA
	}
	dAtA = appendLegacyProposalTag(dAtA, field, 2)
	dAtA = appendLegacyProposalVarint(dAtA, uint64(len(bz)))
	return append(dAtA, bz...)
}

func legacyProposalUvarintSize(v uint64) int {
	n := 0
	for {
		n++
		v >>= 7
		if v == 0 {
			break
		}
	}
	return n
}

func legacyProposalStringSize(field int, s string) int {
	if len(s) == 0 {
		return 0
	}
	return legacyProposalUvarintSize(uint64(field<<3|2)) + legacyProposalUvarintSize(uint64(len(s))) + len(s)
}

func legacyProposalBytesSize(field int, bz []byte) int {
	if len(bz) == 0 {
		return 0
	}
	return legacyProposalUvarintSize(uint64(field<<3|2)) + legacyProposalUvarintSize(uint64(len(bz))) + len(bz)
}

func consumeLegacyProposalVarint(dAtA []byte, i int) (uint64, int, error) {
	var v uint64
	for shift := uint(0); ; shift += 7 {
		if i >= len(dAtA) {
			return 0, 0, fmt.Errorf("proto: unexpected EOF")
		}
		b := dAtA[i]
		i++
		v |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return v, i, nil
		}
	}
}

func skipLegacyProposal(dAtA []byte, i int, wireType int) (int, error) {
	switch wireType {
	case 0:
		_, ni, err := consumeLegacyProposalVarint(dAtA, i)
		return ni, err
	case 1:
		if i+8 > len(dAtA) {
			return 0, fmt.Errorf("proto: unexpected EOF")
		}
		return i + 8, nil
	case 2:
		length, ni, err := consumeLegacyProposalVarint(dAtA, i)
		if err != nil {
			return 0, err
		}
		if ni+int(length) > len(dAtA) {
			return 0, fmt.Errorf("proto: unexpected EOF")
		}
		return ni + int(length), nil
	case 5:
		if i+4 > len(dAtA) {
			return 0, fmt.Errorf("proto: unexpected EOF")
		}
		return i + 4, nil
	default:
		return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
	}
}

// ---- proto marshal/unmarshal implementations ----

func (m *RegisterCoinProposal) Marshal() ([]byte, error) {
	size := m.Size()
	dAtA := make([]byte, 0, size)
	dAtA = appendLegacyProposalString(dAtA, 1, m.Title)
	dAtA = appendLegacyProposalString(dAtA, 2, m.Description)
	meta, err := proto.Marshal(&m.Metadata)
	if err != nil {
		return nil, err
	}
	dAtA = appendLegacyProposalBytes(dAtA, 3, meta)
	return dAtA, nil
}

func (m *RegisterCoinProposal) Size() int {
	n := 0
	n += legacyProposalStringSize(1, m.Title)
	n += legacyProposalStringSize(2, m.Description)
	if metaSize := proto.Size(&m.Metadata); metaSize > 0 {
		n += legacyProposalBytesSize(3, make([]byte, metaSize))
	}
	return n
}

func (m *RegisterCoinProposal) Unmarshal(dAtA []byte) error {
	i := 0
	for i < len(dAtA) {
		wire, ni, err := consumeLegacyProposalVarint(dAtA, i)
		if err != nil {
			return err
		}
		i = ni
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RegisterCoinProposal: wiretype end group for non-group")
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Title = string(dAtA[i : i+int(length)])
			i += int(length)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Description = string(dAtA[i : i+int(length)])
			i += int(length)
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Metadata", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			if err := proto.Unmarshal(dAtA[i:i+int(length)], &m.Metadata); err != nil {
				return err
			}
			i += int(length)
		default:
			ni, err := skipLegacyProposal(dAtA, i, wireType)
			if err != nil {
				return err
			}
			i = ni
		}
	}
	return nil
}

func (m *RegisterERC20Proposal) Marshal() ([]byte, error) {
	dAtA := make([]byte, 0, m.Size())
	dAtA = appendLegacyProposalString(dAtA, 1, m.Title)
	dAtA = appendLegacyProposalString(dAtA, 2, m.Description)
	dAtA = appendLegacyProposalString(dAtA, 3, m.Erc20Address)
	return dAtA, nil
}

func (m *RegisterERC20Proposal) Size() int {
	return legacyProposalStringSize(1, m.Title) + legacyProposalStringSize(2, m.Description) + legacyProposalStringSize(3, m.Erc20Address)
}

// gogo-protobuf generated code and must stay byte-symmetric per type.
//
//nolint:dupl // legacy hand-rolled proto decoding; the duplication mirrors
func (m *RegisterERC20Proposal) Unmarshal(dAtA []byte) error {
	i := 0
	for i < len(dAtA) {
		wire, ni, err := consumeLegacyProposalVarint(dAtA, i)
		if err != nil {
			return err
		}
		i = ni
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RegisterERC20Proposal: wiretype end group for non-group")
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Title = string(dAtA[i : i+int(length)])
			i += int(length)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Description = string(dAtA[i : i+int(length)])
			i += int(length)
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Erc20Address", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Erc20Address = string(dAtA[i : i+int(length)])
			i += int(length)
		default:
			ni, err := skipLegacyProposal(dAtA, i, wireType)
			if err != nil {
				return err
			}
			i = ni
		}
	}
	return nil
}

func (m *ToggleTokenRelayProposal) Marshal() ([]byte, error) {
	dAtA := make([]byte, 0, m.Size())
	dAtA = appendLegacyProposalString(dAtA, 1, m.Title)
	dAtA = appendLegacyProposalString(dAtA, 2, m.Description)
	dAtA = appendLegacyProposalString(dAtA, 3, m.Token)
	return dAtA, nil
}

func (m *ToggleTokenRelayProposal) Size() int {
	return legacyProposalStringSize(1, m.Title) + legacyProposalStringSize(2, m.Description) + legacyProposalStringSize(3, m.Token)
}

// gogo-protobuf generated code and must stay byte-symmetric per type.
//
//nolint:dupl // legacy hand-rolled proto decoding; the duplication mirrors
func (m *ToggleTokenRelayProposal) Unmarshal(dAtA []byte) error {
	i := 0
	for i < len(dAtA) {
		wire, ni, err := consumeLegacyProposalVarint(dAtA, i)
		if err != nil {
			return err
		}
		i = ni
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ToggleTokenRelayProposal: wiretype end group for non-group")
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Title = string(dAtA[i : i+int(length)])
			i += int(length)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Description = string(dAtA[i : i+int(length)])
			i += int(length)
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Token", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Token = string(dAtA[i : i+int(length)])
			i += int(length)
		default:
			ni, err := skipLegacyProposal(dAtA, i, wireType)
			if err != nil {
				return err
			}
			i = ni
		}
	}
	return nil
}

func (m *UpdateTokenPairERC20Proposal) Marshal() ([]byte, error) {
	dAtA := make([]byte, 0, m.Size())
	dAtA = appendLegacyProposalString(dAtA, 1, m.Title)
	dAtA = appendLegacyProposalString(dAtA, 2, m.Description)
	dAtA = appendLegacyProposalString(dAtA, 3, m.Erc20Address)
	dAtA = appendLegacyProposalString(dAtA, 4, m.NewErc20Address)
	return dAtA, nil
}

func (m *UpdateTokenPairERC20Proposal) Size() int {
	return legacyProposalStringSize(1, m.Title) +
		legacyProposalStringSize(2, m.Description) +
		legacyProposalStringSize(3, m.Erc20Address) +
		legacyProposalStringSize(4, m.NewErc20Address)
}

func (m *UpdateTokenPairERC20Proposal) Unmarshal(dAtA []byte) error {
	i := 0
	for i < len(dAtA) {
		wire, ni, err := consumeLegacyProposalVarint(dAtA, i)
		if err != nil {
			return err
		}
		i = ni
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: UpdateTokenPairERC20Proposal: wiretype end group for non-group")
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Title = string(dAtA[i : i+int(length)])
			i += int(length)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Description = string(dAtA[i : i+int(length)])
			i += int(length)
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Erc20Address", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.Erc20Address = string(dAtA[i : i+int(length)])
			i += int(length)
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NewErc20Address", wireType)
			}
			length, ni, err := consumeLegacyProposalVarint(dAtA, i)
			if err != nil {
				return err
			}
			i = ni
			if i+int(length) > len(dAtA) {
				return fmt.Errorf("proto: unexpected EOF")
			}
			m.NewErc20Address = string(dAtA[i : i+int(length)])
			i += int(length)
		default:
			ni, err := skipLegacyProposal(dAtA, i, wireType)
			if err != nil {
				return err
			}
			i = ni
		}
	}
	return nil
}
