package v041

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
)

// fakeUIDIndexStore stands in for the ERC721 keeper, which cannot be asked to
// report damage it did not find: the prune only reports what is in the store, so
// the Error branch below is unreachable through the real keeper unless a test
// first writes corrupt state into it. Injecting the report keeps the branch that
// decides how an operator is told about residual damage covered in CI.
type fakeUIDIndexStore struct {
	report erc721keeper.UIDIndexPruneReport
}

func (s fakeUIDIndexStore) PruneDuplicateUIDIndexEntries(sdk.Context) erc721keeper.UIDIndexPruneReport {
	return s.report
}

// pruneTestCtx returns a context whose Logger() writes into buf as JSON, so a
// test can assert on the level a line was emitted at. sdk.Context{}.Logger()
// returns a nil logger whose Info() panics; the real handler always runs on a
// context that has one.
func pruneTestCtx(buf *bytes.Buffer) sdk.Context {
	return sdk.Context{}.WithLogger(log.NewLogger(buf, log.OutputJSONOption()))
}

// convergedReport is a report for an index that needed one deletion and is
// balanced afterwards: 5 forward keys, 4 reverse entries, 1 shadow removed.
func convergedReport() erc721keeper.UIDIndexPruneReport {
	return erc721keeper.UIDIndexPruneReport{
		Scanned:       4,
		ForwardBefore: 5,
		Deleted:       []string{"1703751205993357472,0x3bc44cb88233f75b858d0748a45d196be8375159"},
	}
}

func TestUIDIndexPruneClean(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		report erc721keeper.UIDIndexPruneReport
		want   bool
	}{
		{"converged after one deletion", convergedReport(), true},
		{"nothing to do at all", erc721keeper.UIDIndexPruneReport{Scanned: 3, ForwardBefore: 3}, true},
		{
			"a conflict alone is damage",
			erc721keeper.UIDIndexPruneReport{Scanned: 3, ForwardBefore: 4, Conflicts: []string{"a"}},
			false,
		},
		{
			"an orphan alone is damage",
			erc721keeper.UIDIndexPruneReport{Scanned: 3, ForwardBefore: 4, Orphans: []string{"a"}},
			false,
		},
		{
			"an unpaired reverse entry alone is damage",
			erc721keeper.UIDIndexPruneReport{Scanned: 4, ForwardBefore: 3, UnpairedNFTs: []string{"a"}},
			false,
		},
		{
			"an incomplete prune is damage",
			erc721keeper.UIDIndexPruneReport{Scanned: 3, ForwardBefore: 4, Deleted: []string{"a", "b"}},
			false,
		},
		{
			"pending work is not damage",
			erc721keeper.UIDIndexPruneReport{Scanned: 3, ForwardBefore: 3, Deleted: []string{"a"}},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, uidIndexPruneClean(tc.report))
		})
	}
}

// The reason the four conditions are listed explicitly instead of leaning on
// the key counts: damage can cancel out in the totals.
func TestUIDIndexPruneClean_CountsCanBalanceWhileTheStoreIsStillDamaged(t *testing.T) {
	t.Parallel()

	report := erc721keeper.UIDIndexPruneReport{
		Scanned:       4,
		ForwardBefore: 4,
		Orphans:       []string{"orphaned-forward-key"},
		UnpairedNFTs:  []string{"unpaired-reverse-entry"},
	}

	require.True(t, report.Balanced(),
		"an orphan adds a forward key and an unpaired reverse entry removes one, so the totals agree")
	require.False(t, uidIndexPruneClean(report),
		"yet two bindings are broken, so the prune must not report a clean index")
}

func TestFirstFew(t *testing.T) {
	t.Parallel()

	require.Empty(t, firstFew(nil))
	require.Equal(t, []string{"a", "b"}, firstFew([]string{"a", "b"}))
	require.Equal(t, []string{"a", "b", "c", "d", "e"},
		firstFew([]string{"a", "b", "c", "d", "e", "f", "g"}))
}

// A clean pass is the expected outcome on both live chains, so it must be
// reported at Info level and must not look like an incident.
func TestPruneErc721UIDIndex_LogsConvergenceAtInfoLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	pruneErc721UIDIndex(pruneTestCtx(&buf), fakeUIDIndexStore{report: convergedReport()})

	out := buf.String()
	require.Contains(t, out, "erc721 conversion index pruned")
	require.Contains(t, out, `"level":"info"`)
	require.NotContains(t, out, `"level":"error"`,
		"a converged index is not an error, and an operator paging on errors must not see one")
	require.Contains(t, out, `"deleted":1`)
	require.Contains(t, out, `"forward_keys_after":4`)
}

// The other half of the signal: residual damage has to be visible at Error
// level, with the counters an operator needs to size the problem. The pass
// still returns normally — an error here would halt the chain at the upgrade
// height over state this handler cannot repair.
func TestPruneErc721UIDIndex_LogsResidualDamageAtErrorLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	pruneErc721UIDIndex(pruneTestCtx(&buf), fakeUIDIndexStore{report: erc721keeper.UIDIndexPruneReport{
		Scanned:       2,
		ForwardBefore: 4,
		Conflicts:     []string{"conflict-key"},
		Orphans:       []string{"orphan-1", "orphan-2", "orphan-3", "orphan-4", "orphan-5", "orphan-6"},
		UnpairedNFTs:  []string{"unpaired-key"},
	}})

	out := buf.String()
	require.Contains(t, out, "erc721 conversion index still degraded after the prune")
	require.Contains(t, out, `"level":"error"`)
	require.Contains(t, out, `"conflicts":1`)
	require.Contains(t, out, `"orphans":6`)
	require.Contains(t, out, `"unpaired_nfts":1`)

	// The sample is capped so the line stays readable, and it is taken from a
	// sorted list so every validator logs the same keys.
	require.Contains(t, out, "orphan-5")
	require.NotContains(t, out, "orphan-6")
	require.Contains(t, out, "conflict-key")
}

// Every list in the report is sorted, which is what makes the sample above the
// same on every validator rather than dependent on map iteration order.
func TestUIDIndexPruneReport_ForwardAfterAndBalance(t *testing.T) {
	t.Parallel()

	report := erc721keeper.UIDIndexPruneReport{Scanned: 5, ForwardBefore: 7, Deleted: []string{"a", "b"}}
	require.Equal(t, 5, report.ForwardAfter())
	require.True(t, report.Balanced())

	report = erc721keeper.UIDIndexPruneReport{Scanned: 5, ForwardBefore: 7, Deleted: []string{"a"}}
	require.Equal(t, 6, report.ForwardAfter())
	require.False(t, report.Balanced(), "one forward key short of the reverse index")
}
