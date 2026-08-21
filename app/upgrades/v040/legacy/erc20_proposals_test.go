package legacy

import (
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
