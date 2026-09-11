package types

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestValidateDenomID(t *testing.T) {
	require.NoError(t, ValidateDenomID("abc"))
	require.NoError(t, ValidateDenomID("uptick-custom-denom"))
	require.Error(t, ValidateDenomID("A-!"))
	require.Error(t, ValidateDenomID("abc\x00def"))
	require.Error(t, ValidateDenomID("uptick-abc/def"))
	require.Error(t, ValidateDenomID("ibc-token"))
}

// A comma inside a denom ID would end up in the classId component of NFT UIDs
// (CreateNFTUID -> "<nftId>,<classId>") and break the last-comma round-trip in
// GetNFTFromUID. The "uptick-" prefix branch of ValidateDenomID bypasses the
// character-set regex, so comma rejection must cover both branches.
func TestValidateDenomID_RejectsComma(t *testing.T) {
	// regex branch (already rejected by the charset, pinned here explicitly)
	require.Error(t, ValidateDenomID("a,b"))
	// uptick- prefixed branch must reject commas
	require.Error(t, ValidateDenomID("uptick-a,b"))
	require.Error(t, ValidateDenomID("uptick-denom,with,commas"))
	// the UID parser round-trips only comma-free classIds
	require.NoError(t, ValidateDenomID("uptick-abcdef"))
}

func TestValidateTokenID(t *testing.T) {
	require.NoError(t, ValidateTokenID("abc"))
	require.Error(t, ValidateTokenID("ab"))
	require.Error(t, ValidateTokenID(strings.Repeat("a", MaxDenomLen+1)))
	require.Error(t, ValidateTokenID("ab\x00c"))
	require.Error(t, ValidateTokenID("ab/c"))
}

func TestValidateTokenURI(t *testing.T) {
	require.NoError(t, ValidateTokenURI("https://example.com/nft/1"))
	require.Error(t, ValidateTokenURI(strings.Repeat("u", MaxTokenURILen+1)))
}

func TestModifyAndModified(t *testing.T) {
	require.False(t, Modified(DoNotModify))
	require.True(t, Modified("new-value"))
	require.Equal(t, "origin", Modify("origin", DoNotModify))
	require.Equal(t, "new-value", Modify("origin", "new-value"))
}

// M-1 (2026-09-04): an empty string must mean "do not modify" so REST/gRPC
// clients that only fill required fields cannot silently wipe metadata.
// Clearing a field requires the explicit [remove] sentinel.
func TestModifyAndModified_EmptyStringKeepsOrigin(t *testing.T) {
	require.False(t, Modified(""))
	require.Equal(t, "origin", Modify("origin", ""))
	require.Equal(t, "origin", Modify("origin", DoNotModify))
	// explicit clear
	require.True(t, Modified(RemoveField))
	require.Equal(t, "", Modify("origin", RemoveField))
	// whitespace-only is a real value (a name of spaces is still suspicious at
	// the msg layer, but Modify itself must not reinterpret it)
	require.True(t, Modified(" "))
	require.Equal(t, " ", Modify("origin", " "))
}

// MsgIssueDenom must reject the "uptick-" prefix, which is
// reserved for module-derived class IDs (erc721/cw721 bridging derives class
// IDs as "uptick-<contract>"). A user pre-minting such a denom would
// permanently block registration of the matching contract.
func TestValidateIssueDenomID_ReservedPrefix(t *testing.T) {
	require.NoError(t, ValidateIssueDenomID("abc"))
	require.NoError(t, ValidateIssueDenomID("mycustomdenom"))
	// module-derived class ID shape
	require.Error(t, ValidateIssueDenomID("uptick-abcdef"))
	require.Error(t, ValidateIssueDenomID("uptick-b37eb5464b45a8097cbbb7c22727a6b259a3d85e"))
	// base rules still apply on top
	require.Error(t, ValidateIssueDenomID("ibc-token"))
	require.Error(t, ValidateIssueDenomID("a,b"))
	require.Error(t, ValidateIssueDenomID("A-!"))
	require.Error(t, ValidateIssueDenomID("ibc/ABCDEF0123"))
	sdk.GetConfig().SetBech32PrefixForAccount("uptick", "uptickpub")
	bech32 := sdk.AccAddress([]byte("cw721contractaddrxx")).String()
	require.Error(t, ValidateIssueDenomID(bech32))
}

func TestValidateDenomID_AllowsICS721Voucher(t *testing.T) {
	require.NoError(t, ValidateDenomID("ibc/ABCDEF0123456789"))
	require.Error(t, ValidateDenomID("ibc/"))
	require.Error(t, ValidateDenomID("ibc-token"))
}

func TestValidateTokenIDForDenom_IBCAllowsShortID(t *testing.T) {
	require.NoError(t, ValidateTokenIDForDenom("ibc/ABCDEF", "1"))
	require.Error(t, ValidateTokenID("1"))
	require.NoError(t, ValidateTokenIDForDenom("kitty", "nft1"))
}

