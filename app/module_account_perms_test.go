package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
)

// Audit F-004: app/app.go granted ibcnfttransfertypes.ModuleName the bank
// permissions {Minter, Burner} although the ICS-721 module moves NFTs and never
// coins. maccPerms was changed to `nil`, and these tests are what makes that
// change stick.
//
// Two properties are being pinned, and they pull in opposite directions:
//
//   - The module must have NO bank permissions. A permission that reappears is
//     the finding coming back.
//   - The module NAME must stay in the map. BlockedAddrs() and
//     ModuleAccountAddrs() (app.go) build addresses from the map's KEYS, so
//     deleting the entry instead of setting it to nil would stop the module
//     address being blocked and make GetModuleAddress return nil. That is the
//     same failure shape that forced a previous round to restore
//     cosmosnft.ModuleName, so "remove the permissions" must not be
//     implemented as "remove the line".
//
// TestICS721ModuleAccountCannotMintOrBurnCoins distinguishes the two by the
// panic it expects: bank's MintCoins/BurnCoins panic with "does not have
// permissions to mint/burn tokens" when the address is registered but
// unprivileged, and with "does not exist" when the name is missing entirely
// (cosmos-sdk x/bank/keeper/keeper.go:351-358 and :388-395). Only the first is
// acceptable here, so the deletion variant fails this test rather than passing
// it for the wrong reason.

// TestICS721ModuleAccountIsRegisteredButUnprivileged pins the exact state the
// audit asked for: the name resolves to an address, and that entry grants
// nothing.
func TestICS721ModuleAccountIsRegisteredButUnprivileged(t *testing.T) {
	app, _ := sharedTestApp(t)

	addr, perms := app.AccountKeeper.GetModuleAddressAndPermissions(ibcnfttransfertypes.ModuleName)

	require.NotNil(t, addr,
		"ibcnfttransfertypes.ModuleName must stay registered in maccPerms: the map's keys are what "+
			"BlockedAddrs/ModuleAccountAddrs derive addresses from, so dropping the entry unblocks the "+
			"module address and makes GetModuleAddress return nil. Set the entry to nil instead.")
	require.Empty(t, perms,
		"%s must not hold bank permissions: the ICS-721 module mints and burns NFTs through the nft "+
			"keeper, not coins through the bank keeper (audit F-004)",
		ibcnfttransfertypes.ModuleName)
}

// TestICS721ModuleAddressStaysBlocked pins the reason the entry is `nil` rather
// than absent. Sending coins to the module address is not something any code
// does, and keeping it blocked is the behaviour the {Minter, Burner} entry had;
// dropping the key would silently change it.
func TestICS721ModuleAddressStaysBlocked(t *testing.T) {
	app, _ := sharedTestApp(t)

	addr := app.AccountKeeper.GetModuleAddress(ibcnfttransfertypes.ModuleName)
	require.NotNil(t, addr, "the ICS-721 module address must still be derivable")
	key := addr.String()

	require.True(t, app.ModuleAccountAddrs()[key],
		"the ICS-721 module address must still be recognised as a module account address")
	require.True(t, app.BlockedAddrs()[key],
		"the ICS-721 module address must stay blocked: it was blocked while the entry carried "+
			"{Minter, Burner}, and removing the key instead of setting it to nil would silently "+
			"let it receive coins")
}

