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
)

// DefaultParams returns production-safe defaults.
// pause_authority defaults to the same address as authority (Q-06 P0).
func DefaultParams() Params {
	auth := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	return Params{
		Authority:             auth,
		PauseAuthority:        auth,
		ChallengeWindowBlocks: DefaultChallengeWindowBlocks,
		ChallengeBondDenom:    DefaultChallengeBondDenom,
		ChallengeBondAmount:   DefaultChallengeBondAmount,
		GasCreditShareBps:     DefaultGasCreditShareBps,
		PauseState:            &PauseState{},
		AllowedProfiles: []AttestationProfile{
			{Profile: DefaultUGEProfile, Version: DefaultUGEProfileVersion, Active: true},
		},
		RewardDenomAllowlist: []string{DefaultChallengeBondDenom},
	}
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
	return nil
}