func TestMsgTransferNFT_AllowsIBCVoucher(t *testing.T) {
	sender := sdk.AccAddress([]byte("remove-sender-addr")).String()
	recipient := sdk.AccAddress([]byte("remove-recipi-addr")).String()
	msg := &MsgTransferNFT{
		Id:        "1",
		DenomId:   "ibc/ABCDEF0123456789",
		Sender:    sender,
		Recipient: recipient,
	}
	require.NoError(t, msg.ValidateBasic())
}

func TestMsgMintNFT_RejectsIBCVoucher(t *testing.T) {
	sender := sdk.AccAddress([]byte("remove-sender-addr")).String()
	msg := &MsgMintNFT{
		Id:        "nft1",
		DenomId:   "ibc/ABCDEF0123456789",
		Name:      "n",
		Sender:    sender,
		Recipient: sender,
	}
	require.Error(t, msg.ValidateBasic())
}

func TestMsgTransferDenom_RejectsIBCVoucher(t *testing.T) {
	sender := sdk.AccAddress([]byte("remove-sender-addr")).String()
	msg := &MsgTransferDenom{
		Id:        "ibc/ABCDEF0123456789",
		Sender:    sender,
		Recipient: sender,
	}
	require.Error(t, msg.ValidateBasic())
}

// The "uptick-" branch of ValidateDenomID used to return after checking only
// that the suffix was non-empty and slash-free, so an oversized or oddly
// punctuated class id in a genesis file was accepted even though
// keeper.SaveDenom documents this function as the one place enforcing the
// charset and the [3,128] bound. The charset is deliberately the loosest
// superset of the shapes the module derives, so nothing the chain can actually
// produce is rejected.
func TestValidateDenomID_UptickBranchEnforcesShape(t *testing.T) {
	// Shapes x/erc721 (40 hex nibbles) and x/cw721 (bech32) derive.
	require.NoError(t, ValidateDenomID("uptick-abcdef"))
	require.NoError(t, ValidateDenomID("uptick-custom-denom"))
	require.NoError(t, ValidateDenomID("uptick-b37eb5464b45a8097cbbb7c22727a6b259a3d85e"))
	require.NoError(t, ValidateDenomID("uptick-uptick1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tpwxqergd3c8g7rusqqwmr9x"))

	// Exactly at the length bound passes; one byte over fails.
	atBound := "uptick-" + strings.Repeat("a", MaxDenomLen-len("uptick-"))
	require.Len(t, atBound, MaxDenomLen)
	require.NoError(t, ValidateDenomID(atBound))
	require.Error(t, ValidateDenomID(atBound+"a"))

	// Punctuation, whitespace and control characters are not part of any
	// module-derived shape.
	for _, bad := range []string{
		"uptick-has space",
		"uptick-line\nbreak",
		"uptick-tab\there",
		"uptick-colon:",
		"uptick-semi;colon",
		"uptick-plus+sign",
		"uptick-emoji\U0001F600",
	} {
		require.Error(t, ValidateDenomID(bad), "expected %q to be rejected", bad)
	}
}

func TestValidateKeywordsAndIsIBCDenom(t *testing.T) {
	require.Error(t, ValidateKeywords("ibc-token"))
	require.NoError(t, ValidateKeywords("custom-token"))
	require.True(t, IsIBCDenom("ibc/ABCDEF"))
	require.False(t, IsIBCDenom("denom"))
}

// TestEditNFTDataRemoveFieldSemantics pins the explicit-clear path for the
// JSON-valued Data field: the "[remove]" sentinel is not itself valid JSON, so
// ValidateBasic must exempt it from the JSON check — otherwise clearing Data
// would be impossible.
func TestEditNFTDataRemoveFieldSemantics(t *testing.T) {
	sender := sdk.AccAddress([]byte("remove-sender-addr")).String()

	edit := func(data string) *MsgEditNFT {
		return &MsgEditNFT{
			Id:      "nft1",
			DenomId: "denom1",
			Data:    data,
			Sender:  sender,
		}
	}

	// Explicit clear must pass validation.
	require.NoError(t, edit(RemoveField).ValidateBasic())
	// Plain JSON replaces.
	require.NoError(t, edit(`{"k":"v"}`).ValidateBasic())
	// Empty means keep.
	require.NoError(t, edit("").ValidateBasic())
	// Other non-JSON values stay rejected.
	require.Error(t, edit("not-json").ValidateBasic())

	// The sentinel actually clears through Modify.
	require.Equal(t, "", Modify(`{"old":true}`, RemoveField))
	// Empty keeps the origin.
	require.Equal(t, `{"old":true}`, Modify(`{"old":true}`, ""))
}
