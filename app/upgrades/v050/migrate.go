package v050

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmibc "github.com/cosmos/evm/ibc"
	erc20types "github.com/cosmos/evm/x/erc20/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
)

// ibcVoucherPrefix marks the bank denominations that are ICS-20 vouchers: an
// ERC20 representation of them is what these migrations hand back to their
// holders.
const ibcVoucherPrefix = "ibc/"

// ibcVoucherPairSettler is the slice of the erc721 keeper the R1-C voucher-pair
// settlement needs. The concrete keeper reports its own counts; the interface
// exists only so the helper below stays unit testable without an app.
type ibcVoucherPairSettler interface {
	SettleIBCVoucherPairs(ctx sdk.Context) (settled, staleCleared, kept, pairsDeleted int)
}

// settleIBCVoucherPairs runs the R1-C legacy voucher-pair settlement (see
// x/erc721/keeper/voucher_settlement.go). The keeper method is fail-safe — it
// never returns an error, because an ambiguous per-token state is kept and left
// to the terminal un-wrap (A3) — and it self-reports its counts, so there is no
// separate summary line here.
func settleIBCVoucherPairs(ctx sdk.Context, erc721 ibcVoucherPairSettler) {
	erc721.SettleIBCVoucherPairs(ctx)
}

// ibcVoucherBankStore is the slice of the bank keeper the v0.5.0 migrations
// need. It is an interface so the migrations stay unit testable: a Toolbox
// embeds the concrete keepers.AppKeepers, so the handler itself can only be run
// on a real application (see app/upgrade_v050_handler_test.go).
//
// The bank module takes a plain context.Context (SDK v0.53 keeper signature);
// sdk.Context satisfies it, which is what the handler passes.
type ibcVoucherBankStore interface {
	IterateTotalSupply(ctx context.Context, cb func(sdk.Coin) bool)
	GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool)
	SetDenomMetaData(ctx context.Context, metadata banktypes.Metadata)
}

// ibcVoucherTokenPairStore is the slice of the erc20 keeper the backfill needs.
type ibcVoucherTokenPairStore interface {
	IsDenomRegistered(ctx sdk.Context, denom string) bool
	RegisterERC20Extension(ctx sdk.Context, denom string) (*erc20types.TokenPair, error)
}

// erc20ParamsStore is the slice of the erc20 keeper the parameter migration
// needs. It is the concrete keeper verbatim, because both methods already exist
// with these signatures.
type erc20ParamsStore interface {
	GetParams(ctx sdk.Context) erc20types.Params
	SetParams(ctx sdk.Context, params erc20types.Params) error
}

// enablePermissionlessRegistration turns x/erc20's PermissionlessRegistration
// parameter on, on both starting states.
//
// # Why the flip has to live here
//
// v0.4.0's migrateErc20Params forces the parameter OFF, deliberately, and a
// chain already past v0.4.0 never re-runs that migration. Changing the value in
// v0.4.0 would therefore move mainnet and leave testnet -- the chain that did
// execute it -- on the old setting forever. A step in this handler is the only
// in-tree place that reaches both starting states. It has to run AFTER the legacy
// stage above, because that stage writes the whole Params struct and would
// overwrite this field.
//
// "In-tree" is the operative word: this is not the only way the parameter can
// move. MsgUpdateParams is authority-gated (keeper/msg_server.go:161-172) and the
// authority is the gov module account, so a passed proposal rewrites it at any
// height. Testnet did exactly that -- proposal 16, "ERC20 Param Change: Enable
// Permissionless Registration", executed 2026-09-17 at height 14974481 -- so a
// chain can hold either value with no migration having run. Read the value; do
// not infer it from the release notes.
//
// # What turning it on changes
//
//   - The inbound ICS-20 callback registers a token pair (and a dynamic
//     precompile account) for any previously unseen `ibc/` denom. That branch
//     does not read the parameter upstream; this repo gates it separately, and
//     with the parameter on the gate (app/keepers/erc20_ibc_gate.go) reports
//     "enabled" and delegates, so the upstream behavior applies again. The
//     callback runs with a zeroed KV gas config (ibc_callbacks.go:53-56), so a
//     counterparty chain can make this chain write a pair at no relayer cost.
//     This is the state growth the original OFF value was bounding.
//   - MsgRegisterERC20 becomes permissionless: any account can map an already
//     deployed ERC20 contract to a new `erc20:0x…` denom (msg_server.go:181-185).
//
// # What it does not do
//
// It is not a retroactive repair. Upstream registers on a *new inbound packet*,
// so every voucher that arrived while the switch was off stays pairless until
// backfillIBCVoucherTokenPairs gives it one: the two migrations are
// complementary, and the backfill is not made redundant by this one.
//
// A read-modify-write: EnableErc20 carries the value v0.4.0 migrated from the
// legacy subspace (and any field a future cosmos/evm adds to Params) instead of
// being rebuilt from DefaultParams, which would silently force EnableErc20 on
// even if the legacy chain had it off -- and EnableErc20 is what makes coin
// conversion possible at all (keeper/mint.go:23). Idempotent: an already-on
// parameter performs no write, so a crash-restart leaves state byte-identical.
func enablePermissionlessRegistration(
	ctx sdk.Context,
	store erc20ParamsStore,
	logger log.Logger,
) error {
	params := store.GetParams(ctx)
	if params.PermissionlessRegistration {
		logger.Info("permissionless erc20 registration already enabled")
		return nil
	}

	params.PermissionlessRegistration = true
	if err := store.SetParams(ctx, params); err != nil {
		return fmt.Errorf("set erc20 params: %w", err)
	}

	logger.Info(
		"permissionless erc20 registration enabled",
		"upgrade", upgradeName,
		"EnableErc20", params.EnableErc20,
	)
	return nil
}

