package types

import (
	"strconv"

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
	_ sdk.Msg = &MsgSetAttestationSubmitter{}
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
	if err := ValidateOptionalCommitment(m.MetadataCommitment); err != nil {
		return err
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

func (m *MsgSetAttestationSubmitter) ValidateBasic() error {
	if err := ValidateAddress(m.Owner); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "owner")
	}
	if err := ValidateAppID(m.AppId); err != nil {
		return err
	}
	if err := ValidateAddress(m.AttestationSubmitter); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "attestation_submitter")
	}
	return nil
}

func (m *MsgSetAttestationSubmitter) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Owner)
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
	if err := ValidateAddress(m.Funder); err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidAddress, "funder")
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
	addr, err := sdk.AccAddressFromBech32(m.Funder)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

// --- V1 entitlement flow msgs (workorder 13 §3) ---

var (
	_ sdk.Msg = &MsgOpenEntitlementPool{}
	_ sdk.Msg = &MsgFundEntitlementPool{}
	_ sdk.Msg = &MsgSetEntitlementPoolPaused{}
	_ sdk.Msg = &MsgRegisterRoleKey{}
	_ sdk.Msg = &MsgSetRoleKeyStatus{}
	_ sdk.Msg = &MsgPublishCandidate{}
	_ sdk.Msg = &MsgOpenChallenge{}
	_ sdk.Msg = &MsgResolveEntitlementChallenge{}
	_ sdk.Msg = &MsgApproveEntitlement{}
	_ sdk.Msg = &MsgClaimEntitlement{}
	_ sdk.Msg = &MsgFreezeEntitlement{}
	_ sdk.Msg = &MsgRevokeEntitlement{}
)

func validateSigner(addr string) error {
	if _, err := sdk.AccAddressFromBech32(addr); err != nil {
		return sdkerrors.Wrap(ErrInvalidAddress, "signer must be valid bech32")
	}
	return nil
}

func (m *MsgOpenEntitlementPool) ValidateBasic() error {
	if err := validateSigner(m.Authority); err != nil {
		return err
	}
	if err := ValidateWireID(m.PoolId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "pool_id")
	}
	if err := sdk.ValidateDenom(m.Denom); err != nil {
		return sdkerrors.Wrap(ErrInvalidDenom, "denom")
	}
	return nil
}

func (m *MsgOpenEntitlementPool) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgFundEntitlementPool) ValidateBasic() error {
	if err := validateSigner(m.Funder); err != nil {
		return err
	}
	if err := ValidateWireID(m.PoolId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "pool_id")
	}
	if err := ValidateAmountString(m.Amount); err != nil {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount")
	}
	if IsZeroAmount(m.Amount) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount must be > 0")
	}
	return nil
}

func (m *MsgFundEntitlementPool) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Funder)
	return []sdk.AccAddress{addr}
}

func (m *MsgSetEntitlementPoolPaused) ValidateBasic() error {
	if err := validateSigner(m.Authority); err != nil {
		return err
	}
	if err := ValidateWireID(m.PoolId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "pool_id")
	}
	return nil
}

func (m *MsgSetEntitlementPoolPaused) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgRegisterRoleKey) ValidateBasic() error {
	if err := validateSigner(m.Authority); err != nil {
		return err
	}
	if m.Role == EntitlementRoleUnspecified {
		return sdkerrors.Wrap(ErrInvalidRole, "role unspecified")
	}
	if err := ValidateWireID(m.KeyId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "key_id")
	}
	if err := ValidateEd25519PubKey(m.Pubkey); err != nil {
		return sdkerrors.Wrap(ErrInvalidCommitment, "pubkey must be 32-byte Ed25519")
	}
	return nil
}

func (m *MsgRegisterRoleKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgSetRoleKeyStatus) ValidateBasic() error {
	if err := validateSigner(m.Authority); err != nil {
		return err
	}
	if m.Role == EntitlementRoleUnspecified {
		return sdkerrors.Wrap(ErrInvalidRole, "role unspecified")
	}
	if err := ValidateWireID(m.KeyId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "key_id")
	}
	if m.Status == RoleKeyStatusUnspecified {
		return sdkerrors.Wrap(ErrInvalidRoleKeyStatus, "status unspecified")
	}
	return nil
}

