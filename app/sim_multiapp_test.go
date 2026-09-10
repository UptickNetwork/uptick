package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	nfttypes "github.com/UptickNetwork/uptick/x/collection/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	cosmoserc20types "github.com/cosmos/evm/x/erc20/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// Multiple application instances cannot coexist in one process.
//
// x/vm seals its EVM coin configuration in a package-level variable the first
// time an app runs InitChain (x/vm/types.setEVMCoinInfo, reached through
// SetGlobalConfigVariables -> EVMConfigurator.Configure). The module's guard is
// a per-AppModule sync.Once, so a second application in the same process gets a
// fresh Once, calls Configure again and panics with "EVM coin info already set".
//
// Upstream solves this with the `test` build tag plus
// EVMConfigurator.ResetTestConfig, but this repository builds its tests without
// that tag (Makefile: build_tags = netgo) and the EVMConfigurator entry point
// is not wired up here, so the simulation tests that need more than one app
// re-execute the test binary instead: the parent keeps the assertions and each
// child runs exactly one application instance.
//
// TestSimHelperProcess is the child entry point. It is inert unless the
// UPTICK_SIM_HELPER_MODE environment variable is set, so it never runs as part
// of a normal test invocation.
const (
	simHelperModeEnv      = "UPTICK_SIM_HELPER_MODE"
	simHelperSeedEnv      = "UPTICK_SIM_HELPER_SEED"
	simHelperStateEnv     = "UPTICK_SIM_HELPER_STATE"
	simHelperSnapshotEnv  = "UPTICK_SIM_HELPER_SNAPSHOT"
	simHelperResultEnv    = "UPTICK_SIM_HELPER_RESULT"
	simHelperNumBlocksEnv = "UPTICK_SIM_HELPER_NUM_BLOCKS"
	simHelperBlockSizeEnv = "UPTICK_SIM_HELPER_BLOCK_SIZE"
)

const (
	// simHelperModeHash runs the simulation and writes the exported-state hash
	// to the result file.
	simHelperModeHash = "hash"
	// simHelperModeExport runs the simulation and writes both the exported
	// state and the committed module-store snapshot to their files.
	simHelperModeExport = "export"
	// simHelperModeImport imports the state at simHelperStateEnv and writes the
	// module-store snapshot of the freshly imported genesis to the snapshot
	// file. The stores are read from the uncommitted InitChain write set, so no
	// block is ever produced and the snapshot is directly comparable with the
	// exporting side.
	simHelperModeImport = "import"
	// simHelperModeContinue imports the state at simHelperStateEnv and keeps
	// simulating from it, then writes the resulting state hash to the result
	// file.
	simHelperModeContinue = "continue"
)

// TestSimHelperProcess is the re-executed child of TestAppStateDeterminism,
// TestAppImportExport and TestAppSimulationAfterImport. Every child does at
// most one simulation so it only ever creates one application.
//
// Results travel through files rather than stdout: the simulation writes its
// progress with bare carriage returns, so anything appended to stdout lands on
// the same line as the last progress update and cannot be parsed reliably.
func TestSimHelperProcess(t *testing.T) {
	mode := os.Getenv(simHelperModeEnv)
	if mode == "" {
		t.Skip("internal helper process entry point; driven by the simulation tests")
	}

	config := simHelperConfig(t)

	switch mode {
	case simHelperModeHash:
		writeSimHelperResult(t, stateHash(t, runSimulation(t, config, nil).AppState))

	case simHelperModeExport:
		var snapshot map[string]simStoreSummary
		exported := runSimulation(t, config, func(app *Uptick) {
			snapshot = simSnapshotCommitted(t, app)
		})

		writeSimHelperExport(t, exported)
		writeSimHelperSnapshot(t, snapshot)

	case simHelperModeImport:
		writeSimHelperSnapshot(t, importAndSnapshot(t, readSimHelperExport(t)))

	case simHelperModeContinue:
		writeSimHelperResult(t, stateHash(t, importAndContinue(t, readSimHelperExport(t).AppState, config)))

	default:
		t.Fatalf("unknown simulation helper mode %q", mode)
	}
}

