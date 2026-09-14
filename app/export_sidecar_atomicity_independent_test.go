package app

// Independent "second path" attacks on the export-diagnostics sidecar.
//
// app/export_sidecar_atomicity_test.go proves the two coupled defects (G-01, O-02)
// through one lever each. This file reaches the SAME invariants through DIFFERENT
// levers, so a green here means the invariant holds for a second, independent
// reason rather than that one test happens to pass:
//
//	branch B of staking.WriteValidators -- a consensus key that unpacks but cannot
//	    be converted to a CometBFT key. The sibling suite only ever hit branch A
//	    (a last-validator-power entry with no validator record at all).
//	the rename(2) leg of writeFileAtomic -- the sibling suite interrupted the WRITE
//	    leg with EFBIG.
//	the CreateTemp leg of writeFileAtomic -- previously uncovered entirely.
//
// Deliberately NOT used: secp256k1 as a "non-consensus" consensus key. It is one
// of the two types ToCmtProtoPublicKey explicitly supports (crypto/codec/cmt.go),
// so injecting it would NOT fail the export and the test would be a false green.
//
// Assertions avoid errnos that are not comparable across platforms: for
// rename(file -> non-empty directory) Linux reports EISDIR and macOS reports
// EEXIST, so the failing leg is pinned by the structural facts instead (the
// "rename " operation string of *LinkError, and the presence of the temp name
// that only an already-created temp file can produce).

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256r1"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

// independentStaleReport is the sidecar of a previous export, used to prove that a
// failed export neither rewrites nor removes it.
func independentStaleReport() []byte {
	return []byte(`{"height":111,"total":1,"issues":[{"module":"erc721",` +
		`"kind":"token_pair_corrupt","key":"0xfrom-the-past","detail":"a previous export"}]}` + "\n")
}

// plantBranchBFailure stores a validator whose consensus pubkey the app's
// interface registry can unpack (so ConsPubKey succeeds and the record is not
// the branch-A failure) but which cmt cannot convert (branch B), plus the
// last-validator-power entry that makes IterateLastValidators visit it. Both
// keys are removed on cleanup so the process-wide shared application is left
// exactly as it was found.
func plantBranchBFailure(t *testing.T) {
	t.Helper()

	sk, err := secp256r1.GenPrivKey()
	require.NoError(t, err)
	consPK, err := codectypes.NewAnyWithValue(sk.PubKey())
	require.NoError(t, err)

	app, ctx := sharedTestApp(t)
	valAddr := sdk.ValAddress(bytes.Repeat([]byte{0xC2}, 20))

	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	valKey := stakingtypes.GetValidatorKey(valAddr)
	powKey := stakingtypes.GetLastValidatorPowerKey(valAddr)

	validator := stakingtypes.Validator{
		ConsensusPubkey: consPK,
		Status:          stakingtypes.Bonded,
	}
	store.Set(valKey, stakingtypes.MustMarshalValidator(app.AppCodec(), &validator))
	// The wire encoding of gogotypes.Int64Value{Value: 1}.
	store.Set(powKey, []byte{0x08, 0x01})

	t.Cleanup(func() {
		store.Delete(valKey)
		store.Delete(powKey)
	})
}

