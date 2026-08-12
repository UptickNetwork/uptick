package internft

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterClass_Getters(t *testing.T) {
	c := InterClass{
		ID:   "class-alpha",
		URI:  "https://metadata.uptick.network/class/alpha",
		Data: `{"name":"Alpha Collection","description":"Test"}`,
	}
	require.Equal(t, "class-alpha", c.GetID())
	require.Equal(t, "https://metadata.uptick.network/class/alpha", c.GetURI())
	require.Equal(t, `{"name":"Alpha Collection","description":"Test"}`, c.GetData())
}

func TestInterClass_Empty(t *testing.T) {
	c := InterClass{}
	require.Empty(t, c.GetID())
	require.Empty(t, c.GetURI())
	require.Empty(t, c.GetData())
}

func TestInterToken_Getters(t *testing.T) {
	token := InterToken{
		ClassID: "class-1",
		ID:      "token-42",
		URI:     "https://metadata.uptick.network/token/42",
		Data:    `{"attributes":[{"trait_type":"Color","value":"Red"}]}`,
	}
	require.Equal(t, "class-1", token.GetClassID())
	require.Equal(t, "token-42", token.GetID())
	require.Equal(t, "https://metadata.uptick.network/token/42", token.GetURI())
	require.Equal(t, `{"attributes":[{"trait_type":"Color","value":"Red"}]}`, token.GetData())
}

func TestInterToken_Empty(t *testing.T) {
	token := InterToken{}
	require.Empty(t, token.GetClassID())
	require.Empty(t, token.GetID())
	require.Empty(t, token.GetURI())
	require.Empty(t, token.GetData())
}

func TestInterNftKeeper_Fields(t *testing.T) {
	// Verify keeper struct can be created with zero values
	k := InterNftKeeper{}
	require.Nil(t, k.cdc)
	require.Nil(t, k.ak)
}

func TestInterClass_IBCClassType(t *testing.T) {
	// Verify InterClass satisfies the IBC class interface
	// (GetID, GetURI, GetData methods)
	var class interface {
		GetID() string
		GetURI() string
		GetData() string
	}

	c := InterClass{ID: "c1", URI: "uri", Data: "data"}
	class = c
	require.Equal(t, "c1", class.GetID())
	require.Equal(t, "uri", class.GetURI())
	require.Equal(t, "data", class.GetData())
}

func TestInterToken_IBCNFTType(t *testing.T) {
	// Verify InterToken satisfies the IBC NFT interface
	// (GetClassID, GetID, GetURI, GetData methods)
	var nft interface {
		GetClassID() string
		GetID() string
		GetURI() string
		GetData() string
	}

	token := InterToken{ClassID: "c1", ID: "t1", URI: "uri", Data: "data"}
	nft = token
	require.Equal(t, "c1", nft.GetClassID())
	require.Equal(t, "t1", nft.GetID())
	require.Equal(t, "uri", nft.GetURI())
	require.Equal(t, "data", nft.GetData())
}
