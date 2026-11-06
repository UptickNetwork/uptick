// Package v051 implements the v0.5.1 upgrade plan.
//
// # Who runs this plan
//
//   - Mainnet: last completed plan is v0.5.0 → upgrades to v0.5.1
//   - Testnet (Origin3): last completed plan is v0.4.1 → upgrades to v0.5.1
//
// # What it does
//
//  1. Refuses a legacy ethermint (pre-v0.4.0) starting state — those chains
//     must take v0.5.0 first.
//  2. Re-applies the v0.5.0 tail migrations (permissionless erc20 registration,
//     voucher decimals, voucher token-pair backfill). Idempotent on mainnet;
//     the only path for testnet, which never ran v0.5.0.
//  3. Adds the x/appmining store and lets RunMigrations InitGenesis the module
//     (ConsensusVersion 1) with default params and the reward-pool module account.
//  4. Overwrites the fresh defaults with the V1 production baseline
//     (applyV1BaselineParams): denom is derived from the live staking bond
//     denom, so the same binary lands auptick on mainnet and auoc on the
//     origin testnet with no per-chain config, and params.Authority is set to
//     the configured ops multisig (v1BaselineAuthority) so the remaining
//     setup -- pool opening, funding, registrations -- runs as plain
//     transactions. After this step the UGE protocol is usable as-is; later
//     adjustments go through MsgUpdateParams.
//
// # Store upgrades
//
// Added: appmining. Deleted: none. Renamed: none.
package v051

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/UptickNetwork/uptick/app/upgrades"
	v050 "github.com/UptickNetwork/uptick/app/upgrades/v050"
	appminingtypes "github.com/UptickNetwork/uptick/x/appmining/types"
)

const upgradeName = "v0.5.1"

// migrationsAppliedKey records that this handler completed, so a replayed plan
// skips the (idempotent) custom steps and only re-runs RunMigrations.
var migrationsAppliedKey = []byte("v0.5.1/migrations-applied")

// Upgrade is the v0.5.1 upgrade plan.
var Upgrade = upgrades.Upgrade{
	UpgradeName:               upgradeName,
	UpgradeHandlerConstructor: upgradeHandlerConstructor,
	StoreUpgrades: &storetypes.StoreUpgrades{
		Added: []string{appminingtypes.StoreKey},
	},
}

func upgradeHandlerConstructor(
	_ *module.Manager,
	c module.Configurator,
	box upgrades.Toolbox,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		logger := sdkCtx.Logger()

		storeKey := box.GetKVStoreKey(upgradetypes.StoreKey)
		if storeKey == nil {
			return nil, fmt.Errorf("x/upgrade store key is not registered")
		}
		upgradeStore := sdkCtx.KVStore(storeKey)

		if upgradeStore.Has(migrationsAppliedKey) {
			logger.Info("v0.5.1 migration set already applied; re-running module migrations only",
				"upgrade", upgradeName)
			return box.ModuleManager.RunMigrations(sdkCtx, c, vm)
		}

		// Refuse pre-v0.4.0 ethermint state. Mainnet reaches v0.5.1 via v0.5.0;
		// testnet is already on cosmos/evm after v0.4.1. Neither should present
		// a missing EvmCoinInfo record.
		if !evmCoinInfoPresent(sdkCtx, box.GetKVStoreKey(evmtypes.StoreKey)) {
			return nil, fmt.Errorf(
				"v0.5.1 refuses a legacy ethermint starting state: EvmCoinInfo is absent; " +
					"mainnet must execute v0.5.0 first, testnet must already be on v0.4.1",
			)
		}

		logger.Info("executing upgrade plan",
			"name", upgradeName,
			"change", "v0.5.0 tail migrations + x/appmining store",
		)

		if err := v050.ApplyTailMigrations(sdkCtx, box, logger); err != nil {
			return nil, fmt.Errorf("v0.5.0 tail migrations: %w", err)
		}

		// RunMigrations InitGenesis's x/appmining (new ConsensusVersion 1) and
		// advances every other module that already matches to a no-op.
		newVM, err := box.ModuleManager.RunMigrations(sdkCtx, c, vm)
		if err != nil {
			return nil, fmt.Errorf("run migrations: %w", err)
		}

		// Overwrite InitGenesis defaults with the V1 production baseline so the
		// protocol is live at the upgrade height. Later tuning goes through
		// MsgUpdateParams, not another upgrade.
		if err := applyV1BaselineParams(sdkCtx, box); err != nil {
			return nil, fmt.Errorf("apply v1 baseline params: %w", err)
		}

		// Ensure segregated fund accounts exist (permissions are in maccPerms).
		if acc := box.AccountKeeper.GetModuleAccount(sdkCtx, appminingtypes.RewardPoolName); acc == nil {
			return nil, fmt.Errorf("appmining_reward_pool module account was not created during upgrade")
		}
		if acc := box.AccountKeeper.GetModuleAccount(sdkCtx, appminingtypes.ChallengeEscrowName); acc == nil {
			return nil, fmt.Errorf("appmining_challenge_escrow module account was not created during upgrade")
		}

		// RC/dev builds once funded the legacy ModuleName account; settleable
		// funds now live in RewardPoolName. Move any leftover balances over.
		if err := migrateLegacyAppminingPool(sdkCtx, box); err != nil {
			return nil, err
		}

		upgradeStore.Set(migrationsAppliedKey, []byte{0x01})
		logger.Info("v0.5.1 migration set complete", "upgrade", upgradeName)
		return newVM, nil
	}
}

