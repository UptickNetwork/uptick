package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/x/nft"
	"github.com/cosmos/cosmos-sdk/x/nft/client/cli"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	nftQueryCmd := &cobra.Command{
		Use:                        nft.ModuleName,
		Short:                      "Querying commands for the nft module",
		Long:                       "",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	nftQueryCmd.AddCommand(
		cli.GetCmdQueryClass(),
		cli.GetCmdQueryClasses(),
		cli.GetCmdQueryNFT(),
		cli.GetCmdQueryNFTs(),
		cli.GetCmdQueryOwner(),
		cli.GetCmdQueryBalance(),
		cli.GetCmdQuerySupply(),
	)
	return nftQueryCmd
}
