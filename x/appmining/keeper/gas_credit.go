package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// ConsumeGasCredit debits gas credit for an authorized payer. Shared by Msg
// server and Cosmos PostHandler auto-consume.
func (k Keeper) ConsumeGasCredit(
	ctx sdk.Context,
	appID []byte,
	gasPayer string,
	amount string,
	sourceTxHash []byte,
) (remaining string, err error) {
	params := k.GetParams(ctx)
	if params.PauseState != nil && params.PauseState.GasCreditPaused {
		return "", types.ErrGasCreditPaused
	}
	app, found := k.GetApplication(ctx, appID)
	if !found {
		return "", types.ErrApplicationNotFound
	}
	if app.Status != types.ApplicationStatusActive {
		return "", types.ErrApplicationNotActive
	}
	authorized := false
	for _, p := range app.GasPayers {
		if p == gasPayer {
			authorized = true
			break
		}
	}
	if !authorized {
		return "", types.ErrGasPayerUnauthorized
	}
	if len(sourceTxHash) != types.SourceTxHashLength {
		return "", types.ErrInvalidHash
	}
	if k.HasGasCreditConsumption(ctx, appID, sourceTxHash) {
		return "", types.ErrSourceTxReplay
	}
	acct := k.GetOrCreateGasCredit(ctx, appID)
	newBal, err := types.CanConsumeCredit(acct.Balance, amount)
	if err != nil {
		return "", err
	}
	newConsumed, err := types.AddAmountStrings(acct.TotalConsumed, amount)
	if err != nil {
		return "", err
	}
	acct.Balance = newBal
	acct.TotalConsumed = newConsumed
	k.SetGasCredit(ctx, acct)
	k.SetGasCreditConsumption(ctx, types.GasCreditConsumption{
		AppId:          appID,
		GasPayer:       gasPayer,
		SourceTxHash:   sourceTxHash,
		Amount:         amount,
		ConsumedHeight: ctx.BlockHeight(),
	})
	ctx.EventManager().EmitEvent(eventWithAppID(
		types.EventTypeGasCreditConsumed, appID,
		sdk.NewAttribute(types.AttributeKeyGasPayer, gasPayer),
		sdk.NewAttribute(types.AttributeKeySourceTxHash, hexBytes(sourceTxHash)),
		sdk.NewAttribute(types.AttributeKeyAmount, amount),
		sdk.NewAttribute(types.AttributeKeyRemaining, newBal),
	))
	return newBal, nil
}

// SetGasPayerIndex binds payer -> app_id. Rejects cross-app rebind.
func (k Keeper) SetGasPayerIndex(ctx sdk.Context, payer string, appID []byte) error {
	if existing, ok := k.GetAppIDByGasPayer(ctx, payer); ok {
		if string(existing) != string(appID) {
			return types.ErrGasPayerAlreadyBound
		}
		return nil
	}
	k.store(ctx).Set(types.GasPayerIndexKey(payer), append([]byte{}, appID...))
	return nil
}

// DeleteGasPayerIndex removes the reverse index entry.
func (k Keeper) DeleteGasPayerIndex(ctx sdk.Context, payer string) {
	k.store(ctx).Delete(types.GasPayerIndexKey(payer))
}

// GetAppIDByGasPayer returns the app bound to this gas payer, if any.
func (k Keeper) GetAppIDByGasPayer(ctx sdk.Context, payer string) ([]byte, bool) {
	bz := k.store(ctx).Get(types.GasPayerIndexKey(payer))
	if len(bz) == 0 {
		return nil, false
	}
	return append([]byte{}, bz...), true
}

// AutoConsumeForTx debits gas credit for an enrolled fee payer after a Cosmos tx.
// Not enrolled → no-op. Insufficient/paused/inactive → skip event, no error
// (ledger path must not roll back an otherwise-valid Cosmos tx until fee
// replacement lands). Unexpected store errors are returned.
func (k Keeper) AutoConsumeForTx(ctx sdk.Context, feePayer sdk.AccAddress, gasUsed uint64, sourceTxHash []byte) error {
	if feePayer == nil || gasUsed == 0 || len(sourceTxHash) != types.SourceTxHashLength {
		return nil
	}
	payer := feePayer.String()
	appID, ok := k.GetAppIDByGasPayer(ctx, payer)
	if !ok {
		return nil
	}
	amount := strconv.FormatUint(gasUsed, 10)
	_, err := k.ConsumeGasCredit(ctx, appID, payer, amount, sourceTxHash)
	if err == nil {
		return nil
	}
	switch {
	case types.ErrCreditInsufficient.Is(err),
		types.ErrGasCreditPaused.Is(err),
		types.ErrApplicationNotActive.Is(err),
		types.ErrApplicationNotFound.Is(err),
		types.ErrGasPayerUnauthorized.Is(err),
		types.ErrSourceTxReplay.Is(err):
		ctx.EventManager().EmitEvent(eventWithAppID(
			types.EventTypeGasCreditAutoSkipped, appID,
			sdk.NewAttribute(types.AttributeKeyGasPayer, payer),
			sdk.NewAttribute(types.AttributeKeySourceTxHash, hexBytes(sourceTxHash)),
			sdk.NewAttribute(types.AttributeKeyAmount, amount),
			sdk.NewAttribute(types.AttributeKeySkipReason, err.Error()),
		))
		return nil
	default:
		return err
	}
}