// migrateLegacyAppminingPool moves balances from the legacy "appmining" module
// account into appmining_reward_pool when present and non-empty.
func migrateLegacyAppminingPool(ctx sdk.Context, box upgrades.Toolbox) error {
	legacy := box.AccountKeeper.GetModuleAccount(ctx, appminingtypes.ModuleName)
	if legacy == nil {
		return nil
	}
	bal := box.BankKeeper.GetAllBalances(ctx, legacy.GetAddress())
	if bal.IsZero() {
		return nil
	}
	if err := box.BankKeeper.SendCoinsFromModuleToModule(
		ctx, appminingtypes.ModuleName, appminingtypes.RewardPoolName, bal,
	); err != nil {
		return fmt.Errorf("migrate legacy appmining balances to reward pool: %w", err)
	}
	ctx.Logger().Info("migrated legacy appmining module balances to reward pool",
		"upgrade", upgradeName,
		"amount", bal.String(),
	)
	return nil
}

func evmCoinInfoPresent(ctx sdk.Context, evmStoreKey *storetypes.KVStoreKey) bool {
	if evmStoreKey == nil {
		return false
	}
	return ctx.KVStore(evmStoreKey).Has(evmtypes.KeyPrefixEvmCoinInfo)
}

// V1 production baseline applied at the upgrade height. The numerals are the
// uptick_7777-1 values verified end-to-end on 2026-10-26/28 (DEC-10 vectors,
// T4 execution). They are BASELINE, not immutable: any of them can be retuned
// later with MsgUpdateParams -- nothing here is load-bearing for correctness,
// only for "the protocol is usable right after the upgrade".
const (
	// funded threshold below which the V1 flow stays disabled (test01: pool-1
	// was funded 10x above this before the e2e run).
	v1BaselineMinFunded = "1000000"
	// challenge deposit (T4: deposit-mode open access). Same numeral on both
	// chains: auptick and auoc are both 18-decimal atto units.
	v1BaselineChallengeDeposit = "500000000000000000000"
	// legacy challenge bond, kept numerically consistent with the deposit
	// (legacy flow itself is paused; the field must merely stay valid).
	v1BaselineChallengeBond = "500000000000000000000"
)

// v1BaselineRuleVersions is the approve rule-version allowlist (F-1: empty
// would deny every approval with error 77). Mirrors test01's live set.
var v1BaselineRuleVersions = []string{
	"rule-1",
	"uge-candidate-rules-0.1",
	"uge-candidate-rules-0.3",
	"uge-candidate-rules-0.4",
}

// v1BaselineAssessmentVersions is the assessment credential-version allowlist
// (AssessmentVersionAllowed is default-deny on an empty list).
var v1BaselineAssessmentVersions = []string{
	"uge-assessment-0.1",
}

// v1BaselineAuthority is written into params.Authority at the upgrade height,
// replacing the gov-module default so pool opening, funding (now
// authority-gated, see MsgFundEntitlementPool), role/signer registration, and
// retuning run as plain transactions from this account instead of governance
// proposals.
//
// RELEASE GATE: this must be a funded multisig address before the v0.5.1
// binaries are built -- applyV1BaselineParams refuses to run on an empty
// value. The same address takes effect on BOTH chains served by this binary
// (mainnet and origin testnet). It stays mutable afterwards: the current
// authority can hand it over via MsgUpdateParams.
//
// Decision 2026-11-06: uptick1r3dfm4lqs9eucv5s3mye98s80luaz3h53t9mcd (the ops
// account, same holder as test01's mykey authority).
var v1BaselineAuthority = "uptick1r3dfm4lqs9eucv5s3mye98s80luaz3h53t9mcd"

