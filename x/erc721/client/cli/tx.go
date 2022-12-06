package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

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
				return fmt.Errorf("classID can not be empty")
			}

			nftID := args[1]
			if len(nftID) == 0 {
				return fmt.Errorf("classID can not be empty")
			}

			contract := args[2]
			fmt.Printf("xxl 02 NewConvertERC721Cmd 001 %v \n", contract)
			if len(contract) == 0 {
				fmt.Printf("xxl 02 NewConvertERC721Cmd come to empty \n")
				contract = types.CreateContractAddressFromClassID(classID)
			}

			tokenID := args[3]
			fmt.Printf("xxl 01 NewConvertERC721Cmd 001 classID : %v \n", tokenID)
			if len(tokenID) == 0 {
				fmt.Printf("xxl 01 NewConvertERC721Cmd 002 %v \n", tokenID)
				tokenID = types.CreateTokenIDFromNFTID(nftID)
			}

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
				ContractAddress: contract,
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

			contract := args[0]
			if err := ethermint.ValidateAddress(contract); err != nil {
				return fmt.Errorf("invalid erc721 contract address %w", err)
			}

			tokenID := args[1]
			if len(tokenID) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}

			from := common.BytesToAddress(cliCtx.GetFromAddress().Bytes())

			classID := args[2]
			fmt.Printf("xxl 01 NewConvertERC721Cmd 001 classID : %v \n", classID)
			if len(classID) == 0 {
				fmt.Printf("xxl 01 NewConvertERC721Cmd 002 %v \n", classID)
				classID = types.CreateClassIDFromContractAddress(contract)
			}

			nftID := args[3]
			fmt.Printf("xxl 01 NewConvertERC721Cmd 001 nftID : %v \n", nftID)
			if len(nftID) == 0 {
				fmt.Printf("xxl 01 NewConvertERC721Cmd 002 %v \n", tokenID)
				nftID = types.CreateNFTIDFromTokenID(tokenID)
			}

			receiver := cliCtx.GetFromAddress()
			if len(args) == 5 {
				receiver, err = sdk.AccAddressFromBech32(args[4])
				if err != nil {
					return err
				}
			}

			msg := &types.MsgConvertERC721{
				ContractAddress: contract,
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

// NewRegisterNFTProposalCmd implements the command to submit a community-pool-spend proposal
func NewRegisterNFTProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-nft [class]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a register nft proposal",
		Long: `Submit a proposal to register a Cosmos nft to the erc721 along with an initial deposit.
Upon passing, the
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal register-nft <path/to/class.json> --from=<key_or_address>`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			class, err := ParseClass(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			content := types.NewRegisterNFTProposal(title, description, class)

			msg, err := gov.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "1auptick", "deposit of proposal")
	if err := cmd.MarkFlagRequired(cli.FlagTitle); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(cli.FlagDescription); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(cli.FlagDeposit); err != nil {
		panic(err)
	}
	return cmd
}

// NewRegisterERC721ProposalCmd implements the command to submit a community-pool-spend proposal
func NewRegisterERC721ProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "register-erc721 [erc721-address]",
		Args:    cobra.ExactArgs(1),
		Short:   "Submit a proposal to register an erc721 token",
		Long:    "Submit a proposal to register an erc721 token to the erc721 along with an initial deposit.",
		Example: fmt.Sprintf("$ %s tx gov submit-proposal register-erc721 <contract_address> --from=<key_or_address>", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			erc721Addr := args[0]
			from := clientCtx.GetFromAddress()
			content := types.NewRegisterERC721Proposal(title, description, erc721Addr)

			msg, err := gov.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "1auptick", "deposit of proposal")
	if err := cmd.MarkFlagRequired(cli.FlagTitle); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(cli.FlagDescription); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(cli.FlagDeposit); err != nil {
		panic(err)
	}
	return cmd
}

// NewToggleTokenConversionProposalCmd implements the command to submit a community-pool-spend proposal
func NewToggleTokenConversionProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "toggle-token-conversion [token]",
		Args:    cobra.ExactArgs(1),
		Short:   "Submit a toggle token conversion proposal",
		Long:    "Submit a proposal to toggle the conversion of a token pair along with an initial deposit.",
		Example: fmt.Sprintf("$ %s tx gov submit-proposal toggle-token-conversion <denom_or_contract> --from=<key_or_address>", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			token := args[0]
			content := types.NewToggleTokenConversionProposal(title, description, token)

			msg, err := gov.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "1aevmos", "deposit of proposal")
	if err := cmd.MarkFlagRequired(cli.FlagTitle); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(cli.FlagDescription); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(cli.FlagDeposit); err != nil {
		panic(err)
	}
	return cmd
}
