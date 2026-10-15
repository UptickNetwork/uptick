package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgRegisterApplication{}
	_ sdk.Msg = &MsgSubmitAttestation{}
	_ sdk.Msg = &MsgChallengeAttestation{}
	_ sdk.Msg = &MsgResolveChallenge{}
	_ sdk.Msg = &MsgFinalizeAttestation{}
	_ sdk.Msg = &MsgGrantGasPayer{}
	_ sdk.Msg = &MsgRevokeGasPayer{}
	_ sdk.Msg = &MsgConsumeGasCredit{}
	_ sdk.Msg = &MsgSetEpochBudget{}
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgSetApplicationStatus{}
	_ sdk.Msg = &MsgPause{}
	_ sdk.Msg = &MsgFundRewardPool{}
)

func (m *MsgRegisterApplication) ValidateBasic() error {
	if err := ValidateAddress(m.Authority); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "authority")
	}
	if err := ValidateAppID(m.AppId); err != nil {
		return err
	}
	if err := ValidateAddress(m.Owner); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "owner")
	}
	if err := ValidateAddress(m.AttestationSubmitter); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "attestation_submitter")
	}
	return nil
}

func (m *MsgRegisterApplication) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgSubmitAttestation) ValidateBasic() error {
	if err := ValidateAddress(m.Submitter); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "submitter")
	}
	if err := validateAttestationFields(m.Attestation); err != nil {
		return err
	}
	if m.Attestation.Submitter != "" && m.Attestation.Submitter != m.Submitter {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "attestation.submitter must match msg.submitter")
	}
	return nil
}

func (m *MsgSubmitAttestation) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Submitter)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgChallengeAttestation) ValidateBasic() error {
	if err := ValidateAddress(m.Challenger); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "challenger")
	}
	if err := ValidateAppID(m.AppId); err != nil {
		return err
	}
	if err := ValidateHash(m.EvidenceHash); err != nil {
		return err
	}
	return nil
}

func (m *MsgChallengeAttestation) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Challenger)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgResolveChallenge) ValidateBasic() error {
	if err := ValidateAddress(m.Authority); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "authority")
	}
	if m.ChallengeId == 0 {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "challenge_id must be > 0")
	}
	if err := ValidateHash(m.ResolutionHash); err != nil {
		return err
	}
	return nil
}

func (m *MsgResolveChallenge) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgFinalizeAttestation) ValidateBasic() error {
	if err := ValidateAddress(m.Caller); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "caller")
	}
	if err := ValidateAppID(m.AppId); err != nil {
		return err
	}
	return nil
}

func (m *MsgFinalizeAttestation) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Caller)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgGrantGasPayer) ValidateBasic() error {
	if err := ValidateAddress(m.Owner); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "owner")
	}
	if err := ValidateAppID(m.AppId); err != nil {
		return err
	}
	if err := ValidateAddress(m.GasPayer); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "gas_payer")
	}
	return nil
}

func (m *MsgGrantGasPayer) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Owner)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgRevokeGasPayer) ValidateBasic() error {
	if err := ValidateAddress(m.Owner); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "owner")
	}
	if err := ValidateAppID(m.AppId); err != nil {
		return err
	}
	if err := ValidateAddress(m.GasPayer); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "gas_payer")
	}
	return nil
}

func (m *MsgRevokeGasPayer) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Owner)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgConsumeGasCredit) ValidateBasic() error {
	if err := ValidateAddress(m.GasPayer); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "gas_payer")
	}
	if err := ValidateAppID(m.AppId); err != nil {
		return err
	}
	if err := ValidateAmountString(m.Amount); err != nil {
		return err
	}
	if IsZeroAmount(m.Amount) {
		return ErrInvalidAmount
	}
	if len(m.SourceTxHash) != SourceTxHashLength {
		return ErrInvalidHash
	}
	return nil
}

func (m *MsgConsumeGasCredit) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.GasPayer)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgSetEpochBudget) ValidateBasic() error {
	if err := ValidateAddress(m.Authority); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "authority")
	}
	if err := sdk.ValidateDenom(m.Budget.Denom); err != nil {
		return ErrInvalidDenom
	}
	if err := ValidateAmountString(m.Budget.Limit); err != nil {
		return err
	}
	if m.Budget.Spent == "" {
		m.Budget.Spent = "0"
	}
	if err := ValidateAmountString(m.Budget.Spent); err != nil {
		return err
	}
	return nil
}

func (m *MsgSetEpochBudget) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgUpdateParams) ValidateBasic() error {
	if err := ValidateAddress(m.Authority); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "authority")
	}
	return m.Params.Validate()
}

func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgSetApplicationStatus) ValidateBasic() error {
	if err := ValidateAddress(m.Authority); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "authority")
	}
	if err := ValidateAppID(m.AppId); err != nil {
		return err
	}
	if m.Status == ApplicationStatusUnspecified {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "status unspecified")
	}
	return nil
}

func (m *MsgSetApplicationStatus) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgPause) ValidateBasic() error {
	if err := ValidateAddress(m.PauseAuthority); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "pause_authority")
	}
	if !PauseBitsHasAnyTrue(m.PauseBits) {
		return ErrPauseOnlyTrue
	}
	return nil
}

func (m *MsgPause) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.PauseAuthority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgFundRewardPool) ValidateBasic() error {
	if err := ValidateAddress(m.Authority); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "authority")
	}
	if err := sdk.ValidateDenom(m.Denom); err != nil {
		return sdkerrors.Wrap(ErrInvalidDenom, err.Error())
	}
	if err := ValidateAmountString(m.Amount); err != nil {
		return err
	}
	amt, err := ParseAmount(m.Amount)
	if err != nil {
		return err
	}
	if !amt.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidAmount, "fund amount must be > 0")
	}
	return nil
}

func (m *MsgFundRewardPool) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}
