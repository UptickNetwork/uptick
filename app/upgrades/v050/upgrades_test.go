package v050

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
)

// TestEvmCoinInfoProbe pins the probe that decides which migration set a v0.5.0
// node runs. It is the counter-test for that decision: a probe that returns a
// constant fails one of the two cases, and so does one keyed on the wrong byte
// -- a probe that reads "any key in the EVM store" would call every v0.3.3 chain
// migrated (the legacy layout writes 0x01-0x03), and the destructive v0.4.0
// replay would run on the one chain that must not see it.
func TestEvmCoinInfoProbe(t *testing.T) {
	t.Parallel()

	key := storetypes.NewKVStoreKey(evmtypes.StoreKey)
	ctx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient")).Ctx

	// A v0.3.3 state: the legacy ethermint key layout stops at prefixParams
	// (0x03) and never writes the 0x05 prefix cosmos/evm reserves for EvmCoinInfo.
	require.False(t, evmCoinInfoInitialized(ctx, key),
		"an EVM store without the EvmCoinInfo record must read as a v0.3.3 state")

	// Every prefix the legacy layout can write leaves the probe on the legacy
	// branch. Written as real entries (prefix + one byte) rather than bare
	// prefixes, which is the shape the legacy store actually holds.
	for _, legacy := range [][]byte{
		evmtypes.KeyPrefixCode,
		evmtypes.KeyPrefixStorage,
		evmtypes.KeyPrefixParams,
		evmtypes.KeyPrefixCodeHash,
	} {
		entry := make([]byte, 0, len(legacy)+1)
		entry = append(entry, legacy...)
		entry = append(entry, 0x01)
		ctx.KVStore(key).Set(entry, []byte{0x01})
	}
	require.False(t, evmCoinInfoInitialized(ctx, key),
		"legacy EVM entries must not be mistaken for the EvmCoinInfo record")

	// A v0.4.x state: the v0.4.0 handler persists the record via
	// EvmKeeper.InitEvmCoinInfo, and nothing afterwards removes it.
	ctx.KVStore(key).Set(evmtypes.KeyPrefixEvmCoinInfo, []byte{0x01})
	require.True(t, evmCoinInfoInitialized(ctx, key),
		"a persisted EvmCoinInfo record must read as a v0.4.x state")
}

// TestEvmCoinInfoProbeWithoutStoreKey covers the guard that keeps a misconfigured
// app from panicking inside the handler: an unresolvable EVM store falls back to
// the full-migration path instead of dereferencing a nil key.
func TestEvmCoinInfoProbeWithoutStoreKey(t *testing.T) {
	t.Parallel()

	ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())

	require.False(t, evmCoinInfoInitialized(ctx, nil),
		"an unregistered EVM store must fall back to the full-migration path, not panic")
}

// TestUpgradeMetadata pins the plan name and the store change mainnet's node
// will follow. The Deleted entry is not decorative: the store loader is keyed on
// the plan name, so this is the only place a "v0.5.0" node learns the capability
// store is gone. The name itself is what the governance proposal must carry
// verbatim.
func TestUpgradeMetadata(t *testing.T) {
	t.Parallel()

	require.Equal(t, "v0.5.0", Upgrade.UpgradeName)
	require.NotNil(t, Upgrade.StoreUpgrades)
	require.Equal(t, []string{"capability"}, Upgrade.StoreUpgrades.Deleted)
	require.Empty(t, Upgrade.StoreUpgrades.Added)
	require.Empty(t, Upgrade.StoreUpgrades.Renamed)
}

// TestMigrationsAppliedKeyIsOutsideTheUpgradeStoreKeySpace guards where the
// replay marker lives. It is written into x/upgrade's own store, whose own keys
// are single low bytes (PlanByte 0x00, DoneByte 0x01, VersionMapByte 0x02,
// ProtocolVersionByte 0x03) plus the "upgradedIBCState" string prefix; a marker
// sharing either would be silently overwritten or misread by the module.
func TestMigrationsAppliedKeyIsOutsideTheUpgradeStoreKeySpace(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, migrationsAppliedKey)
	require.NotEqual(t, byte('u'), migrationsAppliedKey[0],
		"the marker must not sit under x/upgrade's upgradedIBCState prefix")
	require.Greater(t, migrationsAppliedKey[0], byte(0x0f),
		"the replay marker must not share x/upgrade's low-byte key space")
}

// TestStartingStateNames keeps the two operator-facing log labels distinct, so
// an upgrade log tells the operator which branch ran.
func TestStartingStateNames(t *testing.T) {
	t.Parallel()

	require.Contains(t, startingState(true), "v0.3.3")
	require.Contains(t, startingState(false), "v0.4.x")
	require.NotEqual(t, startingState(true), startingState(false))
}

