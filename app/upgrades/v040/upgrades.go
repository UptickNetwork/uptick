package v040

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

	// cosmos/evm imports
	upticktypes "github.com/UptickNetwork/uptick/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
)

const upgradeName = "v0.4.0"

// legacyIBCTransferProvenancePrefix is the KVStore prefix (byte 0x04) used by
// the deprecated uptick x/erc20 module to store IBC transfer provenance records
// (SetIBCTransferProvenance). In cosmos/evm's x/erc20 the same byte 0x04 is
// reserved for KeyPrefixSTRv2Addresses, so the leftover records must be removed
// to avoid a namespace collision and to reclaim dead state.
var legacyIBCTransferProvenancePrefix = []byte{0x04}

// Upgrade implements the v0.4.0 upgrade plan.
//
// This upgrade migrates Uptick from legacy go-ethereum v1.10.x
// to cosmos/evm v0.6.1 (based on cosmos/go-ethereum v1.16.2).
//
// =====================================================================
// MAJOR CHANGES IN THIS UPGRADE
// =====================================================================
//
// 1. EVM Module: x/evm → cosmos/evm x/vm
//   - go-ethereum upgraded from v1.10.17 → v1.16.2
//   - ChainConfig fields changed from Block-based to Time-based
//     (ShanghaiBlock → ShanghaiTime, CancunBlock → CancunTime,
//     PragueBlock → PragueTime)
//   - EIP-7702 SetCodeTx is now natively supported by cosmos/go-ethereum
//     (no custom implementation needed)
//   - ChainConfig is now a global variable (set during NewKeeper),
//     not stored in Params. The upgrade handler re-initializes it.
//
// 2. IBC Module: ibc-go v8 → v10
//   - capability module removed entirely
//   - ScopedKeeper references removed from all keepers
//   - IBCModule interface signatures changed (added channelVersion param)
//   - SendPacket/WriteAcknowledgement no longer require chanCap
//
// 3. SDK: v0.50 → v0.53
//   - x/params module deprecated (params now authority-based)
//   - gov v1beta1 proposals → gov v1 (for erc20 module)
//   - runtime.KVStoreService replaces direct StoreKey in keeper constructors
//
// 4. Wasm: wasmd v0.53 → v0.61
//   - WasmConfig → NodeConfig
//   - wasmvm v2 → v3
//
// 5. EVM Hardfork Activation
//   - In v032, Shanghai/Cancun/Prague were activated via Block height = 0
//   - In cosmos/evm v0.6.1, these are activated via Time (timestamp)
//   - The upgrade sets all fork times to 0 (activated immediately at upgrade)
//   - This ensures all EVM opcodes (PUSH0, BLOBHASH, etc.) are enabled
//   - EIP-7702 SetCodeTx (type 0x04) is enabled via PragueTime
//
// 6. Capability Store Cleanup
//   - The 'capability' module store is deleted (ibc-go v10 doesn't use it)
//
// 7. erc20 Module
//   - Uptick replaces its self-developed x/erc20 with cosmos/evm's x/erc20
//     (v0.6.1). The old MsgTransferERC20 / IBC provenance refund path is no
//     longer wired; IBC coin->ERC20 conversion now goes through cosmos/evm's
//     ERC20 IBC middleware + ibc_callbacks.go.
//   - Params migrated from x/params subspace to authority-based.
//     EnableEVMHook is dropped (cosmos/evm uses PermissionlessRegistration).
//   - Existing OWNER_MODULE pairs are deleted (STRv2 addressing differs).
//   - Existing OWNER_EXTERNAL pairs keep their legacy "erc20/0x…" denom and
//     remain functional; only NEW registrations use the "erc20:0x…" scheme.
//
// =====================================================================
// ROLLBACK NOTE
// =====================================================================
// This upgrade is NOT reversible. Once the capability store is deleted
// and ChainConfig is migrated to Time-based, the chain cannot roll back
// to the legacy implementation. Ensure all validators have upgraded before the upgrade height.
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
		// (both use "evm" store key).
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
				"legacy x/evm → cosmos/evm v0.6.1 (go-ethereum v1.10→v1.16)",
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
		// In the legacy upgrade (v032), Shanghai/Cancun/Prague were activated via:
		//   ChainConfig.ShanghaiBlock = 0
		//   ChainConfig.CancunBlock = 0
		//   ChainConfig.PragueBlock = 0
		//
		// In cosmos/evm v0.6.1, these are now Time-based (timestamp):
		//   ChainConfig.ShanghaiTime = 0  (activated immediately)
		//   ChainConfig.CancunTime = 0
		//   ChainConfig.PragueTime = 0
		//
		// cosmos/evm also adds new fields not in the legacy implementation:
		//   - OsakaTime (not activated, nil)
		//   - VerkleTime (not activated, nil)
		//   - BlobScheduleConfig (Cancun/Prague/Osaka blob configs)
		//
		// EIP-7702 SetCodeTx (type 0x04) is enabled when PragueTime is set.
		// This allows EOA accounts to delegate to smart contract code,
		// enabling account abstraction without protocol-level changes.
		if err := migrateEVMChainConfig(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate EVM chain config: %w", err)
		}

		// Step 1.5: Migrate EVM params + initialize coin info.
		//
		// The legacy ethermint params proto (fields 1-6) has no
		// access_control/history_serve_window/extended_denom_options, so after
		// unmarshaling into cosmos/evm v0.6.1's Params those fields are zero:
		//   - AccessControl = AccessTypeUnspecified -> every create/call is denied
		//   - ExtendedDenomOptions = nil -> LoadEvmCoinInfo fails for non-18-dec
		// Additionally, cosmos/evm v0.6.1 requires EvmCoinInfo to be persisted in
		// the module store; without it the first PreBlock panics on RegisterDenom.
		if err := migrateEVMParams(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate EVM params: %w", err)
		}

		// Step 2: Migrate erc20 params from x/params subspace to authority-based
		//
		// In SDK 0.50 with legacy x/evm, erc20 params were stored in x/params subspace.
		// In SDK 0.53 + cosmos/evm, params are stored directly in the module store
		// and managed via gov v1 authority.
		if err := migrateErc20Params(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate erc20 params: %w", err)
		}

		// Step 3: Delete legacy OWNER_MODULE token pairs
		//
		// The old uptick x/erc20 module stored OWNER_MODULE pairs (module-deployed
		// ERC20 contracts). With cosmos/evm v0.6.1, the STRv2 addressing scheme
		// generates different ERC20 addresses, so old OWNER_MODULE pairs are no
		// longer valid for bidirectional conversion.
		//
		// These legacy pairs are deleted (token pair + byERC20 + byDenom maps)
		// rather than merely disabled: the old ERC20 is being deprecated and real
		// data volume is low, so no backwards compatibility is required. Deleted
		// pairs are logged for auditability.
		if err := deleteLegacyOwnerModulePairs(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("delete legacy OWNER_MODULE pairs: %w", err)
		}

		// Step 3.2: Delete legacy IBC transfer provenance records
		//
		// The deprecated MsgTransferERC20 path wrote provenance records under
		// KVStore prefix byte 0x04. cosmos/evm's x/erc20 reserves that byte for
		// STRv2Addresses, so the leftover records are removed to prevent a
		// namespace collision and reclaim dead state.
		deleteLegacyIBCTransferProvenance(sdkCtx, box, logger)

		// Step 3.5: Migrate erc721 params from x/params subspace to self-contained KV store
		//
		// The old evm-nft-convert module stored params in x/params subspace.
		// The new x/erc721 module stores params directly in its own KV store,
		// following the same pattern as cosmos/evm ERC20.
		if err := migrateErc721Params(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate erc721 params: %w", err)
		}

		// Step 3.6: Migrate cw721 params from x/params subspace to self-contained KV store
		//
		// The old wasm-nft-convert module stored params in x/params subspace.
		// The new x/cw721 module stores params directly in its own KV store,
		// following the same pattern as cosmos/evm ERC20.
		if err := migrateCw721Params(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate cw721 params: %w", err)
		}

		// Step 3.7: Align ICS-721 port with ibc-go v10's alphanumeric router key.
		//
		// Legacy genesis stored PortId "nft-transfer". ibc-go v10 looks up
		// IBCModule callbacks by port ID and panics on non-alphanumeric route
		// keys, so the bound port must equal ModuleName.
		migrateNFTTransferPort(sdkCtx, box, logger)

		// Step 4: Run module migrations
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
// legacy format (Block-based) to the new cosmos/evm format (Time-based).
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

	evmChainID, err := upticktypes.ParseEIP155ChainID(ctx.ChainID())
	if err != nil {
		return fmt.Errorf("parse eip-155 chain id from %q: %w", ctx.ChainID(), err)
	}

	existing := evmtypes.GetChainConfig()
	if existing != nil && existing.ChainId == evmChainID {
		logger.Info(
			"EVM ChainConfig already set for this chain id, skipping SetChainConfig",
			"chain_id", evmChainID,
		)
		return nil
	}

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

	// NOTE: EVM params migration (including EIP-3855 preservation) and
	// EvmCoinInfo initialization are handled separately by migrateEVMParams.

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