// ibcVoucherDenoms returns the ibc/-prefixed denominations that hold a supply,
// sorted so the migration's writes happen in a deterministic order.
//
// The scope is deliberately "what the chain already holds", not a hardcoded
// list. One binary serves both mainnet (which received IBC vouchers for years
// under ibc-go v8) and testnet (11 vouchers, all created after the v0.4.x
// migration), and their denom sets have nothing in common; a compiled-in
// whitelist could only ever be right for one of them. Supply is also the honest
// definition of "存量": a voucher with no supply has no holder whose asset could
// be repaired.
//
// This does NOT reopen the inbound auto-registration the ERC20IBCGate closes.
// That gate limits what a hostile counterparty can make the chain write *in the
// future*; this is a one-time pass over state that already exists, and its size
// is bounded by what the chain accepted before the upgrade.
func ibcVoucherDenoms(ctx sdk.Context, bank ibcVoucherBankStore) []string {
	var denoms []string
	bank.IterateTotalSupply(ctx, func(coin sdk.Coin) bool {
		if strings.HasPrefix(coin.Denom, ibcVoucherPrefix) {
			denoms = append(denoms, coin.Denom)
		}
		return false
	})
	sort.Strings(denoms)
	return denoms
}

// normalizeIBCVoucherERC20Decimals rewrites the bank metadata of every IBC
// voucher into the only shape that satisfies both of the constraints it lives
// under, so that cosmos/evm's ERC20 precompile reports the right decimals.
//
// # What is wrong without it
//
// ibc-go writes a voucher's metadata as:
//
//	Base       = "ibc/3D02…"              (the voucher hash)
//	Display    = "transfer/channel-1/auoc" (the full denom path)
//	DenomUnits = [{Denom: "auoc", Exponent: 0}]
//
// cosmos/evm's precompile (precompiles/erc20/query.go:89-127) reads decimals
// differently for these: with a "ibc/" base it matches the LAST SEGMENT of
// Display against DenomUnits. That segment is "auoc", it matches the only unit,
// and that unit's exponent is 0 -- so decimals() returns 0. Wallets, explorers
// and DEX front-ends then read a 1 auoc balance as 10^18.
//
// THE SHAPE THAT WORKS ("shape C" in the test-side analysis)
//
//	Display    = "auoc"                    (the source denom, no path)
//	DenomUnits = [{Base, 0}, {"auoc", 18}]
//
// which satisfies:
//
//   - x/bank's Metadata.Validate (first unit is the base with exponent 0, units
//     sorted ascending, Display present among them), so the metadata stops
//     being illegal state and the export-side workaround stops masking it; and
//   - the precompile's rule above, whose last-segment match now lands on the
//     unit carrying the real exponent.
//
// There is no other runtime entry point: SDK v0.53 has no MsgSetDenomMetadata,
// only the keeper method used here. That is why this repair has to ride an
// upgrade.
//
// Exponent is taken from an existing non-zero unit for the source denom when
// there is one, and derived from the source denom's prefix otherwise (u… -> 6,
// a… -> 18). A source denom that matches neither is skipped with a log line
// rather than guessed: a wrong exponent is worse than the status quo, because
// it silently rescales a real asset.
//
// Idempotent: a voucher already in shape C produces no write.
//
// It cannot fail, and returns nothing on purpose: every unusable voucher is
// skipped individually (SetDenomMetaData itself has no error return), and halting
// a chain upgrade over one dead voucher would be worse than leaving it as it is.
func normalizeIBCVoucherERC20Decimals(ctx sdk.Context, bank ibcVoucherBankStore, logger log.Logger) {
	var normalized, skipped int

	for _, denom := range ibcVoucherDenoms(ctx, bank) {
		metadata, found := bank.GetDenomMetaData(ctx, denom)
		if !found {
			// No metadata: the precompile falls back to deriving from the source
			// denom, which is already the correct branch.
			continue
		}

		updated, reason, ok := normalizedIBCVoucherMetadata(metadata)
		if !ok {
			skipped++
			logger.Info("skipping ibc voucher metadata", "denom", denom, "reason", reason)
			continue
		}
		if updated.Display == metadata.Display && sameDenomUnits(updated.DenomUnits, metadata.DenomUnits) {
			continue
		}

		bank.SetDenomMetaData(ctx, updated)
		normalized++
		// The old units are dropped, so they have to be readable in the log of
		// the block that did it.
		logger.Info(
			"normalized ibc voucher erc20 decimals",
			"denom", denom,
			"display", updated.Display,
			"decimals", updated.DenomUnits[len(updated.DenomUnits)-1].Exponent,
			"old_display", metadata.Display,
			"old_units", formatDenomUnits(metadata.DenomUnits),
		)
	}

	logger.Info("ibc voucher erc20 decimals normalized", "normalized", normalized, "skipped", skipped)
}

