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
	// RemoveField is the explicit sentinel for CLEARING an optional field.
	// M-1 decision (2026-09-04): an empty string means "do not modify" so
	// REST/gRPC clients that only fill the required fields can no longer
	// silently wipe on-chain metadata. Callers that truly want a field
	// emptied must send this sentinel.
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
// non-empty value (or the RemoveField sentinel) counts as a modification. This
// keeps CLI (defaults to DoNotModify) and REST/gRPC (which leave optional
// fields empty) on the same code path instead of letting empty strings erase
// metadata.
func Modified(target string) bool {
	return target != DoNotModify && target != ""
}

// ValidateIssueDenomID validates a denom ID for user-facing issuance via
// MsgIssueDenom. On top of the base denom rules it rejects the "uptick-"
// prefix: erc721/cw721 bridging derives class IDs as "uptick-<contract>" and
// a user pre-minting such a denom would permanently block registration of
// the matching contract. Module paths create derived denoms directly through
// keeper.SaveDenom and never go through MsgIssueDenom.
func ValidateIssueDenomID(denomID string) error {
	if err := ValidateDenomID(denomID); err != nil {
		return err
	}
	if strings.HasPrefix(denomID, "uptick-") {
		return sdkerrors.Wrapf(ErrInvalidDenom,
			"denomID prefix \"uptick-\" is reserved for module-derived NFT classes and cannot be issued via MsgIssueDenom (%s)", denomID)
	}
	if IsIBCDenom(denomID) {
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
	// module's pair lookup. The base denom regex cannot express this rule, so
	// it is enforced here alongside the other reserved shapes.
	if isHexAddressShape(denomID) {
		return sdkerrors.Wrapf(ErrInvalidDenom,
			"denomID cannot be a hex address shape reserved for ERC721 contract keys (%s)", denomID)
	}
	return nil
}

// hexAddressShapeRe matches EVM address-shaped strings: an optional 0x prefix
// followed by exactly 40 hexadecimal nibbles, any case.
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
// anything else replaces it.
//
// NOTE (documented limitation): because the exact string
// "[remove]" (RemoveField) is the sentinel for clearing a field, a client
// CANNOT set a field to the literal value "[remove]" — it will always be
// interpreted as "clear this field". This applies to token URI, URI hash,
// metadata name and data fields in MsgUpdateNFT / MsgTransferNFT. There is no
// escape sequence by design; treat "[remove]" as a reserved value in clients.
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
