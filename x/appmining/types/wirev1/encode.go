package wirev1

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"
)

const (
	DomainAuth          = "UGE-APP-MINING-WIRE-V1"
	DomainCandidate     = "UGE-APP-MINING-CANDIDATE-WIRE-V1"
	DomainEntitlementID = "UGE-APP-MINING-CANDIDATE-V1"

	maxFieldUTF8Bytes = 4096
)

// frame encodes domain + NUL, then ordered fields as
// u16be(nameLen) || name || u32be(valueLen) || value (UTF-8).
// Matches UGE protocol/app-mining-v1/wire-v1.mjs frame().
func frame(domain string, record map[string]string, expected []string) ([]byte, error) {
	out := make([]byte, 0, 256)
	out = append(out, []byte(domain)...)
	out = append(out, 0x00)
	for _, name := range expected {
		value, ok := record[name]
		if !ok {
			return nil, fmt.Errorf("missing field %q", name)
		}
		// Empty values are allowed only when higher-level validators permit them
		// (UGE DEC-01: APPROVE challenge_resolution_hash may be "").
		if !utf8.ValidString(value) {
			return nil, fmt.Errorf("invalid %s", name)
		}
		key := []byte(name)
		val := []byte(value)
		if len(val) > maxFieldUTF8Bytes {
			return nil, fmt.Errorf("invalid %s", name)
		}
		var hdr [6]byte
		binary.BigEndian.PutUint16(hdr[0:2], uint16(len(key)))
		binary.BigEndian.PutUint32(hdr[2:6], uint32(len(val)))
		out = append(out, hdr[0:2]...)
		out = append(out, key...)
		out = append(out, hdr[2:6]...)
		out = append(out, val...)
	}
	return out, nil
}
