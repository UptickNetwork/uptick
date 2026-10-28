package wirev1_test

import (
	"crypto/ecdsa"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

func samplePublishRecord() map[string]string {
	return map[string]string{
		"protocol_version": "1",
		"operation":        "PUBLISH",
		"role":             "publisher",
		"chain_id":         "uptick_7777-1",
		"pool_id":          "pool-1",
		"key_id":           "mm-pub-1",
		"nonce":            "1",
		"expires_at":       "1893456000",
		"entitlement_id":   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"app_id":           "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"candidate_hash":   "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"rule_version":     "gas-reward-candidate-0.1",
	}
}

func signPersonal(t *testing.T, priv *ecdsa.PrivateKey, frame []byte) string {
	t.Helper()
	hash := accounts.TextHash(frame)
	sig, err := crypto.Sign(hash, priv)
	require.NoError(t, err)
	require.Len(t, sig, 65)
	sig[64] += 27 // personal_sign v
	return base64.StdEncoding.EncodeToString(sig)
}

func TestVerifyAuthorizationSecp256k1_HappyAndNegatives(t *testing.T) {
	priv, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(priv.PublicKey)

	rec := samplePublishRecord()
	frame, err := wirev1.AuthorizationBytes(rec)
	require.NoError(t, err)
	sigB64 := signPersonal(t, priv, frame)

	require.Equal(t, "", wirev1.VerifyAuthorizationSecp256k1(rec, sigB64, addr.Bytes()))

	require.Equal(t, "mismatch", wirev1.VerifyAuthorizationSecp256k1(rec, sigB64, common.HexToAddress("0x0000000000000000000000000000000000000001").Bytes()))

	// high-s: flip s above half-n
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	require.NoError(t, err)
	s := new(big.Int).SetBytes(sig[32:64])
	n := crypto.S256().Params().N
	highS := new(big.Int).Sub(n, s)
	half := new(big.Int).Rsh(n, 1)
	if highS.Cmp(half) <= 0 {
		highS = new(big.Int).Add(highS, half)
		if highS.Cmp(n) >= 0 {
			highS.Sub(highS, half)
		}
	}
	hs := highS.FillBytes(make([]byte, 32))
	copy(sig[32:64], hs)
	highB64 := base64.StdEncoding.EncodeToString(sig)
	require.Equal(t, "high_s", wirev1.VerifyAuthorizationSecp256k1(rec, highB64, addr.Bytes()))

	require.Equal(t, "sig_length", wirev1.VerifyAuthorizationSecp256k1(rec, base64.StdEncoding.EncodeToString(sig[:64]), addr.Bytes()))
}
