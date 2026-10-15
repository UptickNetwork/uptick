package v050

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
)

// ApplyTailMigrations runs the three v0.5.0 migrations that v0.5.1 must also
// carry for testnet (which never executes the v0.5.0 plan):
//
//  1. enablePermissionlessRegistration
//  2. normalizeIBCVoucherERC20Decimals
//  3. backfillIBCVoucherTokenPairs
//
// Each step is idempotent, so a mainnet node that already ran v0.5.0 is
// unaffected by a second application. Failed backfill denoms are recorded the
// same way as in the v0.5.0 handler.
func ApplyTailMigrations(ctx sdk.Context, box upgrades.Toolbox, logger log.Logger) error {
	if err := enablePermissionlessRegistration(ctx, box.Erc20Keeper, logger); err != nil {
		return err
	}
	normalizeIBCVoucherERC20Decimals(ctx, box.BankKeeper, logger)
	failed := backfillIBCVoucherTokenPairs(ctx, box.BankKeeper, box.Erc20Keeper, logger)
	recordBackfillFailures(ctx, box, failed, logger)
	return nil
}