// migrateEVMParams repairs the EVM module params and initializes EvmCoinInfo.
//
// The legacy ethermint params proto predates cosmos/evm v0.6.1's new fields, so
// after unmarshaling the carried-over params have:
//   - AccessControl = AccessTypeUnspecified -> NewRestrictedPermissionPolicy
//     denies all contract creation and calls (including precompiles);
//   - ExtendedDenomOptions = nil -> LoadEvmCoinInfo errors for non-18-decimal
//     denominations;
//   - HistoryServeWindow = 0.
//
// It also persists EvmCoinInfo (base-denom metadata) into the module store;
// without it the x/vm PreBlock calls sdk.RegisterDenom("") and panics on the
// first block after the upgrade.
func migrateEVMParams(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) error {
	logger.Info("migrating EVM params and initializing coin info")

	evmParams := box.EvmKeeper.GetParams(ctx)

	// Restore the fields that are absent from the legacy proto and therefore
	// zero/uninitialized after unmarshaling. EvmDenom ("auptick") carries over
	// correctly because it is still field 1 in both protos.
	evmParams.AccessControl = evmtypes.DefaultAccessControl
	evmParams.HistoryServeWindow = evmtypes.DefaultHistoryServeWindow
	if evmParams.ExtendedDenomOptions == nil {
		evmParams.ExtendedDenomOptions = &evmtypes.ExtendedDenomOptions{
			ExtendedDenom: evmParams.EvmDenom,
		}
	}

	// Preserve EIP-3855 (PUSH0), appended by the legacy v032 upgrade.
	eip3855 := int64(3855)
	hasEip3855 := false
	for _, eip := range evmParams.ExtraEIPs {
		if eip == eip3855 {
			hasEip3855 = true
			break
		}
	}
	if !hasEip3855 {
		evmParams.ExtraEIPs = append(evmParams.ExtraEIPs, eip3855)
	}

	if err := box.EvmKeeper.SetParams(ctx, evmParams); err != nil {
		return fmt.Errorf("set evm params: %w", err)
	}

	// Persist EvmCoinInfo so the x/vm PreBlock can register the base denom.
	if err := box.EvmKeeper.InitEvmCoinInfo(ctx); err != nil {
		return fmt.Errorf("init evm coin info: %w", err)
	}

	logger.Info("EVM params and coin info initialized")
	return nil
}

