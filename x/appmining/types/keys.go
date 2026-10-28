package types

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	ModuleName = "appmining"
	StoreKey   = ModuleName
	RouterKey  = ModuleName

	// RewardPoolName holds settlement-releasable reward funds only.
	RewardPoolName = "appmining_reward_pool"
	// ChallengeEscrowName holds open challenge bonds until resolve.
	ChallengeEscrowName = "appmining_challenge_escrow"

	// AppIDLength is the fixed length of app_id (SHA-256 digest).
	AppIDLength = 32
	// HashLength is the fixed length for attestation/evidence/result hashes.
	HashLength = 32
	// SourceTxHashLength is the fixed length for source transaction hashes.
	SourceTxHashLength = 32

	// EntitlementPoolName holds V1 entitlement pool funds (funded=available+reserved+paid).
	EntitlementPoolName = "appmining_entitlement_pool"
)

// AccModuleAddress is the primary module address (legacy alias of reward pool).
// Prefer AccRewardPoolAddress / AccChallengeEscrowAddress for fund movement.
var (
	AccModuleAddress          sdk.AccAddress
	AccRewardPoolAddress      sdk.AccAddress
	AccChallengeEscrowAddress sdk.AccAddress
)

func init() {
	AccRewardPoolAddress = authtypes.NewModuleAddress(RewardPoolName)
	AccChallengeEscrowAddress = authtypes.NewModuleAddress(ChallengeEscrowName)
	AccModuleAddress = AccRewardPoolAddress
}

const (
	prefixParams byte = iota + 1
	prefixApplication
	prefixEpochBudget
	prefixAttestation
	prefixChallenge
	prefixSettlement
	prefixGasCredit
	prefixGasCreditConsumption
	prefixNextChallengeID
	prefixOpenChallengeIndex // app_id|epoch|challenge_id -> 1
	prefixGasPayerIndex      // bech32 gas_payer -> app_id (unique)
	prefixEntitlementPool
	prefixCandidate
	prefixEntitlement
	prefixEntitlementChallenge
	prefixOpenEntitlementChallenge
	prefixRoleKey
	prefixAuthNonce
	prefixAssessmentSigner
	prefixCredentialIndex // credential_id -> entitlement_id (never released)
)

var (
	KeyPrefixParams                   = []byte{prefixParams}
	KeyPrefixApplication              = []byte{prefixApplication}
	KeyPrefixEpochBudget              = []byte{prefixEpochBudget}
	KeyPrefixAttestation              = []byte{prefixAttestation}
	KeyPrefixChallenge                = []byte{prefixChallenge}
	KeyPrefixSettlement               = []byte{prefixSettlement}
	KeyPrefixGasCredit                = []byte{prefixGasCredit}
	KeyPrefixGasCreditConsumption     = []byte{prefixGasCreditConsumption}
	KeyPrefixNextChallengeID          = []byte{prefixNextChallengeID}
	KeyPrefixOpenChallengeIndex       = []byte{prefixOpenChallengeIndex}
	KeyPrefixGasPayerIndex            = []byte{prefixGasPayerIndex}
	KeyPrefixEntitlementPool          = []byte{prefixEntitlementPool}
	KeyPrefixCandidate                = []byte{prefixCandidate}
	KeyPrefixEntitlement              = []byte{prefixEntitlement}
	KeyPrefixEntitlementChallenge     = []byte{prefixEntitlementChallenge}
	KeyPrefixOpenEntitlementChallenge = []byte{prefixOpenEntitlementChallenge}
	KeyPrefixRoleKey                  = []byte{prefixRoleKey}
	KeyPrefixAuthNonce                = []byte{prefixAuthNonce}
	KeyPrefixAssessmentSigner         = []byte{prefixAssessmentSigner}
	KeyPrefixCredentialIndex          = []byte{prefixCredentialIndex}
)

func ApplicationKey(appID []byte) []byte {
	return append(append([]byte{}, KeyPrefixApplication...), appID...)
}

func EpochBudgetKey(epoch uint64) []byte {
	return append(append([]byte{}, KeyPrefixEpochBudget...), Uint64Bytes(epoch)...)
}

