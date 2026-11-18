package types

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	DefaultChallengeWindowBlocks = 1500
	DefaultChallengeBondDenom    = "auptick"
	DefaultChallengeBondAmount   = "1000000"
	DefaultGasCreditShareBps     = 0
	DefaultUGEProfile            = "uge.pav"
	DefaultUGEProfileVersion     = "pav-0.1"
	DefaultMaxGasPayers          = 16
)

// Production defaults for V1 timing params (workorder 13 §3). Zero on a Params
// field means "use this default" via Effective* helpers so legacy state that
// pre-dates fields 19–22 keeps the same behaviour without a forced migration.
const (
	DefaultChallengeWindowSeconds      = 604800 // 7d
	DefaultMaxFreezeSeconds            = 259200 // 72h
	DefaultMaxWireProofValiditySeconds = 604800 // 7d
	DefaultNonceRetentionSeconds       = 691200 // 8d = validity + 1d grace
	// DefaultChallengeResolutionGraceSeconds is how long a reviewer keeps the
	// exclusive right to decide before anyone may settle an OPEN challenge as
	// EXPIRED (24h after the challenge's own resolution window elapses).
	DefaultChallengeResolutionGraceSeconds = 86400 // 24h
)

// Governance bounds for the V1 timing params. Authority may tune within these
// ranges via MsgUpdateParams; values outside fail Validate.
//
// Challenge window and wire-proof validity allow a 60s floor so short-lived
// testnets can exercise expiry paths (600s window, or a proof that dies within
// a minute) without a binary rebuild. Shrinking wire validity is additionally
// gated by the CLI (--allow-shorten-wire-proof-validity) and emits an event,
// so an accidental 60s on mainnet is visible rather than silent. Freeze and
// nonce retention keep a 1h floor: collapsing those to seconds has no test
// value and only risks stranding in-flight state.
const (
	MinChallengeWindowSeconds        = 60      // 1m (origin e2e / dry-run)
	MinWireProofValiditySeconds      = 60      // 1m (expiry path testing)
	MinTimingSeconds                 = 3600    // 1h (freeze / nonce)
	MaxChallengeWindowSecondsBound   = 2592000 // 30d
	MaxFreezeSecondsBound            = 604800  // 7d
	MaxWireProofValiditySecondsBound = 2592000 // 30d
	MaxNonceRetentionSecondsBound    = 5184000 // 60d
	// Resolution grace keeps the same 60s floor: testnets need to reach the
	// timeout path within a block or two of the window closing.
	MinChallengeResolutionGraceSeconds = 60
	MaxChallengeResolutionGraceSeconds = 2592000 // 30d
)

// DefaultParams returns production-safe defaults.
// pause_authority defaults to the same address as authority (Q-06 P0).
// Legacy attestation/GasCredit/Finalize writes are paused by default (S-2 /
// V1-first private chain); entitlement flow remains enabled at the pause bit
// level and is gated separately by entitlement_enable_min_funded.
func DefaultParams() Params {
	auth := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	ps := &PauseState{}
	PauseLegacyAttestationFlow(ps)
	return Params{
		Authority:             auth,
		PauseAuthority:        auth,
		ChallengeWindowBlocks: DefaultChallengeWindowBlocks,
		ChallengeBondDenom:    DefaultChallengeBondDenom,
		ChallengeBondAmount:   DefaultChallengeBondAmount,
		GasCreditShareBps:     DefaultGasCreditShareBps,
		PauseState:            ps,
		AllowedProfiles: []AttestationProfile{
			{Profile: DefaultUGEProfile, Version: DefaultUGEProfileVersion, Active: true},
		},
		RewardDenomAllowlist:        []string{DefaultChallengeBondDenom},
		MaxGasPayers:                DefaultMaxGasPayers,
		AllowManualGasCreditConsume: false,
		EntitlementEnableMinFunded:  "", // V1 disabled until configured + funded
		// EntitlementAllowedRuleVersions: nil → default-deny; every
		// MsgApproveEntitlement is rejected until governance allows rule
		// versions explicitly (workorder 13 §3 "规则版本允许").
		EntitlementAllowedRuleVersions: nil,
		// T1a publisher-binding placeholders (T1b/T1c activate behavior).
		CredentialRequired:          false,
		ChallengeOpenAccess:         ChallengeOpenAccessOpen,
		ChallengeDepositAmount:      "",
		ChallengeDepositDenom:       "",
		ChallengeWindowSeconds:      DefaultChallengeWindowSeconds,
		MaxFreezeSeconds:            DefaultMaxFreezeSeconds,
		MaxWireProofValiditySeconds: DefaultMaxWireProofValiditySeconds,
		NonceRetentionSeconds:       DefaultNonceRetentionSeconds,
		// Field 23 stays 0 on purpose: InitGenesis of x/appmining runs inside the
		// already-executed v0.5.1 upgrade plan, so writing a non-zero value here
		// would change the params store bytes at that height and break genesis
		// replay. 0 resolves to DefaultChallengeResolutionGraceSeconds via
		// EffectiveChallengeResolutionGraceSeconds.
	}
}

