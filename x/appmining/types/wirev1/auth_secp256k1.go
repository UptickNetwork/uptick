package wirev1

import (
	"encoding/base64"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// Secp256k1SigLength is r(32)||s(32)||v(1) for EIP-191 personal_sign.
	Secp256k1SigLength = 65
)

// secp256k1 half-order (n/2) for EIP-2 low-s.
var secp256k1HalfN = new(big.Int).Rsh(crypto.S256().Params().N, 1)

// VerifyAuthorizationSecp256k1 verifies an EIP-191 personal_sign over the wire
// frame bytes. expectedAddr must be a raw 20-byte EVM address.
// signatureBase64 decodes to 65 bytes with v in {27,28}; high-s is rejected.
//
// Returns "" on success; otherwise a stable reason token:
//
//	"sig_length" | "high_s" | "recover" | "mismatch" | "frame"
func VerifyAuthorizationSecp256k1(record map[string]string, signatureB64 string, expectedAddr []byte) string {
	if len(expectedAddr) != common.AddressLength {
		return "mismatch"
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != Secp256k1SigLength {
		return "sig_length"
	}
	if base64.StdEncoding.EncodeToString(sig) != signatureB64 {
		return "sig_length"
	}
	v := sig[64]
	if v != 27 && v != 28 {
		return "sig_length"
	}
	s := new(big.Int).SetBytes(sig[32:64])
	if s.Cmp(secp256k1HalfN) > 0 {
		return "high_s"
	}
	raw, err := AuthorizationBytes(record)
	if err != nil {
		return "frame"
	}
	hash := accounts.TextHash(raw)
	// go-ethereum expects v as 0/1 for SigToPub.
	sigCopy := append([]byte(nil), sig...)
	sigCopy[64] -= 27
	pub, err := crypto.SigToPub(hash, sigCopy)
	if err != nil {
		return "recover"
	}
	got := crypto.PubkeyToAddress(*pub)
	if got != common.BytesToAddress(expectedAddr) {
		return "mismatch"
	}
	return ""
}