// TestICS721ModuleAccountCannotMintOrBurnCoins is the behavioural half: it does
// not read the permission list, it calls the bank keeper and asserts which
// branch refuses. A wrong implementation cannot pass it by accident.
func TestICS721ModuleAccountCannotMintOrBurnCoins(t *testing.T) {
	app, _ := sharedTestApp(t)

	// CacheContext, not sharedTestAppCtx: GetModuleAccount CREATES the account
	// when it is absent (it allocates an account number), and these calls end in
	// a panic, so the write must not be able to reach the shared application.
	ctx, _ := sharedTestAppCtx.CacheContext()
	amount := sdk.NewCoins(sdk.NewInt64Coin("auptick", 1))

	mintMsg := capturedPanic(t, "BankKeeper.MintCoins on the ICS-721 module account", func() {
		_ = app.BankKeeper.MintCoins(ctx, ibcnfttransfertypes.ModuleName, amount)
	})
	require.Contains(t, mintMsg, "does not have permissions to mint tokens",
		"minting for %s must be refused by the permission check; got: %s",
		ibcnfttransfertypes.ModuleName, mintMsg)
	require.NotContains(t, mintMsg, "does not exist",
		"the refusal must come from the permission check, not from a missing module account: "+
			"'does not exist' means the entry was deleted from maccPerms instead of set to nil; got: %s", mintMsg)

	burnMsg := capturedPanic(t, "BankKeeper.BurnCoins on the ICS-721 module account", func() {
		_ = app.BankKeeper.BurnCoins(ctx, ibcnfttransfertypes.ModuleName, amount)
	})
	require.Contains(t, burnMsg, "does not have permissions to burn tokens",
		"burning for %s must be refused by the permission check; got: %s",
		ibcnfttransfertypes.ModuleName, burnMsg)
	require.NotContains(t, burnMsg, "does not exist",
		"the refusal must come from the permission check, not from a missing module account; got: %s", burnMsg)

	// Collateral check: the edit must not have disarmed a legitimate minter. The
	// `mint` module keeps {minter}, so the same call that just failed above still
	// succeeds for it -- otherwise this test would also pass on a map that
	// granted nothing to anyone.
	require.NoError(t, app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, amount),
		"x/mint must keep its {minter} permission; the ICS-721 entry was the only one changed")
}

// TestICS721ClassCreatorIsNotTheTransferModuleAddress keeps the audit's change
// away from the address ICS-721 actually depends on.
//
// Class creation on the receive path resolves its creator through
// x/internft/keeper.go:64 (and the ClassBuilder's address getter, :29) using
// x/collection/types.ModuleName -- "collection", served by
// nfttypes.ModuleName in the map -- not the transfer module. The two names are
// easy to conflate because the transfer module is the one carrying the ICS-721
// port, so this test states which entry the escrow/creator logic reads.
func TestICS721ClassCreatorIsNotTheTransferModuleAddress(t *testing.T) {
	app, _ := sharedTestApp(t)

	creator := app.AccountKeeper.GetModuleAddress(collectiontypes.ModuleName)
	require.NotNil(t, creator,
		"x/collection/types.ModuleName (%q) must stay registered: ICS-721 class creation resolves its "+
			"creator address through it, and a nil address would write an empty creator",
		collectiontypes.ModuleName)

	transfer := app.AccountKeeper.GetModuleAddress(ibcnfttransfertypes.ModuleName)
	require.NotNil(t, transfer, "the transfer module address must still be derivable")
	require.False(t, creator.Equals(transfer),
		"%q and %q must be different accounts; if they collide, these tests are describing the wrong one",
		collectiontypes.ModuleName, ibcnfttransfertypes.ModuleName)

	// ICS-721 escrow goes to an ADR-028 hash of the channel identifiers, computed
	// by the fork (nft-transfer types/keys.go:51-62) and never resolved through
	// maccPerms. Asserting it is not the module address records that the
	// permission change cannot reach escrowed NFTs.
	escrow := ibcnfttransfertypes.GetEscrowAddress(ibcnfttransfertypes.PortID, "channel-0")
	require.False(t, escrow.Equals(transfer),
		"the ICS-721 escrow address is derived from the port and channel, not from the module account; "+
			"if these ever coincide this test is no longer evidence that escrow is unaffected")
}

