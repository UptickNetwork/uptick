package cli

import (
	"cosmossdk.io/math"
	"fmt"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channelutils "github.com/cosmos/ibc-go/v8/modules/core/04-channel/client/utils"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	//govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/ethereum/go-ethereum/common"
	ethermint "github.com/evmos/ethermint/types"

	"github.com/UptickNetwork/uptick/x/erc20/types"

	"time"
)

// NewTxCmd returns a root CLI command handler for certain modules/erc20 transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "erc20 subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewConvertCoinCmd(),
		NewConvertERC20Cmd(),
		NewTransferERC20Cmd(),
	)
	return txCmd
}

// NewConvertCoinCmd returns a CLI command handler for converting cosmos coins
func NewConvertCoinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-coin [coin] [receiver_hex]",
		Short: "Convert a Cosmos coin to ERC20",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			var receiver string
			sender := cliCtx.GetFromAddress()

			if len(args) == 2 {
				receiver = args[1]
				if err := ethermint.ValidateAddress(receiver); err != nil {
					return fmt.Errorf("invalid receiver hex address %w", err)
				}
			} else {
				receiver = common.BytesToAddress(sender).Hex()
			}

			msg := &types.MsgConvertCoin{
				Coin:     coin,
				Receiver: receiver,
				Sender:   sender.String(),
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

// NewConvertERC20Cmd returns a CLI command handler for converting ERC20s
func NewConvertERC20Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-erc20 [contract-address] [amount] [receiver]",
		Short: "Convert an ERC20 token to Cosmos coin",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contract := args[0]
			if err := ethermint.ValidateAddress(contract); err != nil {
				return fmt.Errorf("invalid ERC20 contract address %w", err)
			}

			amount, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount %s", args[1])
			}

			//from := common.BytesToAddress(cliCtx.GetFromAddress().Bytes())
			from := cliCtx.GetFromAddress()
			receiver := cliCtx.GetFromAddress()
			if len(args) == 3 {
				receiver, err = sdk.AccAddressFromBech32(args[2])
				if err != nil {
					return err
				}
			}

			msg := &types.MsgConvertERC20{
				ContractAddress: contract,
				Amount:          amount,
				Receiver:        receiver.String(),
				Sender:          from.String(),
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

// NewRegisterCoinProposalCmd implements the command to submit a community-pool-spend proposal
func NewRegisterCoinProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-coin [metadata]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a register coin proposal",
		Long: `Submit a proposal to register a Cosmos coin to the erc20 along with an initial deposit.
Upon passing, the
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal register-coin <path/to/metadata.json> --from=<key_or_address>

Where metadata.json contains (example):

{
  "description": "staking, gas and governance token of the Uptick testnets"
  "denom_units": [
		{
			"denom": "auptick",
			"exponent": 0,
			"aliases": ["atto uptick"]
		},
		{
			"denom": "uptick",
			"exponent": 18
		}
	],
	"base": "auptick",
	"display: "uptick",
	"name": "Uptick",
	"symbol": "UPTICK"
}`, version.AppName,
		),
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

			metadata, err := ParseMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			content := types.NewRegisterCoinProposal(title, description, metadata)

			msg, err := govv1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			//if err := msg.ValidateBasic(); err != nil {
			//	return err
			//}

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

// NewRegisterERC20ProposalCmd implements the command to submit a community-pool-spend proposal
func NewRegisterERC20ProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "register-erc20 [erc20-address]",
		Args:    cobra.ExactArgs(1),
		Short:   "Submit a proposal to register an ERC20 token",
		Long:    "Submit a proposal to register an ERC20 token to the erc20 along with an initial deposit.",
		Example: fmt.Sprintf("$ %s tx gov submit-proposal register-erc20 <path/to/proposal.json> --from=<key_or_address>", version.AppName),
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

			erc20Addr := args[0]
			from := clientCtx.GetFromAddress()
			content := types.NewRegisterERC20Proposal(title, description, erc20Addr)

			msg, err := govv1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			//if err := msg.ValidateBasic(); err != nil {
			//	return err
			//}

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

// NewToggleTokenRelayProposalCmd implements the command to submit a community-pool-spend proposal
func NewToggleTokenRelayProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "toggle-token-relay [token]",
		Args:    cobra.ExactArgs(1),
		Short:   "Submit a toggle token relay proposal",
		Long:    "Submit a proposal to toggle the relaying of a token pair along with an initial deposit.",
		Example: fmt.Sprintf("$ %s tx gov submit-proposal toggle-token-relay <denom_or_contract> --from=<key_or_address>", version.AppName),
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
			content := types.NewToggleTokenRelayProposal(title, description, token)

			msg, err := govv1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			//if err := msg.ValidateBasic(); err != nil {
			//	return err
			//}

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

