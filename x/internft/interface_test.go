package internft

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterClassGetters(t *testing.T) {
	c := InterClass{
		ID:   "class-1",
		URI:  "ipfs://class",
		Data: "data",
	}
	require.Equal(t, "class-1", c.GetID())
	require.Equal(t, "ipfs://class", c.GetURI())
	require.Equal(t, "data", c.GetData())
}

func TestInterTokenGetters(t *testing.T) {
	token := InterToken{
		ClassID: "class-1",
		ID:      "token-1",
		URI:     "ipfs://token",
		Data:    "data",
	}
	require.Equal(t, "class-1", token.GetClassID())
	require.Equal(t, "token-1", token.GetID())
	require.Equal(t, "ipfs://token", token.GetURI())
	require.Equal(t, "data", token.GetData())
}
