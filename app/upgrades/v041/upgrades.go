package v041

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/UptickNetwork/uptick/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
)

const upgradeName = "v0.4.1"

// Upgrade is the v0.4.1 upgrade. It repairs state left behind by earlier
// versions: static precompiles, the ICA controller flag, the feemarket base fee
// and the ERC721 conversion index. The Keplr compatibility fix (legacy ethermint
// pubkey, EIP-712 extension option decoding) lives in the binary runtime
// (encoding config + ante handler), so it needs no migration.
var Upgrade = upgrades.Upgrade{
	UpgradeName:               upgradeName,
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades:             &storetypes.StoreUpgrades{},
}

func upgradeHandlerConstructor(
	_ *module.Manager,
	c module.Configurator,
	box upgrades.Toolbox,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// Deliberately no UpgradeAlreadyApplied guard: v0.4.1 bumps no module
		// ConsensusVersion, so a chain from v0.4.0 hands this handler a version
		// map equal to the current one, and the guard would report "applied" on
		// the first (and only legitimate) run -- skipping every repair below.
		// It is unusable here, not omitted by oversight: see the PRECONDITION
		// note on Toolbox.UpgradeAlreadyApplied (app/upgrades/types.go).
		//
		// Do not read the absence as precedent. v0.4.0's handler has no guard
		// either, but that one is an unfixed defect rather than a constraint:
		// its legacy-pair deletion is not idempotent, so a replayed plan
		// deletes pairs registered after the upgrade. Reproducer:
		// app/upgrades/v040/erc20_legacy_replay_test.go.
		//
		// Each repair below is idempotent on its own -- see the per-repair
		// comments -- so a replayed plan is harmless without the guard.

		sdkCtx.Logger().Info(
			"executing upgrade plan",
			"name", upgradeName,
			"change", "activate static precompiles; enable ICA controller; Keplr legacy ethermint pubkey / EIP-712 tx compatibility",
		)

		// Repair the EVM params: v0.4.0 introduced ActiveStaticPrecompiles but
		// left it empty, so every custom static precompile was inactive.
		if err := migrateActiveStaticPrecompiles(sdkCtx, box.EvmKeeper); err != nil {
			return nil, fmt.Errorf("migrate active static precompiles: %w", err)
		}

		// Enable the ICA controller submodule: genesis templates derived from
		// the legacy x/params defaults carry controller_enabled=false, and the
		// ibc-go v10 param migration keeps the stored value as-is, so every
		// ICA register fails with "controller submodule is disabled".
		migrateICAControllerParams(sdkCtx, box.ICAControllerKeeper)

		// Re-scale the feemarket base fee: ethermint stored it as math.Int and
		// cosmos/evm reads it as math.LegacyDec, so the v0.4.0 upgrade left
		// 1 gwei reading back as 10^-9 and BeginBlock made it permanent.
		if err := migrateFeeMarketBaseFee(sdkCtx, box.FeeMarketKeeper); err != nil {
			return nil, fmt.Errorf("migrate feemarket params: %w", err)
		}

		// Collapse the duplicate forward keys the pre-v0.4.1 write path left in
		// the ERC721 conversion index. Its position relative to RunMigrations is
		// not load-bearing (x/erc721 has no pending module migration, so nothing
		// else can reach this state), but it is a repair, so it runs with the
		// repairs and before the module manager gets a say in the same store.
		pruneErc721UIDIndex(sdkCtx, box.Erc721Keeper)

		return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
	}
}

// icaControllerParamsStore is the slice of the ICA controller keeper this
// migration needs. It is an interface for the reason evmParamsStore is, plus one
// of its own: the write is guarded by a read of the value it writes, and against
// the real keeper that guard is invisible -- ibc-go's SetParams returns nothing
// and writing true over an already-true param leaves byte-identical state.
// Recording the calls is what makes dropping the guard fail a test.
type icaControllerParamsStore interface {
	GetParams(ctx sdk.Context) icacontrollertypes.Params
	SetParams(ctx sdk.Context, params icacontrollertypes.Params)
}

// migrateICAControllerParams flips the ICA controller submodule on if it is
// disabled. Chains from legacy genesis templates (e.g. the origin testnet) store
// controller_enabled=false; fresh chains already default to true.
//
// Nothing guards the host submodule's params, because they are unreachable at
// the type level: icaControllerParamsStore is bound to
// icacontrollertypes.Params and nothing in scope exposes icahosttypes.Params, so
// no edit inside this signature can read or write them. The compiler is the
// guard.
//
// The write is a read-modify-write on purpose: it flips one field on the value
// it just read and stores that same value, so a field ibc-go adds to Params
// later is carried through instead of being zeroed. Rebuilding the struct
// (icacontrollertypes.NewParams(true)) compiles and passes every test in this
// repo today, because the one field that exists is the one being set; it only
// diverges once a second field exists.
func migrateICAControllerParams(ctx sdk.Context, store icaControllerParamsStore) {
	params := store.GetParams(ctx)
	if params.ControllerEnabled {
		return
	}
	params.ControllerEnabled = true
	store.SetParams(ctx, params)
	ctx.Logger().Info("ica controller submodule enabled", "upgrade", upgradeName)
}
