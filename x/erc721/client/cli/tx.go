package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	upticktypes "github.com/UptickNetwork/uptick/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/UptickNetwork/uptick/x/erc721/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
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
		Long: "Convert a native Cosmos nft (x/collection) into an ERC721 token.\n" +
			"  - evm_contract_address: empty ('') triggers the module to auto-deploy an ERC721Uptick;\n" +
			"    pass an existing contract address to reuse it instead. A non-empty value must be a valid 0x address.\n" +
			"  - evm_token_ids: leave empty when auto-deploying (the module derives them from cosmos_token_ids).\n" +
			"  - receiver_hex: optional; if omitted the tokens are minted to the sender.",
		Args: cobra.RangeArgs(4, 5),
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
			// if evmContractAddress == "" {
			//	return fmt.Errorf("evm contract address can not be empty")
			//}

			// When evm token ids are omitted (first conversion / auto-deploy), leave
			// the slice empty so the module derives them from the cosmos token ids.
			// A bare strings.Split("", ",") would yield [""], which fails
			// ValidateEVMTokenID in ValidateBasic.
			var evmTokenIds []string
			if args[3] != "" {
				evmTokenIds = strings.Split(args[3], ",")
			}

			var evmReceiver string
			cosmosSender := cliCtx.GetFromAddress()
			if len(args) == 5 {
				evmReceiver = args[4]
				if err := upticktypes.ValidateAddress(evmReceiver); err != nil {
					return fmt.Errorf("invalid receiver hex address %w", err)
				}
			} else {
				evmReceiver = common.BytesToAddress(cosmosSender).Hex()
			}

			msg := &types.MsgConvertNFT{
				EvmContractAddress: strings.ToLower(evmContractAddress),
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
			if err := upticktypes.ValidateAddress(evmContractAddress); err != nil {
				return fmt.Errorf("invalid erc721 contract address %w", err)
			}

			if args[1] == "" {
				return fmt.Errorf("evm tokenids can not be empty")
			}
			evmTokenIds := strings.Split(args[1], ",")
			if len(evmTokenIds) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}

			cosmosSender := cliCtx.GetFromAddress()

			classId := args[2]
			// if classId == "" {
			//	return fmt.Errorf("classId can not be empty")
			//}

			// if args[3] == "" {
			//	return fmt.Errorf("cosmos tokenids can not be empty")
			//}
			cosmosTokenIds := strings.Split(args[3], ",")
			// if len(cosmosTokenIds) == 0 {
			//	return fmt.Errorf("cosmos Token ids can not be empty")
			//}

			cosmosReceiver := cliCtx.GetFromAddress()
			if len(args) == 5 {
				cosmosReceiver, err = sdk.AccAddressFromBech32(args[4])
				if err != nil {
					return err
				}
			}

			msg := &types.MsgConvertERC721{
				EvmContractAddress: strings.ToLower(evmContractAddress),
				EvmTokenIds:        evmTokenIds,
				CosmosSender:       cosmosSender.String(),
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

			cosmosSender := cliCtx.GetFromAddress()

			evmContractAddress := args[0]
			if err := upticktypes.ValidateAddress(evmContractAddress); err != nil {
				return fmt.Errorf("invalid erc721 contract address %w", err)
			}

			if args[1] == "" {
				return fmt.Errorf("cosmos tokenids can not be empty")
			}
			evmTokenIds := strings.Split(args[1], ",")
			if len(evmTokenIds) == 0 {
				return fmt.Errorf("tokenID can not be empty")
			}
			sourcePort := args[2]
			if sourcePort == "" {
				return fmt.Errorf("sourcePort can not be empty")
			}

			sourceChannel := args[3]
			if sourceChannel == "" {
				return fmt.Errorf("sourcePort can not be empty")
			}

			cosmosReceiver := args[4]
			if cosmosReceiver == "" {
				return fmt.Errorf("cosmos receiver can not be empty")
			}

			classId := args[5]
			// if classId == "" {
			//	return fmt.Errorf("classId can not be empty")
			//}

			// if args[6] == "" {
			//	return fmt.Errorf("cosmos tokenids can not be empty")
			//}
			cosmosTokenIds := strings.Split(args[6], ",")
			// if len(cosmosTokenIds) == 0 {
			//	return fmt.Errorf("cosmos token ids cannot be empty")
			//}

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

			msg := &types.MsgTransferERC721{
				EvmContractAddress: strings.ToLower(evmContractAddress),
				EvmTokenIds:        evmTokenIds,
				SourcePort:         sourcePort,
				SourceChannel:      sourceChannel,
				ClassId:            classId,
				CosmosTokenIds:     cosmosTokenIds,
				CosmosSender:       cosmosSender.String(),
				CosmosReceiver:     cosmosReceiver,
				TimeoutHeight:      timeoutHeight,
				TimeoutTimestamp:   timeoutTimestamp,
				Memo:               memo,
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagPacketTimeoutHeight, "0-0", "Absolute packet timeout block height (revision-height). Relative timeouts are not supported after ibc-go v10. The timeout is disabled when set to 0-0.")
	cmd.Flags().Uint64(flagPacketTimeoutTimestamp, 0, "Absolute packet timeout timestamp in nanoseconds since unix epoch. Relative timeouts are not supported after ibc-go v10. The timeout is disabled when set to 0.")
	cmd.Flags().String(flagPacketMemo, "", "Packet memo. Default is empty")
	cmd.Flags().Bool(flagAbsoluteTimeouts, false, "Timeout flags are used as absolute timeouts.")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
