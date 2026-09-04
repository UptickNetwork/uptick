package types

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	DoNotModify = "[do-not-modify]"
	MinDenomLen = 3
	MaxDenomLen = 128

	MaxTokenURILen = 256

	// MaxDenomSchemaLen bounds the denom schema (it is embedded in the first
	// ERC721 deployment calldata, so an unbounded schema can permanently DoS
	// conversion). MaxDenomDataLen bounds arbitrary denom metadata.
	MaxDenomSchemaLen = 8192
	MaxDenomDataLen   = 65536

	ReservedIBC = "ibc"
)

var (
	// IsAlphaNumeric only accepts [a-z0-9]
	IsAlphaNumeric = regexp.MustCompile(`^[a-z0-9]+$`).MatchString
	// IsBeginWithAlpha only begin with [a-z]
	IsBeginWithAlpha = regexp.MustCompile(`^[a-z].*`).MatchString

	idString = `[a-z][a-zA-Z0-9/]{2,127}`
	regexpID = regexp.MustCompile(fmt.Sprintf(`^%s$`, idString)).MatchString

	keywords          = strings.Join([]string{ReservedIBC}, "|")
	regexpKeywordsFmt = fmt.Sprintf("^(%s).*", keywords)
	regexpKeyword     = regexp.MustCompile(regexpKeywordsFmt).MatchString
)

// ValidateDenomID verifies whether the  parameters are legal
func ValidateDenomID(denomID string) error {
	if strings.ContainsRune(denomID, 0) {
		return sdkerrors.Wrapf(ErrInvalidDenom, "denomID contains NUL")
	}
	// A denom ID becomes the classId component of NFT UIDs once the denom is
	// bridged to ERC721/CW721 (CreateNFTUID -> "<nftId>,<classId>"). The UID
	// parser (GetNFTFromUID) splits on the LAST comma because only the second
	// component is guaranteed comma-free; a comma inside the classId would
	// corrupt the round-trip and strand the reverse mapping. The regex branch
	// below already excludes commas, but the "uptick-" prefixed branch does
	// not -- reject commas for every denom ID.
	if strings.Contains(denomID, ",") {
		return sdkerrors.Wrapf(ErrInvalidDenom, "denomID cannot contain comma (%s)", denomID)
	}
	if strings.HasPrefix(denomID, "uptick-") {
		suffix := strings.TrimPrefix(denomID, "uptick-")
		if suffix == "" || strings.Contains(suffix, "/") {
			return sdkerrors.Wrapf(ErrInvalidDenom, "invalid uptick-prefixed denomID (%s)", denomID)
		}
		return ValidateKeywords(denomID)
	}
	if !regexpID(denomID) {
		return sdkerrors.Wrapf(ErrInvalidDenom, "denomID can only accept characters that match the regular expression: (%s),but got (%s)", idString, denomID)
	}
	return ValidateKeywords(denomID)
}

// ValidateTokenID verify that the tokenID is legal
func ValidateTokenID(tokenID string) error {
	if len(tokenID) < MinDenomLen || len(tokenID) > MaxDenomLen {
		return sdkerrors.Wrapf(ErrInvalidTokenID, "the length of nft id(%s) only accepts value [%d, %d]", tokenID, MinDenomLen, MaxDenomLen)
	}
	if strings.ContainsRune(tokenID, 0) || strings.Contains(tokenID, "/") {
		return sdkerrors.Wrapf(ErrInvalidTokenID, "nft id(%s) contains illegal characters", tokenID)
	}
	return nil
}

// ValidateTokenURI verify that the tokenURI is legal
func ValidateTokenURI(tokenURI string) error {
	if len(tokenURI) > MaxTokenURILen {
		return sdkerrors.Wrapf(errortypes.ErrInvalidRequest, "token URI too long; max %d", MaxTokenURILen)
	}
	if strings.ContainsAny(tokenURI, "\n\r\t") {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "token URI contains control characters")
	}
	u, err := url.Parse(tokenURI)
	if err != nil {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "invalid token URI")
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ipfs" {
		return sdkerrors.Wrap(errortypes.ErrInvalidRequest, "token URI must use http(s) or ipfs scheme")
	}
	return nil
}

// Modified returns whether the field is modified
func Modified(target string) bool {
	return target != DoNotModify
}

// ValidateKeywords checks if the given denomID begins with `DenomKeywords`
func ValidateKeywords(denomID string) error {
	if regexpKeyword(denomID) {
		return sdkerrors.Wrapf(ErrInvalidDenom, "invalid denomID: %s, can not begin with keyword: (%s)", denomID, keywords)
	}
	return nil
}

func Modify(origin, target string) string {

	if target == DoNotModify {
		return origin
	}
	return target
}

func IsIBCDenom(denomID string) bool {
	return strings.HasPrefix(denomID, "ibc/")
}