// migrateErc20Params migrates erc20 module parameters from x/params subspace
// to the new authority-based params system.
//
// In SDK 0.50 with legacy x/evm, erc20 params were stored in x/params subspace.
// In SDK 0.53 + cosmos/evm, params are stored directly in the erc20 module store
// and managed via gov v1 authority (authtypes.NewModuleAddress(govtypes.ModuleName)).
//
// Uptick now uses cosmos/evm's x/erc20 (not the self-developed module), so this
// migration initializes the cosmos/evm erc20 keeper's params in the new
// authority-based store. The legacy EnableEVMHook param is intentionally
// dropped; cosmos/evm's default params (EnableErc20 + PermissionlessRegistration)
// are used instead.
func migrateErc20Params(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) error {
	logger.Info("migrating erc20 params to authority-based system")

	erc20Keeper := box.Erc20Keeper
	params := erc20types.DefaultParams()
	params.PermissionlessRegistration = false

	if err := erc20Keeper.SetParams(ctx, params); err != nil {
		return fmt.Errorf("set erc20 params: %w", err)
	}
	logger.Info(
		"erc20 params migrated to authority-based system",
		"EnableErc20", params.EnableErc20,
		"PermissionlessRegistration", params.PermissionlessRegistration,
	)
	return nil
}

