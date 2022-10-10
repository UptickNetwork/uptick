package client

import (
	"github.com/UptickNetwork/uptick/x/erc20/client/cli"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var (
	RegisterCoinProposalHandler         = govclient.NewProposalHandler(cli.NewRegisterCoinProposalCmd)
	RegisterERC20ProposalHandler        = govclient.NewProposalHandler(cli.NewRegisterERC20ProposalCmd)
	ToggleTokenRelayProposalHandler     = govclient.NewProposalHandler(cli.NewToggleTokenRelayProposalCmd)
)

