package appmining_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

type secp256k1GoldenFile struct {
	Vectors []secp256k1GoldenVector `json:"vectors"`
}

type secp256k1GoldenVector struct {
	Name             string            `json:"name"`
	Expect           string            `json:"expect"`
	Record           map[string]string `json:"record"`
	FrameSHA256      string            `json:"frame_sha256"`
	FrameLen         int               `json:"frame_len"`
	SignatureBase64  string            `json:"signature_base64"`
	ExpectedAddress  string            `json:"expected_address"`
}

func TestWireSecp256k1Golden_UGEInterop(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(thisFile), "spec", "appmining-v1-wire-secp256k1-golden-1.0.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var g secp256k1GoldenFile
	require.NoError(t, json.Unmarshal(raw, &g))
	require.Len(t, g.Vectors, 3)

	for _, v := range g.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			frame, err := wirev1.AuthorizationBytes(v.Record)
			require.NoError(t, err)
			require.Equal(t, v.FrameLen, len(frame))
			sum := sha256.Sum256(frame)
			require.Equal(t, v.FrameSHA256, hex.EncodeToString(sum[:]))

			addr := common.HexToAddress(v.ExpectedAddress)
			got := wirev1.VerifyAuthorizationSecp256k1(v.Record, v.SignatureBase64, addr.Bytes())
			switch v.Expect {
			case "ok":
				require.Equal(t, "", got)
			case "high_s", "sig_length":
				require.Equal(t, v.Expect, got)
			default:
				t.Fatalf("unknown expect %q", v.Expect)
			}
		})
	}
}
