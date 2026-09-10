package app

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	nfttypes "github.com/UptickNetwork/uptick/x/collection/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
)

// The simulation harness was missing from this repository (G-05): the Makefile
// targets pointed at TestAppStateDeterminism / TestFullAppSimulation /
// TestAppImportExport / TestAppSimulationAfterImport, none of which existed,
// and they passed -Enabled/-Genesis/-Period flags that no test registered, so
// `make test-sim-nondeterminism` failed with "flag provided but not defined".
// This file restores the harness and the flag contract it depends on.

const simChainID = "uptick_1170-1"

// simGenesisTime is the wall clock the simulated chain starts at.
//
// It must not be the Unix epoch: the first simulated block can be produced at
// exactly the genesis timestamp, and wasmd rejects a contract environment whose
// block time is zero ("Block (unix) time must never be empty or negative").
// A real, non-zero date keeps the harness runnable and, unlike an epoch
// genesis, keeps "expired at genesis" checks meaningful.
var simGenesisTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// The EVM denom and the gas price the simulated transactions must be able to
// pay. Uptick's feemarket accepts a base fee of 1e9 auptick at genesis.
const (
	simFeeDenom    = "auptick"
	simMinGasPrice = 1_000_000_000
)

// Flags consumed by the Makefile simulation targets. They must be registered at
// package-init time, otherwise `go test -Enabled=true` aborts before any test
// runs.
var (
	simEnabled    = flag.Bool("Enabled", false, "run the application simulation")
	simVerbose    = flag.Bool("Verbose", false, "verbose simulation output")
	simCommit     = flag.Bool("Commit", true, "commit the simulated blocks")
	simSeed       = flag.Int64("Seed", 42, "simulation seed")
	simNumBlocks  = flag.Int("NumBlocks", 0, "override the number of simulated blocks")
	simBlockSize  = flag.Int("BlockSize", 0, "override the number of operations per block")
	simGenesis    = flag.String("Genesis", "", "custom simulation genesis file")
	simParamsFile = flag.String("ParamsFile", "", "custom simulation params file")

	// Period is a legacy SDK flag. The simulation config no longer has a
	// concept of "period", but contrib/devtools' runsim (and therefore every
	// Makefile target that shells out to it) still passes -Period, so the flag
	// has to exist or `go test` aborts with "flag provided but not defined"
	// before a single test runs. It is accepted and ignored.
	simPeriod = flag.Int("Period", 0, "deprecated; accepted for runsim compatibility and ignored")

	// Export paths are also passed by runsim. They are honoured by
	// runSimulation so the flags are not silently dead.
	simExportParamsPath = flag.String("ExportParamsPath", "", "file the simulation params are written to")
	simExportStatePath  = flag.String("ExportStatePath", "", "file the exported app state is written to")
)

// simConfig builds the simulation config, letting the Makefile flags override
// the per-test defaults so a short CI run and a long local run share one code
// path.
func simConfig(t testing.TB, defaultBlocks, defaultBlockSize int) simtypes.Config {
	t.Helper()

	numBlocks, blockSize := defaultBlocks, defaultBlockSize
	if *simNumBlocks > 0 {
		numBlocks = *simNumBlocks
	}
	if *simBlockSize > 0 {
		blockSize = *simBlockSize
	}

	config := simtypes.Config{
		ChainID:            simChainID,
		GenesisFile:        *simGenesis,
		ParamsFile:         *simParamsFile,
		ExportParamsPath:   *simExportParamsPath,
		ExportStatePath:    *simExportStatePath,
		Commit:             *simCommit,
		NumBlocks:          numBlocks,
		BlockSize:          blockSize,
		Seed:               *simSeed,
		InitialBlockHeight: 1,
		DBBackend:          "goleveldb",
		BlockMaxGas:        -1,
		Lean:               !*simVerbose,
	}

	// Fixed fuzz seed keeps the harness reproducible: the same -Seed must
	// produce the same chain, which is exactly what the determinism test
	// asserts.
	return config.With(t, config.Seed, simFuzzSeed())
}

// simFuzzSeed is the deterministic fuzz seed every simulation run uses. It is
// a function rather than a package-level slice so a caller cannot mutate the
// shared backing array and silently change the sequence of every later run.
func simFuzzSeed() []byte {
	fuzzSeed := make([]byte, 32)
	for i := range fuzzSeed {
		fuzzSeed[i] = byte(i)
	}
	return fuzzSeed
}

