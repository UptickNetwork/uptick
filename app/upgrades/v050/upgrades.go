// Package v050 implements the v0.5.0 upgrade plan: the single governance plan a
// v0.3.3 chain follows to reach the v0.4.x state.
//
// # Why this package exists
//
// v0.4.0 and v0.4.1 were shipped to testnet as two plans executed in order
// (first v0.4.0, then v0.4.1). Mainnet never ran either of them and is not going
// to: it moves from v0.3.3 to v0.5.0 in ONE proposal. That one plan therefore
// has to replay the whole v0.4.0 change set and then the v0.4.1 repairs, which
// is what this handler does -- it composes the two existing handlers rather than
// restating their migrations, so the v0.4.x code that already passed review is
// what runs on mainnet.
//
// # One handler, two starting states
//
// The probe exists because the v0.4.0 change set must not run twice. A second
// run is not merely wasteful: deleteLegacyOwnerModulePairs selects token pairs
// by ContractOwner == OWNER_MODULE alone, and cosmos/evm's IBC auto-registration
// creates pairs with that same owner, so a replay deletes the STRv2 pairs real
// users hold (the defect pinned by
// app/upgrades/v040/erc20_legacy_replay_test.go). A chain already on v0.4.x has
// to reach the repairs without passing through the v0.4.0 set again.
//
// This is a property of the HANDLER, not a statement about what the v0.5.0
// BINARY serves. The v0.4.0 and v0.4.1 plan names are deliberately not
// registered in this release (see the note on `router` in app/upgrade.go), so
// this binary cannot run on a chain stopped at v0.4.1 -- the testnet -- at all.
// Keeping the handler correct from both starting states is what lets a later
// release register v0.4.1 back and reuse this code unchanged.
//
// # What v0.5.1 reuses from here
//
// v0.5.1 has to carry three migrations forward rather than call this handler,
// because on the testnet this plan never runs:
//
//   - enablePermissionlessRegistration,
//   - normalizeIBCVoucherERC20Decimals,
//   - backfillIBCVoucherTokenPairs.
//
// Testnet stops on v0.4.1 and upgrades straight to v0.5.1, so for it these three
// are the only route to a switch that is on, metadata whose decimals() reads 18
// instead of 0, and token pairs for the vouchers it already holds. Mainnet, past
// v0.5.0, runs the same three a second time and has to be unaffected by it.
//
// Two consequences, both load-bearing for that release:
//
//   - The three are idempotent on their own, which is why they can be re-applied
//     without this plan's replay marker. The marker is keyed to
//     migrationsAppliedKey below, which is v0.5.0's name, so it is not something
//     v0.5.1 inherits. The property is pinned by
//     TestV050TailMigrationsAreSafeToReapplyWithoutTheMarker in
//     app/upgrade_v050_handler_test.go, which deletes the marker and applies the
//     handler twice.
//   - The v0.4.0 stage must NOT be carried forward with them. Both chains have
//     the EvmCoinInfo record by then, so the probe above would never select it
//     anyway; a v0.5.1 handler should refuse a legacy starting state loudly
//     instead of running the repairs and the tail against it.
//
// The starting state is therefore probed from state, never configured: a
// validator running the wrong branch would have to be wrong about its own store
// contents, not about an operator flag.
//
// # The probe: the cosmos/evm EvmCoinInfo record (EVM store prefix 0x05)
//
// The probe is the EvmCoinInfo record in the EVM store. It is the right signal
// for three reasons:
//
//  1. The v0.4.0 handler writes it. migrateEVMParams ends in
//     EvmKeeper.InitEvmCoinInfo, and cosmos/evm's x/vm InitGenesis writes it too,
//     so any chain on the v0.4.x code has it.
//  2. The legacy layout cannot write it. ethermint v0.24.1-uptick's persistent
//     EVM prefixes stop at prefixParams (0x03); cosmos/evm adds prefixCodeHash
//     (0x04) and prefixEvmCoinInfo (0x05). A v0.3.3 store therefore has no key
//     at 0x05 -- this is not "unlikely", it is unreachable.
//  3. Nothing in between rewrites or clears it. x/vm only ever sets the record
//     (SetEvmCoinInfo, from InitGenesis and the v0.4.0 handler); there is no
//     delete path.
//
// So "the key exists" means the v0.4.0 set ran, and "it is absent" means this is
// a v0.3.3 state.
//
// The v0.4.1 repairs need no probe: each is idempotent on its own (see their doc
// comments), so they run on both paths and are harmless when repeated.
//
// # The registration switch
//
// The plan also flips x/erc20's PermissionlessRegistration to true. That is the
// one v0.5.0 step that overrides a v0.4.0 decision instead of replaying it:
// migrateErc20Params forces the parameter off, and a chain already on v0.4.x
// never re-runs that migration, so only a step here reaches both starting
// states. See enablePermissionlessRegistration for what the flag changes and
// why the backfill is still required.
//
// # Replay guard
//
// The v0.4.0 stage is not replay-safe, and x/upgrade does not prevent a replay:
// PreBlocker applies whatever plan is pending with a registered handler name
// (cosmossdk.io/x/upgrade@v0.2.0/abci.go), so re-proposing "v0.5.0" at a new
// height -- or an operator restarting with a stale upgrade-info.json -- reaches
// this handler again. A marker is therefore written to the x/upgrade store once
// every stage has succeeded, and a run that finds it returns early.
//
// The marker lives in the x/upgrade store rather than a module store because it
// is upgrade bookkeeping, not module state, and it starts with 'v' (0x76) so it
// cannot collide with x/upgrade's own keys: those are single low bytes (PlanByte
// 0x00, DoneByte 0x01, VersionMapByte 0x02, ProtocolVersionByte 0x03) plus the
// "upgradedIBCState" string prefix, which starts with 'u'.
//
// # Rollback
//
// Not reversible, for the reasons in app/upgrades/v040: the capability store is
// deleted and EVM ChainConfig moves to the time-based layout.
package v050

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
	v040 "github.com/UptickNetwork/uptick/app/upgrades/v040"
	v041 "github.com/UptickNetwork/uptick/app/upgrades/v041"
)

