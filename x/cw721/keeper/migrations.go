package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates the store from version 1 to 2.
//
// The 1→2 migration is intentionally a no-op: the cw721 module store layout
// did not change between its version-1 and version-2 states — only the module
// ConsensusVersion was bumped (alongside the params handling moving to
// authority-based management, which the v0.4.0 upgrade handler performs). The
// registration exists so a chain whose on-chain module version map still
// records cw721 v1 can advance through RunMigrations without failing. Keep it
// registered; never delete it while any chain can still hold module version 1.
func (m Migrator) Migrate1to2(_ sdk.Context) error {
	return nil
}

// GetParams returns the cw721 module params - used for migration
func (m Migrator) GetParams(ctx sdk.Context) types.Params {
	return m.keeper.GetParams(ctx)
}

// SetParams sets the cw721 module params - used for migration
func (m Migrator) SetParams(ctx sdk.Context, params types.Params) error {
	return m.keeper.SetParams(ctx, params)
}
