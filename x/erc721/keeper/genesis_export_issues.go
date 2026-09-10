package keeper

import (
	"fmt"
	"strings"
)

// GenesisExportIssueKind classifies one reason why a genesis export could not
// represent the live store faithfully.
type GenesisExportIssueKind string

const (
	// GenesisExportIssueTokenPairCorrupt means a registered TokenPair record
	// could not be decoded.
	GenesisExportIssueTokenPairCorrupt GenesisExportIssueKind = "token_pair_corrupt"
	// GenesisExportIssueUIDIndexBackward means a reverse (nftUID -> tokenUID)
	// index entry has no matching forward entry.
	GenesisExportIssueUIDIndexBackward GenesisExportIssueKind = "nft_uid_index_reverse_without_forward"
	// GenesisExportIssueUIDIndexForward means a forward (tokenUID -> nftUID)
	// index entry has no matching reverse entry.
	GenesisExportIssueUIDIndexForward GenesisExportIssueKind = "nft_uid_index_forward_without_reverse"
	// GenesisExportIssueRefundKeyOrphan means a refund receiver record does not
	// belong to any registered contract.
	GenesisExportIssueRefundKeyOrphan GenesisExportIssueKind = "refund_key_orphaned"
)

// GenesisExportIssue is one state record that cannot be exported faithfully.
// Key carries the store key so an operator can point at the exact record.
type GenesisExportIssue struct {
	Kind   GenesisExportIssueKind
	Key    string
	Detail string
}

func (i GenesisExportIssue) String() string {
	return fmt.Sprintf("%s key=%s: %s", i.Kind, i.Key, i.Detail)
}

// MaxReportedExportIssues caps how many individual records are spelled out in
// the error message. Every issue is still counted.
const MaxReportedExportIssues = 20

// GenesisExportError is the fail-closed error raised by ExportGenesis when the
// live store holds state the genesis file cannot represent.
//
// The Cosmos SDK module interface (AppModule.ExportGenesis(ctx, cdc)
// json.RawMessage) has no error channel, so an export cannot hand a structured
// error back to the CLI. Panicking with this value is therefore the only
// mechanism that makes the CLI fail (non-zero exit) AND prints the full,
// actionable list of damaged keys — which is strictly better than both the old
// bare panic (no diagnosis) and a silent skip (data loss).
type GenesisExportError struct {
	Module string
	Issues []GenesisExportIssue
}

func (e *GenesisExportError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b,
		"%s genesis export aborted: %d state record(s) cannot be represented in genesis; "+
			"repair or purge them (see the owning token pair) before exporting again",
		e.Module, len(e.Issues))

	shown := e.Issues
	if len(shown) > MaxReportedExportIssues {
		shown = shown[:MaxReportedExportIssues]
	}
	for _, issue := range shown {
		fmt.Fprintf(&b, "\n  - %s", issue.String())
	}
	if len(e.Issues) > len(shown) {
		fmt.Fprintf(&b, "\n  ... and %d more", len(e.Issues)-len(shown))
	}
	return b.String()
}

// issuesToError converts a non-empty issue list into a plain error, for the
// legacy (error-returning) accessors.
func issuesToError(module string, issues []GenesisExportIssue) error {
	return &GenesisExportError{Module: module, Issues: issues}
}
