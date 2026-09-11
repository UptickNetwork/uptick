package types

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode/utf8"

	sdkerrors "cosmossdk.io/errors"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// (?m)^(\d+) remove leading numbers
	reLeadingNumbers = `(?m)^(\d+)`
	// ^[^A-Za-z] forces first chars to be letters
	// [^a-zA-Z0-9/-] deletes special characters
	reDnmString = `^[^A-Za-z]|[^a-zA-Z0-9/-]`
)

var (
	reLeadingNumbersRegexp = regexp.MustCompile(reLeadingNumbers)
	reDnmStringRegexp      = regexp.MustCompile(reDnmString)
)

func removeLeadingNumbers(str string) string {
	return reLeadingNumbersRegexp.ReplaceAllString(str, "")
}

func removeSpecialChars(str string) string {
	return reDnmStringRegexp.ReplaceAllString(str, "")
}

// recursively remove every invalid prefix
func removeInvalidPrefixes(str string) string {
	if strings.HasPrefix(str, "ibc/") {
		return removeInvalidPrefixes(str[4:])
	}
	if strings.HasPrefix(str, "erc721/") {
		return removeInvalidPrefixes(str[7:])
	}
	return str
}

// SanitizeERC721Name enforces 128 max string length, deletes leading numbers
// removes special characters  (except /)  and spaces from the ERC721 name
func SanitizeERC721Name(name string) string {
	name = removeLeadingNumbers(name)
	name = removeSpecialChars(name)
	if utf8.RuneCountInString(name) > 128 {
		runes := []rune(name)
		name = string(runes[:128])
	}
	name = removeInvalidPrefixes(name)
	return name
}

// EqualMetadata checks if all the fields of the provided coin metadata are equal.
func EqualMetadata(a, b banktypes.Metadata) error {
	if a.Base == b.Base && a.Description == b.Description && a.Display == b.Display && a.Name == b.Name && a.Symbol == b.Symbol {
		if len(a.DenomUnits) != len(b.DenomUnits) {
			return fmt.Errorf("metadata provided has different denom units from stored, %d ≠ %d", len(a.DenomUnits), len(b.DenomUnits))
		}

		for i, v := range a.DenomUnits {
			if (v.Exponent != b.DenomUnits[i].Exponent) || (v.Denom != b.DenomUnits[i].Denom) || !EqualStringSlice(v.Aliases, b.DenomUnits[i].Aliases) {
				return fmt.Errorf("metadata provided has different denom unit from stored, %s ≠ %s", a.DenomUnits[i], b.DenomUnits[i])
			}
		}

		return nil
	}
	return fmt.Errorf("metadata provided is different from stored")
}

// EqualStringSlice checks if two string slices are equal.
func EqualStringSlice(aliasesA, aliasesB []string) bool {
	if len(aliasesA) != len(aliasesB) {
		return false
	}

	for i := 0; i < len(aliasesA); i++ {
		if aliasesA[i] != aliasesB[i] {
			return false
		}
	}

	return true
}

func removeAddress0x(address string) string {

	strAddress := address
	if strings.HasPrefix(address, "0x") {
		strAddress = address[2:]
	}

	return strAddress
}

// CreateClassIDFromContractAddress create classId from erc721 address
func CreateClassIDFromContractAddress(address string) string {

	return fmt.Sprintf("%s-%s", DefaultPrefix, removeAddress0x(address))
}

// CreateContractAddressFromClassID create classId from erc721 address
func CreateContractAddressFromClassID(classID string) string {

	return strings.Replace(classID, DefaultPrefix+"-", "", 1)
}

// CreateNFTIDFromTokenID create classId from erc721 address
func CreateNFTIDFromTokenID(id string) string {

	return fmt.Sprintf("%s%s", DefaultPrefix, removeAddress0x(id))
}

// CreateTokenIDFromNFTID derives a base-10 ERC721 token ID from a Cosmos NFT
// id. The stripped NFT id is first hashed with SHA-256 and then interpreted as
// a big integer, guaranteeing a uint256-compatible decimal token ID regardless
// of the original NFT id length.
func CreateTokenIDFromNFTID(nftID string) string {

	ret := strings.Replace(nftID, DefaultPrefix+"-", "", 1)
	digest := sha256.Sum256([]byte(ret))
	return new(big.Int).SetBytes(digest[:]).String()
}

func CreateTokenUID(contractAddress string, tokenID string) string {

	return fmt.Sprintf("%s,%s", tokenID, contractAddress)
}

func CreateNFTUID(classID string, nftID string) string {

	return fmt.Sprintf("%s,%s", nftID, classID)
}

func GetNFTFromUID(uid string) (string, string) {
	// The UID is "<tokenId>,<contractAddress>" or "<nftId>,<classId>". The second
	// component (contract address / class id) never contains a comma, so splitting
	// on the LAST comma makes the round-trip robust to commas inside the first
	// component without a state migration.
	idx := strings.LastIndex(uid, ",")
	if idx <= 0 || idx == len(uid)-1 {
		return "", ""
	}
	return uid[:idx], uid[idx+1:]
}

