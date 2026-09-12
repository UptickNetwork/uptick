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

// Upgrade is one registered upgrade: its version name, a constructor for the
// handler, and any store upgrades.
type Upgrade struct {
	// Version name this upgrade is registered under, e.g. `v7`.
	UpgradeName string

	// UpgradeHandlerConstructor builds the upgrade handler.
	UpgradeHandlerConstructor func(*module.Manager, module.Configurator, Toolbox) upgradetypes.UpgradeHandler

	// StoreUpgrades is required for any module added, removed or renamed.
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

type UpgradeRouter struct {
	mu map[string]Upgrade
}

// NewUpgradeRouter returns an empty router.
func NewUpgradeRouter() *UpgradeRouter {
	return &UpgradeRouter{make(map[string]Upgrade)}
}

func (r *UpgradeRouter) Register(u Upgrade) *UpgradeRouter {
	if _, has := r.mu[u.UpgradeName]; has {
		panic(u.UpgradeName + " already registered")
	}
	r.mu[u.UpgradeName] = u
	return r
}

func (r *UpgradeRouter) Routers() map[string]Upgrade {
	return r.mu
}

func (r *UpgradeRouter) UpgradeInfo(planName string) Upgrade {
	return r.mu[planName]
}

// UpgradeAlreadyApplied reports whether every module managed by the module
// manager is already stored at its current consensus version -- i.e. this
// upgrade handler has run before.
//
// It is the idempotency guard for one-shot migrations: if a plan is re-scheduled
// by mistake (same name re-registered at a new height, crash-restart replay,
// operator error), re-running non-idempotent migrations can corrupt state or
// hard-stop the chain. Handlers must check this before their custom steps and
// return early when it is true.
//
// Only modules present in the manager are compared: one removed by this upgrade
// (e.g. capability in v0.4.0) is irrelevant, and one missing from vm was added
// by this upgrade, so the first run has not completed and the result is false.
//
// Modules with no ConsensusVersion (IBC light clients 06-solomachine and
// 07-tendermint, plus genesis-only and legacy modules) carry no migratable
// consensus state and must be SKIPPED -- returning false for them makes the
// guard unreachable on any manager containing one, silently disabling
// idempotency protection for every handler that relies on it.
//
// PRECONDITION: only sound for an upgrade that bumps at least one module's
// ConsensusVersion or adds/removes modules. One that changes no consensus version
// leaves a chain from the previous release already satisfying "all stored versions
// == current", so the guard would wrongly report "applied" on the first legitimate
// run (this bit v0.4.1 -- rely on the migrations being idempotent instead).
func (b Toolbox) UpgradeAlreadyApplied(vm module.VersionMap) bool {
	if len(vm) == 0 {
		return false
	}
	for name, mod := range b.ModuleManager.Modules {
		cv, ok := mod.(interface{ ConsensusVersion() uint64 })
		if !ok {
			// No consensus version to compare -- must not block the guard.
			continue
		}
		from, ok := vm[name]
		if !ok || from != cv.ConsensusVersion() {
			return false
		}
	}
	return true
}
