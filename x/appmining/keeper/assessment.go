package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/UptickNetwork/uptick/x/appmining/types"
	"github.com/UptickNetwork/uptick/x/appmining/types/wirev1"
)

// verifyAssessmentCredential validates an AssessmentCredential against the
// published candidate fields and assessment-signer registry (T1b / R3).
// credential_id is permanently one-shot per pool: HasCredentialID remains true
// after candidate REJECTED (D-A final semantics; not a transitional T1 rule).
// Returns the credential_id on success.
func (k Keeper) verifyAssessmentCredential(
	ctx sdk.Context,
	params types.Params,
	msg *types.MsgPublishCandidate,
	cred types.AssessmentCredential,
	nowUnix int64,
) (string, error) {
	if !params.AssessmentVersionAllowed(cred.CredentialVersion) {
		return "", types.ErrAssessmentCredentialInvalid
	}
	// Candidate body must match credential claims.
	if cred.AppId != msg.AppId ||
		cred.PoolId != msg.PoolId ||
		cred.PavId != msg.PavId ||
		cred.EvidenceRoot != msg.EvidenceRoot ||
		cred.GasEvidenceRoot != msg.GasEvidenceRoot ||
		cred.Amount != msg.Amount ||
		cred.Denom != msg.Denom ||
		cred.GasUsed != msg.GasUsed ||
		cred.QualifiedTxCount != msg.QualifiedTxCount ||
		cred.RuleVersion != msg.RuleVersion {
		return "", types.ErrAssessmentCredentialInvalid
	}
	issued, err := strconv.ParseInt(cred.IssuedAt, 10, 64)
	if err != nil {
		return "", types.ErrAssessmentCredentialInvalid
	}
	expires, err := strconv.ParseInt(cred.ExpiresAt, 10, 64)
	if err != nil {
		return "", types.ErrAssessmentCredentialInvalid
	}
	if expires < issued || nowUnix > expires {
		return "", types.ErrAssessmentCredentialInvalid
	}
	record := cred.Record()
	if err := wirev1.RequireCredentialIDBound(record); err != nil {
		return "", types.ErrAssessmentCredentialInvalid
	}
	if k.HasCredentialID(ctx, cred.CredentialId) {
		return "", types.ErrAssessmentCredentialReuse
	}
	signer, found := k.GetAssessmentSigner(ctx, cred.SignerKeyId)
	if !found || signer.Status != types.RoleKeyStatusActive {
		return "", types.ErrAssessmentCredentialInvalid
	}
	if !wirev1.VerifyAssessment(record, cred.SignatureBase64, signer.Pubkey) {
		return "", types.ErrAssessmentCredentialInvalid
	}
	return cred.CredentialId, nil
}

// requirePublishCredential applies T1b credential policy:
//   - app-bound publisher key → credential mandatory
//   - global key → optional unless params.credential_required
func (k Keeper) requirePublishCredential(
	ctx sdk.Context,
	params types.Params,
	msg *types.MsgPublishCandidate,
	publisherAppBound bool,
	nowUnix int64,
) (credentialID string, err error) {
	cred := msg.Credential
	empty := cred.IsEmpty()
	need := publisherAppBound || params.CredentialRequired
	if empty {
		if need {
			return "", types.ErrAssessmentCredentialInvalid
		}
		return "", nil
	}
	return k.verifyAssessmentCredential(ctx, params, msg, cred, nowUnix)
}
