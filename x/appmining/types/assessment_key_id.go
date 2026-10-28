package types

import (
	"regexp"

	sdkerrors "cosmossdk.io/errors"
)

// AssessmentSignerKeyIDPattern: lowercase alnum, optional internal hyphens, length 3–32.
// UGE T1 directive D-B: ^[a-z0-9][a-z0-9-]{2,31}$
var AssessmentSignerKeyIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,31}$`)

// ValidateAssessmentSignerKeyID enforces the D-B key_id naming constraint.
func ValidateAssessmentSignerKeyID(keyID string) error {
	if !AssessmentSignerKeyIDPattern.MatchString(keyID) {
		return sdkerrors.Wrapf(ErrInvalidWireID, "assessment signer key_id %q must match %s", keyID, AssessmentSignerKeyIDPattern.String())
	}
	return nil
}