// simHelperConfig rebuilds the parent's simulation config from the environment.
func simHelperConfig(t *testing.T) simtypes.Config {
	t.Helper()

	numBlocks := intEnv(t, simHelperNumBlocksEnv, 5)
	blockSize := intEnv(t, simHelperBlockSizeEnv, 4)
	config := simConfig(t, numBlocks, blockSize)

	return withSeed(t, config, int64(intEnv(t, simHelperSeedEnv, int(*simSeed))))
}

// writeSimHelperResult publishes the child's result for the parent to read.
func writeSimHelperResult(t *testing.T, hash string) {
	t.Helper()

	path := os.Getenv(simHelperResultEnv)
	require.NotEmpty(t, path, "the helper was not told where to write its result")
	require.NoError(t, os.WriteFile(path, []byte(hash), 0o600))
}

// writeSimHelperSnapshot publishes a store snapshot for the parent to read.
func writeSimHelperSnapshot(t *testing.T, snapshot map[string]simStoreSummary) {
	t.Helper()

	path := os.Getenv(simHelperSnapshotEnv)
	require.NotEmpty(t, path, "the helper was not told where to write its store snapshot")
	require.NotNil(t, snapshot, "the helper produced no store snapshot")

	raw, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
}

// readSimHelperResult reads the result a child published. The child's output is
// only used to build a useful failure message.
func readSimHelperResult(t *testing.T, path string, out []byte) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err, "the simulation helper did not write a result; helper output was:\n%s", out)
	hash := strings.TrimSpace(string(raw))
	require.Len(t, hash, sha256.Size*2, "the simulation helper wrote a malformed result: %q", hash)
	return hash
}

// readSimHelperSnapshot reads the store snapshot a child published.
func readSimHelperSnapshot(t *testing.T, path string) map[string]simStoreSummary {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err, "the simulation helper did not write its store snapshot")

	var snapshot map[string]simStoreSummary
	require.NoError(t, json.Unmarshal(raw, &snapshot), "the store snapshot is not valid JSON")
	require.NotEmpty(t, snapshot, "the store snapshot is empty")
	return snapshot
}

// writeSimHelperExport publishes the exported application for the parent and
// the importing helpers to read.
func writeSimHelperExport(t *testing.T, exported simExport) {
	t.Helper()

	raw, err := json.Marshal(exported)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(os.Getenv(simHelperStateEnv), raw, 0o600))
}

// readSimHelperExport reads the export a child published.
func readSimHelperExport(t *testing.T) simExport {
	t.Helper()

	raw, err := os.ReadFile(os.Getenv(simHelperStateEnv))
	require.NoError(t, err, "reading the exported state handed to the helper failed")

	var exported simExport
	require.NoError(t, json.Unmarshal(raw, &exported), "the exported state is not valid JSON")
	require.NotEmpty(t, exported.AppState, "the helper was handed an empty exported state")
	return exported
}

func intEnv(t *testing.T, key string, fallback int) int {
	t.Helper()

	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	require.NoError(t, err, "environment variable %s is not an integer: %q", key, raw)
	return value
}

// spawnSimHelper re-executes the test binary as a single-application child,
// writing the child's result to a path picked here, and returns the child's
// output for diagnostics.
func spawnSimHelper(t *testing.T, mode string, config simtypes.Config, statePath, snapshotPath string) []byte {
	t.Helper()

	_, out := spawnSimHelperAt(t, mode, config, statePath, snapshotPath,
		filepath.Join(t.TempDir(), "result-"+mode+".txt"))
	return out
}