func (m *MsgSetRoleKeyStatus) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgPublishCandidate) ValidateBasic() error {
	if err := validateSigner(m.Submitter); err != nil {
		return err
	}
	if m.ChainId == "" {
		return sdkerrors.Wrap(ErrChainIDMismatch, "chain_id empty")
	}
	if err := ValidateWireID(m.PoolId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "pool_id")
	}
	if len(m.EntitlementId) != HashLength*2 || ValidateWireID(m.EntitlementId) != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "entitlement_id must be 64-hex")
	}
	if len(m.AppId) != AppIDLength*2 || ValidateWireID(m.AppId) != nil {
		return sdkerrors.Wrap(ErrInvalidAppID, "app_id must be 64-hex")
	}
	if err := ValidateAmountString(m.Amount); err != nil {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount")
	}
	if IsZeroAmount(m.Amount) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount must be > 0")
	}
	if err := sdk.ValidateDenom(m.Denom); err != nil {
		return sdkerrors.Wrap(ErrInvalidDenom, "denom")
	}
	if err := ValidateWireID(m.RuleVersion); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "rule_version")
	}
	if err := ValidateWireHash(m.EvidenceRoot); err != nil {
		return sdkerrors.Wrap(ErrInvalidHash, "evidence_root")
	}
	if m.GasEvidenceRoot != "" {
		if err := ValidateWireHash(m.GasEvidenceRoot); err != nil {
			return sdkerrors.Wrap(ErrInvalidHash, "gas_evidence_root")
		}
	}
	return validateWireProofShape(m.Proof)
}

func (m *MsgPublishCandidate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Submitter)
	return []sdk.AccAddress{addr}
}

func (m *MsgOpenChallenge) ValidateBasic() error {
	if err := validateSigner(m.Challenger); err != nil {
		return err
	}
	if err := ValidateWireID(m.EntitlementId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "entitlement_id")
	}
	if _, ok := EvidenceTypeWire(m.EvidenceType); !ok {
		return ErrInvalidEvidenceType
	}
	if err := ValidateWireHash(m.EvidenceDigest); err != nil {
		return sdkerrors.Wrap(ErrInvalidHash, "evidence_digest")
	}
	return nil
}

func (m *MsgOpenChallenge) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Challenger)
	return []sdk.AccAddress{addr}
}

func (m *MsgResolveEntitlementChallenge) ValidateBasic() error {
	if err := validateSigner(m.Submitter); err != nil {
		return err
	}
	if m.ChainId == "" {
		return sdkerrors.Wrap(ErrChainIDMismatch, "chain_id empty")
	}
	if err := ValidateWireID(m.ChallengeId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "challenge_id")
	}
	if err := ValidateWireID(m.PoolId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "pool_id")
	}
	if m.Outcome != "UPHELD" && m.Outcome != "DISMISSED" {
		return ErrInvalidOutcome
	}
	if err := ValidateWireHash(m.RationaleDigest); err != nil {
		return sdkerrors.Wrap(ErrInvalidHash, "rationale_digest")
	}
	return validateWireProofShape(m.Proof)
}

func (m *MsgResolveEntitlementChallenge) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Submitter)
	return []sdk.AccAddress{addr}
}

func (m *MsgApproveEntitlement) ValidateBasic() error {
	if err := validateSigner(m.Submitter); err != nil {
		return err
	}
	if m.ChainId == "" {
		return sdkerrors.Wrap(ErrChainIDMismatch, "chain_id empty")
	}
	if err := ValidateWireID(m.PoolId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "pool_id")
	}
	if len(m.EntitlementId) != HashLength*2 {
		return sdkerrors.Wrap(ErrInvalidWireID, "entitlement_id must be 64-hex")
	}
	if m.ChallengeResolutionHash != "" {
		if err := ValidateWireHash(m.ChallengeResolutionHash); err != nil {
			return sdkerrors.Wrap(ErrInvalidHash, "challenge_resolution_hash")
		}
	}
	return validateWireProofShape(m.Proof)
}

