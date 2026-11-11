package types

import "testing"

// The two policy functions are the only place that decides which role may hold
// which key type, and which operation that key type may authorize. Keeping the
// matrix pinned here means widening one cell later is a deliberate edit with a
// failing test attached, not a side effect of refactoring a switch.
func TestRoleKeyTypeAllowed(t *testing.T) {
	cases := []struct {
		name    string
		role    EntitlementRole
		keyType RoleKeyType
		want    bool
	}{
		{"publisher ed25519", EntitlementRolePublisher, RoleKeyTypeEd25519, true},
		{"publisher evm", EntitlementRolePublisher, RoleKeyTypeSecp256k1EvmAddress, true},
		{"reviewer ed25519", EntitlementRoleReviewer, RoleKeyTypeEd25519, true},
		{"reviewer evm", EntitlementRoleReviewer, RoleKeyTypeSecp256k1EvmAddress, true},
		{"approver evm", EntitlementRoleApprover, RoleKeyTypeSecp256k1EvmAddress, false},
		{"risk evm", EntitlementRoleRisk, RoleKeyTypeSecp256k1EvmAddress, false},
		{"audit evm", EntitlementRoleAudit, RoleKeyTypeSecp256k1EvmAddress, false},
		{"unspecified evm", EntitlementRoleUnspecified, RoleKeyTypeSecp256k1EvmAddress, false},
		{"reviewer unknown key type", EntitlementRoleReviewer, RoleKeyType(9), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RoleKeyTypeAllowed(tc.role, tc.keyType); got != tc.want {
				t.Fatalf("RoleKeyTypeAllowed(%v, %v) = %v, want %v", tc.role, tc.keyType, got, tc.want)
			}
		})
	}
}

func TestRoleKeyTypeOperationAllowed(t *testing.T) {
	cases := []struct {
		name      string
		role      EntitlementRole
		keyType   RoleKeyType
		operation string
		want      bool
	}{
		// Ed25519 is the unrestricted scheme: the role alone decides scope.
		{"reviewer ed25519 resolve", EntitlementRoleReviewer, RoleKeyTypeEd25519, "RESOLVE", true},
		{"reviewer ed25519 publish", EntitlementRoleReviewer, RoleKeyTypeEd25519, "PUBLISH", true},

		// EVM-address keys are scoped to operations a wallet will personal_sign.
		{"publisher evm publish", EntitlementRolePublisher, RoleKeyTypeSecp256k1EvmAddress, "PUBLISH", true},
		{"publisher evm resolve", EntitlementRolePublisher, RoleKeyTypeSecp256k1EvmAddress, "RESOLVE", false},
		{"reviewer evm resolve", EntitlementRoleReviewer, RoleKeyTypeSecp256k1EvmAddress, "RESOLVE", true},
		{"reviewer evm publish", EntitlementRoleReviewer, RoleKeyTypeSecp256k1EvmAddress, "PUBLISH", false},
		{"reviewer evm approve", EntitlementRoleReviewer, RoleKeyTypeSecp256k1EvmAddress, "APPROVE", false},
		{"risk evm revoke", EntitlementRoleRisk, RoleKeyTypeSecp256k1EvmAddress, "REVOKE", false},
		{"unknown key type", EntitlementRoleReviewer, RoleKeyType(9), "RESOLVE", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RoleKeyTypeOperationAllowed(tc.role, tc.keyType, tc.operation)
			if got != tc.want {
				t.Fatalf("RoleKeyTypeOperationAllowed(%v, %v, %q) = %v, want %v",
					tc.role, tc.keyType, tc.operation, got, tc.want)
			}
		})
	}
}

// Registration must reject what verification would reject anyway, otherwise an
// operator can create a key that no wallet can ever produce a valid proof for.
func TestValidateRoleKeyMaterialMatchesPolicy(t *testing.T) {
	addr := make([]byte, EvmAddressLength)
	addr[19] = 1

	if err := ValidateRoleKeyMaterial(EntitlementRoleReviewer, RoleKeyTypeSecp256k1EvmAddress, addr); err != nil {
		t.Fatalf("reviewer evm address should register: %v", err)
	}
	if err := ValidateRoleKeyMaterial(EntitlementRoleRisk, RoleKeyTypeSecp256k1EvmAddress, addr); err == nil {
		t.Fatal("risk evm address must be rejected")
	}
	if err := ValidateRoleKeyMaterial(EntitlementRoleReviewer, RoleKeyTypeSecp256k1EvmAddress, make([]byte, 32)); err == nil {
		t.Fatal("32-byte material must be rejected for an evm-address key")
	}
}
