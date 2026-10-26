package cli

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

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
		CmdSetApplicationStatus(),
		CmdSetAttestationSubmitter(),
		CmdSubmitAttestation(),
		CmdChallengeAttestation(),
		CmdResolveChallenge(),
		CmdFinalizeAttestation(),
		CmdGrantGasPayer(),
		CmdRevokeGasPayer(),
		CmdConsumeGasCredit(),
		CmdSetEpochBudget(),
		CmdPause(),
		CmdFundRewardPool(),
		CmdUpdateParams(),
	)
	addV1TxCommands(cmd)
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
			if metaHex, _ := cmd.Flags().GetString("metadata-commitment"); metaHex != "" {
				meta, err := hex.DecodeString(metaHex)
				if err != nil {
					return fmt.Errorf("metadata-commitment: %w", err)
				}
				msg.MetadataCommitment = meta
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("metadata-commitment", "", "optional 32-byte metadata commitment (hex)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSetApplicationStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-application-status [app-id-hex] [status]",
		Short: "Set application status: active|suspended|revoked (authority signer)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("app-id-hex: %w", err)
			}
			status, err := parseApplicationStatus(args[1])
			if err != nil {
				return err
			}
			msg := &types.MsgSetApplicationStatus{
				Authority: clientCtx.GetFromAddress().String(),
				AppId:     appID,
				Status:    status,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSetAttestationSubmitter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-attestation-submitter [app-id-hex] [submitter]",
		Short: "Rotate the attestation submitter (owner signer)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("app-id-hex: %w", err)
			}
			msg := &types.MsgSetAttestationSubmitter{
				Owner:                clientCtx.GetFromAddress().String(),
				AppId:                appID,
				AttestationSubmitter: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSubmitAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-attestation [app-id-hex] [epoch] [attestation-hash-hex] [evidence-root-hex] [result-commitment-hex] [reward-requested]",
		Short: "Submit an attestation commitment (submitter signer)",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("app-id-hex: %w", err)
			}
			epoch, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("epoch: %w", err)
			}
			attHash, err := hex.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("attestation-hash: %w", err)
			}
			evidence, err := hex.DecodeString(args[3])
			if err != nil {
				return fmt.Errorf("evidence-root: %w", err)
			}
			result, err := hex.DecodeString(args[4])
			if err != nil {
				return fmt.Errorf("result-commitment: %w", err)
			}
			profile, _ := cmd.Flags().GetString("profile")
			version, _ := cmd.Flags().GetString("profile-version")
			gasCredit, _ := cmd.Flags().GetString("gas-credit-requested")
			msg := &types.MsgSubmitAttestation{
				Submitter: clientCtx.GetFromAddress().String(),
				Attestation: types.AttestationCommitment{
					AppId:              appID,
					Epoch:              epoch,
					Profile:            profile,
					ProfileVersion:     version,
					AttestationHash:    attHash,
					EvidenceRoot:       evidence,
					ResultCommitment:   result,
					RewardRequested:    args[5],
					GasCreditRequested: gasCredit,
					// ValidateBasic runs validateAttestationFields, which requires
					// attestation.submitter to be a valid bech32; the keeper overwrites
					// it from msg.submitter at execution time.
					Submitter: clientCtx.GetFromAddress().String(),
				},
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("profile", types.DefaultUGEProfile, "attestation profile")
	cmd.Flags().String("profile-version", types.DefaultUGEProfileVersion, "attestation profile version")
	cmd.Flags().String("gas-credit-requested", "0", "gas credit requested (pure-credit when reward is 0)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdChallengeAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-attestation [app-id-hex] [epoch] [evidence-hash-hex]",
		Short: "Challenge an attestation (bonds challenger funds)",
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
			epoch, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("epoch: %w", err)
			}
			evidence, err := hex.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("evidence-hash: %w", err)
			}
			msg := &types.MsgChallengeAttestation{
				Challenger:   clientCtx.GetFromAddress().String(),
				AppId:        appID,
				Epoch:        epoch,
				EvidenceHash: evidence,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdResolveChallenge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-challenge [challenge-id] [uphold] [resolution-hash-hex]",
		Short: "Resolve a challenge (authority signer); uphold=true|false",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("challenge-id: %w", err)
			}
			uphold, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("uphold: %w", err)
			}
			resolution, err := hex.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("resolution-hash: %w", err)
			}
			msg := &types.MsgResolveChallenge{
				Authority:      clientCtx.GetFromAddress().String(),
				ChallengeId:    id,
				Uphold:         uphold,
				ResolutionHash: resolution,
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

func CmdGrantGasPayer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant-gas-payer [app-id-hex] [gas-payer]",
		Short: "Grant a gas payer for an application (owner signer)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("app-id-hex: %w", err)
			}
			msg := &types.MsgGrantGasPayer{
				Owner:    clientCtx.GetFromAddress().String(),
				AppId:    appID,
				GasPayer: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRevokeGasPayer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-gas-payer [app-id-hex] [gas-payer]",
		Short: "Revoke a gas payer (owner signer)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			appID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("app-id-hex: %w", err)
			}
			msg := &types.MsgRevokeGasPayer{
				Owner:    clientCtx.GetFromAddress().String(),
				AppId:    appID,
				GasPayer: args[1],
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
		Short: "Manual consume (requires allow_manual_gas_credit_consume; prefer PostHandler)",
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

func CmdSetEpochBudget() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-epoch-budget [epoch] [denom] [limit]",
		Short: "Set or update an epoch budget (authority signer)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			epoch, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("epoch: %w", err)
			}
			closed, _ := cmd.Flags().GetBool("closed")
			msg := &types.MsgSetEpochBudget{
				Authority: clientCtx.GetFromAddress().String(),
				Budget: types.EpochBudget{
					Epoch:  epoch,
					Denom:  args[1],
					Limit:  args[2],
					Spent:  "0",
					Closed: closed,
				},
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Bool("closed", false, "mark the epoch budget closed")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdPause() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause [bits]",
		Short: "OR pause bits to true (pause_authority). bits=registration,attestation,challenge,settlement,gas_credit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			bits, err := parsePauseBits(args[0])
			if err != nil {
				return err
			}
			msg := &types.MsgPause{
				PauseAuthority: clientCtx.GetFromAddress().String(),
				PauseBits:      bits,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params [params-json-file]",
		Short: "Update module params from a full Params JSON file (authority signer; resume is authority-only via UpdateParams)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read params json: %w", err)
			}
			var params types.Params
			if err := clientCtx.Codec.UnmarshalJSON(raw, &params); err != nil {
				return fmt.Errorf("parse params json: %w", err)
			}
			msg := &types.MsgUpdateParams{
				Authority: clientCtx.GetFromAddress().String(),
				Params:    params,
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
		Short: "Fund the appmining reward pool (any allowlisted-denom holder)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgFundRewardPool{
				Funder: clientCtx.GetFromAddress().String(),
				Denom:  args[0],
				Amount: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func parseApplicationStatus(s string) (types.ApplicationStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "active":
		return types.ApplicationStatusActive, nil
	case "suspended":
		return types.ApplicationStatusSuspended, nil
	case "revoked":
		return types.ApplicationStatusRevoked, nil
	default:
		return types.ApplicationStatusUnspecified, fmt.Errorf("status must be active|suspended|revoked")
	}
}

func parsePauseBits(s string) (types.PauseState, error) {
	var bits types.PauseState
	for _, part := range strings.Split(s, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "":
			continue
		case "registration":
			bits.RegistrationPaused = true
		case "attestation":
			bits.AttestationPaused = true
		case "challenge":
			bits.ChallengePaused = true
		case "settlement":
			bits.SettlementPaused = true
		case "gas_credit", "gas-credit":
			bits.GasCreditPaused = true
		default:
			return bits, fmt.Errorf("unknown pause bit %q", part)
		}
	}
	if !types.PauseBitsHasAnyTrue(bits) {
		return bits, fmt.Errorf("at least one pause bit required")
	}
	return bits, nil
}
