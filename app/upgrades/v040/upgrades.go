package v040

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	"github.com/UptickNetwork/uptick/app/upgrades/v040/legacy"
	v2 "github.com/UptickNetwork/uptick/x/collection/migrations/v2"
	collectiontypes "github.com/UptickNetwork/uptick/x/collection/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	// cosmos/evm imports
	upticktypes "github.com/UptickNetwork/uptick/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	evmsecp256k1 "github.com/cosmos/evm/crypto/ethsecp256k1"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
)

const upgradeName = "v0.4.0"

// legacyIBCTransferProvenancePrefix is the KVStore prefix (byte 0x04) the
// deprecated uptick x/erc20 used for IBC transfer provenance records.
// cosmos/evm's x/erc20 reserves 0x04 for KeyPrefixSTRv2Addresses, so the
// leftover records must be removed to avoid a namespace collision.
var legacyIBCTransferProvenancePrefix = []byte{0x04}

// Upgrade implements the v0.4.0 upgrade plan: migration from legacy
// go-ethereum v1.10.x to cosmos/evm v0.6.1 (cosmos/go-ethereum v1.16.2).
//
// Key steps: EVM ChainConfig migrates from Block-based to Time-based fork
// activation (all fork times set to 0); ibc-go v8 → v10 removes the capability
// module entirely; SDK v0.50 → v0.53 deprecates x/params (erc20/erc721/cw721
// params move to module stores); the self-developed x/erc20 is replaced by
// cosmos/evm's x/erc20 (legacy OWNER_MODULE pairs are deleted, OWNER_EXTERNAL
// pairs keep working under the legacy "erc20/0x…" denom).
//
// This upgrade is NOT reversible: once the capability store is deleted and
// ChainConfig is Time-based, the chain cannot roll back. Ensure all validators
// have upgraded before the upgrade height.
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

		// Idempotency guard: re-scheduled plan or crash-restart replay would
		// re-run the one-shot migrations below and hard-stop the chain.
		if box.UpgradeAlreadyApplied(vm) {
			logger.Warn("upgrade plan already applied; skipping one-shot migrations",
				"name", upgradeName,
			)
			return vm, nil
		}

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

		// Step 1: Migrate EVM ChainConfig from Block-based to Time-based.
		// All fork times (Shanghai/Cancun/Prague) set to 0 = activated
		// immediately; enables EIP-7702 SetCodeTx via PragueTime.
		if err := migrateEVMChainConfig(sdkCtx, logger); err != nil {
			return nil, fmt.Errorf("migrate EVM chain config: %w", err)
		}

		// Step 1.5: Repair EVM params (legacy proto has zero-valued
		// AccessControl / ExtendedDenomOptions, which would deny all
		// contract calls) and initialize EvmCoinInfo (required by PreBlock).
		if err := migrateEVMParams(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate EVM params: %w", err)
		}

		// Step 1.6: Migrate legacy Ethermint EthAccount records to standard
		// BaseAccount. This must happen before module migrations so auth can
		// decode every account with the v0.4.0 codec.
		if err := migrateLegacyEVMAccounts(
			sdkCtx,
			box.GetKVStoreKey(authtypes.StoreKey),
			box.AppCodec,
			box.EvmKeeper,
			logger,
		); err != nil {
			return nil, fmt.Errorf("migrate legacy EVM accounts: %w", err)
		}

		// Step 2: Migrate erc20 params from x/params subspace to authority-based.
		if err := migrateErc20Params(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate erc20 params: %w", err)
		}

		// Step 3: Delete legacy OWNER_MODULE token pairs — STRv2 addressing
		// generates different ERC20 addresses, so these pairs are no longer
		// valid for bidirectional conversion. Deleted pairs are logged.
		deleteLegacyOwnerModulePairs(sdkCtx, box, logger)

		// Step 3.2: Delete legacy IBC transfer provenance records — the
		// deprecated MsgTransferERC20 path wrote them under KVStore prefix
		// byte 0x04, which cosmos/evm's x/erc20 reserves for STRv2Addresses.
		deleteLegacyIBCTransferProvenance(sdkCtx, box, logger)

		// Step 3.5: Migrate erc721 params from x/params subspace to the
		// module's own KV store.
		if err := migrateErc721Params(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate erc721 params: %w", err)
		}

		// Step 3.6: Migrate cw721 params from x/params subspace to the
		// module's own KV store.
		if err := migrateCw721Params(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("migrate cw721 params: %w", err)
		}

		// Step 3.7: Align ICS-721 port with ibc-go v10's alphanumeric router
		// key: the bound port must equal ModuleName or the router panics.
		migrateNFTTransferPort(sdkCtx, box, logger)

		// Step 3.8: Remove legacy x/params subspaces for modules that have
		// migrated to self-contained/authority-based params.
		deleteLegacyParamsSubspace(
			sdkCtx,
			box.GetKVStoreKey(paramstypes.StoreKey),
			logger,
			evmtypes.ModuleName,
			erc20types.ModuleName,
			erc721types.ModuleName,
			cw721types.ModuleName,
		)

		// Precheck the legacy collection store read-only before migrating it,
		// so a dirty record produces a complete report before any state change
		// instead of an opaque halt mid-upgrade.
		if problems := v2.PrecheckLegacyStore(sdkCtx, box.GetKVStoreKey(collectiontypes.StoreKey), box.AppCodec); len(problems) > 0 {
			return nil, fmt.Errorf("legacy collection store precheck failed:\n%s", v2.FormatProblems(problems))
		}

		// Step 4: Run module migrations (SDK 0.53, ibc-go v10, cosmos/evm).
		logger.Info("running module migrations")
		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}

