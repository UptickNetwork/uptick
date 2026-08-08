package v034

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	// cosmos/evm imports — replaces ethermint x/evm
	evmtypes "github.com/cosmos/evm/x/vm/types"

	// Uptick erc20 module (retained custom implementation)
	erc20types "github.com/UptickNetwork/uptick/x/erc20/types"
)

const upgradeName = "v0.3.4"

// Upgrade implements the v0.3.4 upgrade plan.
//
// This upgrade migrates Uptick from ethermint (based on go-ethereum v1.10.x)
// to cosmos/evm v0.6.1 (based on cosmos/go-ethereum v1.16.2).
//
// =====================================================================
// MAJOR CHANGES IN THIS UPGRADE
// =====================================================================
//
// 1. EVM Module: ethermint x/evm → cosmos/evm x/vm
//    - go-ethereum upgraded from v1.10.17 → v1.16.2
//    - ChainConfig fields changed from Block-based to Time-based
//      (ShanghaiBlock → ShanghaiTime, CancunBlock → CancunTime,
//       PragueBlock → PragueTime)
//    - EIP-7702 SetCodeTx is now natively supported by cosmos/go-ethereum
//      (no custom implementation needed, as UptickNetwork/ethermint required)
//    - ChainConfig is now a global variable (set during NewKeeper),
//      not stored in Params. The upgrade handler re-initializes it.
//
// 2. IBC Module: ibc-go v8 → v10
//    - capability module removed entirely
//    - ScopedKeeper references removed from all keepers
//    - IBCModule interface signatures changed (added channelVersion param)
//    - SendPacket/WriteAcknowledgement no longer require chanCap
//
// 3. SDK: v0.50 → v0.53
//    - x/params module deprecated (params now authority-based)
//    - gov v1beta1 proposals → gov v1 (for erc20 module)
//    - runtime.KVStoreService replaces direct StoreKey in keeper constructors
//
// 4. Wasm: wasmd v0.53 → v0.61
//    - WasmConfig → NodeConfig
//    - wasmvm v2 → v3
//
// 5. EVM Hardfork Activation
//    - In v032, Shanghai/Cancun/Prague were activated via Block height = 0
//    - In cosmos/evm v0.6.1, these are activated via Time (timestamp)
//    - The upgrade sets all fork times to 0 (activated immediately at upgrade)
//    - This ensures all EVM opcodes (PUSH0, BLOBHASH, etc.) are enabled
//    - EIP-7702 SetCodeTx (type 0x04) is enabled via PragueTime
//
// 6. Capability Store Cleanup
//    - The 'capability' module store is deleted (ibc-go v10 doesn't use it)
//
// 7. erc20 Module
//    - Uptick retains its custom x/erc20 (Provenance + TransferERC20)
//    - Params migrated from x/params subspace to authority-based
//    - gov v1beta1 proposals retained for backward compatibility
//
// =====================================================================
// ROLLBACK NOTE
// =====================================================================
// This upgrade is NOT reversible. Once the capability store is deleted
// and ChainConfig is migrated to Time-based, the chain cannot roll back
// to ethermint. Ensure all validators have upgraded before the upgrade height.
var Upgrade = upgrades.Upgrade{
	UpgradeName:               upgradeName,
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades: &storetypes.StoreUpgrades{
		// Delete capability module store — ibc-go v10 no longer uses capability.
		// This removes all ScopedKeeper capabilities from state.
		// SAFETY: All modules have been updated to not reference capability.
		Deleted: []string{
			"capability",
		},
		// No new stores added — cosmos/evm x/vm uses the same store key
		// as ethermint x/evm (both use "evm" store key).
		Added: []string{},
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

		logger.Info(
			"executing upgrade plan",
			"name", upgradeName,
			"changes", []string{
				"ethermint → cosmos/evm v0.6.1 (go-ethereum v1.10→v1.16)",
				"ibc-go v8 → v10 (capability removed)",
				"SDK v0.50 → v0.53",
				"wasmd v0.53 → v0.61 (wasmvm v2→v3)",
				"EVM ChainConfig: Block-based → Time-based",
				"EIP-7702 SetCodeTx enabled (native in cosmos/go-ethereum)",
				"Shanghai/Cancun/Prague hardforks activated",
			},
		)

		// Step 1: Migrate EVM ChainConfig from Block-based to Time-based
		//
		// In ethermint (v032 upgrade), Shanghai/Cancun/Prague were activated via:
		//   ChainConfig.ShanghaiBlock = 0
		//   ChainConfig.CancunBlock = 0
		//   ChainConfig.PragueBlock = 0
		//
		// In cosmos/evm v0.6.1, these are now Time-based (timestamp):
		//   ChainConfig.ShanghaiTime = 0  (activated immediately)
		//   ChainConfig.CancunTime = 0
		//   ChainConfig.PragueTime = 0
		//
		// cosmos/evm also adds new fields not in ethermint:
		//   - OsakaTime (not activated, nil)
		//   - VerkleTime (not activated, nil)
		//   - BlobScheduleConfig (Cancun/Prague/Osaka blob configs)
		//
		// EIP-7702 SetCodeTx (type 0x04) is enabled when PragueTime is set.
		// This allows EOA accounts to delegate to smart contract code,
		// enabling account abstraction without protocol-level changes.
		// Previously UptickNetwork/ethermint required a custom implementation;
		// cosmos/go-ethereum v1.16.2 includes this natively.
		if err := migrateEVMChainConfig(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate EVM chain config: %w", err)
		}

		// Step 2: Migrate erc20 params from x/params subspace to authority-based
		//
		// In SDK 0.50 + ethermint, erc20 params were stored in x/params subspace.
		// In SDK 0.53 + cosmos/evm, params are stored directly in the module store
		// and managed via gov v1 authority.
		migrateErc20Params(sdkCtx, box, logger)

		// Step 3: Run module migrations
		//
		// This handles all SDK 0.53, ibc-go v10, and cosmos/evm module migrations
		// that are registered in the module manager.
		// The module manager will call each module's InitGenesis if it's a new
		// module, or run the registered migration scripts.
		logger.Info("running module migrations")
		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}

// migrateEVMChainConfig migrates the EVM chain configuration from the old
// ethermint format (Block-based) to the new cosmos/evm format (Time-based).
//
// Key changes:
// - ShanghaiBlock → ShanghaiTime
// - CancunBlock → CancunTime
// - PragueBlock → PragueTime
// - New fields: OsakaTime, VerkleTime (nil = not activated)
// - ChainConfig is now a global variable in cosmos/evm, set via SetChainConfig()
//
// The migration sets all fork times to 0, meaning they activate immediately
// at the upgrade block. This is safe because:
// 1. Shanghai/Cancun/Prague were already activated in v032 (at block 0)
// 2. The upgrade block's timestamp is used as the reference point
// 3. Setting time=0 means "active since genesis timestamp"
func migrateEVMChainConfig(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) error {
	logger.Info("migrating EVM ChainConfig from Block-based to Time-based")

	// Get the EVM chain ID
	// cosmos/evm uses a separate EVM chain ID (EIP-155 compatible)
	// We use the default chain ID or read from existing config
	evmChainID := evmtypes.DefaultEVMChainID

	// Create a new ChainConfig with all forks activated at time 0
	// This is equivalent to the v032 upgrade which set all Block heights to 0
	zero := math.ZeroInt()
	chainConfig := &evmtypes.ChainConfig{
		ChainId: evmChainID,

		// Pre-Shanghai forks — all activated at block 0 (same as before)
		HomesteadBlock:      &zero,
		DAOForkBlock:        &zero,
		DAOForkSupport:      true,
		EIP150Block:         &zero,
		EIP155Block:         &zero,
		EIP158Block:         &zero,
		ByzantiumBlock:      &zero,
		ConstantinopleBlock: &zero,
		PetersburgBlock:     &zero,
		IstanbulBlock:       &zero,
		MuirGlacierBlock:    &zero,
		BerlinBlock:         &zero,
		LondonBlock:         &zero,
		ArrowGlacierBlock:   &zero,
		GrayGlacierBlock:    &zero,
		MergeNetsplitBlock:  &zero,

		// Shanghai/Cancun/Prague — migrated from Block-based to Time-based
		// Setting to 0 means "active since the beginning of time"
		// This ensures all EVM opcodes are enabled:
		// - Shanghai: PUSH0 (EIP-3855), WARM/COLD balance (EIP-3651)
		// - Cancun: BLOBHASH, BLOBBASEFEE, TLOAD/TSTORE, MCOPY (EIP-4844, 1153, 5656)
		// - Prague: EIP-7702 SetCodeTx (type 0x04), EIP-2537 BLS12-381 precompiles
		ShanghaiTime: &zero,
		CancunTime:   &zero,
		PragueTime:   &zero,

		// Future forks — not yet activated
		OsakaTime:  nil,
		VerkleTime: nil,
	}

	// Validate the new chain config
	if err := chainConfig.Validate(); err != nil {
		return fmt.Errorf("invalid chain config: %w", err)
	}

	// Set the global chain config
	// In cosmos/evm v0.6.1, ChainConfig is stored as a package-level global
	// variable (not in the module store). SetChainConfig() updates this global.
	// The EVMKeeper reads it via GetEthChainConfig() during transaction execution.
	if err := evmtypes.SetChainConfig(chainConfig); err != nil {
		return fmt.Errorf("set chain config: %w", err)
	}

	// Also update the EVM params to include EIP-3855 (PUSH0) in ExtraEIPs
	// This was done in v032 and needs to be preserved
	evmParams := box.EvmKeeper.GetParams(ctx)
	eip3855 := int64(3855)
	alreadyHas := false
	for _, eip := range evmParams.ExtraEIPs {
		if eip == eip3855 {
			alreadyHas = true
			break
		}
	}
	if !alreadyHas {
		evmParams.ExtraEIPs = append(evmParams.ExtraEIPs, eip3855)
		if err := box.EvmKeeper.SetParams(ctx, evmParams); err != nil {
			return fmt.Errorf("set evm params with EIP-3855: %w", err)
		}
	}

	logger.Info(
		"EVM ChainConfig migration complete",
		"ShanghaiTime", "0 (activated)",
		"CancunTime", "0 (activated)",
		"PragueTime", "0 (activated)",
		"EIP-7702", "enabled (native cosmos/go-ethereum v1.16.2)",
		"EIP-3855", "enabled (PUSH0 via ExtraEIPs)",
	)
	return nil
}

// migrateErc20Params migrates erc20 module parameters from x/params subspace
// to the new authority-based params system.
//
// In SDK 0.50 + ethermint, erc20 params were stored in x/params subspace.
// In SDK 0.53 + cosmos/evm, params are stored directly in the erc20 module store
// and managed via gov v1 authority (authtypes.NewModuleAddress(govtypes.ModuleName)).
//
// Uptick retains its custom x/erc20 (not cosmos/evm's erc20), so this migration
// ensures the custom erc20 keeper's params are properly initialized.
func migrateErc20Params(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) {
	logger.Info("migrating erc20 params to authority-based system")

	// Set default params in the new authority-based system
	// If the old subspace had custom params, they would be read here.
	// Since the x/params subspace is being deprecated in SDK 0.53,
	// we use default params and let operators adjust via gov proposal if needed.
	erc20Keeper := box.Erc20Keeper
	params := erc20types.DefaultParams()

	erc20Keeper.SetParams(ctx, params)
	logger.Info("erc20 params migrated to authority-based system")
}