// NewUpdateTokenPairERC20ProposalCmd implements the command to submit a community-pool-spend proposal
func NewUpdateTokenPairERC20ProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-token-pair-erc20 [erc20_address] [new_erc20_address]",
		Args:    cobra.ExactArgs(2),
		Short:   "Submit a update token pair ERC20 proposal",
		Long:    `Submit a proposal to update the ERC20 address of a token pair along with an initial deposit.`,
		Example: fmt.Sprintf("$ %s tx gov submit-proposal update-token-pair-erc20 <path/to/proposal.json> --from=<key_or_address>", version.AppName),
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

			erc20Addr := args[0]
			newERC20Addr := args[1]

			from := clientCtx.GetFromAddress()
			content := types.NewUpdateTokenPairERC20Proposal(title, description, erc20Addr, newERC20Addr)
			msg, err := govv1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			//if err := msg.ValidateBasic(); err != nil {
			//	return err
			//}

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

const (
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	flagPacketMemo             = "packet-memo"
	flagAbsoluteTimeouts       = "absolute-timeouts"
)

// NewTransferERC20Cmd returns a CLI command handler for converting an erc20
func NewTransferERC20Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ibc-transfer-erc20  [evm_contract_address] [amount] [src_port] [src_channel] [cosmos_receiver] ",
		Short: "Convert an erc20 token to Cosmos coin and transfer a non-fungible token through IBC " +
			"When the receiver [optional] is omitted, the Cosmos coins are transferred to the sender.",
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			//evmSender := common.BytesToAddress(cliCtx.GetFromAddress().Bytes())
			sender := cliCtx.GetFromAddress().String()
			evmContractAddress := args[0]
			if err := ethermint.ValidateAddress(evmContractAddress); err != nil {
				return fmt.Errorf("invalid erc20 contract address %w", err)
			}

			// amount := args[1]
			amount, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount %s", args[1])
			}
			sourcePort := args[2]
			sourceChannel := args[3]
			cosmosReceiver := args[4]

			timeoutHeightStr, err := cmd.Flags().GetString(flagPacketTimeoutHeight)
			if err != nil {
				return err
			}
			timeoutHeight, err := clienttypes.ParseHeight(timeoutHeightStr)
			if err != nil {
				return err
			}

			timeoutTimestamp, err := cmd.Flags().GetUint64(flagPacketTimeoutTimestamp)
			if err != nil {
				return err
			}

			absoluteTimeouts, err := cmd.Flags().GetBool(flagAbsoluteTimeouts)
			if err != nil {
				return err
			}

			memo, err := cmd.Flags().GetString(flagPacketMemo)
			if err != nil {
				return err
			}

			// if the timeouts are not absolute, retrieve latest block height and block timestamp
			// for the consensus state connected to the destination port/channel
			if !absoluteTimeouts {
				consensusState, height, _, err := channelutils.QueryLatestConsensusState(cliCtx, sourcePort, sourceChannel)
				if err != nil {
					return err
				}

				if !timeoutHeight.IsZero() {
					absoluteHeight := height
					absoluteHeight.RevisionNumber += timeoutHeight.RevisionNumber
					absoluteHeight.RevisionHeight += timeoutHeight.RevisionHeight
					timeoutHeight = absoluteHeight
				}

				if timeoutTimestamp != 0 {
					// use local clock time as reference time if it is later than the
					// consensus state timestamp of the counter party chain, otherwise
					// still use consensus state timestamp as reference
					now := time.Now().UnixNano()
					consensusStateTimestamp := consensusState.GetTimestamp()
					if now > 0 {
						now := uint64(now)
						if now > consensusStateTimestamp {
							timeoutTimestamp = now + timeoutTimestamp
						} else {
							timeoutTimestamp = consensusStateTimestamp + timeoutTimestamp
						}
					} else {
						// return errors.New("local clock time is not greater than Jan 1st, 1970 12:00 AM")
						return fmt.Errorf("tokenIDs cannot be empty")
					}
				}
			}

			msg := &types.MsgTransferERC20{
				EvmContractAddress: evmContractAddress,
				Amount:             amount,
				SourcePort:         sourcePort,
				SourceChannel:      sourceChannel,
				EvmSender:          sender,
				CosmosReceiver:     cosmosReceiver,
				TimeoutHeight:      timeoutHeight,
				TimeoutTimestamp:   timeoutTimestamp,
				Memo:               memo,
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagPacketTimeoutHeight, ibcnfttransfertypes.DefaultRelativePacketTimeoutHeight, "Packet timeout block height. The timeout is disabled when set to 0-0.")
	cmd.Flags().Uint64(flagPacketTimeoutTimestamp, ibcnfttransfertypes.DefaultRelativePacketTimeoutTimestamp, "Packet timeout timestamp in nanoseconds from now. Default is 10 minutes. The timeout is disabled when set to 0.")
	cmd.Flags().String(flagPacketMemo, "", "Packet memo. Default is empty")
	cmd.Flags().Bool(flagAbsoluteTimeouts, false, "Timeout flags are used as absolute timeouts.")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
