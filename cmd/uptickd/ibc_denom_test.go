package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The first vector is the ecosystem's well-known ICS-20 denom for uatom
// arriving over transfer/channel-0 (the ATOM voucher minted on Osmosis and
// friends). It is published outside this repository, so this checks the hash we
// derive against the network rather than against our own helper.
func TestIBCDenom(t *testing.T) {
	t.Run("known ICS-20 vector", func(t *testing.T) {
		got, err := IBCDenom("transfer", "channel-0", "uatom")
		require.NoError(t, err)
		require.Equal(t,
			"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2", got)
	})

	t.Run("an already-prefixed denom is parsed, not concatenated", func(t *testing.T) {
		// The counterparty denom carries its own hop. The voucher hash must
		// cover the local hop followed by the imported path, in that order --
		// which only the parser gets right.
		got, err := IBCDenom("transfer", "channel-1", "transfer/channel-0/uatom")
		require.NoError(t, err)
		require.Equal(t,
			"ibc/FA0006F056DB6719B8C16C551FC392B62F5729978FC0B125AC9A432DBB2AA1A5", got)
	})

	// Before the fix this returned "'transfer/chan0/uatom' is a native denom":
	// the channel id was not recognised as a hop, the parse produced no trace,
	// and the operator was told an unrelated thing about their input.
	t.Run("a malformed channel id is reported as such", func(t *testing.T) {
		_, err := IBCDenom("transfer", "chan0", "uatom")
		require.ErrorContains(t, err, "invalid port/channel")
	})

	t.Run("an empty denom is rejected", func(t *testing.T) {
		_, err := IBCDenom("transfer", "channel-0", "")
		require.ErrorContains(t, err, "denom cannot be empty")
	})
}