// normalizedIBCVoucherMetadata returns the shape-C rewrite of a voucher's
// metadata, or the reason it cannot be repaired. It never mutates its input.
func normalizedIBCVoucherMetadata(metadata banktypes.Metadata) (banktypes.Metadata, string, bool) {
	if upgrades.HasNilDenomUnit(metadata.DenomUnits) {
		// A nil unit means the stored record is structurally corrupt. Refuse to
		// rewrite it: the shape-C rewrite reuses nothing but Base/Display here, so
		// the exponent would have to be guessed, and a wrong exponent silently
		// rescales a real asset. Skip it with a reason and leave it for an
		// operator. (x/bank's own Validate would panic on this record, so every
		// walk below must stay nil-safe too.)
		return metadata, "metadata carries a nil denom unit", false
	}

	if metadata.Display == metadata.Base {
		// The source denom is not recoverable, and deriving one from the hash
		// tail would invent an asset name.
		return metadata, "display is the voucher hash, source denom unknown", false
	}

	sourceDenom := metadata.Display
	if i := strings.LastIndex(sourceDenom, "/"); i >= 0 {
		sourceDenom = sourceDenom[i+1:]
	}
	if sourceDenom == "" || sourceDenom == metadata.Base {
		return metadata, "display carries no usable source denom", false
	}

	decimals, ok := nonZeroUnitExponent(metadata.DenomUnits, sourceDenom)
	if !ok {
		derived, err := evmibc.DeriveDecimalsFromDenom(sourceDenom)
		if err != nil {
			return metadata, fmt.Sprintf("cannot derive decimals for source denom %q: %v", sourceDenom, err), false
		}
		decimals = uint32(derived)
	}

	updated := metadata
	updated.Display = sourceDenom
	updated.DenomUnits = []*banktypes.DenomUnit{
		{Denom: metadata.Base, Exponent: 0},
		{Denom: sourceDenom, Exponent: decimals},
	}

	// Refuse to write metadata the bank module itself would reject: once such a
	// record is on chain, the export-side normalization rewrites Display to the
	// base denom and decimals() goes from 0 to a revert -- worse than the
	// starting point.
	if err := updated.Validate(); err != nil {
		return metadata, fmt.Sprintf("repaired metadata is invalid: %v", err), false
	}
	return updated, "", true
}

// nonZeroUnitExponent returns the exponent the metadata already records for
// denom, if it is non-zero. Shape A carries the source denom with exponent 0,
// which is the value being repaired, so zero is not a usable answer here.
//
// Nil elements are skipped rather than dereferenced: proto repeated message
// fields can carry them, and this migration's job is to survive inconsistent
// historical state, not to panic on it.
func nonZeroUnitExponent(units []*banktypes.DenomUnit, denom string) (uint32, bool) {
	for _, unit := range units {
		if unit == nil {
			continue
		}
		if unit.Denom == denom && unit.Exponent > 0 {
			return unit.Exponent, true
		}
	}
	return 0, false
}

// sameDenomUnits reports whether two unit lists are equal in order and content.
// A nil element compares as such (never dereferenced), so a corrupt list is
// reported as changed rather than crashing the caller.
func sameDenomUnits(a, b []*banktypes.DenomUnit) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if (a[i] == nil) != (b[i] == nil) {
			return false
		}
		if denomUnitDenom(a[i]) != denomUnitDenom(b[i]) || denomUnitExponent(a[i]) != denomUnitExponent(b[i]) {
			return false
		}
	}
	return true
}