// spawnSimHelperAt runs a child and returns the output path together with the
// child's output.
func spawnSimHelperAt(t *testing.T, mode string, config simtypes.Config, statePath, snapshotPath, resultPath string) (string, []byte) {
	t.Helper()

	// -test.v is on so a child that skips (SimulateFromSeed calls tb.Skip when
	// the mock validator set empties) reports why instead of silently producing
	// no result. The output is only surfaced when the parent fails.
	cmd := exec.Command(os.Args[0], "-test.run=^TestSimHelperProcess$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(),
		simHelperModeEnv+"="+mode,
		simHelperSeedEnv+"="+strconv.FormatInt(config.Seed, 10),
		simHelperNumBlocksEnv+"="+strconv.Itoa(config.NumBlocks),
		simHelperBlockSizeEnv+"="+strconv.Itoa(config.BlockSize),
		simHelperStateEnv+"="+statePath,
		simHelperSnapshotEnv+"="+snapshotPath,
		simHelperResultEnv+"="+resultPath,
	)

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "simulation helper %q failed:\n%s", mode, out)
	return resultPath, out
}

// TestAppStateDeterminism is the non-determinism gate: the same seed must
// produce the exact same exported state. A mismatch means consensus-relevant
// state depends on something other than the seed (map iteration, wall clock,
// goroutine ordering).
func TestAppStateDeterminism(t *testing.T) {
	if !*simEnabled {
		t.Skip("simulation disabled: pass -Enabled=true to run")
	}

	config := simConfig(t, 5, 4)
	seeds := []int64{config.Seed, config.Seed + 1, config.Seed + 2}

	for _, seed := range seeds {
		seeded := withSeed(t, config, seed)

		first := runSimHelper(t, simHelperModeHash, seeded, "", "")
		second := runSimHelper(t, simHelperModeHash, seeded, "", "")

		require.Equal(t, first, second,
			"seed %d produced different state across two runs: the chain is non-deterministic", seed)
	}
}

// TestAppImportExport exports the state produced by a simulation, re-imports it
// into a fresh application and compares every module store.
//
// The exporting side hashes the committed stores after the simulated blocks;
// the importing side hashes the store writes InitChain produced from the
// exported genesis, read from the uncommitted finalize-block write set. No
// block runs on the importing side, which is what makes the two directly
// comparable: a module whose genesis is a complete description of its state
// must reproduce its store exactly, and anything InitGenesis drops shows up as
// a missing key or a changed value.
//
// Both sides run in their own process because the EVM module forbids a second
// app in the same process.
func TestAppImportExport(t *testing.T) {
	if !*simEnabled {
		t.Skip("simulation disabled: pass -Enabled=true to run")
	}

	config := simConfig(t, 5, 4)
	dir := t.TempDir()
	statePath := filepath.Join(dir, "exported-state.json")
	exportedSnapshotPath := filepath.Join(dir, "exported-stores.json")
	importedSnapshotPath := filepath.Join(dir, "imported-stores.json")

	spawnSimHelper(t, simHelperModeExport, config, statePath, exportedSnapshotPath)
	require.FileExists(t, statePath, "the export helper did not write the app state")

	spawnSimHelper(t, simHelperModeImport, config, statePath, importedSnapshotPath)

	// The NFT modules are what this gate exists for. If the simulation ever
	// stops exercising them, every store it owns stays empty and the whole
	// comparison silently degrades into "nothing versus nothing", so fail
	// loudly instead of passing vacuously.
	exportedSnapshot := readSimHelperSnapshot(t, exportedSnapshotPath)
	require.NotZero(t, exportedSnapshot[nfttypes.StoreKey].Count,
		"the simulation produced no collection state; the import/export gate would compare nothing")

	simCompareSnapshots(t, exportedSnapshot, readSimHelperSnapshot(t, importedSnapshotPath))
}

