package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassGettersNilReceiver(t *testing.T) {
	var c *Class
	require.Equal(t, "", c.GetId())
	require.Equal(t, "", c.GetName())
	require.Equal(t, "", c.GetSymbol())
	require.Equal(t, "", c.GetDescription())
	require.Equal(t, "", c.GetUri())
	require.Equal(t, "", c.GetUriHash())
	require.Nil(t, c.GetData())
}

func TestNFTGettersNilReceiver(t *testing.T) {
	var n *NFT
	require.Equal(t, "", n.GetClassId())
	require.Equal(t, "", n.GetId())
	require.Equal(t, "", n.GetUri())
	require.Equal(t, "", n.GetUriHash())
	require.Nil(t, n.GetData())
}
