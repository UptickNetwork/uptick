package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethermint "github.com/evmos/ethermint/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
)

// NewTxCmd returns a root CLI command handler for erc721 transaction commands
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "erc721 subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewConvertNFTCmd(),
		NewConvertERC721Cmd(),
	)
	return txCmd
}

// NewConvertNFTCmd returns a CLI command handler for converting a Cosmos coin
func NewConvertNFTCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-nft [class_id] [nft_id] [contract_address] [token_id] [receiver_hex]",
		Short: "Convert a Cosmos nft to erc721. When the receiver [optional] is omitted, the erc721 tokens are transferred to the sender.",
		Args:  cobra.RangeArgs(4, 5),
		RunE: func(cmd *cobra.Command, args []string) error {

			fmt.Printf("xxl 02 NewConvertNFTCmd 000 start \n")
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			classID := args[0]
			if len(classID) == 0 {
				return fmt.Errorf("classId can not be empty")
			}

			nftID := args[1]
			if len(nftID) == 0 {
				return fmt.Errorf("classId can not be empty")
			}

			contractAddress := args[2]
			//fmt.Printf("xxl 02 NewConvertERC721Cmd 001 %v \n", contractAddress)
			//if len(contractAddress) == 0 {
			//	fmt.Printf("xxl 02 NewConvertERC721Cmd come to empty \n")
			//	contractAddress = types.CreateContractAddressFromClassID(classID)
			//}

			tokenID := args[3]
			//fmt.Printf("xxl 01 NewConvertERC721Cmd 001 classID : %v \n", tokenID)
			//if len(tokenID) == 0 {
			//	fmt.Printf("xxl 01 NewConvertERC721Cmd 002 %v \n", tokenID)
			//	tokenID = types.CreateTokenIDFromNFTID(nftID)
			//}

			var receiver string
			sender := cliCtx.GetFromAddress()
			if len(args) == 5 {
				receiver = args[4]
				if err := ethermint.ValidateAddress(receiver); err != nil {
					return fmt.Errorf("invalid receiver hex address %w", err)
				}
			} else {
				receiver = common.BytesToAddress(sender).Hex()
			}

			msg := &types.MsgConvertNFT{
				ContractAddress: contractAddress,
				NftId:           nftID,
				ClassId:         classID,
				TokenId:         tokenID,
				Receiver:        receiver,
				Sender:          sender.String(),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			fmt.Printf("xxl 02 NewConvertNFTCmd 002 msg %v \n", msg)

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewConvertERC721Cmd returns a CLI command handler for converting an erc721
func NewConvertERC721Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "convert-erc721 [contract_address] [token_id] [class_id] [nft_id] [receiver]",
		Short: "Convert an erc721 token to Cosmos coin.  " +
			"When the receiver [optional] is omitted, the Cosmos coins are transferred to the sender.",
		Args: cobra.RangeArgs(4, 5),
		RunE: func(cmd *cobra.Command, args []string) error {

			fmt.Printf("xxl 01 NewConvertERC721Cmd 000 start \n")
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contractAddress := args[0]
			if err := ethermint.ValidateAddress(contractAddress); err != nil {
				return fmt.Errorf("invalid erc721 contract address %w", err)
			}

			tokenID := args[1]
			if len(tokenID) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}

			from := common.BytesToAddress(cliCtx.GetFromAddress().Bytes())

			classID := args[2]
			//fmt.Printf("xxl 01 NewConvertERC721Cmd 001 classID : %v \n", classID)
			//if len(classID) == 0 {
			//	fmt.Printf("xxl 01 NewConvertERC721Cmd 002 %v \n", classID)
			//	classID = types.CreateClassIDFromContractAddress(contractAddress)
			//}

			nftID := args[3]
			//fmt.Printf("xxl 01 NewConvertERC721Cmd 001 nftID : %v \n", nftID)
			//if len(nftID) == 0 {
			//	fmt.Printf("xxl 01 NewConvertERC721Cmd 002 %v \n", tokenID)
			//	nftID = types.CreateNFTIDFromTokenID(tokenID)
			//}

			receiver := cliCtx.GetFromAddress()
			if len(args) == 5 {
				receiver, err = sdk.AccAddressFromBech32(args[4])
				if err != nil {
					return err
				}
			}

			msg := &types.MsgConvertERC721{
				ContractAddress: contractAddress,
				TokenId:         tokenID,
				Receiver:        receiver.String(),
				Sender:          from.Hex(),
				ClassId:         classID,
				NftId:           nftID,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			fmt.Printf("xxl 01 NewConvertERC721Cmd 003 msg,%v \n", msg)
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