// TestAppSimulationAfterImport resumes a simulation from a previously exported
// state, covering the "restart from an exported snapshot" path an operator uses.
// Two runs from the same snapshot and seed must agree, so the resumed chain is
// deterministic too.
func TestAppSimulationAfterImport(t *testing.T) {
	if !*simEnabled {
		t.Skip("simulation disabled: pass -Enabled=true to run")
	}

	config := simConfig(t, 4, 3)
	statePath := filepath.Join(t.TempDir(), "exported-state.json")

	// The export helper always publishes a store snapshot; this test only needs
	// the state, so the snapshot lands in the same temporary directory unused.
	spawnSimHelper(t, simHelperModeExport, config, statePath,
		filepath.Join(t.TempDir(), "exported-stores.json"))

	first := runSimHelper(t, simHelperModeContinue, config, statePath, "")
	second := runSimHelper(t, simHelperModeContinue, config, statePath, "")

	require.Equal(t, first, second,
		"continuing a simulation from an imported state is non-deterministic")
}

// runSimHelper runs a child and returns the state hash it computed.
func runSimHelper(t *testing.T, mode string, config simtypes.Config, statePath, snapshotPath string) string {
	t.Helper()

	resultPath := filepath.Join(t.TempDir(), "result-"+mode+".txt")
	_, out := spawnSimHelperAt(t, mode, config, statePath, snapshotPath, resultPath)
	return readSimHelperResult(t, resultPath, out)
}

// importAndSnapshot imports an exported app state and returns the snapshot of
// the module stores the genesis produced.
func importAndSnapshot(t *testing.T, exported simExport) map[string]simStoreSummary {
	t.Helper()

	db, dir := simDB(t, "import-export")
	defer func() {
		_ = db.Close()
		_ = os.RemoveAll(dir)
	}()

	app := newSimApp(t, db, dir)

	initReq := abci.RequestInitChain{
		ChainId: simChainID,
		Time:    simGenesisTime,
		// A real node receives the genesis consensus parameters from CometBFT
		// rather than from the genesis map, so the export has to be replayed
		// the same way or the consensus store stays empty.
		ConsensusParams: &exported.ConsensusParams,
		AppStateBytes:   exported.AppState,
		InitialHeight:   1,
	}
	_, err := app.InitChain(&initReq)
	require.NoError(t, err, "re-importing the exported genesis failed")

	// InitChain leaves its writes in the finalize-block write set instead of
	// flushing them to the commit store (BaseApp.Commit only flushes what
	// FinalizeBlock wrote). Reading that branched store therefore yields the
	// genesis state exactly, with no block-level transitions applied.
	return simSnapshotAtGenesis(t, app)
}

// importAndContinue re-imports an exported app state and then keeps simulating
// from it. The genesis the simulation starts from is the exported state itself,
// so the resumed chain is the one the export describes rather than a freshly
// generated one.
func importAndContinue(t *testing.T, state []byte, config simtypes.Config) []byte {
	t.Helper()

	db, dir := simDB(t, "after-import")
	defer func() {
		_ = db.Close()
		_ = os.RemoveAll(dir)
	}()

	app := newSimApp(t, db, dir)

	exportedStateFn := func(_ *rand.Rand, accs []simtypes.Account, _ simtypes.Config) (json.RawMessage, []simtypes.Account, string, time.Time) {
		return state, accs, simChainID, simGenesisTime
	}

	_, _, err := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		app.BaseApp,
		exportedStateFn,
		simRandAccFn,
		simOps(app, config),
		app.ModuleAccountAddrs(),
		config,
		app.AppCodec(),
	)
	require.NoError(t, err, "simulation after import failed")

	exported, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err, "export after continuing the simulation failed")
	return exported.AppState
}

// ---------------------------------------------------------------------------
// module store snapshot
// ---------------------------------------------------------------------------

