package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/UptickNetwork/uptick/x/cw721/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewTxCmd returns a root CLI command handler for cw721 transaction commands
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "cw721 subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewConvertNFTCmd(),
		NewConvertCW721Cmd(),
	)
	return txCmd
}

// NewConvertNFTCmd returns a CLI command handler for converting a Cosmos coin
func NewConvertNFTCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-nft [class_id] [nft_ids] [contract_address] [token_ids] [receiver_hex]",
		Short: "Convert a Cosmos nft to cw721. When the receiver [optional] is omitted, the cw721 tokens are transferred to the sender.",
		Args:  cobra.RangeArgs(4, 5),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			classID := args[0]
			if len(classID) == 0 {
				return fmt.Errorf("classId can not be empty")
			}

			nftIDs := strings.Split(args[1], ",")
			if len(nftIDs) == 0 {
				return fmt.Errorf("nftID can not be empty")
			}

			contractAddress := args[2]
			tokenIDs := strings.Split(args[3], ",")

			var receiver string
			sender := cliCtx.GetFromAddress()
			if len(args) == 5 {
				receiver = args[4]
			} else {
				receiver = sender.String()
			}

			msg := &types.MsgConvertNFT{
				ContractAddress: contractAddress,
				NftIds:          nftIDs,
				ClassId:         classID,
				TokenIds:        tokenIDs,
				Receiver:        receiver,
				Sender:          sender.String(),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewConvertCW721Cmd returns a CLI command handler for converting an cw721
func NewConvertCW721Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "convert-cw721 [contract_address] [token_ids] [class_id] [nft_ids] [receiver]",
		Short: "Convert an cw721 token to Cosmos coin.  " +
			"When the receiver [optional] is omitted, the Cosmos coins are transferred to the sender.",
		Args: cobra.RangeArgs(4, 5),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contractAddress := args[0]
			tokenIDs := strings.Split(args[1], ",")
			if len(tokenIDs) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}

			// from := common.BytesToAddress(cliCtx.GetFromAddress().Bytes())
			from := cliCtx.GetFromAddress()

			classID := args[2]
			nftIDs := strings.Split(args[3], ",")

			receiver := cliCtx.GetFromAddress()
			if len(args) == 5 {
				receiver, err = sdk.AccAddressFromBech32(args[4])
				if err != nil {
					return err
				}
			}

			msg := &types.MsgConvertCW721{
				ContractAddress: contractAddress,
				TokenIds:        tokenIDs,
				Receiver:        receiver.String(),
				Sender:          from.String(),
				ClassId:         classID,
				NftIds:          nftIDs,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