// migrateEVMChainConfig migrates the EVM chain configuration from the legacy
// Block-based fork activation to cosmos/evm's Time-based format. All fork
// times (Shanghai/Cancun/Prague) are set to 0, activating them immediately —
// safe because they were already active at block 0 in v032.
func migrateEVMChainConfig(ctx sdk.Context, logger log.Logger) error {
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

		// Shanghai/Cancun/Prague — Time-based activation at 0 ensures all
		// EVM opcodes (incl. EIP-7702 SetCodeTx) remain enabled.
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

// newLegacyAccountCodec builds a codec that can decode every account type a
// live chain may hold (vesting, multisig, legacy EthAccount) while iterating
// the auth store during account migration.
func newLegacyAccountCodec() codec.Codec {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	// Chains may hold vesting accounts (PeriodicVestingAccount,
	// ContinuousVestingAccount, DelayedVestingAccount, ...). The account
	// migration iterates every auth account, so those types must be decodable
	// even though only EthAccount records are rewritten.
	vestingtypes.RegisterInterfaces(interfaceRegistry)
	// Chains may also hold multisig accounts whose pubkey is a legacy amino
	// multisig key (/cosmos.crypto.multisig.LegacyAminoPubKey). Those pubkeys
	// are not rewritten but still have to be decodable while iterating every
	// auth account during the migration.
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	legacy.RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

func getLegacyBoolParam(
	ctx sdk.Context,
	box upgrades.Toolbox,
	logger log.Logger,
	moduleName string,
	key string,
	fallback bool,
) bool {
	subspace, ok := box.ParamsKeeper.GetSubspace(moduleName)
	if ok {
		if raw := subspace.GetRaw(ctx, []byte(key)); len(raw) > 0 {
			return decodeLegacyBoolRaw(ctx, box, logger, moduleName, key, fallback, raw)
		}
	}

	storeKey := box.GetKVStoreKey(paramstypes.StoreKey)
	if storeKey == nil {
		return fallback
	}

	return readLegacyBoolParamRaw(ctx, storeKey, moduleName, key, fallback, logger)
}

func readLegacyBoolParamRaw(
	ctx sdk.Context,
	storeKey *storetypes.KVStoreKey,
	moduleName string,
	key string,
	fallback bool,
	logger log.Logger,
) bool {
	rawKey := append([]byte(moduleName), '/')
	rawKey = append(rawKey, []byte(key)...)
	raw := ctx.KVStore(storeKey).Get(rawKey)
	if len(raw) == 0 {
		return fallback
	}

	return decodeLegacyBoolRaw(ctx, upgrades.Toolbox{}, logger, moduleName, key, fallback, raw)
}

func decodeLegacyBoolRaw(
	_ sdk.Context,
	_ upgrades.Toolbox,
	logger log.Logger,
	moduleName string,
	key string,
	fallback bool,
	raw []byte,
) bool {
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		logger.Error(
			"failed to decode legacy bool param",
			"module", moduleName,
			"key", key,
			"error", err,
		)
		return fallback
	}
	return value
}

// migrateLegacyEVMAccounts rewrites Ethermint v0.3.x EthAccount records into
// standard SDK BaseAccount records (cosmos/evm v0.6.1 no longer uses EthAccount;
// leaving the legacy Any values would make accounts unreadable).
//
// Failure policy: the FULL auth store is scanned, all failures are collected,
// and one aggregated error is returned at the end — a skipped legacy EthAccount
// would be unreadable after the upgrade, so nothing is silently skipped.
func migrateLegacyEVMAccounts(
	ctx sdk.Context,
	storeKey *storetypes.KVStoreKey,
	appCodec codec.Codec,
	evmKeeper *evmkeeper.Keeper,
	logger log.Logger,
) error {
	if storeKey == nil {
		return fmt.Errorf("auth store key not found")
	}

	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(authtypes.AddressStoreKeyPrefix))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	migrationCdc := newLegacyAccountCodec()
	migrated := 0

	// Collected failures: key hex -> reason. We keep scanning after a failure
	// so that every bad account is reported in a single pass.
	type authMigrationFailure struct {
		key    string
		reason string
	}
	var failures []authMigrationFailure
	fail := func(key []byte, format string, args ...interface{}) {
		failures = append(failures, authMigrationFailure{
			key:    fmt.Sprintf("%x", key),
			reason: fmt.Sprintf(format, args...),
		})
	}

	for ; iterator.Valid(); iterator.Next() {
		accountBytes := iterator.Value()
		var accountI sdk.AccountI
		if err := migrationCdc.UnmarshalInterface(accountBytes, &accountI); err != nil {
			fail(iterator.Key(), "decode auth account: %v", err)
			continue
		}

		legacyAccount, ok := accountI.(*legacy.EthAccount)
		if !ok {
			// Retained accounts (vesting, module, ...) keep their type, but a
			// legacy ethermint pubkey Any must be rewritten to the v0.4.0 key
			// type or the account becomes unqueryable after the upgrade.
			if err := migrateRetainedAccountPubKey(appCodec, store, iterator.Key(), accountI, logger); err != nil {
				fail(iterator.Key(), "migrate retained account pubkey: %v", err)
			}
			continue
		}
		if legacyAccount.BaseAccount == nil {
			fail(iterator.Key(), "legacy EthAccount has nil BaseAccount")
			continue
		}

		if legacyAccount.CodeHash != "" && evmKeeper != nil {
			// common.HexToHash silently truncates/zero-pads malformed input,
			// which would persist an unresolvable "ghost" code hash. Validate
			// explicitly: legacy code hashes are keccak256 (32 bytes) hex.
			codeHashBytes := common.FromHex(legacyAccount.CodeHash)
			switch len(codeHashBytes) {
			case 0:
				// Empty / "0x" — nothing to migrate.
			case 32:
				evmKeeper.SetCodeHash(ctx, iterator.Key(), codeHashBytes)
			default:
				fail(iterator.Key(), "legacy EthAccount has malformed code hash %q: expected 32-byte hex, got %d bytes",
					legacyAccount.CodeHash, len(codeHashBytes))
				continue
			}
		}

		// Rewrite the legacy pubkey Any into the v0.4.0 key type —
		// MarshalInterface alone would keep the old type URL inside the Any.
		if legacyAccount.BaseAccount.PubKey != nil {
			var oldPk cryptotypes.PubKey
			if err := migrationCdc.UnpackAny(legacyAccount.BaseAccount.PubKey, &oldPk); err != nil {
				fail(iterator.Key(), "decode legacy pubkey: %v", err)
				continue
			}
			if ethPk, isEth := oldPk.(*legacy.EthSecp256k1PubKey); isEth {
				newPk := &evmsecp256k1.PubKey{Key: ethPk.Key}
				anyPk, err := codectypes.NewAnyWithValue(newPk)
				if err != nil {
					fail(iterator.Key(), "pack migrated pubkey: %v", err)
					continue
				}
				legacyAccount.BaseAccount.PubKey = anyPk
			}
		}

		newBytes, err := appCodec.MarshalInterface(legacyAccount.BaseAccount)
		if err != nil {
			fail(iterator.Key(), "marshal migrated BaseAccount: %v", err)
			continue
		}

		store.Set(iterator.Key(), newBytes)
		migrated++
	}

	if len(failures) > 0 {
		for _, f := range failures {
			logger.Error("legacy auth account failed to migrate",
				"account_key", f.key,
				"reason", f.reason,
			)
		}
		// Abort with an aggregated error: the upgrade tx rolls back atomically,
		// and after patching the accounts the upgrade can be re-run unchanged.
		return fmt.Errorf(
			"%d legacy auth account(s) failed to migrate — the full list was logged above; first failure: account %s: %s",
			len(failures), failures[0].key, failures[0].reason,
		)
	}

	logger.Info("legacy EVM accounts migrated to BaseAccount", "migrated", migrated)
	return nil
}