// simStoreKeys lists the module stores the import/export gate compares.
//
// Every store listed here belongs to a module whose genesis is supposed to be a
// complete description of its state, so a faithful import has to reproduce the
// exported store exactly. That includes the stores this repository owns
// (collection/erc721/cw721, plus the EVM and IBC front ends they lean on) and
// the core asset stores (auth/bank) whose contents those modules move around.
var simStoreKeys = []string{
	// SDK core
	authtypes.StoreKey,
	banktypes.StoreKey,
	stakingtypes.StoreKey,
	minttypes.StoreKey,
	distrtypes.StoreKey,
	slashingtypes.StoreKey,
	govtypes.StoreKey,
	paramstypes.StoreKey,
	consensustypes.StoreKey,
	upgradetypes.StoreKey,
	evidencetypes.StoreKey,
	crisistypes.StoreKey,
	feegrant.StoreKey,
	authzkeeper.StoreKey,
	// IBC
	ibcexported.StoreKey,
	ibctransfertypes.StoreKey,
	ibcnfttransfertypes.StoreKey,
	icahosttypes.StoreKey,
	icacontrollertypes.StoreKey,
	// cosmos/evm
	evmtypes.StoreKey,
	feemarkettypes.StoreKey,
	cosmoserc20types.StoreKey,
	// uptick
	erc721types.StoreKey,
	cw721types.StoreKey,
	nfttypes.StoreKey,
	// wasm
	wasmtypes.StoreKey,
}

// simStorePrefixSkips holds the store prefixes that cannot round trip because
// the owning module's genesis deliberately does not describe them.
var simStorePrefixSkips = map[string][][]byte{
	stakingtypes.StoreKey: {
		// Historical entries accumulate one record per block and are never
		// exported (they exist to let the slashing evidence window resolve
		// consensus addresses), so the post-block store is allowed to have more
		// of them than the re-imported genesis.
		stakingtypes.HistoricalInfoKey,

		// The unbonding-operation registry (the id counter plus the id ->
		// index/type indexes) is assigned by the keeper when an unbonding
		// operation is created and was removed from GenesisState on purpose:
		// staking.InitGenesis re-derives it from the exported unbonding
		// delegations and redelegations, which is why an export carries the
		// "Exported" flag. Ids of operations that already completed cannot be
		// reconstructed, so the counter and the leftover indexes legitimately
		// differ.
		stakingtypes.UnbondingIDKey,
		stakingtypes.UnbondingIndexKey,
		stakingtypes.UnbondingTypeKey,

		// The completion-time queues index unbonding delegations, redelegations
		// and unbonding validators by the block they mature at. When several
		// operations share a completion time their addresses are concatenated
		// into a single value in insertion order, and InitGenesis inserts the
		// operations in exported (key) order rather than in the chronological
		// order the running chain used. The value is a set encoded as a
		// concatenation, so the two orders are the same state written
		// differently.
		stakingtypes.UnbondingQueueKey,
		stakingtypes.RedelegationQueueKey,
		stakingtypes.ValidatorQueueKey,

		// The validator updates staking.EndBlocker hands to CometBFT for the
		// next block. It is per-block state, never exported, and it is what
		// staking.EndBlocker overwrites at the start of every block.
		stakingtypes.ValidatorUpdatesKey,
	},

	wasmtypes.StoreKey: {
		// Per-transaction scratch state used to count the message executions
		// inside a single transaction. It is cleared at the end of the
		// transaction that opened it and is not genesis data.
		wasmtypes.TXCounterPrefix,
	},

	feegrant.StoreKey: {
		// The allowance expiration queue indexes the allowances (key =
		// expiration + grantee + granter, empty value) and InitGenesis rebuilds
		// it by re-granting every exported allowance. It is not part of the
		// exported genesis, and the keeper does not keep it in lockstep with the
		// allowances: UpdateAllowance rewrites an allowance without touching its
		// queue entry, so an entry can outlive the allowance it indexes. Such a
		// leftover is not recoverable from a genesis and holds no data of its
		// own.
		feegrant.FeeAllowanceQueueKeyPrefix,
	},
}

// simSequenceDefault is the big-endian encoding of the value wasmd's
// auto-increment sequences report when they have never been materialised.
var simSequenceDefault = []byte{0, 0, 0, 0, 0, 0, 0, 1}