// Challenge open-access modes (UGE publisher-binding R4 / T1c).
const (
	ChallengeOpenAccessNone    = "none"
	ChallengeOpenAccessOpen    = "open"
	ChallengeOpenAccessDeposit = "deposit"
)

// EffectiveChallengeOpenAccess returns the configured mode (empty → open).
func (p Params) EffectiveChallengeOpenAccess() string {
	if p.ChallengeOpenAccess == "" {
		return ChallengeOpenAccessOpen
	}
	return p.ChallengeOpenAccess
}

// EffectiveMaxGasPayers returns the configured cap (default 16 when unset).
func (p Params) EffectiveMaxGasPayers() uint32 {
	if p.MaxGasPayers == 0 {
		return DefaultMaxGasPayers
	}
	return p.MaxGasPayers
}

// EffectiveChallengeWindowSeconds returns the V1 challenge window (0 → default).
func (p Params) EffectiveChallengeWindowSeconds() int64 {
	return timingSecondsAsInt64(p.ChallengeWindowSeconds, DefaultChallengeWindowSeconds)
}

// EffectiveMaxFreezeSeconds returns the freeze cap (0 → default).
func (p Params) EffectiveMaxFreezeSeconds() int64 {
	return timingSecondsAsInt64(p.MaxFreezeSeconds, DefaultMaxFreezeSeconds)
}

// EffectiveMaxWireProofValiditySeconds returns the wire-proof expiry cap (0 → default).
func (p Params) EffectiveMaxWireProofValiditySeconds() int64 {
	return timingSecondsAsInt64(p.MaxWireProofValiditySeconds, DefaultMaxWireProofValiditySeconds)
}

// EffectiveNonceRetentionSeconds returns the auth-nonce retention window (0 → default).
func (p Params) EffectiveNonceRetentionSeconds() int64 {
	return timingSecondsAsInt64(p.NonceRetentionSeconds, DefaultNonceRetentionSeconds)
}

// EffectiveChallengeResolutionGraceSeconds returns the reviewer grace period
// after a challenge's resolution window before anyone may settle it as EXPIRED
// (0 → default). This is the only knob that makes the timeout path reachable:
// without it an OPEN challenge blocks approval forever.
func (p Params) EffectiveChallengeResolutionGraceSeconds() int64 {
	return timingSecondsAsInt64(p.ChallengeResolutionGraceSeconds, DefaultChallengeResolutionGraceSeconds)
}

// timingSecondsAsInt64 maps a uint64 Params timing field to int64 for wall-clock
// math. Validate() keeps production values well below MaxInt64; the clamp is a
// belt-and-suspenders guard if Effective* is called on unvalidated Params.
func timingSecondsAsInt64(stored, defaultVal uint64) int64 {
	v := stored
	if v == 0 {
		v = defaultVal
	}
	if v > uint64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(v)
}

