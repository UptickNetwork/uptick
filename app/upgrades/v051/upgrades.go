// Package v051 implements the v0.5.1 upgrade plan.
//
// # Who runs this plan
//
//   - Mainnet: last completed plan is v0.5.0 → upgrades to v0.5.1
//   - Testnet (Origin3): last completed plan is v0.4.1 → upgrades to v0.5.1
//
// # What it does
//
//  1. Refuses a legacy ethermint (pre-v0.4.0) starting state — those chains
//     must take v0.5.0 first.
//  2. Re-applies the v0.5.0 tail migrations (permissionless erc20 registration,
//     voucher decimals, voucher token-pair backfill). Idempotent on mainnet;
//     the only path for testnet, which never ran v0.5.0.
//  3. Adds the x/appmining store and lets RunMigrations InitGenesis the module
//     (ConsensusVersion 1) with default params and the reward-pool module account.
//
// # Store upgrades
//
// Added: appmining. Deleted: none. Renamed: none.
package v051

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
	v050 "github.com/UptickNetwork/uptick/app/upgrades/v050"
	appminingtypes "github.com/UptickNetwork/uptick/x/appmining/types"
)

const upgradeName = "v0.5.1"

// migrationsAppliedKey records that this handler completed, so a replayed plan
// skips the (idempotent) custom steps and only re-runs RunMigrations.
var migrationsAppliedKey = []byte("v0.5.1/migrations-applied")

// Upgrade is the v0.5.1 upgrade plan.
var Upgrade = upgrades.Upgrade{
	UpgradeName:               upgradeName,
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades: &storetypes.StoreUpgrades{
		Added: []string{appminingtypes.StoreKey},
	},
}

func upgradeHandlerConstructor(
	_ *module.Manager,
	c module.Configurator,
	box upgrades.Toolbox,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		logger := sdkCtx.Logger()

		storeKey := box.GetKVStoreKey(upgradetypes.StoreKey)
		if storeKey == nil {
			return nil, fmt.Errorf("x/upgrade store key is not registered")
		}
		upgradeStore := sdkCtx.KVStore(storeKey)

		if upgradeStore.Has(migrationsAppliedKey) {
			logger.Info("v0.5.1 migration set already applied; re-running module migrations only",
				"upgrade", upgradeName)
			return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
		}

		// Refuse pre-v0.4.0 ethermint state. Mainnet reaches v0.5.1 via v0.5.0;
		// testnet is already on cosmos/evm after v0.4.1. Neither should present
		// a missing EvmCoinInfo record.
		if !evmCoinInfoPresent(sdkCtx, box.GetKVStoreKey(evmtypes.StoreKey)) {
			return nil, fmt.Errorf(
				"v0.5.1 refuses a legacy ethermint starting state: EvmCoinInfo is absent; " +
					"mainnet must execute v0.5.0 first, testnet must already be on v0.4.1",
			)
		}

		logger.Info("executing upgrade plan",
			"name", upgradeName,
			"change", "v0.5.0 tail migrations + x/appmining store",
		)

		if err := v050.ApplyTailMigrations(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("v0.5.0 tail migrations: %w", err)
		}

		// RunMigrations InitGenesis's x/appmining (new ConsensusVersion 1) and
		// advances every other module that already matches to a no-op.
		newVM, err := box.ModuleManager.RunMigrations(sdkCtx, c, vm)
		if err != nil {
			return nil, fmt.Errorf("run migrations: %w", err)
		}

		// Ensure segregated fund accounts exist (permissions are in maccPerms).
		if acc := box.AccountKeeper.GetModuleAccount(sdkCtx, appminingtypes.RewardPoolName); acc == nil {
			return nil, fmt.Errorf("appmining_reward_pool module account was not created during upgrade")
		}
		if acc := box.AccountKeeper.GetModuleAccount(sdkCtx, appminingtypes.ChallengeEscrowName); acc == nil {
			return nil, fmt.Errorf("appmining_challenge_escrow module account was not created during upgrade")
		}

		// RC/dev builds once funded the legacy ModuleName account; settleable
		// funds now live in RewardPoolName. Move any leftover balances over.
		if err := migrateLegacyAppminingPool(sdkCtx, box); err != nil {
			return nil, err
		}

		upgradeStore.Set(migrationsAppliedKey, []byte{0x01})
		logger.Info("v0.5.1 migration set complete", "upgrade", upgradeName)
		return newVM, nil
	}
}

// migrateLegacyAppminingPool moves balances from the legacy "appmining" module
// account into appmining_reward_pool when present and non-empty.
func migrateLegacyAppminingPool(ctx sdk.Context, box upgrades.Toolbox) error {
	legacy := box.AccountKeeper.GetModuleAccount(ctx, appminingtypes.ModuleName)
	if legacy == nil {
		return nil
	}
	bal := box.BankKeeper.GetAllBalances(ctx, legacy.GetAddress())
	if bal.IsZero() {
		return nil
	}
	if err := box.BankKeeper.SendCoinsFromModuleToModule(
		ctx, appminingtypes.ModuleName, appminingtypes.RewardPoolName, bal,
	); err != nil {
		return fmt.Errorf("migrate legacy appmining balances to reward pool: %w", err)
	}
	ctx.Logger().Info("migrated legacy appmining module balances to reward pool",
		"upgrade", upgradeName,
		"amount", bal.String(),
	)
	return nil
}

func evmCoinInfoPresent(ctx sdk.Context, evmStoreKey *storetypes.KVStoreKey) bool {
	if evmStoreKey == nil {
		return false
	}
	return ctx.KVStore(evmStoreKey).Has(evmtypes.KeyPrefixEvmCoinInfo)
}
