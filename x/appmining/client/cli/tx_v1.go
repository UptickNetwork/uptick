package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/UptickNetwork/uptick/x/appmining/types"
)

func addV1TxCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		CmdOpenEntitlementPool(),
		CmdFundEntitlementPool(),
		CmdSetEntitlementPoolPaused(),
		CmdRegisterRoleKey(),
		CmdSetRoleKeyStatus(),
		CmdClaimEntitlement(),
	)
	addV1WireTxCommands(cmd)
}

func CmdOpenEntitlementPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open-entitlement-pool [pool-id] [denom] [version]",
		Short: "Open a V1 entitlement pool (authority)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			ver, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("version: %w", err)
			}
			msg := &types.MsgOpenEntitlementPool{
				Authority: clientCtx.GetFromAddress().String(),
				PoolId:    args[0],
				Denom:     args[1],
				Version:   ver,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdFundEntitlementPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-entitlement-pool [pool-id] [amount]",
		Short: "Fund a V1 entitlement pool (no mint)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgFundEntitlementPool{
				Funder: clientCtx.GetFromAddress().String(),
				PoolId: args[0],
				Amount: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSetEntitlementPoolPaused() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-entitlement-pool-paused [pool-id] [true|false]",
		Short: "Pause or unpause a V1 entitlement pool (authority)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			paused, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("paused: %w", err)
			}
			msg := &types.MsgSetEntitlementPoolPaused{
				Authority: clientCtx.GetFromAddress().String(),
				PoolId:    args[0],
				Paused:    paused,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRegisterRoleKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-role-key [role] [key-id] [pubkey-hex] [version]",
		Short: "Register a V1 Ed25519 role key (authority). role=publisher|approver|reviewer|risk|audit",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			role, err := parseEntitlementRole(args[0])
			if err != nil {
				return err
			}
			pk, err := hex.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("pubkey-hex: %w", err)
			}
			ver, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("version: %w", err)
			}
			msg := &types.MsgRegisterRoleKey{
				Authority: clientCtx.GetFromAddress().String(),
				Role:      role,
				KeyId:     args[1],
				Pubkey:    pk,
				Version:   ver,
			}
			if appHex, _ := cmd.Flags().GetString("app-id-hex"); appHex != "" {
				appID, err := hex.DecodeString(appHex)
				if err != nil {
					return fmt.Errorf("app-id-hex: %w", err)
				}
				msg.AppId = appID
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("app-id-hex", "", "optional 32-byte app binding (hex); empty = global key")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSetRoleKeyStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-role-key-status [role] [key-id] [active|paused|revoked]",
		Short: "Set V1 role key status (authority)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			role, err := parseEntitlementRole(args[0])
			if err != nil {
				return err
			}
			status, err := parseRoleKeyStatus(args[2])
			if err != nil {
				return err
			}
			msg := &types.MsgSetRoleKeyStatus{
				Authority: clientCtx.GetFromAddress().String(),
				Role:      role,
				KeyId:     args[1],
				Status:    status,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdClaimEntitlement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-entitlement [entitlement-id-hex]",
		Short: "Claim a V1 entitlement (must be current app owner)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgClaimEntitlement{
				Owner:         clientCtx.GetFromAddress().String(),
				EntitlementId: args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func parseEntitlementRole(s string) (types.EntitlementRole, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "publisher":
		return types.EntitlementRolePublisher, nil
	case "approver":
		return types.EntitlementRoleApprover, nil
	case "reviewer":
		return types.EntitlementRoleReviewer, nil
	case "risk":
		return types.EntitlementRoleRisk, nil
	case "audit":
		return types.EntitlementRoleAudit, nil
	default:
		return types.EntitlementRoleUnspecified, fmt.Errorf("role must be publisher|approver|reviewer|risk|audit")
	}
}

func parseRoleKeyStatus(s string) (types.RoleKeyStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "active":
		return types.RoleKeyStatusActive, nil
	case "paused":
		return types.RoleKeyStatusPaused, nil
	case "revoked":
		return types.RoleKeyStatusRevoked, nil
	default:
		return types.RoleKeyStatusUnspecified, fmt.Errorf("status must be active|paused|revoked")
	}
}