// withSeed returns a copy of config bound to another seed, rebuilding the fuzz
// seed so the byte source is identical to the one simConfig installs.
func withSeed(t testing.TB, config simtypes.Config, seed int64) simtypes.Config {
	t.Helper()
	return config.With(t, seed, simFuzzSeed())
}

// newSimApp builds a fresh application over the given database.
func newSimApp(t testing.TB, db dbm.DB, home string) *Uptick {
	t.Helper()

	app := NewUptick(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(home),
		nil,
		baseapp.SetChainID(simChainID),
		// The simulated operations must be able to pay the feemarket base fee,
		// otherwise every generated transaction is rejected with "gas prices
		// too low" and the harness degenerates into empty blocks.
		baseapp.SetMinGasPrices(fmt.Sprintf("%d%s", simMinGasPrice, simFeeDenom)),
	)
	require.NotNil(t, app)
	require.NotEmpty(t, app.Name())
	return app
}

// simDB creates an isolated database + directory for one simulation run.
func simDB(t testing.TB, name string) (dbm.DB, string) {
	t.Helper()

	dir, err := os.MkdirTemp("", "uptick-sim-"+name)
	require.NoError(t, err)

	db, err := dbm.NewDB(name, dbm.GoLevelDBBackend, dir)
	require.NoError(t, err)
	return db, dir
}

// simAppStateFn builds the genesis the simulation starts from.
//
// It deliberately does NOT use simtestutil.AppStateRandomizedFn: the per-module
// randomized genesis generators leave bank.Supply inconsistent for this app's
// denom configuration (supply counts NumBonded stakes that never appear as
// balances), so InitChain panics before a single block is simulated. The
// upstream simapp helper GenesisStateWithValSet produces an auth/bank/staking
// genesis that is self-consistent for the accounts we hand it, which is what
// makes the harness actually runnable.
func simAppStateFn(app *Uptick) simtypes.AppStateFn {
	return func(r *rand.Rand, accs []simtypes.Account, config simtypes.Config) (json.RawMessage, []simtypes.Account, string, time.Time) {
		genesisTimestamp := simGenesisTime

		genesisState := app.bm.DefaultGenesis(app.AppCodec())
		CustomizeDefaultGenesis(app.AppCodec(), genesisState)

		if config.GenesisFile != "" {
			bz, err := os.ReadFile(config.GenesisFile)
			if err != nil {
				panic(fmt.Sprintf("failed to read simulation genesis file %s: %v", config.GenesisFile, err))
			}
			if err := json.Unmarshal(bz, &genesisState); err != nil {
				panic(fmt.Sprintf("failed to parse simulation genesis file %s: %v", config.GenesisFile, err))
			}
		}

		valSet, genAccs, balances := simGenesisAccounts(accs)

		// GenesisStateWithValSet rebuilds the bank genesis (new params, new
		// balances, empty denom-metadata list). The EVM module resolves its EVM
		// denom through that metadata and panics without it, so the Uptick
		// metadata injected above has to be carried across.
		metadata := bankDenomMetadata(app.AppCodec(), genesisState)

		builtState, err := simtestutil.GenesisStateWithValSet(app.AppCodec(), genesisState, valSet, genAccs, balances...)
		if err != nil {
			panic(fmt.Sprintf("failed to build the simulation genesis: %v", err))
		}
		setBankDenomMetadata(app.AppCodec(), builtState, metadata)
		setBondedPoolBalance(app.AppCodec(), builtState, int64(len(valSet.Validators)))
		addValidatorSigningInfo(app.AppCodec(), builtState, valSet)
		relaxFeeMarket(app.AppCodec(), builtState)
		seedNFTGenesis(app.AppCodec(), builtState, accs)

		appState, err := json.Marshal(builtState)
		if err != nil {
			panic(fmt.Sprintf("failed to marshal the simulation genesis: %v", err))
		}
		return appState, accs, simChainID, genesisTimestamp
	}
}

// Deterministic identities for the NFT fixture the simulation genesis carries.
// The collection module's token IDs must be 3 to 128 characters long, which is
// why the Cosmos NFT IDs are not plain numbers while the ERC721 token id is.
const (
	simCollectionDenomID = "upticksim"
	simSeedNFTID         = "nft001"
	simSeedSecondNFTID   = "nft002"
	simSeedEVMTokenID    = "1"
	simERC721Contract    = "0x1234567890abcdef1234567890abcdef12345678"
	simSeedRefundOwner   = "0x00000000000000000000000000000000000000a1"
)

