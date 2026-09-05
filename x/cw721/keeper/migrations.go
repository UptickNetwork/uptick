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

// Migrate1to2 migrates the store from version 1 to 2
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
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