// ERC721 token ids reach chain state in two spellings:
//
//   - legacy (<= v0.3.3, evm-nft-convert): "0x" followed by the lowercase hex
//     of the UTF-8 bytes of the Cosmos NFT id with the "uptick-" prefix
//     stripped, so nft id "nftmp123456789" becomes
//     "0x6e66746d70313233343536373839".
//   - canonical (v0.4.0+): the base-10 string of the uint256 token id, which
//     for a freshly converted NFT is sha256(nft id).
//
// For one and the same NFT the two spellings denote DIFFERENT uint256 values
// (the id bytes read as an integer, versus the digest of those bytes), so a
// legacy binding cannot be re-hashed into the canonical value: the value
// itself lives in the ERC721 contract on the EVM side and must not move. What
// this module can and does unify is the KEY SPELLING of that value in its own
// store, and the compatibility layer below is deliberately one-way:
//
//   - ValidateEVMTokenID runs from every Msg*.ValidateBasic, i.e. wherever a
//     caller CREATES a binding: canonical base-10 only.
//   - ParseEVMTokenID is for values read back out of state or handed to the
//     ERC721 contract: both spellings, yielding the value.
//   - CanonicalEVMTokenID / LegacyEVMTokenID / EVMTokenIDKeyVariants let the
//     keeper find a value under either spelling and rewrite it under the
//     canonical one, so a legacy key is upgraded in place on first touch
//     instead of being duplicated or orphaned.
//
// Values wider than 256 bits stay rejected in both spellings. That is not a
// regression: such a token id cannot be represented by the uint256 the ERC721
// contract stores, so the state is already broken and must not be rewritten.

// ValidateEVMTokenID ensures an ERC721 token ID is a base-10 uint256 string.
// fmt.Sscan historically accepted hexadecimal token IDs, which could create
// duplicate NFT-pair keys for the same numerical token ID.
//
// This is the creation-time check — deliberately stricter than
// ParseEVMTokenID, which also accepts the legacy "0x"+hex spelling for values
// that are only being read back.
func ValidateEVMTokenID(tokenID string) error {
	if _, _, err := parseEVMTokenIDValue(tokenID, false); err != nil {
		return err
	}
	return nil
}

// ParseEVMTokenID parses an ERC721 token id in either the canonical base-10
// spelling or the legacy "0x"+hex spelling and returns its uint256 value. Use
// it for values that came out of state or that feed an ERC721 contract call;
// use ValidateEVMTokenID when creating a binding.
func ParseEVMTokenID(tokenID string) (*big.Int, error) {
	n, _, err := parseEVMTokenIDValue(tokenID, true)
	return n, err
}

// CanonicalEVMTokenID returns the canonical base-10 spelling of tokenID and
// reports whether the input used the legacy "0x"+hex spelling. The result has
// no leading zeros, so "007", "7" and "0x07" all canonicalise to "7" and can
// never produce two store keys for one value.
func CanonicalEVMTokenID(tokenID string) (canonical string, legacy bool, err error) {
	n, legacy, err := parseEVMTokenIDValue(tokenID, true)
	if err != nil {
		return "", false, err
	}
	return n.String(), legacy, nil
}

// LegacyEVMTokenID returns the legacy "0x"+hex spelling that denotes the same
// uint256 value as the given canonical base-10 token id.
//
// hex.EncodeToString always emits an even number of nibbles, so an odd-length
// Text(16) result means the leading byte was 0x01..0x0f and its dropped high
// nibble has to be restored — otherwise "0x"+"b616263" would not match the
// stored "0x0b616263". A token id whose bytes start with NUL would need more
// than one nibble restored; those are rejected at mint time (the collection
// token id rules forbid NUL), so the single-nibble case is the only one.
func LegacyEVMTokenID(tokenID string) (string, bool) {
	n, ok := new(big.Int).SetString(tokenID, 10)
	if !ok || n.Sign() < 0 || n.BitLen() > 256 {
		return "", false
	}
	// Zero has no legacy spelling: evm-nft-convert hex-encoded the bytes of a
	// non-empty nft id, and the collection token id rules reject an empty one.
	if n.Sign() == 0 {
		return "", false
	}

	hexDigits := n.Text(16)
	if len(hexDigits)%2 == 1 {
		hexDigits = "0" + hexDigits
	}
	return "0x" + hexDigits, true
}

// EVMTokenIDKeyVariants returns the distinct token-id spellings under which
// the same uint256 value may have been written to the NFT-pair store, most
// canonical first. Readers probe them in order, which is what makes a legacy
// key readable without a state migration; writers use the first entry as the
// key to write and drop the rest, which is what keeps one value from ending
// up with two keys.
//
// Input that is not a valid uint256 in either spelling yields itself as the
// only variant, so a caller still performs an exact-match lookup against
// already-broken state instead of skipping the read.
func EVMTokenIDKeyVariants(tokenID string) []string {
	canonical, _, err := CanonicalEVMTokenID(tokenID)
	if err != nil {
		return []string{tokenID}
	}

	variants := []string{canonical}
	if legacy, ok := LegacyEVMTokenID(canonical); ok && legacy != canonical {
		variants = append(variants, legacy)
	}
	return variants
}