const upgradeName = "v0.5.0"

// migrationsAppliedKey records, in the x/upgrade store, that this handler has
// completed its migration sequence.
//
// It exists to make the plan safe to replay, which the v0.4.0 stage is not:
// re-running deleteLegacyOwnerModulePairs deletes every STRv2 pair registered
// since the upgrade, i.e. the assets users are actively bridging. The marker is
// written only after every stage succeeds, so a run that crashes or returns an
// error leaves no marker and re-executes the whole sequence -- which is safe,
// because a failed handler aborts the block and its writes are never committed.
var migrationsAppliedKey = []byte("v0.5.0/migrations-applied")

// Upgrade implements the v0.5.0 upgrade plan.
//
// StoreUpgrades carries the v0.4.0 tree change: the capability module store is
// gone under ibc-go v10. It is repeated here rather than inherited because the
// store loader is keyed on the *plan name* -- a mainnet node following "v0.5.0"
// only ever consults this entry -- so dropping it would silently boot the node
// into the v0.4.x layout with the legacy store still mounted. On a chain that
// already ran v0.4.0 it is inert: the loader iterates the stores this binary
// registers, and capability is not one of them, so the deletion matches nothing.
var Upgrade = upgrades.Upgrade{
	UpgradeName:               upgradeName,
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades: &storetypes.StoreUpgrades{
		Deleted: []string{"capability"},
	},
}

func upgradeHandlerConstructor(
	mm *module.Manager,
	c module.Configurator,
	box upgrades.Toolbox,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		logger := sdkCtx.Logger()

		marker, err := replayMarker(sdkCtx, box)
		if err != nil {
			return nil, err
		}
		if marker.Has(migrationsAppliedKey) {
			logger.Info(
				"v0.5.0 migration set already applied; skipping the replay-unsafe stages",
				"upgrade", upgradeName,
			)
			// Still hand the version map to the module manager: on a replay it is
			// already current, so this is a no-op, but it keeps the handler's
			// return value identical to the first run's.
			return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
		}

		fromLegacyState := !evmCoinInfoInitialized(sdkCtx, box.GetKVStoreKey(evmtypes.StoreKey))
		logger.Info(
			"executing upgrade plan",
			"name", upgradeName,
			"starting_state", startingState(fromLegacyState),
		)

		vm, err = runMigrationSet(ctx, plan, vm, fromLegacyState, stages{
			legacy:  v040.Upgrade.UpgradeHandlerConstructor(mm, c, box),
			repairs: v041.Upgrade.UpgradeHandlerConstructor(mm, c, box),
		})
		if err != nil {
			return nil, err
		}

		// v0.5.0's own migrations, in the order they have to run: the
		// registration switch first, then the metadata shape, so a holder
		// querying decimals() in the same block already reads the repaired
		// exponent, then the token pairs that expose the voucher to the EVM at
		// all.
		//
		// All three run on BOTH starting states, unlike the v0.4.0 change set
		// above. The repairs are not repair-the-migration work: a testnet
		// already on v0.4.x has exactly the same IBC vouchers with decimals()
		// == 0 and no token pair, because ibc-go writes that metadata shape
		// whoever receives the packet and the inbound auto-registration is
		// gated off -- which is also why the switch has to be set here rather
		// than in v0.4.0. See migrate.go.
		if err := enablePermissionlessRegistration(sdkCtx, box.Erc20Keeper, logger); err != nil {
			return nil, fmt.Errorf("enable permissionless erc20 registration: %w", err)
		}
		normalizeIBCVoucherERC20Decimals(sdkCtx, box.BankKeeper, logger)
		backfillIBCVoucherTokenPairs(sdkCtx, box.BankKeeper, box.Erc20Keeper, logger)

		marker.Set(migrationsAppliedKey, []byte{0x01})
		logger.Info("v0.5.0 migration set complete", "upgrade", upgradeName)
		return vm, nil
	}
}

