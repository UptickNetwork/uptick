package app

import (
	"encoding/json"
	"errors"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *Uptick) ExportAppStateAndValidators(
	forZeroHeight bool, jailAllowedAddrs []string, modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// Creates context with current height and checks txs for ctx to be usable by start of next block
	ctx := app.NewContextLegacy(true, tmproto.Header{Height: app.LastBlockHeight()})
	// We export at last height + 1, because that's the height at which
	// Tendermint will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0

		if err := app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	genState, err := app.mm.ExportGenesisForModules(ctx, app.AppCodec(), modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	// A module export never fails on damaged state: it degrades, logs, and
	// still returns its genesis (decision 第 22 轮: one export policy for the
	// whole repository). Persist that degradation list next to the node home so
	// it outlives the process -- otherwise the only trace would be a log line,
	// and an operator restoring from a partially degraded backup would have no
	// way to know what was missing.
	if diags := app.collectExportDiagnostics(ctx); len(diags) > 0 {
		if path, writeErr := app.writeExportDiagnosticsReport(height, diags); writeErr != nil {
			// Never fail the export over the sidecar: the degradations are
			// already logged by each module, and the genesis itself is valid.
			ctx.Logger().Error("failed to write the export diagnostics report", "err", writeErr)
		} else {
			ctx.Logger().Error(
				"genesis export was degraded; the full list of affected records was written to the diagnostics report",
				"path", path,
				"affected_records", len(diags),
			)
		}
	} else {
		app.removeStaleExportDiagnosticsReport()
	}

	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          height,
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, nil
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//
//	in favor of export at a block height
func (app *Uptick) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) error {
	applyAllowedAddrs := len(jailAllowedAddrs) > 0

	allowedAddrsMap := make(map[string]bool)

	for _, addr := range jailAllowedAddrs {
		_, err := sdk.ValAddressFromBech32(addr)
		if err != nil {
			return err
		}
		allowedAddrsMap[addr] = true
	}

	/* Just to be safe, assert the invariants on current state. */
	app.CrisisKeeper.AssertInvariants(ctx)

	/* Handle fee distribution state. */

	// Every withdrawal below must succeed before any irreversible cleanup
	// runs: the slash-event and historical-reward purges further down cannot be
	// undone by re-running the export, so a failure that used to be
	// log-and-continue could ship a genesis whose distribution accounting is
	// silently incomplete. Errors are collected (not just the first one) so a
	// single run reports everything that needs attention, and returned before
	// the destructive steps.
	var distributionErrs []error

	// withdraw all validator commission
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			ctx.Logger().Error("failed to decode validator operator address", "operator", val.GetOperator(), "err", err)
			distributionErrs = append(distributionErrs,
				fmt.Errorf("decode validator operator %s: %w", val.GetOperator(), err))
			return false
		}
		if _, err := app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz); err != nil {
			ctx.Logger().Error("withdraw validator commission failed", "operator", val.GetOperator(), "err", err)
			distributionErrs = append(distributionErrs,
				fmt.Errorf("withdraw validator commission for %s: %w", val.GetOperator(), err))
		}
		return false
	})
	if err != nil {
		return err
	}

	// withdraw all delegator rewards
	dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	if err != nil {
		return err
	}
	for _, delegation := range dels {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			distributionErrs = append(distributionErrs,
				fmt.Errorf("decode validator address %s: %w", delegation.ValidatorAddress, err))
			continue
		}

		delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
		if err != nil {
			distributionErrs = append(distributionErrs,
				fmt.Errorf("decode delegator address %s: %w", delegation.DelegatorAddress, err))
			continue
		}
		if _, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr); err != nil {
			ctx.Logger().Error("withdraw delegation rewards failed", "delegator", delegation.DelegatorAddress, "validator", delegation.ValidatorAddress, "err", err)
			distributionErrs = append(distributionErrs,
				fmt.Errorf("withdraw delegation rewards for %s/%s: %w",
					delegation.DelegatorAddress, delegation.ValidatorAddress, err))
		}
	}

	if len(distributionErrs) > 0 {
		return fmt.Errorf(
			"zero-height export aborted before the irreversible distribution cleanup: %w",
			errors.Join(distributionErrs...))
	}

	// clear validator slash events
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)

	// clear validator historical rewards
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)

	// set context height to zero
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// reinitialize all validators
	//
	// IterateValidators takes a `func(...) bool`, so the closure has no error
	// channel. The previous version panicked at each of the five failure
	// points, which crashes the node process and diverges from the fail-closed
	// error style this same function uses everywhere else (errors.Join for the
	// commission/reward passes above, fmt.Errorf further down). Capture the
	// first failure, stop the iteration and return it instead.
	var reinitErr error
	if err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		// donate any unwithdrawn outstanding reward fraction tokens to the community pool
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			ctx.Logger().Error("failed to decode validator operator address", "operator", val.GetOperator(), "err", err)
			reinitErr = fmt.Errorf("decode validator operator %s: %w", val.GetOperator(), err)
			return true
		}
		scraps, err := app.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valBz)
		if err != nil {
			ctx.Logger().Error("get validator outstanding rewards failed", "operator", val.GetOperator(), "err", err)
			reinitErr = fmt.Errorf("get outstanding rewards for %s: %w", val.GetOperator(), err)
			return true
		}
		feePool, err := app.DistrKeeper.FeePool.Get(ctx)
		if err != nil {
			ctx.Logger().Error("get fee pool failed", "err", err)
			reinitErr = fmt.Errorf("get fee pool: %w", err)
			return true
		}
		feePool.CommunityPool = feePool.CommunityPool.Add(scraps...)
		if err := app.DistrKeeper.FeePool.Set(ctx, feePool); err != nil {
			ctx.Logger().Error("set fee pool failed", "err", err)
			reinitErr = fmt.Errorf("set fee pool: %w", err)
			return true
		}
		if err := app.DistrKeeper.Hooks().AfterValidatorCreated(ctx, valBz); err != nil {
			ctx.Logger().Error("AfterValidatorCreated hook failed", "operator", val.GetOperator(), "err", err)
			reinitErr = fmt.Errorf("AfterValidatorCreated hook for %s: %w", val.GetOperator(), err)
			return true
		}
		return false
	}); err != nil {
		return fmt.Errorf("failed to iterate validators: %w", err)
	}
	if reinitErr != nil {
		return fmt.Errorf("zero-height export failed to reinitialise validators: %w", reinitErr)
	}

	// reinitialize all delegations
	//
	// By the time this runs the slash events and historical rewards have
	// already been purged, so a hook failure can no longer be downgraded to a
	// log line: the export must fail rather than emit a genesis whose
	// delegation indexes were never rebuilt.
	for _, del := range dels {
		valAddr, err := sdk.ValAddressFromBech32(del.ValidatorAddress)
		if err != nil {
			return err
		}
		delAddr, err := sdk.AccAddressFromBech32(del.DelegatorAddress)
		if err != nil {
			return err
		}
		if err := app.DistrKeeper.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
			ctx.Logger().Error("BeforeDelegationCreated hook failed", "delegator", del.DelegatorAddress, "validator", del.ValidatorAddress, "err", err)
			return fmt.Errorf("rebuild delegation index (BeforeDelegationCreated) for %s/%s: %w",
				del.DelegatorAddress, del.ValidatorAddress, err)
		}
		if err := app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
			ctx.Logger().Error("AfterDelegationModified hook failed", "delegator", del.DelegatorAddress, "validator", del.ValidatorAddress, "err", err)
			return fmt.Errorf("rebuild delegation index (AfterDelegationModified) for %s/%s: %w",
				del.DelegatorAddress, del.ValidatorAddress, err)
		}
	}

	// reset context height
	ctx = ctx.WithBlockHeight(height)

	/* Handle staking state. */

	// iterate through redelegations, reset creation height
	var resetErrs []error
	if err := app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		if err := app.StakingKeeper.SetRedelegation(ctx, red); err != nil {
			ctx.Logger().Error("SetRedelegation failed", "delegator", red.DelegatorAddress, "validator", red.ValidatorSrcAddress, "err", err)
			resetErrs = append(resetErrs,
				fmt.Errorf("reset redelegation %s/%s: %w", red.DelegatorAddress, red.ValidatorSrcAddress, err))
		}
		return false
	}); err != nil {
		return fmt.Errorf("failed to iterate redelegations: %w", err)
	}
	if len(resetErrs) > 0 {
		return fmt.Errorf("failed to reset redelegation creation heights: %w", errors.Join(resetErrs...))
	}

	// iterate through unbonding delegations, reset creation height
	if err := app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for i := range ubd.Entries {
			ubd.Entries[i].CreationHeight = 0
		}
		if err := app.StakingKeeper.SetUnbondingDelegation(ctx, ubd); err != nil {
			ctx.Logger().Error("SetUnbondingDelegation failed", "delegator", ubd.DelegatorAddress, "validator", ubd.ValidatorAddress, "err", err)
			resetErrs = append(resetErrs,
				fmt.Errorf("reset unbonding delegation %s/%s: %w", ubd.DelegatorAddress, ubd.ValidatorAddress, err))
		}
		return false
	}); err != nil {
		return fmt.Errorf("failed to iterate unbonding delegations: %w", err)
	}
	if len(resetErrs) > 0 {
		return fmt.Errorf("failed to reset unbonding delegation creation heights: %w", errors.Join(resetErrs...))
	}

	// Iterate through validators by power descending, reset bond heights, and
	// update bond intra-tx counters.
	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	iter := storetypes.KVStoreReversePrefixIterator(store, stakingtypes.ValidatorsKey)
	counter := int16(0)

	for ; iter.Valid(); iter.Next() {
		addr := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iter.Key()))
		validator, err := app.StakingKeeper.GetValidator(ctx, addr)
		if err != nil {
			return fmt.Errorf("expected validator %s, not found: %w", addr, err)
		}
		validator.UnbondingHeight = 0
		if applyAllowedAddrs && !allowedAddrsMap[addr.String()] {
			validator.Jailed = true
		}
		if err := app.StakingKeeper.SetValidator(ctx, validator); err != nil {
			return fmt.Errorf("SetValidator failed for %s: %w", addr, err)
		}
		counter++
	}

	if err := iter.Close(); err != nil {
		return err
	}

	if _, err := app.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx); err != nil {
		return err
	}

	/* Handle slashing state. */

	// reset start height on signing infos
	var signingInfoErrs []error
	if err := app.SlashingKeeper.IterateValidatorSigningInfos(
		ctx,
		func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			if err := app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info); err != nil {
				ctx.Logger().Error("SetValidatorSigningInfo failed", "consensus_addr", addr.String(), "err", err)
				signingInfoErrs = append(signingInfoErrs,
					fmt.Errorf("reset signing info for %s: %w", addr.String(), err))
			}
			return false
		},
	); err != nil {
		return fmt.Errorf("failed to iterate validator signing infos: %w", err)
	}
	if len(signingInfoErrs) > 0 {
		return fmt.Errorf("failed to reset validator signing infos: %w", errors.Join(signingInfoErrs...))
	}
	return nil
}
