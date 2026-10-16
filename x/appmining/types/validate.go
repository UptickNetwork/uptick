package types

import (
	"encoding/base64"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateAppID requires exactly 32 bytes.
func ValidateAppID(appID []byte) error {
	if len(appID) != AppIDLength {
		return ErrInvalidAppID
	}
	return nil
}

// AppIDDisplay returns the canonical base64url-no-padding 43-char form of a
// 32-byte app_id for events (Q-02). Panics if length is wrong — callers must
// ValidateAppID first.
func AppIDDisplay(appID []byte) string {
	return base64.RawURLEncoding.EncodeToString(appID)
}

// ValidateHash requires exactly 32 bytes.
func ValidateHash(h []byte) error {
	if len(h) != HashLength {
		return ErrInvalidHash
	}
	return nil
}

// ValidateOptionalCommitment accepts empty or exactly 32 bytes.
func ValidateOptionalCommitment(h []byte) error {
	if len(h) == 0 {
		return nil
	}
	return ValidateHash(h)
}

// ValidateAddress checks bech32 account address. Returns ErrInvalidAddress
// (not ErrUnauthorized) so that malformed addresses are distinguishable from
// signer-identity failures in ValidateBasic paths.
func ValidateAddress(addr string) error {
	if _, err := sdk.AccAddressFromBech32(addr); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "malformed bech32 address %q: %v", addr, err)
	}
	return nil
}

// ProfileAllowed reports whether profile/version is in the active allowlist.
func ProfileAllowed(params Params, profile, version string) bool {
	for _, p := range params.AllowedProfiles {
		if p.Active && p.Profile == profile && p.Version == version {
			return true
		}
	}
	return false
}

// DenomAllowed reports whether denom may be used for epoch budgets.
func DenomAllowed(params Params, denom string) bool {
	if denom == "" {
		return false
	}
	for _, d := range params.RewardDenomAllowlist {
		if d == denom {
			return true
		}
	}
	return false
}

// PauseBitsHasAnyTrue reports whether any pause dimension is requested.
func PauseBitsHasAnyTrue(p PauseState) bool {
	return p.RegistrationPaused || p.AttestationPaused || p.ChallengePaused ||
		p.SettlementPaused || p.GasCreditPaused
}

// ApplyPauseOR merges true bits from req onto cur (pause-only-true).
func ApplyPauseOR(cur, req PauseState) PauseState {
	return PauseState{
		RegistrationPaused: cur.RegistrationPaused || req.RegistrationPaused,
		AttestationPaused:  cur.AttestationPaused || req.AttestationPaused,
		ChallengePaused:    cur.ChallengePaused || req.ChallengePaused,
		SettlementPaused:   cur.SettlementPaused || req.SettlementPaused,
		GasCreditPaused:    cur.GasCreditPaused || req.GasCreditPaused,
	}
}

// PauseClearsAny reports whether req would clear any currently-true bit on cur.
func PauseClearsAny(cur, req PauseState) bool {
	return (cur.RegistrationPaused && !req.RegistrationPaused) ||
		(cur.AttestationPaused && !req.AttestationPaused) ||
		(cur.ChallengePaused && !req.ChallengePaused) ||
		(cur.SettlementPaused && !req.SettlementPaused) ||
		(cur.GasCreditPaused && !req.GasCreditPaused)
}