// TestNoBankMintOrBurnCallerTargetsTheICS721Module is the caller inventory, as
// a test. The audit's F-004 rests on "nothing calls MintCoins/BurnCoins with
// this module name", which is a fact about the source, so it is pinned where it
// can be re-checked rather than only in prose.
//
// Comments are stripped first, so naming the module in a comment cannot satisfy
// or break it.
func TestNoBankMintOrBurnCallerTargetsTheICS721Module(t *testing.T) {
	srcs := repoNonTestGoSources(t)

	var sites []string
	for path, raw := range srcs {
		code := stripLineComments(raw)
		for i, line := range strings.Split(code, "\n") {
			if !strings.Contains(line, "MintCoins(") && !strings.Contains(line, "BurnCoins(") {
				continue
			}
			sites = append(sites, fmt.Sprintf("%s:%d", path, i+1))

			require.NotContains(t, line, "ibcnfttransfertypes.ModuleName",
				"%s must not mint or burn coins: the ICS-721 module holds no bank permissions after "+
					"audit F-004, so this call would panic at runtime", sites[len(sites)-1])
			require.NotContains(t, line, `"nonfungibletokentransfer"`,
				"%s must not mint or burn coins: the ICS-721 module holds no bank permissions after "+
					"audit F-004, so this call would panic at runtime", sites[len(sites)-1])
		}
	}
	sort.Strings(sites)

	// The inventory is asserted, not just the absence of this module: a new bank
	// mint/burn caller is exactly the thing that would make the removed
	// permission load-bearing again, and it deserves a deliberate look at this
	// test rather than a silent pass.
	require.Len(t, sites, 2,
		"the repo is expected to contain exactly two bank mint/burn call sites (both in testutil/signer.go); "+
			"found %v. If a caller was added on purpose, confirm it does not target the ICS-721 module and "+
			"update this expectation", sites)
	for _, site := range sites {
		require.Contains(t, site, filepath.Join("testutil", "signer.go"),
			"unexpected bank mint/burn call site %s; audit F-004's inventory has to be re-checked", site)
	}
}

// TestICS721ModuleKeepsItsModuleOrderEntries is the guard against the other
// tempting "cleanup": the four ordering lists in app.go are unrelated to
// permissions, and the module still has to be initialised and exported like
// every other module.
func TestICS721ModuleKeepsItsModuleOrderEntries(t *testing.T) {
	raw, err := os.ReadFile("app.go")
	require.NoError(t, err)

	code := stripLineComments(string(raw))
	flat := strings.Join(strings.Fields(code), " ")

	// One map entry plus four ordering lists: SetOrderBeginBlockers,
	// SetOrderEndBlockers, SetOrderInitGenesis, SetOrderExportGenesis.
	require.Equal(t, 5, strings.Count(code, "ibcnfttransfertypes.ModuleName"),
		"expected exactly one maccPerms entry plus four module-ordering entries for the ICS-721 module")

	// require.Contains prints the whole flattened file on failure; strings.Contains
	// does not, and the message below says more than the diff would.
	require.True(t,
		strings.Contains(flat, "ibcnfttransfertypes.ModuleName: nil"),
		"the ICS-721 permissions entry must be the literal nil after audit F-004")
}

// capturedPanic runs f and returns the text of the panic it raised, failing the
// test if it did not panic.
//
// The panic value is formatted with %v rather than compared: bank panics with an
// errorsmod-wrapped sdk error whose Error() is the description, so %v carries
// exactly the two messages TestICS721ModuleAccountCannotMintOrBurnCoins tells
// apart.
func capturedPanic(t *testing.T, what string, f func()) string {
	t.Helper()

	var msg string
	func() {
		defer func() {
			if r := recover(); r != nil {
				msg = fmt.Sprintf("%v", r)
			}
		}()
		f()
	}()

	require.NotEmpty(t, msg, "%s must panic; it returned normally instead", what)
	return msg
}

// repoNonTestGoSources reads every non-test .go file in the repository, keyed by
// the path as walked from the app package directory.
//
// Build/vendor-shaped and documentation directories are skipped: their contents
// are not compiled into the chain, and documents quoting the audit would
// otherwise be searched as if they were code. Test files are excluded because a
// test double for the bank keeper would show up as a call site; this scan is
// about production callers.
func repoNonTestGoSources(t *testing.T) map[string]string {
	t.Helper()

	skipDirs := map[string]bool{
		"deliverables": true,
		"reviewDoc":    true,
		"node_modules": true,
		"testdata":     true,
		"vendor":       true,
	}

	srcs := map[string]string{}
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			// Every dot directory (.git, .github, .workbuddy, ...) and every
			// skipped name is out of scope; .github holds no Go source.
			if path != ".." && (strings.HasPrefix(name, ".") || skipDirs[name]) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		bz, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		srcs[path] = string(bz)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, srcs, "the repository source scan found no Go files; the working directory is wrong")
	return srcs
}
