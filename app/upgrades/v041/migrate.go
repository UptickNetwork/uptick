package v041

import (
	"fmt"

	"cosmossdk.io/math"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
)

// defaultActiveStaticPrecompiles is cosmos/evm's DefaultStaticPrecompiles list as
// Uptick wires it. It excludes the vesting precompile (0x803), which this chain
// does not register, to avoid the "precompiled contract not stored in memory"
// panic when the EVM resolves it.
var defaultActiveStaticPrecompiles = []string{
	evmtypes.P256PrecompileAddress,
	evmtypes.Bech32PrecompileAddress,
	evmtypes.StakingPrecompileAddress,
	evmtypes.DistributionPrecompileAddress,
	evmtypes.ICS20PrecompileAddress,
	evmtypes.BankPrecompileAddress,
	evmtypes.GovPrecompileAddress,
	evmtypes.SlashingPrecompileAddress,
}

// ConfigureDefaultStaticPrecompiles overrides the EVM module's default params so
// fresh chains also activate the static precompiles (the cosmos/evm default is
// empty). It must run before anything calls evmtypes.DefaultParams() -- i.e.
// before app.go builds the module manager -- and is called explicitly rather than
// from a package init(), so the wiring is visible at the call site.
func ConfigureDefaultStaticPrecompiles() {
	evmtypes.DefaultStaticPrecompiles = defaultActiveStaticPrecompiles
}

// evmParamsStore is the slice of the EVM keeper this migration needs. It is an
// interface so the SetParams failure path is injectable: the real keeper only
// fails on invalid params and this migration always writes a known-good list, so
// with the concrete type that branch would be unreachable from any test.
type evmParamsStore interface {
	GetParams(ctx sdk.Context) evmtypes.Params
	SetParams(ctx sdk.Context, params evmtypes.Params) error
}

// migrateActiveStaticPrecompiles repairs the EVM params so the static precompiles
// are actually active. v0.4.0 introduced the ActiveStaticPrecompiles field (the
// legacy ethermint params proto had none) but never populated it, so
// IsAvailableStaticPrecompile returned false and every precompile call failed.
func migrateActiveStaticPrecompiles(ctx sdk.Context, store evmParamsStore) error {
	params := store.GetParams(ctx)
	updated, changed := withDefaultActiveStaticPrecompiles(params)
	if !changed {
		return nil
	}
	if err := store.SetParams(ctx, updated); err != nil {
		return fmt.Errorf("set evm params: %w", err)
	}
	return nil
}

// withDefaultActiveStaticPrecompiles fills in the default precompile list when the
// stored one is empty; a populated list is preserved so governance removals are
// not silently reverted. "Empty" is indistinguishable from "never configured" (a
// proto round-trip collapses an explicitly-empty list to nil), so a governance
// decision to disable every precompile also resets here.
func withDefaultActiveStaticPrecompiles(params evmtypes.Params) (evmtypes.Params, bool) {
	if len(params.ActiveStaticPrecompiles) != 0 {
		return params, false
	}
	params.ActiveStaticPrecompiles = append([]string(nil), defaultActiveStaticPrecompiles...)
	return params, true
}

// legacyDecScale is 10^18, the fixed-point scale of math.LegacyDec: its numeric
// value is the raw big.Int divided by this. LegacyNewDec takes an int64 and 10^18
// fits, so it is written as a literal.
var legacyDecScale = math.LegacyNewDec(1_000_000_000_000_000_000)

// feeMarketParamsStore is the slice of the feemarket keeper this migration needs.
// It is an interface for the injectable-failure reason evmParamsStore is.
type feeMarketParamsStore interface {
	GetParams(ctx sdk.Context) feemarkettypes.Params
	SetParams(ctx sdk.Context, params feemarkettypes.Params) error
}

// migrateFeeMarketBaseFee repairs feemarket Params.base_fee, which the
// ethermint -> cosmos/evm replacement silently re-encoded by a factor of 10^18.
//
// The two modules agree on the store layout (StoreKey "feemarket", ParamsKey
// "Params") and on proto field 6 for base_fee, but not on the field's Go type:
// ethermint v0.24.1-uptick held a math.Int, cosmos/evm v0.6.2 holds a
// math.LegacyDec. Both marshal as the ASCII of their raw big.Int, and for a
// math.Int that integer IS the value while for a LegacyDec it is the value scaled
// by 10^18 -- so the legacy bytes "1000000000" (a 1 gwei base fee, the shipping
// DefaultBaseFee) decode as 10^-9. The wire form stays valid protobuf, so
// GetParams does not fail: it returns the wrong number, and feemarket's
// BeginBlock then makes it permanent by writing CalculateBaseFee's result back
// through SetParams every block (EnableHeight 0, NoBaseFee false).
//
// The guard is on magnitude, because nothing in the store records which encoding
// produced the bytes: the legacy Int "1000000000" and a hypothetical LegacyDec of
// 10^-9 are byte-identical, and MinGasPrice / MinGasMultiplier are LegacyDec in
// both versions and carry no signal either. Legacy-encoded means the stored value
// is below 1 wei -- true for every base fee this chain has had, since those are
// all below 10^18 wei; repaired means it is a whole number of wei, i.e. >= 1.
// That guard is what makes the transform idempotent, which v0.4.1 needs because it
// deliberately carries no UpgradeAlreadyApplied guard: a replay without it would
// multiply by another 10^18.
//
// Known limits, both unreachable between v0.4.0 and v0.4.1, recorded so the next
// reader re-checks rather than rediscovers them: a legacy base fee >= 10^18 wei
// would land on >= 1 and go unrepaired; a legitimately sub-unit base fee would be
// rescaled. cosmos/evm's CalcGasBaseFee can decay base_fee below 1 on a quiet
// chain, but not while v0.4.0 is running -- v0.4.0 leaves the value at ~10^-9 from
// its first block, and this handler runs in PreBlocker before that block's
// BeginBlock.
//
// It could not be done in v0.4.0: that handler never touched feemarket, and it is
// frozen at the released build.
func migrateFeeMarketBaseFee(ctx sdk.Context, store feeMarketParamsStore) error {
	params := store.GetParams(ctx)
	repaired, changed := withRepairedBaseFee(params)
	if !changed {
		return nil
	}
	if err := store.SetParams(ctx, repaired); err != nil {
		return fmt.Errorf("set feemarket params: %w", err)
	}
	ctx.Logger().Info(
		"feemarket base fee re-scaled after the ethermint -> cosmos/evm switch",
		"upgrade", upgradeName,
		"was", params.BaseFee.String(),
		"now", repaired.BaseFee.String(),
	)
	return nil
}