// stages are the two change sets a v0.5.0 node may have to run, in order.
type stages struct {
	// legacy is the v0.4.0 change set: module migrations (SDK v0.50 -> v0.53,
	// ibc-go v8 -> v10, cosmos/evm v0.6), the EVM ChainConfig move to the
	// time-based layout and the legacy state clean-ups. It is replayed only from
	// a v0.3.3 state, because its legacy-pair deletion is not replay-safe.
	legacy upgradetypes.UpgradeHandler

	// repairs is the v0.4.1 change set: static precompiles, the ICA controller
	// flag, the feemarket base fee and the ERC721 conversion index. Every repair
	// is idempotent, so it runs on both starting states.
	repairs upgradetypes.UpgradeHandler
}

// runMigrationSet runs the change sets a v0.5.0 node needs for its probed
// starting state, and names the stage that failed so a halt points at one of
// them.
//
// The v0.4.1 handler ends in RunMigrations as well. On the mainnet path that
// second call is a no-op, because the v0.4.0 stage already returned the migrated
// version map and the module manager only moves each module to its current
// consensus version; on the testnet path it is the only one, exactly as it was
// when v0.4.1 was the plan.
//
// It is a separate function so the sequencing is testable without an app: a
// Toolbox embeds the concrete keepers, so the handler itself can only be
// exercised on a real application (see app/upgrade_v050_handler_test.go).
func runMigrationSet(
	ctx context.Context,
	plan upgradetypes.Plan,
	vm module.VersionMap,
	fromLegacyState bool,
	s stages,
) (module.VersionMap, error) {
	if fromLegacyState {
		// Mainnet path: v0.3.3 -> v0.5.0 has to replay the whole v0.4.0 change
		// set, because that state transition never ran there.
		var err error
		vm, err = s.legacy(ctx, plan, vm)
		if err != nil {
			return nil, fmt.Errorf("v0.4.0 migration set: %w", err)
		}
	}

	vm, err := s.repairs(ctx, plan, vm)
	if err != nil {
		return nil, fmt.Errorf("v0.4.1 repairs: %w", err)
	}
	return vm, nil
}

// replayMarker returns the x/upgrade store, where the replay marker lives.
//
// A missing store key is unreachable while x/upgrade is wired into the app, but
// it must fail the upgrade rather than silently skip the guard: without the
// marker the plan would run its non-idempotent stage again on a replay.
func replayMarker(ctx sdk.Context, box upgrades.Toolbox) (storetypes.KVStore, error) {
	storeKey := box.GetKVStoreKey(upgradetypes.StoreKey)
	if storeKey == nil {
		return nil, fmt.Errorf("x/upgrade store key is not registered; cannot guard the v0.4.0 stage against a replay")
	}
	return ctx.KVStore(storeKey), nil
}

// startingState names the probed starting state for the operator log, so an
// upgrade log says which branch a node took.
func startingState(fromLegacy bool) string {
	if fromLegacy {
		return "v0.3.3 (legacy ethermint state) -- replaying the v0.4.0 change set"
	}
	return "v0.4.x (cosmos/evm state) -- v0.4.0 change set already applied"
}

// evmCoinInfoInitialized reports whether the cosmos/evm EvmCoinInfo record is
// present in the EVM store, i.e. whether the v0.4.0 migration set has run.
//
// The byte-level argument for the probe is in the package doc; the guard below
// covers the case where the EVM store is not registered, which would make a
// store read panic. An unresolvable store is treated as "not initialized" so the
// node takes the full-migration path: for a v0.3.3 chain that is the correct
// branch, and for any other chain the app would be misconfigured well beyond the
// reach of this handler.
//
// It takes the store key rather than the Toolbox so the probe stays unit
// testable: a Toolbox carries the whole AppKeepers set and cannot be built from
// outside the keepers package, while a store key can be paired with an in-memory
// store.
func evmCoinInfoInitialized(ctx sdk.Context, evmStoreKey *storetypes.KVStoreKey) bool {
	if evmStoreKey == nil {
		return false
	}
	return ctx.KVStore(evmStoreKey).Has(evmtypes.KeyPrefixEvmCoinInfo)
}
