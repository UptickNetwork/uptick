package app

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingsim "github.com/cosmos/cosmos-sdk/x/staking/simulation"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Uptick enforces a chain-wide minimum validator commission of 5% in the ante
// handler (app/ante/comission.go). The stock x/staking simulator does not know
// about that policy: SimulateMsgCreateValidator draws the commission rate
// uniformly from [0, maxRate] and SimulateMsgEditValidator draws it from
// [0, MaxRate], so roughly one in five generated messages is rejected by
// ValidatorCommissionDecorator. A rejected message is returned as a delivery
// error, which makes SimulateFromSeed call tb.Fatalf and aborts the whole run —
// the simulation could therefore never complete a single non-empty block.
//
// The fix belongs in the harness, not in the ante handler: the policy is a real
// consensus rule and must not be relaxed for tests. Instead the two offending
// operations are re-implemented here with the commission clamped to the same
// 5% floor a well-behaved client would use. Every other staking operation is
// taken verbatim from the SDK.
const (
	// simMinCommissionRate mirrors minCommission() in app/ante/comission.go.
	simMinCommissionRate = 5
	simCommissionPrec    = 2
)

// simStateFor builds the module.SimulationState the SDK's
// simtestutil.BuildSimulationOperations would build, so the staking operations
// can be assembled separately from the rest of the weighted operation list.
func simStateFor(app *Uptick, config simtypes.Config) module.SimulationState {
	simState := module.SimulationState{
		AppParams: make(simtypes.AppParams),
		Cdc:       app.AppCodec(),
		TxConfig:  moduletestutil.MakeTestTxConfig(),
		BondDenom: sdk.DefaultBondDenom,
	}

	if config.ParamsFile != "" {
		bz, err := os.ReadFile(config.ParamsFile)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(bz, &simState.AppParams); err != nil {
			panic(err)
		}
	}

	simState.LegacyProposalContents = app.SimulationManager().GetProposalContents(simState) //nolint:staticcheck // legacy proposal contents
	simState.ProposalMsgs = app.SimulationManager().GetProposalMsgs(simState)

	return simState
}

// stakingSimOps returns x/staking's weighted operations with the two
// commission-bearing operations replaced by clamped versions.
func stakingSimOps(app *Uptick, simState module.SimulationState) []simtypes.WeightedOperation {
	var (
		weightCreateValidator int
		weightEditValidator   int
		weightDelegate        int
		weightUndelegate      int
		weightRedelegate      int
		weightCancelUnbond    int
	)

	params := simState.AppParams
	params.GetOrGenerate(stakingsim.OpWeightMsgCreateValidator, &weightCreateValidator, nil, func(_ *rand.Rand) {
		weightCreateValidator = stakingsim.DefaultWeightMsgCreateValidator
	})
	params.GetOrGenerate(stakingsim.OpWeightMsgEditValidator, &weightEditValidator, nil, func(_ *rand.Rand) {
		weightEditValidator = stakingsim.DefaultWeightMsgEditValidator
	})
	params.GetOrGenerate(stakingsim.OpWeightMsgDelegate, &weightDelegate, nil, func(_ *rand.Rand) {
		weightDelegate = stakingsim.DefaultWeightMsgDelegate
	})
	params.GetOrGenerate(stakingsim.OpWeightMsgUndelegate, &weightUndelegate, nil, func(_ *rand.Rand) {
		weightUndelegate = stakingsim.DefaultWeightMsgUndelegate
	})
	params.GetOrGenerate(stakingsim.OpWeightMsgBeginRedelegate, &weightRedelegate, nil, func(_ *rand.Rand) {
		weightRedelegate = stakingsim.DefaultWeightMsgBeginRedelegate
	})
	params.GetOrGenerate(stakingsim.OpWeightMsgCancelUnbondingDelegation, &weightCancelUnbond, nil, func(_ *rand.Rand) {
		weightCancelUnbond = stakingsim.DefaultWeightMsgCancelUnbondingDelegation
	})

	ak, bk, k, txGen := app.AccountKeeper, app.BankKeeper, app.StakingKeeper, simState.TxConfig

	return []simtypes.WeightedOperation{
		simulation.NewWeightedOperation(weightCreateValidator, simMsgCreateValidator(txGen, ak, bk, k)),
		simulation.NewWeightedOperation(weightEditValidator, simMsgEditValidator(txGen, ak, bk, k)),
		simulation.NewWeightedOperation(weightDelegate, stakingsim.SimulateMsgDelegate(txGen, ak, bk, k)),
		simulation.NewWeightedOperation(weightUndelegate, stakingsim.SimulateMsgUndelegate(txGen, ak, bk, k)),
		simulation.NewWeightedOperation(weightRedelegate, stakingsim.SimulateMsgBeginRedelegate(txGen, ak, bk, k)),
		simulation.NewWeightedOperation(weightCancelUnbond, stakingsim.SimulateMsgCancelUnbondingDelegate(txGen, ak, bk, k)),
	}
}

// simMinCommissionDec returns the 5% floor as a LegacyDec.
func simMinCommissionDec() math.LegacyDec {
	return math.LegacyNewDecWithPrec(simMinCommissionRate, simCommissionPrec)
}