func (m *MsgApproveEntitlement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Submitter)
	return []sdk.AccAddress{addr}
}

func (m *MsgClaimEntitlement) ValidateBasic() error {
	if err := validateSigner(m.Owner); err != nil {
		return err
	}
	if len(m.EntitlementId) != HashLength*2 {
		return sdkerrors.Wrap(ErrInvalidWireID, "entitlement_id must be 64-hex")
	}
	return nil
}

func (m *MsgClaimEntitlement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}

func (m *MsgFreezeEntitlement) ValidateBasic() error {
	if err := validateSigner(m.Submitter); err != nil {
		return err
	}
	if m.ChainId == "" {
		return sdkerrors.Wrap(ErrChainIDMismatch, "chain_id empty")
	}
	if err := ValidateWireID(m.PoolId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "pool_id")
	}
	if len(m.EntitlementId) != HashLength*2 {
		return sdkerrors.Wrap(ErrInvalidWireID, "entitlement_id must be 64-hex")
	}
	if _, ok := EvidenceTypeWire(m.EvidenceType); !ok {
		return ErrInvalidEvidenceType
	}
	if err := ValidateWireHash(m.EvidenceDigest); err != nil {
		return sdkerrors.Wrap(ErrInvalidHash, "evidence_digest")
	}
	if _, err := strconv.ParseInt(m.FreezeUntil, 10, 64); err != nil {
		return sdkerrors.Wrap(ErrFreezeDuration, "freeze_until must be unix seconds")
	}
	return validateWireProofShape(m.Proof)
}

func (m *MsgFreezeEntitlement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Submitter)
	return []sdk.AccAddress{addr}
}

func (m *MsgRevokeEntitlement) ValidateBasic() error {
	if err := validateSigner(m.Submitter); err != nil {
		return err
	}
	if m.ChainId == "" {
		return sdkerrors.Wrap(ErrChainIDMismatch, "chain_id empty")
	}
	if err := ValidateWireID(m.PoolId); err != nil {
		return sdkerrors.Wrap(ErrInvalidWireID, "pool_id")
	}
	if len(m.EntitlementId) != HashLength*2 {
		return sdkerrors.Wrap(ErrInvalidWireID, "entitlement_id must be 64-hex")
	}
	if _, ok := EvidenceTypeWire(m.EvidenceType); !ok {
		return ErrInvalidEvidenceType
	}
	if err := ValidateWireHash(m.EvidenceDigest); err != nil {
		return sdkerrors.Wrap(ErrInvalidHash, "evidence_digest")
	}
	if err := ValidateWireHash(m.RationaleDigest); err != nil {
		return sdkerrors.Wrap(ErrInvalidHash, "rationale_digest")
	}
	if m.RiskProof.KeyId == m.AuditProof.KeyId {
		return ErrDualAuthRequired
	}
	if err := validateWireProofShape(m.RiskProof); err != nil {
		return err
	}
	return validateWireProofShape(m.AuditProof)
}

func (m *MsgRevokeEntitlement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Submitter)
	return []sdk.AccAddress{addr}
}

// validateWireProofShape checks the structural fields of a wire-v1 proof
// (key_id / nonce / expires_at present). Signature validity is checked by the
// keeper against the registered role key.
func validateWireProofShape(p WireProof) error {
	if p.KeyId == "" {
		return sdkerrors.Wrap(ErrAuthMalformed, "key_id empty")
	}
	if err := ValidateWireID(p.KeyId); err != nil {
		return sdkerrors.Wrap(ErrAuthMalformed, "key_id")
	}
	if p.Nonce == "" {
		return sdkerrors.Wrap(ErrAuthMalformed, "nonce empty")
	}
	if _, err := strconv.ParseInt(p.ExpiresAt, 10, 64); err != nil {
		return sdkerrors.Wrap(ErrAuthMalformed, "expires_at must be unix seconds")
	}
	if p.SignatureBase64 == "" {
		return sdkerrors.Wrap(ErrAuthMalformed, "signature empty")
	}
	return nil
}
