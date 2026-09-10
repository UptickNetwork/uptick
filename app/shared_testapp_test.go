package app

import (
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// sharedTestApp returns the one application instance this test binary is
// allowed to build, together with a context committed on top of its first
// block.
//
// THE SINGLETON IS NOT AN OPTIMISATION. cosmos/evm publishes the chain's coin
// and EVM configuration through package-level state: the second
// x/vm.SetGlobalConfigVariables call in a process panics with "EVM coin info
// already set" (x/vm/types/denom_config.go). Every test in this package that
// needs a real app - the ERC721 EVM suite and the upgrade handler suite - must
// therefore go through this function. Adding a second `NewUptick` call
// anywhere below is a race that shows up as a panic, not a test failure.
//
// The app is initialised to the same state the simulation harness uses: a
// self-consistent auth/bank/staking genesis, one committed block so the EVM
// account bookkeeping is consistent, and a context that reads the committed
// store without writing back to it (so keeper mutations performed by one test
// cannot leak into the next).
func sharedTestApp(t *testing.T) (*Uptick, sdk.Context) {
	t.Helper()

	sharedTestAppOnce.Do(func() {
		db, dir := simDB(t, "shared-testapp")
		t.Cleanup(func() {
			_ = db.Close()
			_ = os.RemoveAll(dir)
		})

		app := newSimApp(t, db, dir)

		r := rand.New(rand.NewSource(7))
		accs := simRandAccFn(r, 3)
		appState, _, chainID, genesisTime := simAppStateFn(app)(r, accs, simtypes.Config{})

		// simGenesisAccounts derives the genesis validators from these fixed
		// secrets; reuse the first one as the block proposer so the EVM keeper
		// can resolve a coinbase.
		proposer := sdk.ConsAddress(cmted25519.GenPrivKeyFromSecret([]byte("uptick-simulation-validator-0")).PubKey().Address())

		consensusParams := cmttypes.DefaultConsensusParams().ToProto()
		_, err := app.InitChain(&abci.RequestInitChain{
			ChainId:         chainID,
			Time:            genesisTime,
			ConsensusParams: &consensusParams,
			AppStateBytes:   appState,
			InitialHeight:   1,
		})
		require.NoError(t, err, "InitChain failed")

		// The EVM module publishes its global coin/chain configuration from
		// InitGenesis, and its account bookkeeping only becomes consistent once
		// a block has been produced. Drive one real block before touching the
		// keepers.
		_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:          1,
			Time:            genesisTime.Add(time.Second),
			ProposerAddress: proposer,
		})
		require.NoError(t, err, "FinalizeBlock failed")
		_, err = app.Commit()
		require.NoError(t, err, "Commit failed")

		sharedTestAppApp = app
		sharedTestAppCtx = app.NewUncachedContext(false, cmtproto.Header{
			ChainID:         chainID,
			Height:          app.LastBlockHeight(),
			Time:            genesisTime.Add(2 * time.Second),
			ProposerAddress: proposer,
		})
	})

	require.NotNil(t, sharedTestAppApp, "the shared test application failed to initialise")
	return sharedTestAppApp, sharedTestAppCtx
}

var (
	sharedTestAppOnce sync.Once
	sharedTestAppApp  *Uptick
	sharedTestAppCtx  sdk.Context
)
