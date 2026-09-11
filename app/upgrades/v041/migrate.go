package v041

import (
	"fmt"

	"cosmossdk.io/math"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	erc721keeper "github.com/UptickNetwork/uptick/x/erc721/keeper"
)

// defaultActiveStaticPrecompiles is the list of static precompile addresses
// registered by cosmos/evm's precompilestypes.DefaultStaticPrecompiles, which
// Uptick wires into the EVM keeper. It intentionally excludes the vesting
// precompile (0x803), which Uptick does not register, to avoid the
// "precompiled contract not stored in memory" panic when the EVM resolves it.
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

// ConfigureDefaultStaticPrecompiles overrides the EVM module's default params
// so fresh chains also activate the static precompiles (the cosmos/evm default
// is an empty list). It must be called before any evmtypes.DefaultParams()
// invocation — i.e. before the module manager (and its DefaultGenesis) is
// built in app.go. It replaces the former package init(): a global mutation
// as an import side effect is invisible to readers and easy to lose during
// refactors, so the app wires it explicitly.
func ConfigureDefaultStaticPrecompiles() {
	evmtypes.DefaultStaticPrecompiles = defaultActiveStaticPrecompiles
}

// evmParamsStore is the slice of the EVM keeper this migration needs.
//
// It is an interface rather than the concrete keeper on purpose: the real keeper
// can only fail SetParams on invalid params, and this migration always writes a
// known-good list, so with the concrete type the error branch below would be
// unreachable from any test. Narrowing the dependency makes the failure path
// injectable instead of untested.
type evmParamsStore interface {
	GetParams(ctx sdk.Context) evmtypes.Params
	SetParams(ctx sdk.Context, params evmtypes.Params) error
}

// migrateActiveStaticPrecompiles repairs the EVM params so the static
// precompiles are actually activated. The v0.4.0 upgrade introduced the
// ActiveStaticPrecompiles params field (the legacy ethermint params proto had no
// such field) but never populated it, leaving every custom precompile inactive:
// IsAvailableStaticPrecompile returns false, so GetStaticPrecompileInstance
// never loads the contract and precompile calls fail.
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

// withDefaultActiveStaticPrecompiles returns params with the default precompile
// list filled in when the stored list is empty. An already-populated list is
// preserved so governance removals are not silently reverted.
func withDefaultActiveStaticPrecompiles(params evmtypes.Params) (evmtypes.Params, bool) {
	// An empty list means "reset to the default set". Note that proto round-trips
	// collapse an explicitly-empty list to nil, so a governance decision to clear
	// the list (disable all static precompiles) is not distinguishable from
	// "never configured" here; both reset to the default.
	if len(params.ActiveStaticPrecompiles) != 0 {
		return params, false
	}
	params.ActiveStaticPrecompiles = append([]string(nil), defaultActiveStaticPrecompiles...)
	return params, true
}

// legacyDecScale is the fixed-point scale of cosmossdk.io/math.LegacyDec: the
// numeric value of a LegacyDec is its raw big.Int divided by 10^18. Written out
// as a literal because math.LegacyNewDec takes an int64 and 10^18 fits.
var legacyDecScale = math.LegacyNewDec(1_000_000_000_000_000_000)

// feeMarketParamsStore is the slice of the feemarket keeper this migration
// needs.
//
// It is an interface rather than the concrete keeper for the same reason as
// evmParamsStore above: the real keeper can only fail SetParams on invalid
// params and this migration always writes a valid one, so the error branch
// would be unreachable from any test.
type feeMarketParamsStore interface {
	GetParams(ctx sdk.Context) feemarkettypes.Params
	SetParams(ctx sdk.Context, params feemarkettypes.Params) error
}

// migrateFeeMarketBaseFee repairs feemarket Params.base_fee, which the
// ethermint -> cosmos/evm replacement silently re-encoded by a factor of 10^18.
//
// BACKGROUND. The two modules agree on the store layout (StoreKey "feemarket",
// ParamsKey "Params") and on proto field 6 for base_fee, but not on the field's
// Go type:
//
//	ethermint v0.24.1-uptick  Params.BaseFee cosmossdk.io/math.Int
//	cosmos/evm    v0.6.2      Params.BaseFee cosmossdk.io/math.LegacyDec
//
// Both custom types marshal as the ASCII of their raw big.Int (math.Int.Marshal
// and LegacyDec.Marshal both end in i.MarshalText(), cosmossdk.io/math v1.5.3
// int.go:466 and legacy_dec.go:837). For math.Int the raw integer *is* the
// value; for LegacyDec it is the value scaled by 10^18. So the legacy bytes
// "1000000000" - a 1 gwei base fee, the shipping DefaultBaseFee - decode as
// 0.000000001000000000 instead of 1000000000.000000000000000000.
//
// The wire form stays valid protobuf, so GetParams does not fail; it returns
// the wrong number. It is then persisted and made permanent: feemarket's
// BeginBlock runs CalculateBaseFee every block (EnableHeight is 0 and
// NoBaseFee is false, so IsBaseFeeEnabled is true) and SetBaseFee writes the
// result straight back through SetParams.
//
// WHY HERE AND NOT IN v0.4.0. The v0.4.0 handler never touched feemarket (its
// whole execute list is evm/erc20/erc721/cw721/nft-transfer/legacy params
// subspaces), and it is frozen at the released build. v0.4.1 is the next plan
// able to carry the repair.
//
// WHY A VALUE-DOMAIN GUARD INSTEAD OF A VERSION MARKER. Nothing in the store
// says which encoding produced the bytes: the legacy Int "1000000000" and a
// (hypothetical) LegacyDec of 0.000000001 are byte-identical, and MinGasPrice /
// MinGasMultiplier are LegacyDec in both versions and so carry no signal
// either. The discriminator therefore has to be the value itself, and the one
// property that separates the two states is magnitude:
//
//   - legacy-encoded: the stored value is legacyInt / 10^18, which is < 1 for
//     every legacy base fee below 10^18 wei (1 token) - i.e. for every value
//     this chain has ever had.
//   - already repaired (or written by cosmos/evm itself): the value is a whole
//     number of wei, i.e. >= 1.
//
// The guard is what makes the transform idempotent, which v0.4.1 needs because
// it deliberately carries no UpgradeAlreadyApplied guard. Without it a replay
// would multiply by another 10^18.
//
// KNOWN LIMITS, both unreachable between v0.4.0 and v0.4.1 and both listed here
// so the next reader re-checks them rather than rediscovering them:
//   - a legacy base fee >= 10^18 wei would land on >= 1 and go unrepaired;
//   - a legitimately sub-unit base fee would be rescaled. cosmos/evm's
//     CalcGasBaseFee can decay base_fee below 1 on a quiet chain, but not while
//     v0.4.0 is running: v0.4.0 leaves the value at ~10^-9 from its first
//     block, and this handler runs in PreBlocker before that block's
//     BeginBlock.
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

