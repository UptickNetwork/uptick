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
// flow for audit-only retention, ensures the V1 entitlement pool module account
// exists, and rebuilds the V1 challenge derived indexes. It does NOT convert POC
// settlements into Entitlements.
//
// The index backfill is why this migration must run before any per-candidate
// read is trusted: Read paths (approve's OPEN/UPHELD gate, the settled-digest
// check, the timeout sweeper) only look at the candidate index and the deadline
// buckets. A challenge written by a v1 binary has neither, so skipping the
// backfill would make an existing OPEN challenge invisible to approve.
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

	// BackfillChallengeIndexes walks the whole table and marks the indexes ready,
	// so the EndBlocker rebuild becomes a no-op on chains that upgrade through a
	// plan instead of a bare binary swap.
	indexed, err := m.keeper.BackfillChallengeIndexes(ctx)
	if err != nil {
		return err
	}
	ctx.Logger().Info("appmining migrated 1→2: legacy attestation/gas-credit/settlement paused; entitlement pool ready; challenge indexes rebuilt",
		"challenges_indexed", indexed)
	return nil
}
