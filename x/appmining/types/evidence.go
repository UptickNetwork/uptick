package types

// EvidenceTypeWire returns the UGE wire string for an evidence type enum.
func EvidenceTypeWire(t EvidenceType) (string, bool) {
	switch t {
	case EvidenceTypeTxMissingOrUnfinalized:
		return "TX_MISSING_OR_UNFINALIZED", true
	case EvidenceTypeReceiptQbtMismatch:
		return "RECEIPT_QBT_MISMATCH", true
	case EvidenceTypeDuplicateOrFamilyConflict:
		return "DUPLICATE_OR_FAMILY_CONFLICT", true
	case EvidenceTypeConfirmedRefundOrReversal:
		return "CONFIRMED_REFUND_OR_REVERSAL", true
	case EvidenceTypeAppOrEmitterIneligible:
		return "APP_OR_EMITTER_INELIGIBLE", true
	case EvidenceTypeReproducibleCalculationError:
		return "REPRODUCIBLE_CALCULATION_ERROR", true
	default:
		return "", false
	}
}

// RoleWire returns the wire role string for an entitlement role.
func RoleWire(r EntitlementRole) (string, bool) {
	switch r {
	case EntitlementRolePublisher:
		return "publisher", true
	case EntitlementRoleApprover:
		return "approver", true
	case EntitlementRoleReviewer:
		return "reviewer", true
	case EntitlementRoleRisk:
		return "risk", true
	case EntitlementRoleAudit:
		return "audit", true
	default:
		return "", false
	}
}

// RoleFromWire maps a wire role string to the enum.
func RoleFromWire(s string) (EntitlementRole, bool) {
	switch s {
	case "publisher":
		return EntitlementRolePublisher, true
	case "approver":
		return EntitlementRoleApprover, true
	case "reviewer":
		return EntitlementRoleReviewer, true
	case "risk":
		return EntitlementRoleRisk, true
	case "audit":
		return EntitlementRoleAudit, true
	default:
		return EntitlementRoleUnspecified, false
	}
}