// RuleVersionAllowed reports whether the candidate rule_version may be
// approved under the current params. Empty allowlist denies all (§3).
func (p Params) RuleVersionAllowed(v string) bool {
	for _, allowed := range p.EntitlementAllowedRuleVersions {
		if allowed == v {
			return true
		}
	}
	return false
}

// Validate performs params validation.
func (p Params) Validate() error {
	if err := ValidateAddress(p.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if err := ValidateAddress(p.PauseAuthority); err != nil {
		return fmt.Errorf("invalid pause_authority: %w", err)
	}
	if p.ChallengeWindowBlocks == 0 {
		return fmt.Errorf("challenge_window_blocks must be > 0")
	}
	if err := sdk.ValidateDenom(p.ChallengeBondDenom); err != nil {
		return fmt.Errorf("invalid challenge_bond_denom: %w", err)
	}
	if err := ValidateAmountString(p.ChallengeBondAmount); err != nil {
		return fmt.Errorf("invalid challenge_bond_amount: %w", err)
	}
	if IsZeroAmount(p.ChallengeBondAmount) {
		return fmt.Errorf("challenge_bond_amount must be > 0")
	}
	if p.GasCreditShareBps > 10000 {
		return fmt.Errorf("gas_credit_share_bps must be <= 10000")
	}
	if p.PauseState == nil {
		return fmt.Errorf("pause_state must not be nil")
	}
	if len(p.AllowedProfiles) == 0 {
		return fmt.Errorf("allowed_profiles must not be empty")
	}
	seen := make(map[string]struct{}, len(p.AllowedProfiles))
	for _, ap := range p.AllowedProfiles {
		if ap.Profile == "" || ap.Version == "" {
			return fmt.Errorf("allowed profile fields must be non-empty")
		}
		key := ap.Profile + "/" + ap.Version
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate allowed profile %s", key)
		}
		seen[key] = struct{}{}
	}
	if len(p.RewardDenomAllowlist) == 0 {
		return fmt.Errorf("reward_denom_allowlist must not be empty")
	}
	allow := make(map[string]struct{}, len(p.RewardDenomAllowlist))
	for _, d := range p.RewardDenomAllowlist {
		if err := sdk.ValidateDenom(d); err != nil {
			return fmt.Errorf("invalid reward_denom_allowlist entry %q: %w", d, err)
		}
		allow[d] = struct{}{}
	}
	if _, ok := allow[p.ChallengeBondDenom]; !ok {
		return fmt.Errorf("challenge_bond_denom %q must be in reward_denom_allowlist", p.ChallengeBondDenom)
	}
	if p.EntitlementEnableMinFunded != "" {
		if err := ValidateAmountString(p.EntitlementEnableMinFunded); err != nil {
			return fmt.Errorf("invalid entitlement_enable_min_funded: %w", err)
		}
	}
	seenRV := make(map[string]struct{}, len(p.EntitlementAllowedRuleVersions))
	for _, rv := range p.EntitlementAllowedRuleVersions {
		if err := ValidateWireID(rv); err != nil {
			return fmt.Errorf("invalid entitlement_allowed_rule_versions entry %q: %w", rv, err)
		}
		if _, ok := seenRV[rv]; ok {
			return fmt.Errorf("duplicate entitlement_allowed_rule_versions entry %q", rv)
		}
		seenRV[rv] = struct{}{}
	}
	mode := p.EffectiveChallengeOpenAccess()
	switch mode {
	case ChallengeOpenAccessOpen, ChallengeOpenAccessNone, ChallengeOpenAccessDeposit:
	default:
		return fmt.Errorf("invalid challenge_open_access %q (want none|open|deposit)", p.ChallengeOpenAccess)
	}
	if p.ChallengeDepositAmount != "" {
		if err := ValidateAmountString(p.ChallengeDepositAmount); err != nil {
			return fmt.Errorf("invalid challenge_deposit_amount: %w", err)
		}
	}
	if p.ChallengeDepositDenom != "" {
		if err := sdk.ValidateDenom(p.ChallengeDepositDenom); err != nil {
			return fmt.Errorf("invalid challenge_deposit_denom: %w", err)
		}
		if !DenomAllowed(p, p.ChallengeDepositDenom) {
			return fmt.Errorf("challenge_deposit_denom %q must be in reward_denom_allowlist", p.ChallengeDepositDenom)
		}
	}
	if mode == ChallengeOpenAccessDeposit {
		if p.ChallengeDepositAmount == "" || IsZeroAmount(p.ChallengeDepositAmount) {
			return fmt.Errorf("challenge_deposit_amount must be > 0 when challenge_open_access=deposit")
		}
		if p.ChallengeDepositDenom == "" {
			return fmt.Errorf("challenge_deposit_denom required when challenge_open_access=deposit")
		}
	}
	seenAV := make(map[string]struct{}, len(p.EntitlementAllowedAssessmentVersions))
	for _, av := range p.EntitlementAllowedAssessmentVersions {
		if err := ValidateWireID(av); err != nil {
			return fmt.Errorf("invalid entitlement_allowed_assessment_versions entry %q: %w", av, err)
		}
		if _, ok := seenAV[av]; ok {
			return fmt.Errorf("duplicate entitlement_allowed_assessment_versions entry %q", av)
		}
		seenAV[av] = struct{}{}
	}
	if err := validateTimingSeconds("challenge_window_seconds", p.ChallengeWindowSeconds, MinChallengeWindowSeconds, MaxChallengeWindowSecondsBound); err != nil {
		return err
	}
	if err := validateTimingSeconds("max_freeze_seconds", p.MaxFreezeSeconds, MinTimingSeconds, MaxFreezeSecondsBound); err != nil {
		return err
	}
	if err := validateTimingSeconds("max_wire_proof_validity_seconds", p.MaxWireProofValiditySeconds, MinWireProofValiditySeconds, MaxWireProofValiditySecondsBound); err != nil {
		return err
	}
	if err := validateTimingSeconds("nonce_retention_seconds", p.NonceRetentionSeconds, MinTimingSeconds, MaxNonceRetentionSecondsBound); err != nil {
		return err
	}
	if err := validateTimingSeconds("challenge_resolution_grace_seconds", p.ChallengeResolutionGraceSeconds, MinChallengeResolutionGraceSeconds, MaxChallengeResolutionGraceSeconds); err != nil {
		return err
	}
	if p.EffectiveNonceRetentionSeconds() < p.EffectiveMaxWireProofValiditySeconds() {
		return fmt.Errorf("nonce_retention_seconds (%d) must be >= max_wire_proof_validity_seconds (%d)",
			p.EffectiveNonceRetentionSeconds(), p.EffectiveMaxWireProofValiditySeconds())
	}
	return nil
}

// validateTimingSeconds accepts 0 (Effective default) or a value in [min, max].
func validateTimingSeconds(name string, v uint64, min, max uint64) error {
	if v == 0 {
		return nil
	}
	if v < min || v > max {
		return fmt.Errorf("%s must be 0 (default) or in [%d, %d], got %d", name, min, max, v)
	}
	return nil
}

// PauseLegacyAttestationFlow sets the legacy attestation-line pause bits
// (attestation, challenge, settlement, gas credit). Registration stays open
// (control plane; V1 setup depends on it) and the V1 entitlement pause bit is
// independent (Q-06) — neither is touched here.
func PauseLegacyAttestationFlow(ps *PauseState) {
	ps.AttestationPaused = true
	ps.ChallengePaused = true
	ps.SettlementPaused = true
	ps.GasCreditPaused = true
}
