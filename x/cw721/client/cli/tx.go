package cli

import (
	"fmt"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channelutils "github.com/cosmos/ibc-go/v7/modules/core/04-channel/client/utils"
	"strings"

	"github.com/spf13/cobra"

	"github.com/UptickNetwork/uptick/x/cw721/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"time"
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
		NewTransferCW721Cmd(),
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

const (
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	flagPacketMemo             = "packet-memo"
	flagAbsoluteTimeouts       = "absolute-timeouts"
)

// NewTransferCW721Cmd returns a CLI command handler for converting an cw721
func NewTransferCW721Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ibc-transfer-cw721 [cw_contract_address] [cw_token_ids] [src_port] [src_channel] [cosmos_receiver] [class_id] [cosmos_token_ids]",
		Short: "Convert an cw721 token to Cosmos coin and transfer a non-fungible token through IBC " +
			"When the receiver [optional] is omitted, the Cosmos coins are transferred to the sender.",
		Args: cobra.RangeArgs(7, 8),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cwSender := cliCtx.GetFromAddress()

			cwContractAddress := args[0]
			cwTokenIds := strings.Split(args[1], ",")
			if len(cwTokenIds) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}
			sourcePort := args[2]
			sourceChannel := args[3]
			cosmosReceiver := args[4]
			classId := args[5]
			cosmosTokenIds := strings.Split(args[6], ",")

			if len(cwTokenIds) == 0 {
				return fmt.Errorf("tokenIDs cannot be empty")
			}

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

			msg := &types.MsgTransferCW721{
				CwContractAddress: cwContractAddress,
				CwTokenIds:        cwTokenIds,
				SourcePort:        sourcePort,
				SourceChannel:     sourceChannel,
				ClassId:           classId,
				CosmosTokenIds:    cosmosTokenIds,
				CwSender:          cwSender.String(),
				CosmosReceiver:    cosmosReceiver,
				TimeoutHeight:     timeoutHeight,
				TimeoutTimestamp:  timeoutTimestamp,
				Memo:              memo,
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
