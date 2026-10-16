package types

import "fmt"

// CanSetApplicationStatus enforces permanent Revoked and valid transitions.
func CanSetApplicationStatus(from, to ApplicationStatus) error {
	if to == ApplicationStatusUnspecified {
		return ErrInvalidStatusTransition
	}
	if from == to {
		return nil
	}
	if from == ApplicationStatusRevoked {
		return ErrApplicationRevoked
	}
	switch to {
	case ApplicationStatusActive, ApplicationStatusSuspended, ApplicationStatusRevoked:
		return nil
	default:
		return ErrInvalidStatusTransition
	}
}

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
		// Keep Challenged while other OPENs may remain; caller restores
		// CHALLENGE_WINDOW only after confirming HasOpenChallenge is false.
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

// AttestationBudgetCharge is the amount counted against epoch budget for an
// attestation: reward_requested, or gas_credit_requested when reward is zero.
func AttestationBudgetCharge(att AttestationCommitment) (string, error) {
	if err := ValidateAmountString(att.RewardRequested); err != nil {
		return "", err
	}
	if IsZeroAmount(att.RewardRequested) {
		if err := ValidateAmountString(att.GasCreditRequested); err != nil {
			return "", err
		}
		if IsZeroAmount(att.GasCreditRequested) {
			return "", ErrBudgetUnavailable
		}
		return att.GasCreditRequested, nil
	}
	return att.RewardRequested, nil
}

// AttestationCountsTowardBudget reports whether the attestation still holds a
// budget claim (not yet settled or rejected).
func AttestationCountsTowardBudget(status AttestationStatus) bool {
	switch status {
	case AttestationStatusFinalized, AttestationStatusRejected:
		return false
	default:
		return true
	}
}

// ComputeSettlementAmounts validates budget and returns reward/credit split.
//
// When reward_requested > 0: credit = share_bps of reward_requested (D-09);
// gas_credit_requested is informational. Budget spent increases by reward_requested.
//
// When reward_requested = 0 (Q-03 pure-credit): credit = gas_credit_requested
// (must be > 0); no bank reward. Budget spent increases by gas_credit_requested
// so epoch limits still cap credit issuance.
func ComputeSettlementAmounts(
	att AttestationCommitment,
	budget EpochBudget,
	shareBps uint32,
) (rewardReleased, creditIssued, newSpent string, err error) {
	if budget.Closed {
		return "", "", "", ErrInvalidEpoch
	}
	if IsZeroAmount(budget.Limit) {
		return "", "", "", ErrBudgetUnavailable
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

	if IsZeroAmount(att.RewardRequested) {
		if err := ValidateAmountString(att.GasCreditRequested); err != nil {
			return "", "", "", err
		}
		if IsZeroAmount(att.GasCreditRequested) {
			return "", "", "", ErrBudgetUnavailable
		}
		creditAmt, err := ParseAmount(att.GasCreditRequested)
		if err != nil {
			return "", "", "", err
		}
		if creditAmt.GT(remaining) {
			return "", "", "", ErrBudgetExceeded
		}
		return "0", att.GasCreditRequested, AmountString(spent.Add(creditAmt)), nil
	}

	requested, err := ParseAmount(att.RewardRequested)
	if err != nil {
		return "", "", "", err
	}
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
