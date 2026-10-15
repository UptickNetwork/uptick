package types

import "fmt"

// DefaultGenesisState returns an empty genesis with default params.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                DefaultParams(),
		Applications:          []Application{},
		EpochBudgets:          []EpochBudget{},
		Attestations:          []AttestationCommitment{},
		Challenges:            []Challenge{},
		Settlements:           []Settlement{},
		GasCreditAccounts:     []GasCreditAccount{},
		GasCreditConsumptions: []GasCreditConsumption{},
		NextChallengeId:       1,
	}
}

// ValidateGenesis validates genesis state.
func ValidateGenesis(gs GenesisState) error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if gs.NextChallengeId == 0 {
		return fmt.Errorf("next_challenge_id must be >= 1")
	}

	appSeen := make(map[string]struct{}, len(gs.Applications))
	payerSeen := make(map[string]struct{})
	for _, app := range gs.Applications {
		if err := ValidateAppID(app.AppId); err != nil {
			return err
		}
		key := string(app.AppId)
		if _, ok := appSeen[key]; ok {
			return fmt.Errorf("duplicate application")
		}
		appSeen[key] = struct{}{}
		if err := ValidateAddress(app.Owner); err != nil {
			return fmt.Errorf("invalid owner: %w", err)
		}
		if err := ValidateAddress(app.AttestationSubmitter); err != nil {
			return fmt.Errorf("invalid attestation_submitter: %w", err)
		}
		for _, p := range app.GasPayers {
			if err := ValidateAddress(p); err != nil {
				return fmt.Errorf("invalid gas_payer: %w", err)
			}
			if _, ok := payerSeen[p]; ok {
				return fmt.Errorf("gas payer %s bound to multiple applications", p)
			}
			payerSeen[p] = struct{}{}
		}
		if app.Status == ApplicationStatusUnspecified {
			return fmt.Errorf("application status unspecified")
		}
	}

	epochSeen := make(map[uint64]struct{}, len(gs.EpochBudgets))
	for _, b := range gs.EpochBudgets {
		if _, ok := epochSeen[b.Epoch]; ok {
			return fmt.Errorf("duplicate epoch budget %d", b.Epoch)
		}
		epochSeen[b.Epoch] = struct{}{}
		if !DenomAllowed(gs.Params, b.Denom) {
			return fmt.Errorf("epoch %d denom %q not in reward_denom_allowlist", b.Epoch, b.Denom)
		}
		if err := ValidateAmountString(b.Limit); err != nil {
			return err
		}
		if err := ValidateAmountString(b.Spent); err != nil {
			return err
		}
		cmp, err := CompareAmountStrings(b.Spent, b.Limit)
		if err != nil {
			return err
		}
		if cmp > 0 {
			return fmt.Errorf("epoch %d spent > limit", b.Epoch)
		}
	}

	attSeen := make(map[string]struct{}, len(gs.Attestations))
	for _, a := range gs.Attestations {
		if err := validateAttestationFields(a); err != nil {
			return err
		}
		key := string(a.AppId) + "/" + fmt.Sprint(a.Epoch)
		if _, ok := attSeen[key]; ok {
			return fmt.Errorf("duplicate attestation")
		}
		attSeen[key] = struct{}{}
	}

	chalSeen := make(map[uint64]struct{}, len(gs.Challenges))
	for _, c := range gs.Challenges {
		if _, ok := chalSeen[c.ChallengeId]; ok {
			return fmt.Errorf("duplicate challenge id")
		}
		chalSeen[c.ChallengeId] = struct{}{}
		if c.ChallengeId >= gs.NextChallengeId {
			return fmt.Errorf("challenge id >= next_challenge_id")
		}
		if err := ValidateAppID(c.AppId); err != nil {
			return err
		}
		if err := ValidateHash(c.EvidenceHash); err != nil {
			return err
		}
		if err := ValidateAddress(c.Challenger); err != nil {
			return err
		}
		if err := ValidateAmountString(c.BondAmount); err != nil {
			return err
		}
	}

	for _, s := range gs.Settlements {
		if err := ValidateAppID(s.AppId); err != nil {
			return err
		}
		if err := ValidateHash(s.AttestationHash); err != nil {
			return err
		}
		if err := ValidateAmountString(s.RewardReleased); err != nil {
			return err
		}
		if err := ValidateAmountString(s.GasCreditIssued); err != nil {
			return err
		}
	}

	for _, a := range gs.GasCreditAccounts {
		if err := ValidateAppID(a.AppId); err != nil {
			return err
		}
		for _, field := range []string{a.Balance, a.TotalIssued, a.TotalConsumed} {
			if err := ValidateAmountString(field); err != nil {
				return err
			}
		}
	}

	consSeen := make(map[string]struct{}, len(gs.GasCreditConsumptions))
	for _, c := range gs.GasCreditConsumptions {
		if err := ValidateAppID(c.AppId); err != nil {
			return err
		}
		if len(c.SourceTxHash) != SourceTxHashLength {
			return ErrInvalidHash
		}
		key := string(c.AppId) + "/" + string(c.SourceTxHash)
		if _, ok := consSeen[key]; ok {
			return fmt.Errorf("duplicate gas credit consumption")
		}
		consSeen[key] = struct{}{}
		if err := ValidateAmountString(c.Amount); err != nil {
			return err
		}
		if err := ValidateAddress(c.GasPayer); err != nil {
			return err
		}
	}
	return nil
}

func validateAttestationFields(a AttestationCommitment) error {
	if err := ValidateAppID(a.AppId); err != nil {
		return err
	}
	if err := ValidateHash(a.AttestationHash); err != nil {
		return err
	}
	if err := ValidateHash(a.EvidenceRoot); err != nil {
		return err
	}
	if err := ValidateHash(a.ResultCommitment); err != nil {
		return err
	}
	if err := ValidateAmountString(a.RewardRequested); err != nil {
		return err
	}
	if err := ValidateAmountString(a.GasCreditRequested); err != nil {
		return err
	}
	if err := ValidateAddress(a.Submitter); err != nil {
		return err
	}
	if a.Profile == "" || a.ProfileVersion == "" {
		return ErrInvalidCommitment
	}
	return nil
}