// migrateRetainedAccountPubKey rewrites the legacy Ethermint pubkey Any inside
// a retained (non-EthAccount) auth account into the v0.4.0 key type, so the
// account stays queryable and signable after the upgrade. Accounts without a
// legacy pubkey are left untouched.
func migrateRetainedAccountPubKey(
	appCodec codec.Codec,
	store storetypes.KVStore,
	key []byte,
	account sdk.AccountI,
	logger log.Logger,
) error {
	pk := account.GetPubKey()
	if pk == nil {
		return nil
	}

	ethPk, isEth := pk.(*legacy.EthSecp256k1PubKey)
	if !isEth {
		return nil
	}

	newPk := &evmsecp256k1.PubKey{Key: ethPk.Key}
	if err := account.SetPubKey(newPk); err != nil {
		return fmt.Errorf("set migrated pubkey on retained account %x: %w", key, err)
	}

	newBytes, err := appCodec.MarshalInterface(account)
	if err != nil {
		return fmt.Errorf("marshal retained account %x: %w", key, err)
	}
	store.Set(key, newBytes)
	logger.Info("migrated legacy pubkey of retained account", "address", account.GetAddress().String())
	return nil
}

// migrateEVMParams repairs the EVM module params (the legacy ethermint proto
// zero-fills AccessControl, which would deny all contract calls) and persists
// EvmCoinInfo, without which the first PreBlock after the upgrade panics.
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

	// Repair malformed denom metadata (Display unit missing from DenomUnits
	// would resolve decimals to 0 and panic) before initializing EvmCoinInfo.
	repairEvmDenomMetadata(ctx, box, evmParams.EvmDenom, logger)

	// Persist EvmCoinInfo so the x/vm PreBlock can register the base denom.
	if err := box.EvmKeeper.InitEvmCoinInfo(ctx); err != nil {
		return fmt.Errorf("init evm coin info: %w", err)
	}

	logger.Info("EVM params and coin info initialized")
	return nil
}

