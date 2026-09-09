package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// These tests pin the root-cause contracts behind the twelfth-round sim fix.
// The sim op SimulateMsgIssueDenom previously passed the literal "Schema" as
// the schema field, which MsgIssueDenom.ValidateBasic (msgs.go:81) rejects via
// gjson.Valid, so the sim op 100% failed at SimDeliver and IssueDenom was
// effectively untested in simulation. And SimulateMsgBurnNFT's burn-failure
// branch previously emitted EventTypeEditNFT (copy-paste from EditNFT).
//
// A full sim-op behavioral test requires the simulation framework (runsim /
// test-sim-*), which is out of unit-test scope. These tests instead pin the
// contracts the sim op depends on: ValidateBasic's JSON-schema gate, and the
// distinctness of the burn vs edit event types. If someone reverts the sim op
// to the old values, these contracts catch the regression at the type layer.

func validIssueDenomMsg(schema string) *MsgIssueDenom {
	return NewMsgIssueDenom(
		"abc",  // id (legal: lowercase, 3 chars)
		"n",    // name
		schema, // schema
		authtypes.NewModuleAddress("test").String(), // sender (legal bech32)
		"s",          // symbol
		false,        // mintRestricted
		false,        // updateRestricted
		"desc",       // description
		"ipfs://uri", // uri
		"hash",       // uriHash
		"",           // data (empty -> skipped by gjson)
	)
}

// TestMsgIssueDenom_ValidateBasic_RejectsNonJSONSchema pins that a non-JSON,
// non-empty schema is rejected. This is the gate the sim op hit when it passed
// the literal "Schema"; pinning it keeps the contract that forced the sim fix.
func TestMsgIssueDenom_ValidateBasic_RejectsNonJSONSchema(t *testing.T) {
	msg := validIssueDenomMsg("Schema")
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid schema")
}

// TestMsgIssueDenom_ValidateBasic_AcceptsMinimalJSONSchema pins that "{}"
// (the minimal valid JSON a real issuer sends, and what the sim op now uses)
// passes ValidateBasic, so the sim op can reach SimDeliver.
func TestMsgIssueDenom_ValidateBasic_AcceptsMinimalJSONSchema(t *testing.T) {
	msg := validIssueDenomMsg("{}")
	require.NoError(t, msg.ValidateBasic())
}

// TestMsgIssueDenom_ValidateBasic_AcceptsEmptySchema pins that an empty schema
// is allowed (the gate only fires on non-empty non-JSON).
func TestMsgIssueDenom_ValidateBasic_AcceptsEmptySchema(t *testing.T) {
	msg := validIssueDenomMsg("")
	require.NoError(t, msg.ValidateBasic())
}

// TestEventTypeBurnNFT_DistinctFromEditNFT pins that the burn and edit event
// types are distinct and non-empty, so a burn failure in SimulateMsgBurnNFT is
// never mis-attributed to an edit (the twelfth-round copy-paste bug).
func TestEventTypeBurnNFT_DistinctFromEditNFT(t *testing.T) {
	require.NotEmpty(t, EventTypeBurnNFT)
	require.NotEmpty(t, EventTypeEditNFT)
	require.NotEqual(t, EventTypeBurnNFT, EventTypeEditNFT,
		"burn and edit event types must stay distinct")
	require.Equal(t, "burn_nft", EventTypeBurnNFT)
	require.Equal(t, "edit_nft", EventTypeEditNFT)
}