// simMsgCreateValidator is a copy of x/staking/simulation.SimulateMsgCreateValidator
// (SDK v0.53.6) with the commission rate floored at the chain minimum. Two
// changes are made: maxRate is drawn from [0.10, 1.00) instead of [0, 1.00) so
// it can never sit below the floor, and rate is raised to the floor when the
// random draw falls under it.
func simMsgCreateValidator(
	txGen client.TxConfig,
	ak stakingtypes.AccountKeeper,
	bk stakingtypes.BankKeeper,
	k *stakingkeeper.Keeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		msgType := sdk.MsgTypeURL(&stakingtypes.MsgCreateValidator{})

		simAccount, _ := simtypes.RandomAcc(r, accs)
		address := sdk.ValAddress(simAccount.Address)

		// ensure the validator doesn't exist already
		if _, err := k.GetValidator(ctx, address); err == nil {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "validator already exists"), nil, nil
		}

		denom, err := k.BondDenom(ctx)
		if err != nil {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "bond denom not found"), nil, err
		}

		balance := bk.GetBalance(ctx, simAccount.Address, denom).Amount
		if !balance.IsPositive() {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "balance is negative"), nil, nil
		}

		amount, err := simtypes.RandPositiveInt(r, balance)
		if err != nil {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "unable to generate positive amount"), nil, err
		}
		selfDelegation := sdk.NewCoin(denom, amount)

		account := ak.GetAccount(ctx, simAccount.Address)
		spendable := bk.SpendableCoins(ctx, account.GetAddress())

		var fees sdk.Coins
		coins, hasNeg := spendable.SafeSub(selfDelegation)
		if !hasNeg {
			fees, err = simtypes.RandomFees(r, ctx, coins)
			if err != nil {
				return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "unable to generate fees"), nil, err
			}
		}

		description := stakingtypes.NewDescription(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
		)

		minRate := simMinCommissionDec()
		// Draw maxRate from [0.10, 1.00) so it is always at or above the floor.
		maxCommission := math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 10, 100)), simCommissionPrec)
		if maxCommission.LT(minRate) {
			maxCommission = minRate
		}

		rate := simtypes.RandomDecAmount(r, maxCommission)
		if rate.LT(minRate) {
			rate = minRate
		}

		commission := stakingtypes.NewCommissionRates(rate, maxCommission, minRate)

		msg, err := stakingtypes.NewMsgCreateValidator(
			address.String(), simAccount.ConsKey.PubKey(), selfDelegation, description, commission, math.OneInt(),
		)
		if err != nil {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "unable to create CreateValidator message"), nil, err
		}

		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         txGen,
			Cdc:           nil,
			Msg:           msg,
			Context:       ctx,
			SimAccount:    simAccount,
			AccountKeeper: ak,
			ModuleName:    stakingtypes.ModuleName,
		}

		return simulation.GenAndDeliverTx(txCtx, fees)
	}
}

// simMsgEditValidator is a copy of x/staking/simulation.SimulateMsgEditValidator
// (SDK v0.53.6) with the new commission rate floored at the chain minimum. The
// SDK's own ValidateNewRate guard still runs afterwards, so a value the chain
// would reject on other grounds degrades to a no-op instead of an error.
func simMsgEditValidator(
	txGen client.TxConfig,
	ak stakingtypes.AccountKeeper,
	bk stakingtypes.BankKeeper,
	k *stakingkeeper.Keeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		msgType := sdk.MsgTypeURL(&stakingtypes.MsgEditValidator{})

		vals, err := k.GetAllValidators(ctx)
		if err != nil {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "unable to get validators"), nil, err
		}
		if len(vals) == 0 {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "number of validators equal zero"), nil, nil
		}

		val, ok := testutil.RandSliceElem(r, vals)
		if !ok {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "unable to pick a validator"), nil, nil
		}

		address := val.GetOperator()

		newCommissionRate := simtypes.RandomDecAmount(r, val.Commission.MaxRate)
		if minRate := simMinCommissionDec(); newCommissionRate.LT(minRate) {
			newCommissionRate = minRate
		}

		if err := val.Commission.ValidateNewRate(newCommissionRate, ctx.BlockHeader().Time); err != nil {
			// skip as the commission is invalid
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "invalid commission rate"), nil, nil
		}

		bz, err := k.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "error getting validator address bytes"), nil, err
		}

		simAccount, found := simtypes.FindAccount(accs, sdk.AccAddress(bz))
		if !found {
			return simtypes.NoOpMsg(stakingtypes.ModuleName, msgType, "unable to find account"), nil,
				fmt.Errorf("validator %s not found", val.GetOperator())
		}

		account := ak.GetAccount(ctx, simAccount.Address)
		spendable := bk.SpendableCoins(ctx, account.GetAddress())

		description := stakingtypes.NewDescription(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
		)

		msg := stakingtypes.NewMsgEditValidator(address, description, &newCommissionRate, nil)

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           txGen,
			Cdc:             nil,
			Msg:             msg,
			Context:         ctx,
			SimAccount:      simAccount,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      stakingtypes.ModuleName,
			CoinsSpentInMsg: spendable,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}