// denomUnitDenom / denomUnitExponent read a possibly-nil denom unit. They exist
// because x/bank's Metadata.Validate dereferences every unit unconditionally, so
// any path that walks a stored Metadata must tolerate nil first.
func denomUnitDenom(u *banktypes.DenomUnit) string {
	if u == nil {
		return ""
	}
	return u.Denom
}

func denomUnitExponent(u *banktypes.DenomUnit) uint32 {
	if u == nil {
		return 0
	}
	return u.Exponent
}

// formatDenomUnits renders a unit list for the audit log, marking nil elements
// instead of dereferencing them.
func formatDenomUnits(units []*banktypes.DenomUnit) string {
	parts := make([]string, 0, len(units))
	for _, unit := range units {
		if unit == nil {
			parts = append(parts, "<nil>")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%d", unit.Denom, unit.Exponent))
	}
	return strings.Join(parts, ",")
}

// backfillIBCVoucherTokenPairs gives every IBC voucher already on the chain the
// STRv2 token pair it never had.
//
// # Why they are missing
//
// Two design decisions stacked up: the v0.4.0 handler deletes the legacy
// OWNER_MODULE pairs (the old module's contracts are not convertible any more
// under cosmos/evm), and the ERC20IBCGate suppresses the inbound
// auto-registration that would have replaced them. The gate stays closed on
// purpose -- it bounds unbounded state growth from a hostile counterparty -- but
// it also means no denom that arrived before the upgrade can ever acquire a
// pair on its own. RegisterERC20Extension is a keeper method that reads no
// governance parameter, so the backfill can hand out those pairs from here
// without reopening the gate.
//
// The pairs are derived addresses (NewTokenPairSTRv2 hashes the voucher), not
// deployed contracts, so nothing is deployed and no foreign address is trusted.
//
// A denom that already has a pair is left alone (RegisterERC20Extension requires
// it), and one that fails is logged and skipped rather than aborting the chain
// upgrade: the failure modes are per-denom (a hand-deployed contract sitting on
// the derived address, a malformed voucher denom), and halting mainnet at the
// upgrade height over one unreachable voucher would be far worse than leaving
// that voucher without an EVM representation. The summary line carries the
// counts so a systemic failure is visible in the upgrade log.
//
// Not aborting must not mean "unrecoverable", though. The handler writes its
// completion marker unconditionally, so a denom that fails here would never be
// retried by a replay. The failed denoms are therefore RETURNED, and the caller
// persists them (see recordBackfillFailures / RetryIBCVoucherBackfill) so the
// failure is enumerable on chain and has a retry path, rather than surviving only
// as a log line an operator may never read.
func backfillIBCVoucherTokenPairs(
	ctx sdk.Context,
	bank ibcVoucherBankStore,
	pairs ibcVoucherTokenPairStore,
	logger log.Logger,
) (failed []string) {
	var registered, skipped int

	for _, denom := range ibcVoucherDenoms(ctx, bank) {
		if pairs.IsDenomRegistered(ctx, denom) {
			skipped++
			continue
		}

		pair, err := pairs.RegisterERC20Extension(ctx, denom)
		if err != nil {
			failed = append(failed, denom)
			logger.Error("failed to backfill ibc voucher token pair", "denom", denom, "err", err)
			continue
		}

		registered++
		logger.Info("backfilled ibc voucher token pair", "denom", denom, "erc20", pair.Erc20Address)
	}

	logger.Info(
		"ibc voucher token pair backfill complete",
		"registered", registered,
		"already_registered", skipped,
		"failed", len(failed),
	)
	return failed
}

// RetryIBCVoucherBackfill re-runs the token-pair backfill for exactly the denoms
// a previous run recorded as failed, returning the count it registered and the
// denoms that are still unmatched.
//
// It is the recovery path for the per-denom skip above: the v0.5.0 handler writes
// its completion marker unconditionally, so without an explicit retry a voucher
// whose registration failed once would keep no EVM representation forever. A
// v0.5.x tail migration (or an authority-gated tool) calls this with the denoms
// stored at backfillFailuresKey; it is idempotent -- a denom that has since
// acquired a pair is skipped -- so it is safe to call repeatedly.
func RetryIBCVoucherBackfill(
	ctx sdk.Context,
	pairs ibcVoucherTokenPairStore,
	denoms []string,
	logger log.Logger,
) (registered int, stillFailed []string) {
	for _, denom := range denoms {
		if pairs.IsDenomRegistered(ctx, denom) {
			continue
		}

		pair, err := pairs.RegisterERC20Extension(ctx, denom)
		if err != nil {
			stillFailed = append(stillFailed, denom)
			logger.Error("retry: failed to backfill ibc voucher token pair", "denom", denom, "err", err)
			continue
		}

		registered++
		logger.Info("retry: backfilled ibc voucher token pair", "denom", denom, "erc20", pair.Erc20Address)
	}
	return registered, stillFailed
}
