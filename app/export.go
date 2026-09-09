package app

import (
	"encoding/json"
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

	// withdraw all validator commission
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			ctx.Logger().Error("failed to decode validator operator address", "operator", val.GetOperator(), "err", err)
			panic(err)
		}
		if _, err := app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz); err != nil {
			ctx.Logger().Error("withdraw validator commission failed", "operator", val.GetOperator(), "err", err)
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
			return err
		}

		delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
		if err != nil {
			return err
		}
		if _, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr); err != nil {
			ctx.Logger().Error("withdraw delegation rewards failed", "delegator", delegation.DelegatorAddress, "validator", delegation.ValidatorAddress, "err", err)
		}
	}

	// clear validator slash events
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)

	// clear validator historical rewards
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)

	// set context height to zero
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// reinitialize all validators
	_ = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		// donate any unwithdrawn outstanding reward fraction tokens to the community pool
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			ctx.Logger().Error("failed to decode validator operator address", "operator", val.GetOperator(), "err", err)
			panic(err)
		}
		scraps, err := app.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valBz)
		if err != nil {
			ctx.Logger().Error("get validator outstanding rewards failed", "operator", val.GetOperator(), "err", err)
			panic(err)
		}
		feePool, err := app.DistrKeeper.FeePool.Get(ctx)
		if err != nil {
			ctx.Logger().Error("get fee pool failed", "err", err)
			panic(err)
		}
		feePool.CommunityPool = feePool.CommunityPool.Add(scraps...)
		if err := app.DistrKeeper.FeePool.Set(ctx, feePool); err != nil {
			ctx.Logger().Error("set fee pool failed", "err", err)
			panic(err)
		}
		if err := app.DistrKeeper.Hooks().AfterValidatorCreated(ctx, valBz); err != nil {
			ctx.Logger().Error("AfterValidatorCreated hook failed", "operator", val.GetOperator(), "err", err)
			panic(err)
		}
		return false
	})

	// reinitialize all delegations
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
		}
		if err := app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
			ctx.Logger().Error("AfterDelegationModified hook failed", "delegator", del.DelegatorAddress, "validator", del.ValidatorAddress, "err", err)
		}
	}

	// reset context height
	ctx = ctx.WithBlockHeight(height)

	/* Handle staking state. */

	// iterate through redelegations, reset creation height
	_ = app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		if err := app.StakingKeeper.SetRedelegation(ctx, red); err != nil {
			ctx.Logger().Error("SetRedelegation failed", "delegator", red.DelegatorAddress, "validator", red.ValidatorSrcAddress, "err", err)
		}
		return false
	})

	// iterate through unbonding delegations, reset creation height
	_ = app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for i := range ubd.Entries {
			ubd.Entries[i].CreationHeight = 0
		}
		if err := app.StakingKeeper.SetUnbondingDelegation(ctx, ubd); err != nil {
			ctx.Logger().Error("SetUnbondingDelegation failed", "delegator", ubd.DelegatorAddress, "validator", ubd.ValidatorAddress, "err", err)
		}
		return false
	})

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
	_ = app.SlashingKeeper.IterateValidatorSigningInfos(
		ctx,
		func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			if err := app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info); err != nil {
				ctx.Logger().Error("SetValidatorSigningInfo failed", "consensus_addr", addr.String(), "err", err)
			}
			return false
		},
	)
	return nil
}