// EqualEVMTokenID reports whether two token-id strings denote the same uint256
// value in either spelling. Input that is not a valid uint256 in either form
// has to match byte for byte, so already-broken state is never silently
// widened into a match.
func EqualEVMTokenID(a, b string) bool {
	if a == b {
		return true
	}

	na, _, errA := parseEVMTokenIDValue(a, true)
	if errA != nil {
		return false
	}
	nb, _, errB := parseEVMTokenIDValue(b, true)
	if errB != nil {
		return false
	}
	return na.Cmp(nb) == 0
}

// Contract addresses are the SECOND component of the token-UID key, and they
// reach state in two spellings just like the token id does: the canonical
// all-lowercase form and the EIP-55 checksummed form that the pre-v0.4.0
// module wrote. Both mainnet and testnet still hold checksummed addresses.
//
// The parallel with EVMTokenIDKeyVariants stops at the mechanism, though. The
// two token-id spellings denote DIFFERENT uint256 values — the legacy one is
// already minted into the ERC721 contract and must not move — so there all the
// module can unify is the key spelling. The two address spellings are the SAME
// 20-byte value: common.HexToAddress round-trips either one to the same
// address. Normalising an address is therefore safe, and is what lets the
// module converge on one key per binding.

// CanonicalContractAddress returns the spelling used for every store key
// derived from an ERC721 contract address: lowercase for a valid hex address,
// and the input unchanged otherwise (a malformed value still has to match
// itself exactly rather than being rewritten).
func CanonicalContractAddress(address string) string {
	if common.IsHexAddress(address) {
		return strings.ToLower(address)
	}
	return address
}

// ChecksumContractAddress returns the EIP-55 checksummed spelling of an EVM
// contract address, i.e. the spelling the pre-v0.4.0 module stored. It reports
// false when the input is not a hex address.
func ChecksumContractAddress(address string) (string, bool) {
	if !common.IsHexAddress(address) {
		return "", false
	}
	return common.HexToAddress(address).Hex(), true
}

// ContractAddressKeyVariants returns the distinct spellings under which the
// same contract address may have been written to the store, canonical
// (lowercase) first. A reader probes them in order; a writer uses the first as
// the key to write and drops the rest.
func ContractAddressKeyVariants(address string) []string {
	canonical := CanonicalContractAddress(address)

	variants := []string{canonical}
	if checksummed, ok := ChecksumContractAddress(address); ok && checksummed != canonical {
		variants = append(variants, checksummed)
	}
	return variants
}

// EqualContractAddress reports whether two strings denote the same EVM
// contract address regardless of spelling. Anything that is not a hex address
// falls back to an exact comparison, so a malformed value is not silently
// widened into a match.
func EqualContractAddress(a, b string) bool {
	if !common.IsHexAddress(a) || !common.IsHexAddress(b) {
		return a == b
	}
	return common.HexToAddress(a) == common.HexToAddress(b)
}

// TokenUIDKeyVariants returns every spelling a forward (token id, contract
// address) key may have been written under, canonical spelling first.
//
// The forward key is "<tokenId>,<contractAddress>" and BOTH of its components
// have more than one historical spelling — the token id is either the v0.3.3
// "0x"+hex form or the v0.4.0 base-10 one, and the address is either EIP-55
// checksummed or lowercase — so a key has to be looked up under the cross
// product of the two. This is the same enumeration SetNFTPairs and
// ResolveNFTUIDPair perform inline; it is factored out here because the index
// prune needs it a third time.
//
// Note the argument order of CreateTokenUID: it takes (contract, tokenID) while
// the string it builds puts the token id first.
//
// A key that cannot be split yields itself as its only variant, so already
// broken state is compared exactly instead of being widened into a match.
func TokenUIDKeyVariants(tokenUID string) []string {
	tokenID, address := GetNFTFromUID(tokenUID)
	if tokenID == "" || address == "" {
		return []string{tokenUID}
	}

	tokenVariants := EVMTokenIDKeyVariants(tokenID)
	addressVariants := ContractAddressKeyVariants(address)

	variants := make([]string, 0, len(tokenVariants)*len(addressVariants))
	for _, tokenVariant := range tokenVariants {
		for _, addressVariant := range addressVariants {
			variants = append(variants, CreateTokenUID(addressVariant, tokenVariant))
		}
	}
	return variants
}

// parseEVMTokenIDValue parses tokenID and enforces the uint256 range. With
// allowLegacy the legacy "0x"+hex spelling is accepted; without it only the
// canonical base-10 spelling is.
func parseEVMTokenIDValue(tokenID string, allowLegacy bool) (*big.Int, bool, error) {
	digits, base, legacy := tokenID, 10, false
	if allowLegacy && (strings.HasPrefix(tokenID, "0x") || strings.HasPrefix(tokenID, "0X")) {
		digits, base, legacy = tokenID[2:], 16, true
	}

	n, ok := new(big.Int).SetString(digits, base)
	if !ok || n.Sign() < 0 || n.BitLen() > 256 {
		return nil, false, sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "invalid ERC721 token id %q", tokenID)
	}
	return n, legacy, nil
}
