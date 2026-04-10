package simulation

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenNFTID(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	id := genNFTID(r, 3, 16)
	require.GreaterOrEqual(t, len(id), 3)
	require.Less(t, len(id), 16)
	require.Equal(t, strings.ToLower(id), id)
}

func TestGenDenomID(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	id := genDenomID(r)
	require.NotEmpty(t, id)
	require.Equal(t, strings.ToLower(id), id)
	require.NotContains(t, id, "ibc")
}

func TestRandData(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	got := randData(r)
	require.Contains(t, data, got)
}

func TestGenRandomBool(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	seenTrue := false
	seenFalse := false
	for i := 0; i < 20; i++ {
		v := genRandomBool(r)
		if v {
			seenTrue = true
		} else {
			seenFalse = true
		}
	}
	require.True(t, seenTrue)
	require.True(t, seenFalse)
}