// seedNFTGenesis writes a deterministic NFT fixture into the simulation
// genesis: a collection with two NFTs plus one ERC721 and one CW721 token pair,
// each with its per-token conversion binding and IBC refund receiver.
//
// The randomized operations only touch the NFT modules when they happen to pick
// the collection messages, which many seeds never do. The import/export gate
// would then compare empty NFT stores and pass without protecting anything, so
// the genesis seeds exactly the records those modules own: the collection
// denom/NFT entries whose export path decodes metadata, and the pair,
// conversion-binding and refund-receiver records the erc721/cw721 genesis
// round-trip has to restore.
func seedNFTGenesis(cdc codec.JSONCodec, genesis map[string]json.RawMessage, accs []simtypes.Account) {
	if len(accs) < 2 {
		panic(fmt.Sprintf("seeding the NFT fixture needs at least 2 simulation accounts, got %d", len(accs)))
	}

	owner := accs[0].Address.String()
	cw721Contract := accs[1].Address.String()

	var collectionGen nfttypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[nfttypes.ModuleName], &collectionGen)
	collectionGen.Collections = append(collectionGen.Collections, nfttypes.Collection{
		Denom: nfttypes.Denom{
			Id:          simCollectionDenomID,
			Name:        "Simulation Collection",
			Schema:      "{}",
			Creator:     owner,
			Symbol:      "SIM",
			Description: "deterministic NFT fixture for the import/export gate",
		},
		NFTs: []nfttypes.BaseNFT{
			// The first NFT carries Data so the export path that decodes NFT
			// metadata runs, the second one is bare.
			{Id: simSeedNFTID, Name: "seeded one", Owner: owner, Data: `{"seed":1}`},
			{Id: simSeedSecondNFTID, Name: "seeded two", Owner: owner},
		},
	})
	genesis[nfttypes.ModuleName] = cdc.MustMarshalJSON(&collectionGen)

	var erc721Gen erc721types.GenesisState
	cdc.MustUnmarshalJSON(genesis[erc721types.ModuleName], &erc721Gen)
	erc721Gen.TokenPairs = append(erc721Gen.TokenPairs,
		erc721types.NewTokenPair(common.HexToAddress(simERC721Contract), simCollectionDenomID))
	erc721Gen.NftUidPairs = append(erc721Gen.NftUidPairs, erc721types.NFTUIDPair{
		TokenUid: erc721types.CreateTokenUID(simERC721Contract, simSeedEVMTokenID),
		NftUid:   erc721types.CreateNFTUID(simCollectionDenomID, simSeedNFTID),
	})
	erc721Gen.RefundReceivers = append(erc721Gen.RefundReceivers, erc721types.RefundReceiver{
		EvmContractAddress: simERC721Contract,
		TokenId:            simSeedEVMTokenID,
		EvmAddress:         simSeedRefundOwner,
	})
	genesis[erc721types.ModuleName] = cdc.MustMarshalJSON(&erc721Gen)

	var cw721Gen cw721types.GenesisState
	cdc.MustUnmarshalJSON(genesis[cw721types.ModuleName], &cw721Gen)
	cw721Gen.TokenPairs = append(cw721Gen.TokenPairs,
		cw721types.NewTokenPair(cw721Contract, simCollectionDenomID))
	cw721Gen.NftUidPairs = append(cw721Gen.NftUidPairs, cw721types.NFTUIDPair{
		TokenUid: seedUID(simSeedEVMTokenID, cw721Contract),
		NftUid:   seedUID(simSeedNFTID, simCollectionDenomID),
	})
	cw721Gen.RefundReceivers = append(cw721Gen.RefundReceivers, cw721types.RefundReceiver{
		ContractAddress: cw721Contract,
		TokenId:         simSeedEVMTokenID,
		Owner:           owner,
	})
	genesis[cw721types.ModuleName] = cdc.MustMarshalJSON(&cw721Gen)
}

// seedUID builds the "<id>,<contract-or-class>" form both the CW721 module's
// own UID helpers and the token UID helpers use. The CW721 package ships
// GetNFTFromUID without the matching constructors, so the format is spelled out
// here rather than duplicated from the collection side by accident.
func seedUID(id, contractOrClass string) string {
	return fmt.Sprintf("%s,%s", id, contractOrClass)
}

// bankDenomMetadata reads the bank metadata out of a genesis map.
func bankDenomMetadata(cdc codec.JSONCodec, genesis map[string]json.RawMessage) []banktypes.Metadata {
	var bankGen banktypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[banktypes.ModuleName], &bankGen)
	return bankGen.DenomMetadata
}