// v1BaselinePauseAuthority is written into params.PauseAuthority at the
// upgrade height. The kill switch (MsgPause) is a one-way fuse: it can only
// set pause bits, never clear them, and cannot touch params -- so pointing it
// at the same multisig is safe, and an on-call EOA can be separated later via
// MsgUpdateParams without touching the admin authority. Empty falls back to
// v1BaselineAuthority (decision 2026-11-06: same address, independent slot).
var v1BaselinePauseAuthority = "uptick1r3dfm4lqs9eucv5s3mye98s80luaz3h53t9mcd"

// SetBaselineAuthorityForTest overrides the release authority for handler
// tests and restores the previous value when the test finishes.
func SetBaselineAuthorityForTest(t interface{ Cleanup(func()) }, addr string) {
	prev := v1BaselineAuthority
	v1BaselineAuthority = addr
	t.Cleanup(func() { v1BaselineAuthority = prev })
}

// SetBaselinePauseAuthorityForTest overrides the release kill-switch authority
// for handler tests and restores the previous value when the test finishes.
func SetBaselinePauseAuthorityForTest(t interface{ Cleanup(func()) }, addr string) {
	prev := v1BaselinePauseAuthority
	v1BaselinePauseAuthority = addr
	t.Cleanup(func() { v1BaselinePauseAuthority = prev })
}

// applyV1BaselineParams rewrites the just-initialized x/appmining params into
// the production baseline. The denom is read from the live staking bond denom
// rather than hardcoded: one binary serves mainnet (auptick) and the origin
// testnet (auoc), and DenomAllowed gates pool opening, funding, and epoch
// budgets off RewardDenomAllowlist -- leaving the InitGenesis default
// ("auptick" everywhere) would brick the V1 fund path on any chain whose
// native denom differs.
//
// Called inside the migrationsAppliedKey-protected region: a replayed plan
// skips it, and a crash before the marker re-runs it with identical values
// (all writes here are idempotent sets).
func applyV1BaselineParams(ctx sdk.Context, box upgrades.Toolbox) error {
	if v1BaselineAuthority == "" {
		return fmt.Errorf("v1BaselineAuthority is empty: set the ops multisig address before building a release binary")
	}
	denom, err := box.StakingKeeper.BondDenom(ctx)
	if err != nil {
		return fmt.Errorf("read staking bond denom: %w", err)
	}
	if denom == "" {
		return fmt.Errorf("staking bond denom is empty; cannot derive the v1 baseline denom")
	}

	params := box.AppminingKeeper.GetParams(ctx)
	params.Authority = v1BaselineAuthority
	pauseAuthority := v1BaselinePauseAuthority
	if pauseAuthority == "" {
		// Same kill-switch holder as the admin authority (2026-11-06 decision);
		// a dedicated on-call address can be configured or handed over later
		// via MsgUpdateParams.
		pauseAuthority = v1BaselineAuthority
	}
	params.PauseAuthority = pauseAuthority
	params.ChallengeBondDenom = denom
	params.ChallengeBondAmount = v1BaselineChallengeBond
	params.ChallengeDepositDenom = denom
	params.ChallengeDepositAmount = v1BaselineChallengeDeposit
	params.RewardDenomAllowlist = []string{denom}
	params.EntitlementEnableMinFunded = v1BaselineMinFunded
	params.EntitlementAllowedRuleVersions = append([]string(nil), v1BaselineRuleVersions...)
	params.EntitlementAllowedAssessmentVersions = append([]string(nil), v1BaselineAssessmentVersions...)
	params.CredentialRequired = true
	params.ChallengeOpenAccess = appminingtypes.ChallengeOpenAccessDeposit

	// SetParams validates again internally; the explicit check first keeps the
	// failure message attributable to this step rather than a generic store error.
	if err := params.Validate(); err != nil {
		return fmt.Errorf("baseline params invalid: %w", err)
	}
	if err := box.AppminingKeeper.SetParams(ctx, params); err != nil {
		return err
	}

	ctx.Logger().Info("v1 baseline params applied",
		"upgrade", upgradeName,
		"denom", denom,
		"min_funded", v1BaselineMinFunded,
		"challenge_deposit", v1BaselineChallengeDeposit,
		"authority", params.Authority,
		"pause_authority", params.PauseAuthority,
	)
	return nil
}
