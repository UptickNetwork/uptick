package params

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// MakeEncodingConfigChecked already had an error path; the interface registry
// constructor's error was simply discarded with `_`, which turned a
// registration/SDK-compatibility defect into a nil-registry panic inside
// NewProtoCodec or RegisterInterfaces — far from the call site and with no
// indication that the registry was the problem. Guard the handling rather than
// the (hard to reach) failure: the constructor can only fail on a genuinely
// broken build.
func TestMakeEncodingConfigCheckedHandlesRegistryError(t *testing.T) {
	src, err := os.ReadFile("proto.go")
	require.NoError(t, err)

	const call = "types.NewInterfaceRegistryWithOptions("
	var seen bool
	for i, line := range strings.Split(string(src), "\n") {
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		if !strings.Contains(line, call) {
			continue
		}
		seen = true
		require.NotContains(t, line, ", _",
			"proto.go:%d discards a return value of the interface registry constructor; its error returns (nil, err) on failure and a nil registry panics inside NewProtoCodec", i+1)
	}
	require.True(t, seen, "the interface registry constructor moved; update this guard")
}

// The behavioural half: the checked constructor must hand back a fully wired
// config, so the error path cannot be "fixed" by returning an empty value.
func TestMakeEncodingConfigCheckedReturnsUsableConfig(t *testing.T) {
	enc, err := MakeEncodingConfigChecked()
	require.NoError(t, err)

	require.NotNil(t, enc.InterfaceRegistry)
	require.NotNil(t, enc.Codec)
	require.NotNil(t, enc.TxConfig)
	require.NotNil(t, enc.LegacyAmino)
}