// setBankDenomMetadata writes the bank metadata back into a genesis map.
func setBankDenomMetadata(cdc codec.JSONCodec, genesis map[string]json.RawMessage, metadata []banktypes.Metadata) {
	var bankGen banktypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[banktypes.ModuleName], &bankGen)
	bankGen.DenomMetadata = metadata
	genesis[banktypes.ModuleName] = cdc.MustMarshalJSON(&bankGen)
}

// setBondedPoolBalance funds the bonded pool module account with the full
// bonded amount.
//
// simtestutil.GenesisStateWithValSet (SDK v0.53.6) writes one delegation per
// validator and adds one bond amount per delegation to the total supply, but it
// only ever funds the bonded pool with a single bond amount. Its genesis is
// therefore balanced only when the validator set has exactly one member; with
// more, bank.InitGenesis aborts with
//
//	genesis supply is incorrect, expected <supply>, got <sum of balances>
//
// because the supply exceeds the balances by (validators-1) bond amounts.
func setBondedPoolBalance(cdc codec.JSONCodec, genesis map[string]json.RawMessage, validatorCount int64) {
	poolAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String()
	balance := banktypes.Balance{
		Address: poolAddr,
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.DefaultPowerReduction.MulRaw(validatorCount))),
	}

	var bankGen banktypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[banktypes.ModuleName], &bankGen)

	for i, existing := range bankGen.Balances {
		if existing.Address == poolAddr {
			bankGen.Balances[i] = balance
			genesis[banktypes.ModuleName] = cdc.MustMarshalJSON(&bankGen)
			return
		}
	}

	bankGen.Balances = append(bankGen.Balances, balance)
	genesis[banktypes.ModuleName] = cdc.MustMarshalJSON(&bankGen)
}

// addValidatorSigningInfo registers the simulated validator with the slashing
// module. GenesisStateWithValSet writes validators directly into the staking
// genesis, bypassing the genutil gentx path that normally records signing info;
// without this FinalizeBlock aborts with "no validator signing info found".
func addValidatorSigningInfo(cdc codec.JSONCodec, genesis map[string]json.RawMessage, valSet *cmttypes.ValidatorSet) {
	var slashingGen slashingtypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[slashingtypes.ModuleName], &slashingGen)

	for _, val := range valSet.Validators {
		consAddr := sdk.ConsAddress(val.Address).String()
		slashingGen.SigningInfos = append(slashingGen.SigningInfos, slashingtypes.SigningInfo{
			Address: consAddr,
			ValidatorSigningInfo: slashingtypes.ValidatorSigningInfo{
				Address:             consAddr,
				StartHeight:         0,
				JailedUntil:         simGenesisTime,
				Tombstoned:          false,
				MissedBlocksCounter: 0,
			},
		})
	}
	genesis[slashingtypes.ModuleName] = cdc.MustMarshalJSON(&slashingGen)
}

// relaxFeeMarket disables the base fee for the simulated chain. The generated
// operations sign with a zero gas price, so a live base fee rejects every one
// of them with "gas prices too low" and the simulation would only ever produce
// empty blocks.
func relaxFeeMarket(cdc codec.JSONCodec, genesis map[string]json.RawMessage) {
	var fmGen feemarkettypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[feemarkettypes.ModuleName], &fmGen)
	fmGen.Params.NoBaseFee = true
	fmGen.Params.MinGasPrice = math.LegacyZeroDec()
	fmGen.Params.BaseFee = math.LegacyZeroDec()
	genesis[feemarkettypes.ModuleName] = cdc.MustMarshalJSON(&fmGen)
}

// simGenesisValidatorCount is the number of validators the simulated chain
// starts with.
//
// A single validator is not enough: SimulateFromSeed drives its own mock
// CometBFT validator set from the app's EndBlock updates and calls tb.Skip the
// moment that set becomes empty. One generated operation that zeroes the only
// validator's power (a full undelegation, say) is enough to trigger it, and
// because tb.Skip unwinds the test goroutine with runtime.Goexit the
// post-simulation export never runs — the determinism and import/export gates
// would silently degrade into no-ops. Several validators make that unreachable.
const simGenesisValidatorCount = 5

