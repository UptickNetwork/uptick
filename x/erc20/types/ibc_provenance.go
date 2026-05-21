package types

import "fmt"

// IBCTransferProvenanceKey builds a deterministic store key for an outbound
// ERC20-originated ICS-20 transfer. The key binds port, channel, sequence, sender,
// denom, and amount so memo substrings cannot authorize refunds.
func IBCTransferProvenanceKey(
	sourcePort, sourceChannel string,
	sequence uint64,
	sender, denom, amount string,
) []byte {
	return []byte(fmt.Sprintf(
		"%s/%s/%d/%s/%s/%s",
		sourcePort,
		sourceChannel,
		sequence,
		sender,
		denom,
		amount,
	))
}
