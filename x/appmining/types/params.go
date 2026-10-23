package types

import (
	"fmt"

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

// V1 entitlement timing constants (workorder 13 §3): 7-day challenge window,
// single freeze capped at 72h.
const (
	ChallengeWindowSeconds = 604800
	MaxFreezeSeconds       = 259200
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
	}
}

// EffectiveMaxGasPayers returns the configured cap (default 16 when unset).
func (p Params) EffectiveMaxGasPayers() uint32 {
	if p.MaxGasPayers == 0 {
		return DefaultMaxGasPayers
	}
	return p.MaxGasPayers
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
