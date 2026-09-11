package legacy

import (
	"bytes"
	"math"
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestLegacyERC20ProposalTypeNames(t *testing.T) {
	require.Equal(t, "uptick.erc20.v1.RegisterCoinProposal", proto.MessageName(&RegisterCoinProposal{}))
	require.Equal(t, "uptick.erc20.v1.RegisterERC20Proposal", proto.MessageName(&RegisterERC20Proposal{}))
	require.Equal(t, "uptick.erc20.v1.ToggleTokenRelayProposal", proto.MessageName(&ToggleTokenRelayProposal{}))
	require.Equal(t, "uptick.erc20.v1.UpdateTokenPairERC20Proposal", proto.MessageName(&UpdateTokenPairERC20Proposal{}))
}

func TestLegacyERC20ProposalRoundTrip(t *testing.T) {
	coin := &RegisterCoinProposal{
		Title:       "register coin",
		Description: "legacy coin registration",
		Metadata: banktypes.Metadata{
			Description: "test",
			Base:        "testcoin",
			Display:     "testcoin",
			Name:        "TestCoin",
			Symbol:      "TST",
			DenomUnits: []*banktypes.DenomUnit{
				{Denom: "testcoin", Exponent: 0, Aliases: nil},
			},
		},
	}
	bz, err := proto.Marshal(coin)
	require.NoError(t, err)
	var coinGot RegisterCoinProposal
	require.NoError(t, proto.Unmarshal(bz, &coinGot))
	require.Equal(t, *coin, coinGot)

	erc20 := &RegisterERC20Proposal{
		Title:        "register erc20",
		Description:  "legacy erc20 registration",
		Erc20Address: "0x55d3302a0e779ea4782d8bc5ea3d33cf0af3a7bd",
	}
	bz, err = proto.Marshal(erc20)
	require.NoError(t, err)
	var erc20Got RegisterERC20Proposal
	require.NoError(t, proto.Unmarshal(bz, &erc20Got))
	require.Equal(t, *erc20, erc20Got)

	toggle := &ToggleTokenRelayProposal{
		Title:       "toggle relay",
		Description: "legacy toggle",
		Token:       "testcoin",
	}
	bz, err = proto.Marshal(toggle)
	require.NoError(t, err)
	var toggleGot ToggleTokenRelayProposal
	require.NoError(t, proto.Unmarshal(bz, &toggleGot))
	require.Equal(t, *toggle, toggleGot)

	update := &UpdateTokenPairERC20Proposal{
		Title:           "update pair",
		Description:     "legacy update",
		Erc20Address:    "0x1111111111111111111111111111111111111111",
		NewErc20Address: "0x2222222222222222222222222222222222222222",
	}
	bz, err = proto.Marshal(update)
	require.NoError(t, err)
	var updateGot UpdateTokenPairERC20Proposal
	require.NoError(t, proto.Unmarshal(bz, &updateGot))
	require.Equal(t, *update, updateGot)
}

// The codec above is hand-written -- there is no .proto left to generate from,
// because v0.4.0 deleted the module that owned the original -- and the round
// trip test only feeds it bytes it produced itself. The input is not
// necessarily ours: app/params/proto.go calls v041.RegisterCompatInterfaces,
// which registers all four types as govv1beta1.Content implementations on the
// *application* registry, so any Any value carrying one of these type URLs is
// decoded by this parser. That includes a genesis file -- uptickd genesis
// validate-genesis, or start, on a file someone else produced.
//
// So the property under test is: malformed bytes are rejected with an error,
// never with a panic. A panic here is not a failed assertion, it is a node that
// cannot start or cannot export, and it reports a Go stack trace instead of
// saying which field was wrong.
func TestUnmarshalRejectsMalformedInputWithoutPanicking(t *testing.T) {
	lengthPrefixed := func(length uint64) []byte {
		// 0x0a is field 1 (Title) with wire type 2 for every type here.
		return appendLegacyProposalVarint([]byte{0x0a}, length)
	}

	for _, tc := range []struct {
		name  string
		input []byte
	}{
		// A varint that never terminates must stop at the 10-byte width limit
		// instead of running to the end of the buffer. The old loop kept
		// shifting past 64 bits (which Go defines as zero, so it did not trap)
		// and swallowed this input as a valid length of 0.
		{"terminated but 11-byte length varint",
			append(append([]byte{0x0a}, bytes.Repeat([]byte{0x80}, 10)...), 0x01)},
		{"truncated tag varint", []byte{0x80}},
		{"truncated length varint", []byte{0x0a, 0x80}},

		// The two below are why this test exists. int(2^63) is negative, so
		// the old `i+int(length) > len(dAtA)` guard passed, and the
		// dAtA[i:i+int(length)] that followed panicked. int(math.MaxInt64)
		// overflows the same addition.
		{"declared length is 2^63", lengthPrefixed(uint64(1) << 63)},
		{"declared length is math.MaxInt64", lengthPrefixed(math.MaxInt64)},
		{"declared length is 2^64-1", lengthPrefixed(math.MaxUint64)},

		{"declared length exceeds the buffer", []byte{0x0a, 0x05, 'a'}},
		{"illegal wire type 7", []byte{0x0f}},
		{"illegal wire type 6", []byte{0x0e}},
		{"end-group wire type 4", []byte{0x0c}},
		{"length-delimited skip beyond the buffer", []byte{0xa2, 0x01, 0x7f}},
		{"fixed32 skip beyond the buffer", []byte{0xa5, 0x01, 0x01}},
		{"fixed64 skip beyond the buffer", []byte{0xa1, 0x01, 0x01}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			require.NotPanics(t, func() {
				var p UpdateTokenPairERC20Proposal
				err = p.Unmarshal(tc.input)
			}, "a malformed Any value must be rejected, not crash the process")
			require.Error(t, err, "malformed input must be rejected")
		})
	}
}
