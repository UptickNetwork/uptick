package client

import (
	"github.com/cosmos/cosmos-sdk/client"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	cosrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"

	"github.com/UptickNetwork/uptick/x/erc20/client/cli"
	"github.com/UptickNetwork/uptick/x/erc20/client/rest"
)

func EmptyProposalRESTHandler(client.Context) cosrest.ProposalRESTHandler {
	return cosrest.ProposalRESTHandler{}
}

var (
	RegisterCoinProposalHandler         = govclient.NewProposalHandler(cli.NewRegisterCoinProposalCmd, rest.RegisterCoinProposalRESTHandler)
	RegisterERC20ProposalHandler        = govclient.NewProposalHandler(cli.NewRegisterERC20ProposalCmd, rest.RegisterERC20ProposalRESTHandler)
	ToggleTokenRelayProposalHandler     = govclient.NewProposalHandler(cli.NewToggleTokenRelayProposalCmd, rest.ToggleTokenRelayRESTHandler)
	UpdateTokenPairERC20ProposalHandler = govclient.NewProposalHandler(cli.NewUpdateTokenPairERC20ProposalCmd, rest.UpdateTokenPairERC20ProposalRESTHandler)
)