func AttestationKey(appID []byte, epoch uint64) []byte {
	key := append(append([]byte{}, KeyPrefixAttestation...), appID...)
	return append(key, Uint64Bytes(epoch)...)
}

func ChallengeKey(id uint64) []byte {
	return append(append([]byte{}, KeyPrefixChallenge...), Uint64Bytes(id)...)
}

func SettlementKey(appID []byte, epoch uint64) []byte {
	key := append(append([]byte{}, KeyPrefixSettlement...), appID...)
	return append(key, Uint64Bytes(epoch)...)
}

func GasCreditKey(appID []byte) []byte {
	return append(append([]byte{}, KeyPrefixGasCredit...), appID...)
}

func GasCreditConsumptionKey(appID, sourceTxHash []byte) []byte {
	key := append(append([]byte{}, KeyPrefixGasCreditConsumption...), appID...)
	return append(key, sourceTxHash...)
}

func OpenChallengeIndexKey(appID []byte, epoch, challengeID uint64) []byte {
	key := append(append([]byte{}, KeyPrefixOpenChallengeIndex...), appID...)
	key = append(key, Uint64Bytes(epoch)...)
	return append(key, Uint64Bytes(challengeID)...)
}

func OpenChallengeIndexPrefix(appID []byte, epoch uint64) []byte {
	key := append(append([]byte{}, KeyPrefixOpenChallengeIndex...), appID...)
	return append(key, Uint64Bytes(epoch)...)
}

// GasPayerIndexKey maps a gas payer bech32 address to a single app_id.
func GasPayerIndexKey(payer string) []byte {
	return append(append([]byte{}, KeyPrefixGasPayerIndex...), []byte(payer)...)
}

func Uint64Bytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// --- V1 entitlement flow keys ---

// EntitlementPoolKey stores the Pool record by pool_id.
func EntitlementPoolKey(poolID string) []byte {
	return append(append([]byte{}, KeyPrefixEntitlementPool...), []byte(poolID)...)
}

// CandidateKey stores the Candidate record by entitlement_id.
func CandidateKey(entitlementID string) []byte {
	return append(append([]byte{}, KeyPrefixCandidate...), []byte(entitlementID)...)
}

// EntitlementKey stores the Entitlement record by entitlement_id.
func EntitlementKey(entitlementID string) []byte {
	return append(append([]byte{}, KeyPrefixEntitlement...), []byte(entitlementID)...)
}

// EntitlementChallengeKey stores the EntitlementChallenge record by challenge_id.
func EntitlementChallengeKey(challengeID string) []byte {
	return append(append([]byte{}, KeyPrefixEntitlementChallenge...), []byte(challengeID)...)
}

// OpenEntitlementChallengeKey indexes the open challenge of one challenger for
// one candidate: entitlement_id | challenger -> challenge_id.
func OpenEntitlementChallengeKey(entitlementID, challenger string) []byte {
	key := append(append([]byte{}, KeyPrefixOpenEntitlementChallenge...), []byte(entitlementID)...)
	return append(key, []byte(challenger)...)
}

// RoleKeyStoreKey stores a registered role key: role_byte | key_id -> RoleKey.
func RoleKeyStoreKey(role EntitlementRole, keyID string) []byte {
	key := append(append([]byte{}, KeyPrefixRoleKey...), byte(role))
	return append(key, []byte(keyID)...)
}

// AuthNonceKey stores consumed wire-v1 nonces: role_byte | key_id | 0 | nonce -> 1.
func AuthNonceKey(role EntitlementRole, keyID, nonce string) []byte {
	key := append(append([]byte{}, KeyPrefixAuthNonce...), byte(role))
	key = append(key, []byte(keyID)...)
	key = append(key, []byte{0}...)
	return append(key, []byte(nonce)...)
}

// AssessmentSignerKey stores an assessment signer by key_id.
func AssessmentSignerKey(keyID string) []byte {
	return append(append([]byte{}, KeyPrefixAssessmentSigner...), []byte(keyID)...)
}

// CredentialIndexKey maps credential_id -> entitlement_id.
// Permanent one-shot (D-A): entries are never released after REJECTED.
func CredentialIndexKey(credentialID string) []byte {
	return append(append([]byte{}, KeyPrefixCredentialIndex...), []byte(credentialID)...)
}
