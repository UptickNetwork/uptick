package types

import "fmt"

// CanSubmitAttestation enforces uniqueness and non-overwrite (BLOCK-1 fix).
// existing may be nil when no record exists.
func CanSubmitAttestation(app Application, existing *AttestationCommitment, epoch uint64) error {
	if app.Status != ApplicationStatusActive {
		return ErrApplicationNotActive
	}
	if existing == nil {
		return nil
	}
	// Any existing record for (app, epoch) blocks resubmit — including challenged/finalized.
	return ErrAttestationDuplicate
}

// CanChallenge reports whether an attestation can accept a new challenge.
func CanChallenge(att AttestationCommitment, height int64) error {
	switch att.Status {
	case AttestationStatusChallengeWindow, AttestationStatusChallenged:
		// ok
	default:
		return ErrAttestationFinalized
	}
	if height > att.ChallengeDeadlineHeight {
		return ErrChallengeWindow
	}
	return nil
}

// ApplyChallenge transitions attestation to Challenged.
func ApplyChallenge(att AttestationCommitment) AttestationCommitment {
	att.Status = AttestationStatusChallenged
	return att
}

// ApplyResolveChallenge updates challenge and optionally attestation on uphold.
func ApplyResolveChallenge(
	att AttestationCommitment,
	chal Challenge,
	uphold bool,
	resolutionHash []byte,
	height int64,
) (AttestationCommitment, Challenge, error) {
	if chal.Status != ChallengeStatusOpen {
		return att, chal, ErrChallengeNotOpen
	}
	chal.ResolutionHash = resolutionHash
	chal.ResolvedHeight = height
	if uphold {
		chal.Status = ChallengeStatusUpheld
		att.Status = AttestationStatusRejected
	} else {
		chal.Status = ChallengeStatusRejected
		// Restore challenge window status if no other open challenges remain —
		// caller must decide based on open-challenge index.
		if att.Status == AttestationStatusChallenged {
			att.Status = AttestationStatusChallengeWindow
		}
	}
	return att, chal, nil
}

// CanFinalize checks attestation settlement eligibility (without budget/pool).
func CanFinalize(att AttestationCommitment, height int64, hasOpenChallenge bool) error {
	switch att.Status {
	case AttestationStatusFinalized, AttestationStatusRejected:
		return ErrAttestationFinalized
	case AttestationStatusUnspecified:
		return ErrAttestationNotFound
	}
	if height <= att.ChallengeDeadlineHeight {
		return ErrChallengeWindow
	}
	if hasOpenChallenge {
		return ErrOpenChallenge
	}
	if att.Status == AttestationStatusChallenged {
		// Challenged with no open challenges after all rejected → treat as window.
		// If still marked challenged but no open, allow finalize.
	}
	return nil
}

// ComputeSettlementAmounts validates budget and returns reward/credit split of
// reward_requested. gas_credit_requested is informational; issued credit comes
// from gas_credit_share_bps of reward_requested (chain-enforced D-09).
func ComputeSettlementAmounts(
	att AttestationCommitment,
	budget EpochBudget,
	shareBps uint32,
) (rewardReleased, creditIssued, newSpent string, err error) {
	if budget.Closed {
		return "", "", "", ErrInvalidEpoch
	}
	if IsZeroAmount(budget.Limit) || IsZeroAmount(att.RewardRequested) {
		return "", "", "", ErrBudgetUnavailable
	}
	requested, err := ParseAmount(att.RewardRequested)
	if err != nil {
		return "", "", "", err
	}
	spent, err := ParseAmount(budget.Spent)
	if err != nil {
		return "", "", "", err
	}
	limit, err := ParseAmount(budget.Limit)
	if err != nil {
		return "", "", "", err
	}
	remaining := limit.Sub(spent)
	if requested.GT(remaining) {
		return "", "", "", ErrBudgetExceeded
	}
	reward, credit, err := SplitRewardAndCredit(att.RewardRequested, shareBps)
	if err != nil {
		return "", "", "", err
	}
	newSpent = AmountString(spent.Add(requested))
	return reward, credit, newSpent, nil
}

// BuildSettlement constructs the unique settlement record.
func BuildSettlement(att AttestationCommitment, reward, credit string, height int64) Settlement {
	return Settlement{
		AppId:           att.AppId,
		Epoch:           att.Epoch,
		Profile:         att.Profile,
		ProfileVersion:  att.ProfileVersion,
		AttestationHash: att.AttestationHash,
		RewardReleased:  reward,
		GasCreditIssued: credit,
		FinalizedHeight: height,
	}
}

// ApplyFinalize marks attestation finalized.
func ApplyFinalize(att AttestationCommitment) AttestationCommitment {
	att.Status = AttestationStatusFinalized
	return att
}

// CanConsumeCredit validates consume preconditions on amounts (not auth).
func CanConsumeCredit(balance, amount string) (newBalance string, err error) {
	if err := ValidateAmountString(amount); err != nil {
		return "", err
	}
	if IsZeroAmount(amount) {
		return "", fmt.Errorf("%w: zero consume", ErrInvalidAmount)
	}
	return SubAmountStrings(balance, amount)
}
