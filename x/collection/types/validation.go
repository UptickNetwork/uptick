package types

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	DoNotModify = "[do-not-modify]"
	// RemoveField is the sentinel for CLEARING an optional field: an empty
	// string means "do not modify", so removing a field requires this value.
	RemoveField = "[remove]"
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

	// uptickSuffix matches the address component of a module-derived class id.
	// x/erc721 derives "uptick-<hex contract address>" and x/cw721 derives
	// "uptick-<bech32 contract address>", so alphanumerics plus hyphens cover
	// every shape the chain itself can produce while still rejecting
	// whitespace, punctuation and control characters.
	regexpUptickSuffix = regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString

	keywords          = strings.Join([]string{ReservedIBC}, "|")
	regexpKeywordsFmt = fmt.Sprintf("^(%s).*", keywords)
	regexpKeyword     = regexp.MustCompile(regexpKeywordsFmt).MatchString
)

// ValidateDenomID verifies whether the  parameters are legal
func ValidateDenomID(denomID string) error {
	if strings.ContainsRune(denomID, 0) {
		return sdkerrors.Wrapf(ErrInvalidDenom, "denomID contains NUL")
	}
	// Commas are rejected for every denom ID: a bridged denom becomes the
	// classId component of NFT UIDs ("<nftId>,<classId>"), which are parsed
	// by splitting on the last comma, so a comma inside a classId would
	// corrupt the round-trip.
	if strings.Contains(denomID, ",") {
		return sdkerrors.Wrapf(ErrInvalidDenom, "denomID cannot contain comma (%s)", denomID)
	}
	if strings.HasPrefix(denomID, "uptick-") {
		suffix := strings.TrimPrefix(denomID, "uptick-")
		if suffix == "" || strings.Contains(suffix, "/") {
			return sdkerrors.Wrapf(ErrInvalidDenom, "invalid uptick-prefixed denomID (%s)", denomID)
		}
		// This branch used to return right here after the two shape checks
		// above, so an oversized or oddly punctuated class id in a genesis
		// file was accepted even though keeper.SaveDenom documents
		// ValidateDenomID as the single place enforcing the charset and the
		// [3,128] bound. Enforce both here too. The lower bound needs no
		// check: the "uptick-" prefix alone is 7 bytes. The charset is
		// deliberately the loosest superset of the shapes the module derives
		// (40 hex nibbles from x/erc721, bech32 from x/cw721) so that no class
		// the chain can actually produce is rejected.
		if len(denomID) > MaxDenomLen {
			return sdkerrors.Wrapf(ErrInvalidDenom, "denomID length exceeds %d (%s)", MaxDenomLen, denomID)
		}
		if !regexpUptickSuffix(suffix) {
			return sdkerrors.Wrapf(ErrInvalidDenom,
				"uptick-prefixed denomID(%s) may only contain alphanumerics and hyphens after the prefix", denomID)
		}
		return ValidateKeywords(denomID)
	}
	if IsIBCDenom(denomID) {
		return validateIBCClassID(denomID)
	}
	if !regexpID(denomID) {
		return sdkerrors.Wrapf(ErrInvalidDenom, "denomID can only accept characters that match the regular expression: (%s),but got (%s)", idString, denomID)
	}
	return ValidateKeywords(denomID)
}

// validateIBCClassID accepts ICS-721 voucher class ids of the form ibc/{hash}.
// User issuance still goes through ValidateIssueDenomID, which rejects this prefix.
func validateIBCClassID(denomID string) error {
	suffix := strings.TrimPrefix(denomID, "ibc/")
	if suffix == "" {
		return sdkerrors.Wrapf(ErrInvalidDenom, "invalid ICS-721 class id (%s)", denomID)
	}
	if len(denomID) > MaxDenomLen {
		return sdkerrors.Wrapf(ErrInvalidDenom, "denomID length exceeds %d (%s)", MaxDenomLen, denomID)
	}
	for _, r := range suffix {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '/' || r == '.' || r == '_' || r == '-' {
			continue
		}
		return sdkerrors.Wrapf(ErrInvalidDenom, "invalid ICS-721 class id (%s)", denomID)
	}
	return nil
}