// repairEvmDenomMetadata normalizes the bank denom metadata for the EVM denom
// so InitEvmCoinInfo can derive a supported decimals value: points Display at
// the highest-exponent unit (or appends an 18-decimal display unit) when the
// current Display unit is absent from DenomUnits.
func repairEvmDenomMetadata(ctx sdk.Context, box upgrades.Toolbox, evmDenom string, logger log.Logger) {
	metadata, found := box.BankKeeper.GetDenomMetaData(ctx, evmDenom)
	if !found {
		// Let InitEvmCoinInfo surface the missing-metadata error.
		return
	}

	displayInUnits := false
	maxExponent := uint32(0)
	maxUnitDenom := metadata.Base
	for _, unit := range metadata.DenomUnits {
		if unit.Denom == metadata.Display {
			displayInUnits = true
		}
		if unit.Exponent > maxExponent {
			maxExponent = unit.Exponent
			maxUnitDenom = unit.Denom
		}
	}
	if displayInUnits {
		return
	}

	if maxExponent > 0 {
		metadata.Display = maxUnitDenom
	} else {
		metadata.DenomUnits = append(metadata.DenomUnits, &banktypes.DenomUnit{
			Denom:    metadata.Display,
			Exponent: 18,
			Aliases:  []string{},
		})
	}
	box.BankKeeper.SetDenomMetaData(ctx, metadata)
	logger.Info("repaired evm denom metadata", "display", metadata.Display)
}

