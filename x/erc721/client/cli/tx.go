package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethermint "github.com/evmos/ethermint/types"

	"github.com/UptickNetwork/uptick/x/erc721/types"
	ibcnfttransfertypes "github.com/bianjieai/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channelutils "github.com/cosmos/ibc-go/v7/modules/core/04-channel/client/utils"
	"time"
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
		NewTransferERC721Cmd(),
	)
	return txCmd
}

// NewConvertNFTCmd returns a CLI command handler for converting a Cosmos coin
func NewConvertNFTCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-nft [class_id] [cosmos_token_ids] [evm_contract_address] [evm_token_ids] [receiver_hex]",
		Short: "Convert a Cosmos nft to erc721. When the receiver [optional] is omitted, the erc721 tokens are transferred to the sender.",
		Args:  cobra.RangeArgs(4, 5),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			classId := args[0]
			if len(classId) == 0 {
				return fmt.Errorf("classId can not be empty")
			}

			cosmosTokenIds := strings.Split(args[1], ",")
			if len(cosmosTokenIds) == 0 {
				return fmt.Errorf("nftID can not be empty")
			}

			evmContractAddress := args[2]
			evmTokenIds := strings.Split(args[3], ",")

			var evmReceiver string
			cosmosSender := cliCtx.GetFromAddress()
			if len(args) == 5 {
				evmReceiver = args[4]
				if err := ethermint.ValidateAddress(evmReceiver); err != nil {
					return fmt.Errorf("invalid receiver hex address %w", err)
				}
			} else {
				evmReceiver = common.BytesToAddress(cosmosSender).Hex()
			}

			msg := &types.MsgConvertNFT{
				EvmContractAddress: evmContractAddress,
				CosmosTokenIds:     cosmosTokenIds,
				ClassId:            classId,
				EvmTokenIds:        evmTokenIds,
				EvmReceiver:        evmReceiver,
				CosmosSender:       cosmosSender.String(),
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

// NewConvertERC721Cmd returns a CLI command handler for converting an erc721
func NewConvertERC721Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "convert-erc721 [evm_contract_address] [evm_token_ids] [class_id] [cosmos_token_ids] [cosmos_receiver]",
		Short: "Convert an erc721 token to Cosmos coin.  " +
			"When the receiver [optional] is omitted, the Cosmos coins are transferred to the sender.",
		Args: cobra.RangeArgs(4, 5),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			evmContractAddress := args[0]
			if err := ethermint.ValidateAddress(evmContractAddress); err != nil {
				return fmt.Errorf("invalid erc721 contract address %w", err)
			}

			evmTokenIds := strings.Split(args[1], ",")
			if len(evmTokenIds) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}

			evmSender := common.BytesToAddress(cliCtx.GetFromAddress().Bytes())

			classId := args[2]
			cosmosTokenIds := strings.Split(args[3], ",")

			cosmosReceiver := cliCtx.GetFromAddress()
			if len(args) == 5 {
				cosmosReceiver, err = sdk.AccAddressFromBech32(args[4])
				if err != nil {
					return err
				}
			}

			msg := &types.MsgConvertERC721{
				EvmContractAddress: evmContractAddress,
				EvmTokenIds:        evmTokenIds,
				EvmSender:          evmSender.Hex(),
				CosmosReceiver:     cosmosReceiver.String(),
				ClassId:            classId,
				CosmosTokenIds:     cosmosTokenIds,
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

// NewTransferERC721Cmd returns a CLI command handler for converting an erc721
func NewTransferERC721Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ibc-transfer-erc721 [evm_contract_address] [evm_token_ids] [src_port] [src_channel] [cosmos_receiver] [class_id] [cosmos_token_ids]",
		Short: "Convert an erc721 token to Cosmos coin and transfer a non-fungible token through IBC " +
			"When the receiver [optional] is omitted, the Cosmos coins are transferred to the sender.",
		Args: cobra.RangeArgs(7, 8),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			evmSender := common.BytesToAddress(cliCtx.GetFromAddress().Bytes())

			evmContractAddress := args[0]
			if err := ethermint.ValidateAddress(evmContractAddress); err != nil {
				return fmt.Errorf("invalid erc721 contract address %w", err)
			}

			evmTokenIds := strings.Split(args[1], ",")
			if len(evmTokenIds) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}
			sourcePort := args[2]
			sourceChannel := args[3]
			cosmosReceiver := args[4]
			classId := args[5]
			cosmosTokenIds := strings.Split(args[6], ",")

			if len(evmTokenIds) == 0 {
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

			msg := &types.MsgTransferERC721{
				EvmContractAddress: evmContractAddress,
				EvmTokenIds:        evmTokenIds,
				SourcePort:         sourcePort,
				SourceChannel:      sourceChannel,
				ClassId:            classId,
				CosmosTokenIds:     cosmosTokenIds,
				EvmSender:          evmSender.Hex(),
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
