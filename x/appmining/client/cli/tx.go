package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

// NewTxCmd returns the transaction commands for this module.
func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Application mining transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdRegisterApplication(),
		CmdFinalizeAttestation(),
		CmdConsumeGasCredit(),
		CmdFundRewardPool(),
	)
	return cmd
}

func CmdRegisterApplication() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-application [app-id-hex] [owner] [submitter]",
		Short: "Register an application (authority signer)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("app-id-hex: %w", err)
			}
			msg := &types.MsgRegisterApplication{
				Authority:            clientCtx.GetFromAddress().String(),
				AppId:                appID,
				Owner:                args[1],
				AttestationSubmitter: args[2],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdFinalizeAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finalize-attestation [app-id-hex] [epoch]",
		Short: "Finalize an attestation after the challenge window",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}
			epoch, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			msg := &types.MsgFinalizeAttestation{
				Caller: clientCtx.GetFromAddress().String(),
				AppId:  appID,
				Epoch:  epoch,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdConsumeGasCredit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consume-gas-credit [app-id-hex] [amount] [source-tx-hash-hex]",
		Short: "Consume application gas credit bound to a source tx",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}
			source, err := hex.DecodeString(args[2])
			if err != nil {
				return err
			}
			msg := &types.MsgConsumeGasCredit{
				GasPayer:     clientCtx.GetFromAddress().String(),
				AppId:        appID,
				Amount:       args[1],
				SourceTxHash: source,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdFundRewardPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-reward-pool [denom] [amount]",
		Short: "Fund the appmining reward pool (authority signer)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgFundRewardPool{
				Authority: clientCtx.GetFromAddress().String(),
				Denom:     args[0],
				Amount:    args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
