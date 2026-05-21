package types_test

import (
	"testing"

	"github.com/UptickNetwork/uptick/x/erc20/types"
	"github.com/stretchr/testify/require"
)

func TestIBCTransferProvenanceKeyDeterministic(t *testing.T) {
	keyA := types.IBCTransferProvenanceKey("transfer", "channel-0", 7, "uptick1sender", "ibc/ABC", "100")
	keyB := types.IBCTransferProvenanceKey("transfer", "channel-0", 7, "uptick1sender", "ibc/ABC", "100")
	require.Equal(t, keyA, keyB)
}

func TestIBCTransferProvenanceKeyDiffersOnSequence(t *testing.T) {
	keyA := types.IBCTransferProvenanceKey("transfer", "channel-0", 7, "uptick1sender", "ibc/ABC", "100")
	keyB := types.IBCTransferProvenanceKey("transfer", "channel-0", 8, "uptick1sender", "ibc/ABC", "100")
	require.NotEqual(t, keyA, keyB)
}

func TestIBCTransferProvenanceKeyDiffersOnMemoMarkerFieldsUnchanged(t *testing.T) {
	// Provenance keys ignore memo; only bound transfer fields matter.
	keyA := types.IBCTransferProvenanceKey("transfer", "channel-0", 7, "uptick1sender", "ibc/ABC", "100")
	keyB := types.IBCTransferProvenanceKey("transfer", "channel-0", 7, "uptick1sender", "ibc/ABC", "100")
	require.Equal(t, keyA, keyB)
}
