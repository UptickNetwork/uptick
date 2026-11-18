package cli

import (
	"encoding/hex"
	"encoding/json"
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
			if pubHex, _ := cmd.Flags().GetString("publisher-pubkey-hex"); pubHex != "" {
				pk, err := hex.DecodeString(pubHex)
				if err != nil {
					return fmt.Errorf("publisher-pubkey-hex: %w", err)
				}
				msg.PublisherPubkey = pk
			}
			if kid, _ := cmd.Flags().GetString("publisher-key-id"); kid != "" {
				msg.PublisherKeyId = kid
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("metadata-commitment", "", "optional 32-byte metadata commitment (hex)")
	cmd.Flags().String("publisher-pubkey-hex", "", "optional embedded PUBLISHER Ed25519 pubkey (hex)")
	cmd.Flags().String("publisher-key-id", "", "optional embedded publisher key_id (required with pubkey)")
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

const (
	flagFullParams                    = "full-params"
	flagAllowShortenWireProofValidity = "allow-shorten-wire-proof-validity"
)

func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params [params-json-file]",
		Short: "Update module params (authority signer; resume is authority-only via UpdateParams)",
		Long: `By default, keys from params-json-file are deep-merged onto the live on-chain
Params (nested objects like pause_state merge key-by-key; arrays replace).
A field-level diff is printed before the tx is built so resume/pause flips are
visible to the signer. Timing fields show stored and Effective values (stored 0
means the module default).

Shortening max_wire_proof_validity_seconds (Effective) is allowed on-chain but
invalidates already-issued unused proofs (fail-closed). The CLI refuses that
change unless --allow-shorten-wire-proof-validity is set.

When tightening both wire-proof validity and nonce retention in an emergency,
put both keys in one patch (merged Params is validated together), or lower
wire first then nonce. Lowering nonce alone while wire Effective is still
larger fails Validate (nonce_retention must be >= max_wire_proof_validity).

Use --full-params for air-gapped / --generate-only flows: skip the node query
and treat the file as a complete Params object (legacy whole-document semantics).
With --full-params the shorten guard is skipped (no before-state); verify
max_wire_proof_validity_seconds manually against the last known on-chain value.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read params json: %w", err)
			}
			fullParams, err := cmd.Flags().GetBool(flagFullParams)
			if err != nil {
				return err
			}
			allowShorten, err := cmd.Flags().GetBool(flagAllowShortenWireProofValidity)
			if err != nil {
				return err
			}

			var params types.Params
			if fullParams {
				if err := validateFullParamsPauseState(raw); err != nil {
					return err
				}
				if err := clientCtx.Codec.UnmarshalJSON(raw, &params); err != nil {
					return fmt.Errorf("parse full params json: %w", err)
				}
				_, _ = fmt.Fprint(cmd.ErrOrStderr(),
					"params mode: --full-params (no on-chain merge; diff vs chain skipped)\n"+
						"shorten guard skipped; verify max_wire_proof_validity_seconds manually\n")
			} else {
				var before types.Params
				var beforeJSON, afterJSON []byte
				params, before, beforeJSON, afterJSON, err = mergeParamsJSON(cmd, clientCtx, raw)
				if err != nil {
					return err
				}
				beforeNorm, nerr := normalizeParamsJSONForDiff(beforeJSON)
				if nerr != nil {
					return fmt.Errorf("normalize before params: %w", nerr)
				}
				afterNorm, nerr := normalizeParamsJSONForDiff(afterJSON)
				if nerr != nil {
					return fmt.Errorf("normalize after params: %w", nerr)
				}
				lines, derr := paramsFieldDiffLines(beforeNorm, afterNorm, before, params)
				if derr != nil {
					return fmt.Errorf("params diff: %w", derr)
				}
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), formatParamsDiff(lines))
				if wireProofValidityDecreases(before, params) && !allowShorten {
					return fmt.Errorf("%s", formatWireProofShortenError(before, params))
				}
			}

			msg := &types.MsgUpdateParams{
				Authority: clientCtx.GetFromAddress().String(),
				Params:    params,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Bool(flagFullParams, false, "Treat file as a complete Params JSON; skip on-chain query/merge (offline / air-gap)")
	cmd.Flags().Bool(flagAllowShortenWireProofValidity, false, "Allow decreasing Effective max_wire_proof_validity_seconds (invalidates unused proofs)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// mergeParamsJSON loads on-chain Params and deep-merges keys present in patchJSON.
// Returns after/before Params plus before/after codec JSON for diffing.
func mergeParamsJSON(cmd *cobra.Command, clientCtx client.Context, patchJSON []byte) (types.Params, types.Params, []byte, []byte, error) {
	queryClient := types.NewQueryClient(clientCtx)
	res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
	if err != nil {
		return types.Params{}, types.Params{}, nil, nil, fmt.Errorf("query on-chain params: %w", err)
	}
	before := res.Params
	baseJSON, err := clientCtx.Codec.MarshalJSON(&before)
	if err != nil {
		return types.Params{}, types.Params{}, nil, nil, fmt.Errorf("marshal on-chain params: %w", err)
	}
	var base map[string]json.RawMessage
	if err := json.Unmarshal(baseJSON, &base); err != nil {
		return types.Params{}, types.Params{}, nil, nil, fmt.Errorf("decode on-chain params json: %w", err)
	}
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(patchJSON, &patch); err != nil {
		return types.Params{}, types.Params{}, nil, nil, fmt.Errorf("decode params patch json: %w", err)
	}
	if len(patch) == 0 {
		return types.Params{}, types.Params{}, nil, nil, fmt.Errorf("params patch json is empty")
	}
	mergedMap, err := deepMergeJSONObjects(base, patch)
	if err != nil {
		return types.Params{}, types.Params{}, nil, nil, err
	}
	merged, err := json.Marshal(mergedMap)
	if err != nil {
		return types.Params{}, types.Params{}, nil, nil, fmt.Errorf("marshal merged params: %w", err)
	}
	var after types.Params
	if err := clientCtx.Codec.UnmarshalJSON(merged, &after); err != nil {
		return types.Params{}, types.Params{}, nil, nil, fmt.Errorf("parse merged params json: %w", err)
	}
	afterJSON, err := clientCtx.Codec.MarshalJSON(&after)
	if err != nil {
		return types.Params{}, types.Params{}, nil, nil, err
	}
	return after, before, baseJSON, afterJSON, nil
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
