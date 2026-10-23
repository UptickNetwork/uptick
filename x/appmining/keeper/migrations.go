package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// Migrator is a helper for in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(k Keeper) Migrator {
	return Migrator{keeper: k}
}

// Migrate1to2 pauses the legacy attestation / GasCredit / Finalize settlement
// flow for audit-only retention and ensures the V1 entitlement pool module
// account exists. It does NOT convert POC settlements into Entitlements.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	params := m.keeper.GetParams(ctx)
	if params.PauseState == nil {
		params.PauseState = &types.PauseState{}
	}
	types.PauseLegacyAttestationFlow(params.PauseState)
	if err := m.keeper.SetParams(ctx, params); err != nil {
		return err
	}
	// Touch module account so x/auth materializes it from maccPerms if missing.
	_ = m.keeper.ak.GetModuleAccount(ctx, types.EntitlementPoolName)
	ctx.Logger().Info("appmining migrated 1→2: legacy attestation/gas-credit/settlement paused; entitlement pool ready")
	return nil
}