// ---------------------------------------------------------------------------
// G-01 / O-02: a DEGRADED export that fails in its LAST step must return an
// error rather than panic, must not publish a report, and must not destroy the
// report of the previous export.
//
// This is strictly stronger than the sibling suite's late-failure test, which
// plants a CLEAN export: a degraded export is exactly the case in which a
// wrongly ordered implementation would publish a report describing an export
// that never happened -- the original O-02 defect.
//
// Reverse control: move finalizeExportDiagnostics back above
// MarshalIndent/WriteValidators in app/export.go. The degraded run then publishes
// a report and clears the stale one before WriteValidators fails, so the
// byte-for-byte assertion fails.
// ---------------------------------------------------------------------------
func TestExportIndependentLateFailureViaBranchB(t *testing.T) {
	const nftUID = "qa-independent-branch-b-nft"

	app, ctx := sharedTestApp(t)
	plantDegradation(t, nftUID)

	// Precondition: the plant really does degrade the export, otherwise the run
	// below would not exercise the degraded-then-failed combination at all.
	require.NotEmpty(t, app.collectExportDiagnostics(ctx),
		"the planted reverse-only index entry must degrade the export")

	plantBranchBFailure(t)

	// --- Preconditions pinning the failure to the LAST step of the export ---
	valAddr := sdk.ValAddress(bytes.Repeat([]byte{0xC2}, 20))
	got, err := app.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err, "the injected validator must read back: this is NOT branch A")
	pk, err := got.ConsPubKey()
	require.NoError(t, err, "the consensus pubkey must unpack: this is NOT a decode failure")
	_, cerr := cryptocodec.ToCmtPubKeyInterface(pk)
	require.Error(t, cerr, "branch B: the cmt conversion must fail")
	require.Contains(t, cerr.Error(), "cannot convert")

	_, verr := staking.WriteValidators(ctx, app.StakingKeeper)
	require.Error(t, verr, "the export's last step must fail")
	require.Contains(t, verr.Error(), "cannot convert", "and for the branch-B reason, not another")

	// Every earlier step still succeeds -- that is what makes this a LATE failure.
	genState, err := app.mm.ExportGenesisForModules(ctx, app.AppCodec(), nil)
	require.NoError(t, err, "the module export must succeed; only the last step fails")
	require.NotEmpty(t, genState)

	home := t.TempDir()
	stale := filepath.Join(home, ExportDiagnosticsFileName)
	staleBody := independentStaleReport()
	require.NoError(t, os.WriteFile(stale, staleBody, 0o600))
	exportTestHome(t, app, home)

	// --- The export itself ---
	var exp servertypes.ExportedApp
	require.NotPanics(t, func() {
		exp, err = app.ExportAppStateAndValidators(false, nil, nil)
	}, "a late failure must surface as an error, never as a panic")

	require.Error(t, err, "an export that failed in its last step must fail")
	require.Contains(t, err.Error(), "cannot convert")
	require.Empty(t, exp.AppState, "no genesis may be handed back for a failed export")
	require.Zero(t, exp.Height)

	// The invariant: no report of a failed export may be published, and the
	// report of the previous export must survive untouched.
	gotBody, rerr := os.ReadFile(stale)
	require.NoError(t, rerr)
	require.Equal(t, string(staleBody), string(gotBody),
		"a failed export must not rewrite or delete the report of a previous one")
	require.Equal(t, []string{ExportDiagnosticsFileName}, exportDirEntries(t, home),
		"and it must leave no temporary file behind")

	// The degradation is not lost -- it is still faithfully collectable. It is
	// simply not PUBLISHED, because the export it would describe never happened.
	require.NotEmpty(t, app.collectExportDiagnostics(ctx),
		"the degradation must remain faithfully collectable after a failed export")
}

// ---------------------------------------------------------------------------
// O-02 (atomicity, publish leg): when the publish step itself is the leg that
// fails, the atomic write must still leave the directory exactly as it found it.
//
// A non-empty directory occupying the sidecar path makes CreateTemp/Write/Sync/
// Close all succeed and only rename(2) fail -- a different leg, and a different
// errno, from the sibling suite's EFBIG write-leg test.
//
// Reverse control: remove the deferred os.Remove(tmp) from writeFileAtomic. The
// temporary file then survives the failed rename and the "exactly one entry"
// assertion fails.
// ---------------------------------------------------------------------------
func TestExportIndependentRenameLegFailure(t *testing.T) {
	const nftUID = "qa-independent-rename-nft"

	app, ctx := sharedTestApp(t)
	plantDegradation(t, nftUID)
	require.NotEmpty(t, app.collectExportDiagnostics(ctx))

	home := t.TempDir()
	blocked := filepath.Join(home, ExportDiagnosticsFileName)
	require.NoError(t, os.MkdirAll(filepath.Join(blocked, "keepme"), 0o755))
	exportTestHome(t, app, home)
	notices := captureExportNotices(t)

	exp, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err, "the sidecar must never fail an export")
	require.NotEmpty(t, exp.AppState)

	st := LastExportDiagnosticsStatus()
	require.True(t, st.Degraded)
	require.False(t, st.ReportWritten)
	require.Error(t, st.WriteErr)

	// Stage pin, portable across platforms. The "rename " operation string comes
	// from *LinkError and is produced only by os.Rename; the temp name proves the
	// preceding CreateTemp/Write/Sync/Close legs all succeeded.
	require.Contains(t, st.WriteErr.Error(), "rename ",
		"the failing leg must be rename, not CreateTemp or Write: %v", st.WriteErr)
	require.Contains(t, st.WriteErr.Error(), "."+ExportDiagnosticsFileName+".tmp-",
		"rename implies the temporary file already existed: %v", st.WriteErr)
	require.Contains(t, st.WriteErr.Error(), ExportDiagnosticsFileName)
	require.False(t, errors.Is(st.WriteErr, syscall.ENOTDIR),
		"that would be the CreateTemp leg, i.e. a different failure")
	require.NotContains(t, st.WriteErr.Error(), "file size exceeds",
		"that would be the EFBIG write leg, i.e. a different failure")

	// Whatever occupied the path is untouched, and the failed atomic write left
	// no temporary file behind.
	_, serr := os.Stat(filepath.Join(blocked, "keepme"))
	require.NoError(t, serr, "the blocking directory must be left untouched")
	require.Equal(t, []string{ExportDiagnosticsFileName}, exportDirEntries(t, home),
		"the directory must hold exactly the blocking entry: no leftover temporary file")

	// The stale-cleanup leg fails too, and that failure is visible rather than
	// swallowed.
	require.Error(t, st.StaleCleanupErr)
	require.True(t, errors.Is(st.StaleCleanupErr, syscall.ENOTEMPTY),
		"removing a non-empty directory must report ENOTEMPTY: %v", st.StaleCleanupErr)

	// And the operator-facing notice carries both failures.
	var notice ExportDiagnosticsNotice
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(notices.Bytes()), &notice))
	require.Equal(t, exportDiagnosticsEventNotPersisted, notice.Event)
	require.Equal(t, st.ExportID, notice.ExportID)
	require.NotEmpty(t, notice.Error)
	require.NotEmpty(t, notice.StaleReportRemovalError)
}

