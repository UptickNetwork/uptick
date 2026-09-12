package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	"github.com/UptickNetwork/uptick/x/cw721/types"
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

			if args[1] == "" {
				return fmt.Errorf("nft IDs can not be empty")
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
			if len(contractAddress) == 0 {
				return fmt.Errorf("contract address can not be empty")
			}

			if args[1] == "" {
				return fmt.Errorf("token IDs can not be empty")
			}
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

// defaultPacketTimeout is the fallback applied by the CLI when both timeout
// flags are left at their zero defaults: a 10 minute relative timestamp,
// mirroring the pre-migration behavior of ibc-go's transfer CLI.
const defaultPacketTimeout = 10 * time.Minute

// resolvePacketTimeouts fills in a relative timeout timestamp when both the
// timeout height and the timeout timestamp are zero. Without this fallback the
// default CLI invocation fails MsgTransferCW721.ValidateBasic ("timeout height
// and timeout timestamp cannot both be zero"). The msg-level guard is kept so
// that packets submitted directly on-chain cannot live forever.
func resolvePacketTimeouts(timeoutHeight clienttypes.Height, timeoutTimestamp uint64) (clienttypes.Height, uint64) {
	if timeoutHeight.IsZero() && timeoutTimestamp == 0 {
		return timeoutHeight, uint64(time.Now().Add(defaultPacketTimeout).UnixNano())
	}
	return timeoutHeight, timeoutTimestamp
}

// NewTransferCW721Cmd returns a CLI command handler for converting an cw721
func NewTransferCW721Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ibc-transfer-cw721 [cw_contract_address] [cw_token_ids] [src_port] [src_channel] [cosmos_receiver] [class_id] [cosmos_token_ids]",
		Short: "Convert an cw721 token to Cosmos coin and transfer a non-fungible token through IBC",
		Args:  cobra.ExactArgs(7),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cwSender := cliCtx.GetFromAddress()

			cwContractAddress := args[0]
			if cwContractAddress == "" {
				return fmt.Errorf("contract address can not be empty")
			}

			if args[1] == "" {
				return fmt.Errorf("wasm token IDs can not be empty")
			}
			cwTokenIds := strings.Split(args[1], ",")
			if len(cwTokenIds) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}

			sourcePort := args[2]
			if sourcePort == "" {
				return fmt.Errorf("source port address can not be empty")
			}

			sourceChannel := args[3]
			if sourceChannel == "" {
				return fmt.Errorf("source channel address can not be empty")
			}

			cosmosReceiver := args[4]
			if cosmosReceiver == "" {
				return fmt.Errorf("cosmos receiver can not be empty")
			}

			classId := args[5]
			cosmosTokenIds := strings.Split(args[6], ",")

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
				return fmt.Errorf("relative timeouts not supported after ibc-go v10 migration")
			}

			timeoutHeight, timeoutTimestamp = resolvePacketTimeouts(timeoutHeight, timeoutTimestamp)

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

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagPacketTimeoutHeight, "0-0", "Absolute packet timeout block height (revision-height). When both height and timestamp are zero, a 10 minute relative timeout timestamp is applied.")
	cmd.Flags().Uint64(flagPacketTimeoutTimestamp, 0, "Absolute packet timeout timestamp in nanoseconds since unix epoch. When both height and timestamp are zero, a 10 minute relative timeout timestamp is applied.")
	cmd.Flags().String(flagPacketMemo, "", "Packet memo. Default is empty")
	cmd.Flags().Bool(flagAbsoluteTimeouts, true, "Timeout flags are used as absolute timeouts (relative timeouts are not supported).")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