// simGenesisAccounts derives deterministic validators plus a funded genesis
// account for every simulated account, so the generated operations can actually
// pay fees instead of failing on an empty balance.
func simGenesisAccounts(accs []simtypes.Account) (*cmttypes.ValidatorSet, []authtypes.GenesisAccount, []banktypes.Balance) {
	bondDenom := sdk.DefaultBondDenom

	stake := sdk.DefaultPowerReduction.MulRaw(1_000_000)

	genAccs := make([]authtypes.GenesisAccount, 0, len(accs)+simGenesisValidatorCount)
	balances := make([]banktypes.Balance, 0, len(accs)+simGenesisValidatorCount)
	for _, acc := range accs {
		genAccs = append(genAccs, authtypes.NewBaseAccountWithAddress(acc.Address))
		balances = append(balances, banktypes.Balance{
			Address: acc.Address.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(bondDenom, stake)),
		})
	}

	// Deterministic validators: fixed secrets keep the harness reproducible,
	// which the determinism test relies on.
	validators := make([]*cmttypes.Validator, 0, simGenesisValidatorCount)
	for i := 0; i < simGenesisValidatorCount; i++ {
		privKey := cmted25519.GenPrivKeyFromSecret([]byte(fmt.Sprintf("uptick-simulation-validator-%d", i)))
		pubKey := privKey.PubKey()
		valAddr := sdk.AccAddress(pubKey.Address())

		genAccs = append(genAccs, authtypes.NewBaseAccountWithAddress(valAddr))
		balances = append(balances, banktypes.Balance{
			Address: valAddr.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(bondDenom, stake)),
		})

		validators = append(validators, cmttypes.NewValidator(pubKey, 1))
	}

	return cmttypes.NewValidatorSet(validators), genAccs, balances
}

func simRandAccFn(r *rand.Rand, n int) []simtypes.Account {
	return simtypes.RandomAccounts(r, n)
}

// simOps builds the weighted operation list from every module that registered
// simulation operations with the app's SimulationManager. x/staking is
// substituted with stakingSimOps so its create/edit-validator operations carry
// a commission the chain's ante handler accepts (see sim_staking_ops_test.go).
func simOps(app *Uptick, config simtypes.Config) simulation.WeightedOperations {
	simState := simStateFor(app, config)

	ops := make(simulation.WeightedOperations, 0, 128)
	for _, m := range app.SimulationManager().Modules {
		if named, ok := m.(module.AppModule); ok && named.Name() == stakingtypes.ModuleName {
			ops = append(ops, stakingSimOps(app, simState)...)
			continue
		}
		ops = append(ops, m.WeightedOperations(simState)...)
	}
	return ops
}

// simExport is what a simulation hands to the importing side: the genesis state
// plus the consensus parameters. ExportAppStateAndValidators keeps the
// consensus parameters out of the genesis map because a real node receives them
// from CometBFT in the InitChain request, so they have to travel separately or
// the re-imported chain loses them.
type simExport struct {
	AppState        json.RawMessage          `json:"app_state"`
	ConsensusParams cmtproto.ConsensusParams `json:"consensus_params"`
}

// runSimulation executes one randomized simulation and returns the exported
// application, which the determinism and import/export tests consume.
//
// observe, when set, is called with the live application right before the
// database is torn down. The import/export gate uses it to hash the committed
// module stores, which is the only way to compare them with the stores a fresh
// application rebuilds from the exported genesis.
func runSimulation(t testing.TB, config simtypes.Config, observe func(*Uptick)) simExport {
	t.Helper()

	db, dir := simDB(t, "sim")
	defer func() {
		_ = db.Close()
		_ = os.RemoveAll(dir)
	}()

	app := newSimApp(t, db, dir)

	_, params, err := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		app.BaseApp,
		simAppStateFn(app),
		simRandAccFn,
		simOps(app, config),
		app.ModuleAccountAddrs(),
		config,
		app.AppCodec(),
	)
	require.NoError(t, err, "simulation failed")

	// Write -ExportStatePath / -ExportParamsPath when runsim asked for them.
	if err := simtestutil.CheckExportSimulation(app, config, params); err != nil {
		t.Fatalf("exporting the simulation artefacts failed: %v", err)
	}

	exported, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err, "export after simulation failed")

	if observe != nil {
		observe(app)
	}

	return simExport{AppState: exported.AppState, ConsensusParams: exported.ConsensusParams}
}

func stateHash(t testing.TB, state []byte) string {
	t.Helper()
	sum := sha256.Sum256(state)
	return fmt.Sprintf("%x", sum)
}

// TestFullAppSimulation runs the randomized simulation for a fixed number of
// blocks. It is the smoke test of the whole harness: app construction, genesis
// generation, operation execution and export must all succeed.
func TestFullAppSimulation(t *testing.T) {
	if !*simEnabled {
		t.Skip("simulation disabled: pass -Enabled=true to run")
	}

	state := runSimulation(t, simConfig(t, 10, 5), nil)
	require.NotEmpty(t, state.AppState)
}