// deleteLegacyOwnerModulePairs deletes all existing OWNER_MODULE token pairs.
//
// In the legacy uptick x/erc20 module, OWNER_MODULE pairs were created for
// Cosmos-native coins that had module-deployed ERC20 contracts. With the
// migration to cosmos/evm v0.6.1, the STRv2 addressing scheme generates
// different ERC20 contract addresses, making these old pairs invalid for
// bidirectional conversion.
//
// What this migration does:
//   - Iterates all existing token pairs in the erc20 store
//   - For pairs with ContractOwner == OWNER_MODULE: deletes the pair and its
//     byERC20 + byDenom maps (DeleteTokenPair)
//   - Preserves OWNER_EXTERNAL pairs (external ERC20 → Cosmos coin mappings)
//
// Deleted pairs are logged (denom + erc20 address) for auditability. The
// Cosmos-native coin itself is NOT removed from the bank module — only the
// ERC20↔coin mapping is dropped.
func deleteLegacyOwnerModulePairs(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) error {
	logger.Info("deleting legacy OWNER_MODULE token pairs")

	erc20Keeper := box.Erc20Keeper
	allPairs := erc20Keeper.GetTokenPairs(ctx)

	var deletedCount int
	var preservedCount int
	var details []string

	for _, pair := range allPairs {
		if pair.ContractOwner != erc20types.OWNER_MODULE {
			// External ERC20 pairs are still valid under STRv2
			preservedCount++
			continue
		}

		// Delete the pair and its byERC20 + byDenom maps.
		erc20Keeper.DeleteTokenPair(ctx, pair)
		deletedCount++

		details = append(details,
			fmt.Sprintf("  - denom=%s erc20=%s",
				pair.Denom, pair.Erc20Address),
		)
	}

	logger.Info(
		"legacy OWNER_MODULE pair deletion complete",
		"total_pairs", len(allPairs),
		"deleted_owner_module", deletedCount,
		"preserved_external", preservedCount,
	)

	for _, d := range details {
		logger.Debug(d)
	}

	return nil
}