// withRepairedBaseFee converts base_fee from the legacy math.Int encoding to the
// cosmos/evm math.LegacyDec encoding, reporting whether anything changed. A value
// that is already a whole number of wei is left alone, and so is an absent one: a
// nil base fee means neither module wrote the store, and inventing a fee is out of
// scope for a repair.
func withRepairedBaseFee(params feemarkettypes.Params) (feemarkettypes.Params, bool) {
	fee := params.BaseFee
	// IsNil first, and not for style: LegacyDec.IsPositive/IsNegative/LT
	// dereference the unexported big.Int, so a zero-value LegacyDec panics on a nil
	// pointer instead of answering false -- and a Params read back with field 6
	// absent is exactly that zero value.
	if fee.IsNil() || !fee.IsPositive() || !fee.LT(math.LegacyOneDec()) {
		return params, false
	}

	// Undo the mis-decoding: the stored value is the legacy integer divided by
	// 10^18, so scaling back up recovers it. Exact for any value below 10^18 (the
	// guard's range), because LegacyDec keeps 18 decimals.
	legacyInt := fee.Mul(legacyDecScale).TruncateInt()
	if !legacyInt.IsPositive() {
		return params, false
	}

	params.BaseFee = math.LegacyNewDecFromInt(legacyInt)
	return params, true
}

// erc721UIDIndexStore is the slice of the ERC721 keeper this migration needs. The
// interface is here to make the dependency legible, not for the injectable-failure
// reason the params stores above have -- the prune cannot fail, it only reports.
// Its result type is the module's own, because summarizing it would throw away the
// keys an operator needs.
type erc721UIDIndexStore interface {
	PruneDuplicateUIDIndexEntries(ctx sdk.Context) erc721keeper.UIDIndexPruneReport
}

// pruneErc721UIDIndex collapses the duplicate forward entries in the ERC721
// conversion index: the one-shot half of the key-spelling work, and the only part
// of it that needs a governance upgrade.
//
// Why a handler is needed at all: the runtime fix (read either spelling of a token
// id or contract address, always write the canonical one, compare by value) is
// already in the release and upgrades a pair in place the first time a write path
// touches it -- but it cannot reach a key no write path will ever touch again. A
// leftover forward key is not harmless: it has no reverse partner, so every
// genesis export reports it as degraded and real corruption drowns in the fixed
// noise.
//
// Why it cannot fail the upgrade: this runs from PreBlocker, where an error panics
// the node at the upgrade height and needs a coordinated restart. Residual damage
// is pre-existing state the handler is not required to repair, so the pass deletes
// what it can prove redundant and logs the rest at Error level -- the same posture
// the genesis export takes, and the reason the log carries counters rather than a
// boolean.
func pruneErc721UIDIndex(ctx sdk.Context, store erc721UIDIndexStore) {
	report := store.PruneDuplicateUIDIndexEntries(ctx)

	if uidIndexPruneClean(report) {
		ctx.Logger().Info(
			"erc721 conversion index pruned",
			"upgrade", upgradeName,
			"reverse_entries", report.Scanned,
			"forward_keys_before", report.ForwardBefore,
			"forward_keys_after", report.ForwardAfter(),
			"deleted", len(report.Deleted),
		)
		return
	}

	// Each condition is listed rather than leaning on Balanced() alone: an orphan in
	// one binding can cancel an unpaired entry in another in the totals.
	ctx.Logger().Error(
		"erc721 conversion index still degraded after the prune",
		"upgrade", upgradeName,
		"reverse_entries", report.Scanned,
		"forward_keys_before", report.ForwardBefore,
		"forward_keys_after", report.ForwardAfter(),
		"deleted", len(report.Deleted),
		"conflicts", len(report.Conflicts),
		"orphans", len(report.Orphans),
		"unpaired_nfts", len(report.UnpairedNFTs),
		"conflict_sample", firstFew(report.Conflicts),
		"orphan_sample", firstFew(report.Orphans),
		"unpaired_nft_sample", firstFew(report.UnpairedNFTs),
	)
}

// uidIndexPruneClean reports whether the pass left the index in the state a
// healthy store is in: one forward key per reverse entry, and nothing refused.
func uidIndexPruneClean(report erc721keeper.UIDIndexPruneReport) bool {
	return report.Balanced() &&
		len(report.Conflicts) == 0 &&
		len(report.Orphans) == 0 &&
		len(report.UnpairedNFTs) == 0
}

// firstFew caps a diagnostic list at five entries. The prune sorts every list,
// so the sample is the same on every validator — a log line that differs
// between nodes is worse than no log line.
func firstFew(keys []string) []string {
	const maxSample = 5
	if len(keys) <= maxSample {
		return keys
	}
	return keys[:maxSample]
}