// ValidateTokenIDForDenom validates a token id in the context of its class.
// ICS-721 tokens are frequently short ("1"); user-minted collection tokens
// keep the [3,128] bound.
func ValidateTokenIDForDenom(denomID, tokenID string) error {
	if IsIBCDenom(denomID) {
		return validateIBCTokenID(tokenID)
	}
	return ValidateTokenID(tokenID)
}

func validateIBCTokenID(tokenID string) error {
	if tokenID == "" || len(tokenID) > MaxDenomLen {
		return sdkerrors.Wrapf(ErrInvalidTokenID, "the length of nft id(%s) only accepts value [1, %d]", tokenID, MaxDenomLen)
	}
	if strings.ContainsRune(tokenID, 0) || strings.Contains(tokenID, "/") {
		return sdkerrors.Wrapf(ErrInvalidTokenID, "nft id(%s) contains illegal characters", tokenID)
	}
	return nil
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

// Modified reports whether an incoming optional field carries a change.
// An empty string and DoNotModify both mean "keep the current value"; only a
// non-empty value (or the RemoveField sentinel) counts as a modification.
func Modified(target string) bool {
	return target != DoNotModify && target != ""
}

// ValidateIssueDenomID validates a denom ID for user-facing issuance via
// MsgIssueDenom. On top of the base denom rules it rejects reserved shapes:
// the "uptick-" prefix (module-derived class IDs), "ibc/" (ICS-721 vouchers),
// bech32 addresses (CW721 key collision) and hex addresses (ERC721 key
// collision). Module paths bypass this and go through keeper.SaveDenom.
func ValidateIssueDenomID(denomID string) error {
	if strings.HasPrefix(denomID, "uptick-") {
		return sdkerrors.Wrapf(ErrInvalidDenom,
			"denomID prefix \"uptick-\" is reserved for module-derived NFT classes and cannot be issued via MsgIssueDenom (%s)", denomID)
	}
	if strings.HasPrefix(denomID, "ibc/") {
		return sdkerrors.Wrapf(ErrInvalidDenom,
			"denomID prefix \"ibc/\" is reserved for ICS-721 voucher classes and cannot be issued via MsgIssueDenom (%s)", denomID)
	}
	// A user denom whose id is a valid bech32 account address would collide
	// with CW721 contract keys. Module-created classes never go through
	// MsgIssueDenom.
	if _, err := sdk.AccAddressFromBech32(denomID); err == nil {
		return sdkerrors.Wrapf(ErrInvalidDenom,
			"denomID cannot be a bech32 account address (%s)", denomID)
	}
	// A user denom whose id is a 40-nibble hex address string (optionally
	// 0x-prefixed) would collide with ERC721 contract keys in the erc721
	// module's pair lookup.
	if isHexAddressShape(denomID) {
		return sdkerrors.Wrapf(ErrInvalidDenom,
			"denomID cannot be a hex address shape reserved for ERC721 contract keys (%s)", denomID)
	}
	// Reserved-shape checks passed; defer to the base validation for the
	// remaining rules (NUL, comma, regex, length, keywords).
	return ValidateDenomID(denomID)
}

// hexAddressShapeRe matches EVM address-shaped strings: an optional 0x prefix
// plus exactly 40 hex nibbles, any case (the bare-hex form is what the pair
// lookup faces). Pure shape check; semantic validation belongs to the EVM
// helper package.
var hexAddressShapeRe = regexp.MustCompile(`^(0x)?[0-9a-fA-F]{40}$`)

func isHexAddressShape(s string) bool {
	return hexAddressShapeRe.MatchString(s)
}

// ValidateKeywords checks if the given denomID begins with `DenomKeywords`
func ValidateKeywords(denomID string) error {
	if regexpKeyword(denomID) {
		return sdkerrors.Wrapf(ErrInvalidDenom, "invalid denomID: %s, can not begin with keyword: (%s)", denomID, keywords)
	}
	return nil
}

// Modify merges an incoming optional field into the stored value.
// Empty string and DoNotModify keep the origin; RemoveField clears it;
// anything else replaces it. Note: the literal value "[remove]" is reserved —
// clients cannot set a field to that exact string.
func Modify(origin, target string) string {
	switch target {
	case DoNotModify, "":
		return origin
	case RemoveField:
		return ""
	default:
		return target
	}
}

func IsIBCDenom(denomID string) bool {
	return strings.HasPrefix(denomID, "ibc/")
}