// deleteLegacyIBCTransferProvenance removes the IBC transfer provenance records
// written by the deprecated MsgTransferERC20 path (uptick x/erc20).
//
// In the legacy uptick x/erc20 store, KVStore prefix byte 0x04 held
// IBCTransferProvenance records keyed by
// "port/channel/sequence/sender/denom/amount". cosmos/evm's x/erc20 reserves
// the same byte 0x04 for KeyPrefixSTRv2Addresses, so the leftover records are
// dead state that must be cleared to avoid a namespace collision.
func deleteLegacyIBCTransferProvenance(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) {
	storeKey := box.GetKVStoreKey(erc20types.StoreKey)
	if storeKey == nil {
		logger.Error("erc20 store key is nil, skipping provenance cleanup")
		return
	}
	store := ctx.KVStore(storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, legacyIBCTransferProvenancePrefix)
	defer iterator.Close()

	var deleted int
	var skipped int
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		// 0x04 + 20-byte address is the cosmos/evm STRv2 layout; leave it alone.
		if len(key) == 1+common.AddressLength {
			skipped++
			continue
		}
		store.Delete(key)
		deleted++
	}

	logger.Info(
		"deleted legacy IBC transfer provenance records",
		"deleted", deleted,
		"skipped_strv2", skipped,
	)
}

// migrateErc721Params migrates erc721 module parameters from x/params subspace
// to the new self-contained KV store (aligned with cosmos/evm ERC20 pattern).
//
// In the old evm-nft-convert module, params were stored in x/params subspace
// via paramtypes.ParamSet interface. The new x/erc721 module stores params
// directly in its own KV store (using the same store key prefix), eliminating
// the dependency on x/params.
//
// Since the old params subspace is being deprecated in SDK 0.53, we use
// default params and let operators adjust via gov proposal if needed.
func migrateErc721Params(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) error {
	logger.Info("migrating erc721 params to self-contained KV store")

	storeKey := box.GetKVStoreKey(erc721types.StoreKey)
	if storeKey != nil {
		store := ctx.KVStore(storeKey)
		if store.Has(erc721types.KeyPrefixParams) {
			logger.Info("erc721 params already present, skipping overwrite")
			return nil
		}
	}

	erc721Keeper := box.Erc721Keeper
	params := erc721types.DefaultParams()

	if err := erc721Keeper.SetParams(ctx, params); err != nil {
		return err
	}
	logger.Info(
		"erc721 params migrated to self-contained KV store",
		"EnableErc721", params.EnableErc721,
		"EnableEVMHook", params.EnableEVMHook,
	)
	return nil
}

// migrateCw721Params migrates cw721 module parameters from x/params subspace
// to the new self-contained KV store (aligned with cosmos/evm ERC20 pattern).
//
// In the old wasm-nft-convert module, params were stored in x/params subspace
// via paramtypes.ParamSet interface. The new x/cw721 module stores params
// directly in its own KV store (using the same store key prefix), eliminating
// the dependency on x/params.
//
// Since the old params subspace is being deprecated in SDK 0.53, we use
// default params and let operators adjust via gov proposal if needed.
func migrateCw721Params(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) error {
	logger.Info("migrating cw721 params to self-contained KV store")

	storeKey := box.GetKVStoreKey(cw721types.StoreKey)
	if storeKey != nil {
		store := ctx.KVStore(storeKey)
		if store.Has(cw721types.KeyPrefixParams) {
			logger.Info("cw721 params already present, skipping overwrite")
			return nil
		}
	}

	cw721Keeper := box.Cw721Keeper
	params := cw721types.DefaultParams()

	if err := cw721Keeper.SetParams(ctx, params); err != nil {
		return err
	}
	logger.Info(
		"cw721 params migrated to self-contained KV store",
		"EnableCw721", params.EnableCw721,
		"EnableEVMHook", params.EnableEVMHook,
	)
	return nil
}

// migrateNFTTransferPort rewrites the ICS-721 bound port from the legacy
// hyphenated identifier to ModuleName, which ibc-go v10 can register.
func migrateNFTTransferPort(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) {
	current := box.IBCNFTTransferKeeper.GetPort(ctx)
	want := ibcnfttransfertypes.PortID
	if current == want {
		logger.Info("ICS-721 port already alphanumeric, skipping", "port", current)
		return
	}
	box.IBCNFTTransferKeeper.SetPort(ctx, want)
	logger.Info("ICS-721 port migrated for ibc-go v10 router", "from", current, "to", want)
}