// ---------------------------------------------------------------------------
// O-02 (atomicity, earliest leg): when the temporary file cannot even be
// created, nothing may be created anywhere, the configured home must not be
// touched, and a failing stale-removal must be reported rather than mistaken for
// "there was nothing to remove".
//
// The lever is a plain file where the node home should be: os.CreateTemp(dir)
// then fails with ENOTDIR, which comes from a path component not being a
// directory and therefore involves no permission check -- so, unlike a 0o500
// directory, it cannot be bypassed by running as root and needs no t.Skip.
//
// Reverse control: make writeFileAtomic return nil when CreateTemp fails. The
// swallow dies one assertion earlier than the errno ones -- at ReportWritten,
// which is the point: a caller told "written" is the failure mode, so the
// WriteErr assertions are never reached. Measured, not assumed.
// ---------------------------------------------------------------------------
func TestExportIndependentCreateTempLegFailure(t *testing.T) {
	const nftUID = "qa-independent-notdir-nft"
	const originalHome = "NOT-A-DIRECTORY"

	app, ctx := sharedTestApp(t)
	plantDegradation(t, nftUID)
	require.NotEmpty(t, app.collectExportDiagnostics(ctx))

	base := t.TempDir()
	homeAsFile := filepath.Join(base, "home-is-not-a-directory")
	require.NoError(t, os.WriteFile(homeAsFile, []byte(originalHome), 0o600))
	exportTestHome(t, app, homeAsFile)

	exp, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err, "the sidecar must never fail an export")
	require.NotEmpty(t, exp.AppState)

	st := LastExportDiagnosticsStatus()
	require.True(t, st.Degraded)
	require.False(t, st.ReportWritten)
	require.Error(t, st.WriteErr)
	require.True(t, errors.Is(st.WriteErr, syscall.ENOTDIR),
		"a home that is not a directory must fail with ENOTDIR: %v", st.WriteErr)

	// Nothing was created anywhere, and the configured home is byte-unchanged.
	bz, rerr := os.ReadFile(homeAsFile)
	require.NoError(t, rerr)
	require.Equal(t, originalHome, string(bz), "the node home must not be touched")
	require.Equal(t, []string{filepath.Base(homeAsFile)}, exportDirEntries(t, base),
		"no temporary file may appear next to the node home")

	// removeStaleExportDiagnosticsReport must report this, not claim success.
	require.Error(t, st.StaleCleanupErr)
	require.True(t, errors.Is(st.StaleCleanupErr, syscall.ENOTDIR),
		"the stale-removal leg must fail with ENOTDIR too: %v", st.StaleCleanupErr)
	require.False(t, errors.Is(st.StaleCleanupErr, os.ErrNotExist),
		"ENOTDIR must not be mistaken for 'there was nothing to remove'")
}
