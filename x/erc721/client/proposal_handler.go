package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"github.com/UptickNetwork/uptick/x/erc721/client/cli"
)

var (
	RegisterNFTProposalHandler           = govclient.NewProposalHandler(cli.NewRegisterNFTProposalCmd)
	RegisterERC721ProposalHandler        = govclient.NewProposalHandler(cli.NewRegisterERC721ProposalCmd)
	ToggleTokenConversionProposalHandler = govclient.NewProposalHandler(cli.NewToggleTokenConversionProposalCmd)
)