// TestRunMigrationSetMainnetPath pins the composition a mainnet node performs:
// the v0.4.0 change set first, the v0.4.1 repairs second, and the repairs see
// the version map the first stage returned -- the second RunMigrations is only
// harmless because it is handed an already-migrated map.
func TestRunMigrationSetMainnetPath(t *testing.T) {
	t.Parallel()

	var order []string
	legacy := func(_ context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		order = append(order, "v0.4.0")
		require.Empty(t, vm, "the legacy stage must be the first to see the version map")
		return module.VersionMap{"evm": 2}, nil
	}
	repairs := func(_ context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		order = append(order, "v0.4.1")
		require.Equal(t, module.VersionMap{"evm": 2}, vm,
			"the repairs must run on the map the legacy stage returned")
		return module.VersionMap{"evm": 2, "erc721": 3}, nil
	}

	vm, err := runMigrationSet(context.Background(), upgradetypes.Plan{}, module.VersionMap{},
		true, stages{legacy: legacy, repairs: repairs})
	require.NoError(t, err)
	require.Equal(t, []string{"v0.4.0", "v0.4.1"}, order)
	require.Equal(t, module.VersionMap{"evm": 2, "erc721": 3}, vm)
}

// TestRunMigrationSetTestnetPath is the other side of the probe: a chain already
// on v0.4.x must not have the v0.4.0 change set replayed over it. Only the
// idempotent repairs run.
func TestRunMigrationSetTestnetPath(t *testing.T) {
	t.Parallel()

	legacy := func(context.Context, upgradetypes.Plan, module.VersionMap) (module.VersionMap, error) {
		t.Fatal("the v0.4.0 change set must not run from a v0.4.x state")
		return nil, nil
	}
	repairs := func(_ context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	}

	vm, err := runMigrationSet(context.Background(), upgradetypes.Plan{}, module.VersionMap{"evm": 2},
		false, stages{legacy: legacy, repairs: repairs})
	require.NoError(t, err)
	require.Equal(t, module.VersionMap{"evm": 2}, vm)
}

// TestRunMigrationSetNamesTheFailingStage: a halt has to point at the change set
// that failed, and a failed legacy stage must stop the sequence rather than run
// the repairs over a half-applied chain.
func TestRunMigrationSetNamesTheFailingStage(t *testing.T) {
	t.Parallel()

	legacyErr := errors.New("legacy pairs")
	legacy := func(context.Context, upgradetypes.Plan, module.VersionMap) (module.VersionMap, error) {
		return nil, legacyErr
	}
	repairsRan := false
	repairs := func(context.Context, upgradetypes.Plan, module.VersionMap) (module.VersionMap, error) {
		repairsRan = true
		return nil, nil
	}

	vm, err := runMigrationSet(context.Background(), upgradetypes.Plan{}, module.VersionMap{},
		true, stages{legacy: legacy, repairs: repairs})
	require.ErrorIs(t, err, legacyErr)
	require.Contains(t, err.Error(), "v0.4.0 migration set")
	require.Nil(t, vm)
	require.False(t, repairsRan, "a failed change set must stop the sequence")

	repairsErr := errors.New("base fee")
	repairsFails := func(context.Context, upgradetypes.Plan, module.VersionMap) (module.VersionMap, error) {
		return nil, repairsErr
	}

	_, err = runMigrationSet(context.Background(), upgradetypes.Plan{}, module.VersionMap{},
		false, stages{legacy: legacy, repairs: repairsFails})
	require.ErrorIs(t, err, repairsErr)
	require.Contains(t, err.Error(), "v0.4.1 repairs")
}

// TestBackfillFailuresKeyIsOutsideTheUpgradeStoreKeySpace keeps the failure
// record in the same safe key space as the replay marker, and distinct from it:
// sharing x/upgrade's own low-byte keys, or the marker's key, would let one
// record silently overwrite or corrupt the other.
func TestBackfillFailuresKeyIsOutsideTheUpgradeStoreKeySpace(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, backfillFailuresKey)
	require.NotEqual(t, byte('u'), backfillFailuresKey[0],
		"the failure record must not sit under x/upgrade's upgradedIBCState prefix")
	require.Greater(t, backfillFailuresKey[0], byte(0x0f),
		"the failure record must not share x/upgrade's low-byte key space")
	require.NotEqual(t, string(migrationsAppliedKey), string(backfillFailuresKey),
		"the failure record must not overwrite the replay marker")
}