// migrateErc20Params migrates erc20 params from x/params subspace to the
// authority-based store of cosmos/evm's x/erc20. The legacy EnableEVMHook
// param is intentionally dropped; cosmos/evm defaults are used instead.
func migrateErc20Params(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) error {
	logger.Info("migrating erc20 params to authority-based system")

	erc20Keeper := box.Erc20Keeper
	params := erc20types.DefaultParams()
	params.PermissionlessRegistration = false
	params.EnableErc20 = getLegacyBoolParam(
		ctx,
		box,
		logger,
		erc20types.ModuleName,
		string(erc20types.ParamStoreKeyEnableErc20),
		params.EnableErc20,
	)

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

// deleteLegacyOwnerModulePairs deletes all OWNER_MODULE token pairs (pair +
// byERC20 + byDenom maps); STRv2 addressing makes these old bindings invalid.
// OWNER_EXTERNAL pairs are preserved, and the Cosmos-native coins themselves
// stay untouched in the bank module. Deleted pairs are logged.
func deleteLegacyOwnerModulePairs(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) {
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

}

// deleteLegacyIBCTransferProvenance removes the provenance records written by
// the deprecated MsgTransferERC20 path. The legacy store used KVStore prefix
// byte 0x04, which cosmos/evm's x/erc20 reserves for STRv2Addresses, so the
// leftover records must be cleared to avoid a namespace collision.
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

// migrateErc721Params migrates erc721 params from the deprecated x/params
// subspace to the module's own KV store; defaults apply if unset.
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
	params.EnableErc721 = getLegacyBoolParam(
		ctx,
		box,
		logger,
		erc721types.ModuleName,
		"EnableErc721",
		params.EnableErc721,
	)
	params.EnableEVMHook = getLegacyBoolParam(
		ctx,
		box,
		logger,
		erc721types.ModuleName,
		"EnableEVMHook",
		params.EnableEVMHook,
	)

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

// migrateCw721Params migrates cw721 params from the deprecated x/params
// subspace to the module's own KV store; defaults apply if unset.
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
	params.EnableCw721 = getLegacyBoolParam(
		ctx,
		box,
		logger,
		cw721types.ModuleName,
		"EnableCw721",
		params.EnableCw721,
	)
	params.EnableEVMHook = getLegacyBoolParam(
		ctx,
		box,
		logger,
		cw721types.ModuleName,
		"EnableEVMHook",
		params.EnableEVMHook,
	)

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

// deleteLegacyParamsSubspace removes raw legacy x/params entries for modules
// that no longer use the x/params module. The values have already been migrated
// into each module's self-contained store, so this only removes dead state.
func deleteLegacyParamsSubspace(
	ctx sdk.Context,
	storeKey *storetypes.KVStoreKey,
	logger log.Logger,
	moduleNames ...string,
) {
	if storeKey == nil {
		logger.Error("params store key is nil, skipping legacy param cleanup")
		return
	}

	store := ctx.KVStore(storeKey)
	for _, moduleName := range moduleNames {
		prefixKey := append([]byte(moduleName), '/')
		iterator := storetypes.KVStorePrefixIterator(store, prefixKey)

		deleted := 0
		for ; iterator.Valid(); iterator.Next() {
			store.Delete(iterator.Key())
			deleted++
		}
		iterator.Close()

		logger.Info(
			"legacy params subspace removed",
			"module", moduleName,
			"deleted", deleted,
		)
	}
}
