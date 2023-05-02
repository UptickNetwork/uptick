package types

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	// (?m)^(\d+) remove leading numbers
	reLeadingNumbers = `(?m)^(\d+)`
	// ^[^A-Za-z] forces first chars to be letters
	// [^a-zA-Z0-9/-] deletes special characters
	reDnmString = `^[^A-Za-z]|[^a-zA-Z0-9/-]`
)

func removeLeadingNumbers(str string) string {
	re := regexp.MustCompile(reLeadingNumbers)
	return re.ReplaceAllString(str, "")
}

func removeSpecialChars(str string) string {
	re := regexp.MustCompile(reDnmString)
	return re.ReplaceAllString(str, "")
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
	if len(name) > 128 {
		name = name[:128]
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

// CreateTokenIDFromNFTID create classId from erc721 address
func CreateTokenIDFromNFTID(nftID string) string {

	ret := strings.Replace(nftID, DefaultPrefix+"-", "", 1)
	return "0x" + hex.EncodeToString([]byte(ret))
	// return strings.Replace(nftID, DefaultPrefix, "", 1)
}

//
func CreateTokenUID(contractAddress string, tokenID string) string {

	return fmt.Sprintf("%s,%s", tokenID, contractAddress)
}

func CreateNFTUID(classID string, nftID string) string {

	return fmt.Sprintf("%s,%s", nftID, classID)
}

func GetNFTFromUID(uid string) (string, string) {

	uidArray := strings.Split(uid, ",")

	if len(uidArray) != 2 {
		return "", ""
	} else {
		return uidArray[0], uidArray[1]
	}
}