// simStoreDroppedEntries lists key/value pairs that are semantically identical
// to being absent, so comparing them would report a difference that does not
// exist. Each entry receives the raw key and value and reports whether the pair
// should be ignored on both sides.
var simStoreDroppedEntries = map[string]func(key, value []byte) bool{
	wasmtypes.StoreKey: func(key, value []byte) bool {
		// wasmd's code/contract ID sequences are lazy: PeekAutoIncrementID
		// returns 1 for a missing key, and InitGenesis materialises the key with
		// that same value. "Missing" and "present with the default" are the same
		// state, so only counters that actually advanced are compared.
		if !bytes.HasPrefix(key, wasmtypes.SequenceKeyPrefix) {
			return false
		}
		return bytes.Equal(value, simSequenceDefault)
	},

	authzkeeper.StoreKey: func(key, value []byte) bool {
		// A queue entry whose GrantQueueItem lists no message types is an empty
		// shell: the SDK shrinks the list when a grant is revoked but only
		// deletes the queue key once the expiration passes. ExportGenesis never
		// writes queue entries at all (InitGenesis rebuilds them from the
		// grants it restores), so such a shell cannot survive a round trip and
		// carries no state: DequeueAndDeleteExpiredGrants treats a missing grant
		// as a no-op.
		if !bytes.HasPrefix(key, authzkeeper.GrantQueuePrefix) {
			return false
		}
		return len(value) == 0
	},
}

// simStoreSummary is a comparable digest of one module store: how many entries
// it holds, a hash over all of them, and every entry as "<key>=<value hash>"
// sorted by key. The per-entry list makes a mismatch reportable in terms of the
// exact keys involved instead of only "this store differs".
type simStoreSummary struct {
	Count   int      `json:"count"`
	Hash    string   `json:"hash"`
	Entries []string `json:"entries"`
}

// simSnapshotCommitted hashes the module stores as they were committed after
// the simulated blocks.
func simSnapshotCommitted(t *testing.T, app *Uptick) map[string]simStoreSummary {
	t.Helper()

	header := cmtproto.Header{ChainID: simChainID, Height: app.LastBlockHeight()}
	return simSnapshot(t, app.NewUncachedContext(false, header), app)
}

// simSnapshotAtGenesis hashes the module stores an InitChain produced. It must
// only be called while the finalize-block write set is still around, i.e. after
// InitChain and before Commit.
func simSnapshotAtGenesis(t *testing.T, app *Uptick) map[string]simStoreSummary {
	t.Helper()

	header := cmtproto.Header{ChainID: simChainID, Height: 1}
	return simSnapshot(t, app.NewContextLegacy(false, header), app)
}

// simSnapshot digests every store in simStoreKeys through the given context.
func simSnapshot(t *testing.T, ctx sdk.Context, app *Uptick) map[string]simStoreSummary {
	t.Helper()

	names := append([]string(nil), simStoreKeys...)
	sort.Strings(names)

	snapshot := make(map[string]simStoreSummary, len(names))
	for _, name := range names {
		key := app.GetKey(name)
		require.NotNil(t, key, "store %q is not mounted on the application", name)

		iter := ctx.KVStore(key).Iterator(nil, nil)
		snapshot[name] = simStoreDigest(iter, simStoreSkipPredicate(name))
		require.NoError(t, iter.Close(), "closing the iterator over store %q failed", name)
	}
	return snapshot
}

// simStoreSkipPredicate combines the prefix and key/value rules that decide
// which entries of a store cannot round trip.
func simStoreSkipPredicate(store string) func(key, value []byte) bool {
	prefixes := simStorePrefixSkips[store]
	dropped := simStoreDroppedEntries[store]

	return func(key, value []byte) bool {
		if hasAnyPrefix(key, prefixes) {
			return true
		}
		return dropped != nil && dropped(key, value)
	}
}