// withRepairedBaseFee returns params with base_fee converted from the legacy
// math.Int encoding to the cosmos/evm math.LegacyDec encoding, and reports
// whether anything changed. A value that is already a whole number of wei is
// left alone, and so is an absent one: a nil (zero-value) base fee means the
// store was written by something other than the ethermint or the cosmos/evm
// module, and inventing a fee there is out of scope for a repair migration. See
// migrateFeeMarketBaseFee for why magnitude is the only available
// discriminator.
func withRepairedBaseFee(params feemarkettypes.Params) (feemarkettypes.Params, bool) {
	fee := params.BaseFee
	// IsNil must be tested first, and not for style: cosmossdk.io/math v1.5.3
	// LegacyDec.IsPositive/IsNegative/LT all dereference the unexported big.Int
	// directly (legacy_dec.go:220), so a zero-value LegacyDec panics with a nil
	// pointer dereference instead of answering false. A Params read back from a
	// store whose field 6 was absent leaves exactly that zero value.
	if fee.IsNil() || !fee.IsPositive() || !fee.LT(math.LegacyOneDec()) {
		return params, false
	}

	// Undo the mis-decoding: the current value is the legacy integer divided by
	// 10^18, so scaling back up recovers the integer the math.Int held. The
	// multiplication is exact for any legacy value below 10^18 (the guard's
	// range), because LegacyDec keeps 18 decimals.
	legacyInt := fee.Mul(legacyDecScale).TruncateInt()
	if !legacyInt.IsPositive() {
		return params, false
	}

	params.BaseFee = math.LegacyNewDecFromInt(legacyInt)
	return params, true
}

// erc721UIDIndexStore is the slice of the ERC721 keeper this migration needs.
//
// The interface is here for the ordinary reason (it makes the dependency of the
// migration legible) rather than for the injectable-failure reason the two
// params stores above have: the prune cannot fail, it only reports. Its result
// type is the module's own, because a summarised version of it would throw away
// exactly the keys an operator needs.
type erc721UIDIndexStore interface {
	PruneDuplicateUIDIndexEntries(ctx sdk.Context) erc721keeper.UIDIndexPruneReport
}

// pruneErc721UIDIndex collapses the duplicate forward entries in the ERC721
// conversion index, which is the one-shot half of the key-spelling work and the
// only part of it that needs a governance upgrade.
//
// WHY THIS IS IN A HANDLER. The runtime fix (read either spelling of a token id
// or a contract address, always write the canonical one, compare by value) is
// already in the release and needs no migration: it upgrades a pair in place the
// first time a write path touches it. What it cannot do is reach a key that no
// write path will ever touch again, and a forward key left behind by the
// pre-v0.4.1 module is not harmless — it has no reverse partner, so every
// genesis export reports it as a degraded export and real corruption drowns in
// the fixed noise.
//
// WHY IT CANNOT FAIL THE UPGRADE. This runs from PreBlocker, where returning an
// error panics the node at the upgrade height and requires a coordinated
// restart. Residual damage is pre-existing state the handler is not required to
// be able to repair, so the pass reports what it can prove is redundant, deletes
// exactly that, and logs the rest at Error level. That is the same posture the
// genesis export takes, and it is the reason the counters are logged rather than
// just a boolean.
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

	// Every condition is listed explicitly instead of leaning on Balanced()
	// alone: an orphan in one binding and an unpaired reverse entry in another
	// would cancel out in the totals while the store is still damaged.
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

// uidIndexPruneClean reports whether the pass left the two indexes in the state
// a healthy store is in: one forward key per reverse entry, and nothing the
// prune had to refuse to touch.
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
	const max = 5
	if len(keys) <= max {
		return keys
	}
	return keys[:max]
}
