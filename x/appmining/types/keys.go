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

	// AppIDLength is the fixed length of app_id (SHA-256 digest).
	AppIDLength = 32
	// HashLength is the fixed length for attestation/evidence/result hashes.
	HashLength = 32
	// SourceTxHashLength is the fixed length for source transaction hashes.
	SourceTxHashLength = 32
)

// AccModuleAddress is the module account that holds the reward pool.
var AccModuleAddress sdk.AccAddress

func init() {
	AccModuleAddress = authtypes.NewModuleAddress(ModuleName)
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
)

var (
	KeyPrefixParams               = []byte{prefixParams}
	KeyPrefixApplication          = []byte{prefixApplication}
	KeyPrefixEpochBudget          = []byte{prefixEpochBudget}
	KeyPrefixAttestation          = []byte{prefixAttestation}
	KeyPrefixChallenge            = []byte{prefixChallenge}
	KeyPrefixSettlement           = []byte{prefixSettlement}
	KeyPrefixGasCredit            = []byte{prefixGasCredit}
	KeyPrefixGasCreditConsumption = []byte{prefixGasCreditConsumption}
	KeyPrefixNextChallengeID      = []byte{prefixNextChallengeID}
	KeyPrefixOpenChallengeIndex   = []byte{prefixOpenChallengeIndex}
	KeyPrefixGasPayerIndex        = []byte{prefixGasPayerIndex}
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