// simStoreDigest hashes every compared key/value pair of a store.
func simStoreDigest(iter storetypes.Iterator, skip func(key, value []byte) bool) simStoreSummary {
	hash := sha256.New()
	summary := simStoreSummary{}

	for ; iter.Valid(); iter.Next() {
		key, value := iter.Key(), iter.Value()
		if skip(key, value) {
			continue
		}

		valueHash := sha256.Sum256(value)
		summary.Count++
		summary.Entries = append(summary.Entries, fmt.Sprintf("%s=%x", hex.EncodeToString(key), valueHash[:8]))
		hash.Write(key)
		hash.Write([]byte{0})
		hash.Write(value)
		hash.Write([]byte{0})
	}

	sort.Strings(summary.Entries)
	summary.Hash = hex.EncodeToString(hash.Sum(nil))
	return summary
}

func hasAnyPrefix(key []byte, prefixes [][]byte) bool {
	for _, prefix := range prefixes {
		if bytes.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// simCompareSnapshots asserts that the importing side reproduced every store of
// the exporting side.
func simCompareSnapshots(t *testing.T, exported, imported map[string]simStoreSummary) {
	t.Helper()

	names := make([]string, 0, len(exported)+len(imported))
	seen := make(map[string]bool, len(exported)+len(imported))
	for _, snapshot := range []map[string]simStoreSummary{exported, imported} {
		for name := range snapshot {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)

	// Log what was actually compared. An import/export gate is only as good as
	// the state it sees, and an empty store silently passes every check.
	for _, name := range names {
		t.Logf("store %-12s exported=%4d imported=%4d", name, exported[name].Count, imported[name].Count)
	}

	var failures []string
	for _, name := range names {
		want, wantOK := exported[name]
		got, gotOK := imported[name]

		switch {
		case !wantOK || !gotOK:
			failures = append(failures, fmt.Sprintf(
				"%s: store missing from one side (exported=%t, imported=%t)", name, wantOK, gotOK))

		case want.Hash != got.Hash:
			failures = append(failures, fmt.Sprintf(
				"%s: %d entry/entries exported, %d after import; %s",
				name, want.Count, got.Count, simEntryDiff(want.Entries, got.Entries)))
		}
	}

	require.Empty(t, failures,
		"re-importing the exported state did not restore it:\n%s", strings.Join(failures, "\n"))
}

// simEntryDiff summarises how two "<key>=<value hash>" entry lists differ, which
// is what points at the state InitGenesis failed to restore.
func simEntryDiff(exported, imported []string) string {
	importedByKey := make(map[string]string, len(imported))
	for _, entry := range imported {
		key, value, _ := strings.Cut(entry, "=")
		importedByKey[key] = value
	}
	exportedByKey := make(map[string]string, len(exported))
	for _, entry := range exported {
		key, value, _ := strings.Cut(entry, "=")
		exportedByKey[key] = value
	}

	var missing, extra, changed []string
	for _, entry := range exported {
		key, value, _ := strings.Cut(entry, "=")
		importedValue, ok := importedByKey[key]
		switch {
		case !ok:
			missing = append(missing, key)
		case importedValue != value:
			changed = append(changed, fmt.Sprintf("%s (%s -> %s)", key, value, importedValue))
		}
	}
	for _, entry := range imported {
		key, _, _ := strings.Cut(entry, "=")
		if _, ok := exportedByKey[key]; !ok {
			extra = append(extra, key)
		}
	}

	parts := []string{
		fmt.Sprintf("%d key(s) lost by the import %s", len(missing), simSample(missing)),
		fmt.Sprintf("%d key(s) added by the import %s", len(extra), simSample(extra)),
	}
	if len(changed) > 0 {
		parts = append(parts, fmt.Sprintf("%d key(s) restored with a different value %s",
			len(changed), simSample(changed)))
	}
	return strings.Join(parts, ", ")
}

func simSample(entries []string) string {
	const limit = 3

	if len(entries) == 0 {
		return "[]"
	}

	sample := entries
	suffix := ""
	if len(sample) > limit {
		sample = sample[:limit]
		suffix = fmt.Sprintf(", ... %d more", len(entries)-limit)
	}
	return "[" + strings.Join(sample, ", ") + suffix + "]"
}
