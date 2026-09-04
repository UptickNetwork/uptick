package upgrades

import (
	"github.com/UptickNetwork/uptick/app/keepers"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	store "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// Upgrade defines a struct containing necessary fields that a SoftwareUpgradeProposal
// must have written, in order for the state migration to go smoothly.
// An upgrade must implement this struct, and then set it in the app.go.
// The app.go will then define the handler.
type Upgrade struct {
	// Upgrade version name, for the upgrade handler, e.g. `v7`
	UpgradeName string

	// UpgradeHandlerConstructor defines the function that creates an upgrade handler
	UpgradeHandlerConstructor func(*module.Manager, module.Configurator, Toolbox) upgradetypes.UpgradeHandler

	// Store upgrades, should be used for any new modules introduced, new modules deleted, or store names renamed.
	StoreUpgrades *store.StoreUpgrades
}

// ConsensusParamsReaderWriter defines the interface for reading and writing consensus params
type ConsensusParamsReaderWriter interface {
	StoreConsensusParams(ctx sdk.Context, cp tmproto.ConsensusParams) error
	GetConsensusParams(ctx sdk.Context) tmproto.ConsensusParams
}

// Toolbox contains all the modules necessary for an upgrade
type Toolbox struct {
	AppCodec      codec.Codec
	ModuleManager *module.Manager
	ReaderWriter  ConsensusParamsReaderWriter
	keepers.AppKeepers
}

type upgradeRouter struct {
	mu map[string]Upgrade
}

// NewUpgradeRouter creates a new upgrade router.
//
// No parameters.
// Returns a pointer to upgradeRouter.
func NewUpgradeRouter() *upgradeRouter {
	return &upgradeRouter{make(map[string]Upgrade)}
}

func (r *upgradeRouter) Register(u Upgrade) *upgradeRouter {
	if _, has := r.mu[u.UpgradeName]; has {
		panic(u.UpgradeName + " already registered")
	}
	r.mu[u.UpgradeName] = u
	return r
}

func (r *upgradeRouter) Routers() map[string]Upgrade {
	return r.mu
}

func (r *upgradeRouter) UpgradeInfo(planName string) Upgrade {
	return r.mu[planName]
}

// UpgradeAlreadyApplied reports whether every module managed by the module
// manager is already stored at its current consensus version — i.e. this
// upgrade handler has run before.
//
// It is the idempotency guard for one-shot upgrade migrations: if an upgrade
// plan is re-scheduled by mistake (same name re-registered at a new height,
// crash-restart replay, operator error), re-running non-idempotent migrations
// can corrupt state or hard-stop the whole chain. Handlers must check this
// before executing their custom migration steps and return early when true.
//
// The comparison only covers modules present in the manager; versions of
// modules removed by the upgrade (e.g. capability in v0.4.0) are irrelevant.
// A module missing from vm (newly added by this upgrade) means the first run
// has not completed, so the result is false.
func (b Toolbox) UpgradeAlreadyApplied(vm module.VersionMap) bool {
	if len(vm) == 0 {
		return false
	}
	for name, mod := range b.ModuleManager.Modules {
		cv, ok := mod.(interface{ ConsensusVersion() uint64 })
		if !ok {
			// Module does not expose a consensus version (legacy or
			// genesis-only module) — treat as not-yet-migrated so the
			// handler stays conservative and re-runs.
			return false
		}
		from, ok := vm[name]
		if !ok || from != cv.ConsensusVersion() {
			return false
		}
	}
	return true
}
